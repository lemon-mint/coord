package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/internal/llmutils"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var _ llm.Model = (*openAIModel)(nil)

var (
	errEmptyContent   error = errors.New("convertContentCoord2OpenAI: empty content")
	errInvalidContent error = errors.New("convertContentCoord2OpenAI: invalid content")
)

func convertContentCoord2OpenAI(dst []openai.ChatCompletionMessage, content *llm.Content) ([]openai.ChatCompletionMessage, error) {
	if content == nil {
		return dst, errEmptyContent
	}

	coordContent := content

	var role string
	switch content.Role {
	case llm.RoleUser:
		role = openai.ChatMessageRoleUser
	case llm.RoleModel:
		role = openai.ChatMessageRoleAssistant
	case llm.RoleFunc:
		role = openai.ChatMessageRoleTool
	default:
		return dst, errInvalidContent
	}

	if len(coordContent.Parts) == 0 {
		return dst, errEmptyContent
	}

	if len(coordContent.Parts) == 1 {
		switch p := coordContent.Parts[0].(type) {
		case llm.Text:
			dst = append(dst, openai.ChatCompletionMessage{
				Role:    role,
				Content: string(p),
			})
			return dst, nil
		}
	}

	var msg openai.ChatCompletionMessage
	const (
		stateTypeClean  = 0
		stateTypeClient = 1
		stateTypeServer = 2
	)
	state := stateTypeClean

	flush := func() {
		if state == stateTypeClean {
			return
		}
		dst = append(dst, msg)
		msg = openai.ChatCompletionMessage{}
		state = stateTypeClean
	}

	for _, seg := range coordContent.Parts {
		switch p := seg.(type) {
		case llm.Text:
			switch coordContent.Role {
			case llm.RoleUser:
				if state != stateTypeClean && state != stateTypeClient {
					flush()
				}
				state = stateTypeClient
			case llm.RoleModel:
				if state != stateTypeClean && state != stateTypeServer {
					flush()
				}
				state = stateTypeServer
			default:
				return dst, errInvalidContent
			}

			msg.Role = role
			msg.MultiContent = append(msg.MultiContent, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeText,
				Text: string(p),
			})
		case *llm.InlineData:
			switch coordContent.Role {
			case llm.RoleUser:
				if state != stateTypeClean && state != stateTypeClient {
					flush()
				}
				state = stateTypeClient
			default:
				return dst, errInvalidContent
			}

			msg.Role = role
			msg.MultiContent = append(msg.MultiContent, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				Text: "data:" + p.MIMEType + ";base64," + base64.URLEncoding.EncodeToString(p.Data),
			})
		case *llm.FileData:
			switch coordContent.Role {
			case llm.RoleUser:
				if state != stateTypeClean && state != stateTypeClient {
					flush()
				}
				state = stateTypeClient
			default:
				return dst, errInvalidContent
			}

			msg.Role = role
			msg.MultiContent = append(msg.MultiContent, openai.ChatMessagePart{
				Type: openai.ChatMessagePartTypeImageURL,
				Text: p.FileURI,
			})
		case *llm.FunctionCall:
			switch coordContent.Role {
			case llm.RoleModel:
				if state != stateTypeClean && state != stateTypeServer {
					flush()
				}
				state = stateTypeServer
			default:
				return dst, errInvalidContent
			}

			jsonData, err := json.Marshal(p.Args)
			if err != nil {
				jsonData = []byte("{\"error\": \"RPCError: Failed to marshal the args (HTTP 500)\"}")
			}

			msg.Role = role
			msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
				Type: openai.ToolTypeFunction,
				ID:   p.ID,
				Function: openai.FunctionCall{
					Name:      p.Name,
					Arguments: string(jsonData),
				},
			})
		case *llm.FunctionResponse:
			switch coordContent.Role {
			case llm.RoleFunc:
				if state != stateTypeClean {
					flush()
				}
				state = stateTypeClient
			default:
				return dst, errInvalidContent
			}

			jsonData, err := json.Marshal(p.Content)
			if err != nil {
				jsonData = []byte("{\"error\": \"RPCError: Failed to serialize response (HTTP 500)\"}")
			}

			msg.Role = role
			msg.ToolCallID = p.ID
			msg.Content = string(jsonData)
		}
	}

	flush()

	return dst, nil
}

func convertContextCoord2OpenAI(ctx *llm.ChatContext, prompt ...*llm.Content) ([]openai.ChatCompletionMessage, error) {
	var dst []openai.ChatCompletionMessage
	var err error

	if ctx != nil {
		for _, c := range ctx.Contents {
			dst, err = convertContentCoord2OpenAI(dst, c)
			if err != nil {
				return dst, err
			}
		}
	}

	for _, p := range prompt {
		dst, err = convertContentCoord2OpenAI(dst, p)
		if err != nil {
			return dst, err
		}
	}

	return dst, nil
}

func convertFunctionCoord2OpenAI(f *llm.FunctionDeclaration) *openai.FunctionDefinition {
	return &openai.FunctionDefinition{
		Name:        f.Name,
		Description: f.Description,
		Parameters:  convertSchemaCoord2OpenAI(f.Schema, nil),
	}
}

func convTypeCoord2OpenAI(t llm.OpenAPIType) jsonschema.DataType {
	switch t {
	case llm.OpenAPITypeString:
		return jsonschema.String
	case llm.OpenAPITypeNumber:
		return jsonschema.Number
	case llm.OpenAPITypeInteger:
		return jsonschema.Integer
	case llm.OpenAPITypeBoolean:
		return jsonschema.Boolean
	case llm.OpenAPITypeArray:
		return jsonschema.Array
	case llm.OpenAPITypeObject:
		return jsonschema.Object
	}

	return jsonschema.Null
}

func convertSchemaCoord2OpenAI(s *llm.Schema, cache map[*llm.Schema]*jsonschema.Definition) *jsonschema.Definition {
	if s == nil {
		return nil
	}

	if cache == nil {
		cache = make(map[*llm.Schema]*jsonschema.Definition)
	}

	if c, ok := cache[s]; ok {
		return c
	}

	schema := &jsonschema.Definition{
		Type:        convTypeCoord2OpenAI(s.Type),
		Description: s.Description,
	}
	cache[s] = schema

	switch s.Type {
	case llm.OpenAPITypeString:
		schema.Enum = make([]string, 0, len(s.Enum))
		for i := range s.Enum {
			if v, ok := s.Enum[i].(string); ok {
				schema.Enum = append(schema.Enum, v)
			}
		}
	case llm.OpenAPITypeNumber:
	case llm.OpenAPITypeInteger:
	case llm.OpenAPITypeBoolean:
	case llm.OpenAPITypeArray:
		schema.Items = convertSchemaCoord2OpenAI(s.Items, cache)
	case llm.OpenAPITypeObject:
		schema.Properties = make(map[string]jsonschema.Definition, len(s.Properties))
		for k, v := range s.Properties {
			schema.Properties[k] = *convertSchemaCoord2OpenAI(v, cache)
		}
		schema.Required = s.Required
	}

	return schema
}

type streamingOpenAI2CoordConverter struct {
	content   *llm.StreamContent
	streamOut chan llm.Segment

	pendingToolCalls map[int]openai.ToolCall
}

func (g *streamingOpenAI2CoordConverter) feed(ctx context.Context, chat openai.ChatCompletionStreamChoiceDelta) error {
	var role llm.Role
	switch chat.Role {
	case openai.ChatMessageRoleUser:
		g.content.Content.Role = llm.RoleUser
	case openai.ChatMessageRoleAssistant:
		g.content.Content.Role = llm.RoleModel
	case openai.ChatMessageRoleTool:
		g.content.Content.Role = role
	}

	if chat.Content != "" {
		seg := llm.Text(chat.Content)
		g.content.Content.Parts = append(g.content.Content.Parts, seg)
		select {
		case g.streamOut <- seg:
		case <-ctx.Done():
			return ctx.Err()
		}
	} else if len(chat.ToolCalls) > 0 {
		for _, p := range chat.ToolCalls {
			if g.pendingToolCalls == nil {
				g.pendingToolCalls = make(map[int]openai.ToolCall)
			}

			if p.Index == nil {
				// standalone tool call
				seg := &llm.FunctionCall{
					ID:   p.ID,
					Name: p.Function.Name,
				}

				err := json.Unmarshal([]byte(p.Function.Arguments), &seg.Args)
				if err != nil {
					return err
				}

				select {
				case g.streamOut <- seg:
				case <-ctx.Done():
					return ctx.Err()
				}
			} else {
				prev, ok := g.pendingToolCalls[*p.Index]
				if !ok {
					prev = p
					g.pendingToolCalls[*p.Index] = prev
				} else {
					prev.Function.Arguments += p.Function.Arguments
					g.pendingToolCalls[*p.Index] = prev
				}

				// check if it is parsable as JSON
				var args map[string]any
				err := json.Unmarshal([]byte(prev.Function.Arguments), &args)
				if err == nil {
					// dispatch tool call
					seg := &llm.FunctionCall{
						ID:   prev.ID,
						Name: prev.Function.Name,
						Args: args,
					}

					g.content.Content.Parts = append(g.content.Content.Parts, seg)
					select {
					case g.streamOut <- seg:
					case <-ctx.Done():
						return ctx.Err()
					}
				}
			}
			continue
		}
	}

	return nil
}

func (g *openAIModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	contents, err := convertContextCoord2OpenAI(chat, input)
	if err != nil {
		stream := make(chan llm.Segment)
		close(stream)
		v := &llm.StreamContent{
			Content: &llm.Content{},
			Stream:  stream,
			Err:     err,
		}
		return v
	}

	var otools []openai.Tool
	if chat != nil {
		otools = make([]openai.Tool, len(chat.Tools))
		for i := range chat.Tools {
			otools[i] = openai.Tool{
				Type:     openai.ToolTypeFunction,
				Function: convertFunctionCoord2OpenAI(chat.Tools[i]),
			}
		}
	}

	if g.config.SystemInstruction != "" {
		contents = append([]openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleSystem,
			Content: g.config.SystemInstruction,
		}}, contents...)
	}

	if chat != nil {
		if chat.SystemInstruction != "" {
			contents = append([]openai.ChatCompletionMessage{{
				Role:    openai.ChatMessageRoleSystem,
				Content: chat.SystemInstruction,
			}}, contents...)
		}
	}

	model_request := openai.ChatCompletionRequest{
		Model:    g.model,
		Messages: contents,
		Tools:    otools,
		Stop:     g.config.StopSequences,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}

	if g.config.MaxOutputTokens == nil || *g.config.MaxOutputTokens <= 0 {
		model_request.MaxTokens = 2048
	} else {
		model_request.MaxTokens = *g.config.MaxOutputTokens
	}

	if g.config.Temperature != nil {
		model_request.Temperature = *g.config.Temperature
	}

	if g.config.TopP != nil {
		model_request.TopP = *g.config.TopP
	}

	iter, err := g.client.CreateChatCompletionStream(ctx, model_request)
	if err != nil {
		ch := make(chan llm.Segment)
		close(ch)
		v := &llm.StreamContent{Err: err, Content: &llm.Content{}, Stream: ch}
		return v
	}

	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Content: &llm.Content{},
		Stream:  stream,
	}

	converter := &streamingOpenAI2CoordConverter{
		content:   v,
		streamOut: stream,
	}

	go func() {
		defer close(stream)
		defer func() {
			v.Content.Parts = llmutils.Normalize(v.Content.Parts)
		}()

		for {
			resp, err := iter.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}

				v.Err = err
				return
			}

			if resp.Usage != nil {
				if v.UsageData == nil {
					v.UsageData = new(llm.UsageData)
				}

				v.UsageData.InputTokens += resp.Usage.PromptTokens
				v.UsageData.OutputTokens += resp.Usage.CompletionTokens
				v.UsageData.TotalTokens += resp.Usage.TotalTokens
			}

			if len(resp.Choices) > 0 {
				if resp.Choices[0].FinishReason != "" {
					switch resp.Choices[0].FinishReason {
					case openai.FinishReasonNull:
						v.FinishReason = llm.FinishReasonUnknown
					case openai.FinishReasonLength:
						v.FinishReason = llm.FinishReasonMaxTokens
					case openai.FinishReasonFunctionCall, openai.FinishReasonToolCalls:
						v.FinishReason = llm.FinishReasonToolUse
					case openai.FinishReasonContentFilter:
						v.FinishReason = llm.FinishReasonSafety
					case openai.FinishReasonStop:
						v.FinishReason = llm.FinishReasonStop
					default:
						v.FinishReason = llm.FinishReasonUnknown
					}
				}

				err := converter.feed(ctx, resp.Choices[0].Delta)
				if err != nil {
					v.Err = err
					return
				}
			} else {
				v.Err = llm.ErrNoResponse
				return
			}
		}
	}()

	return v
}

func ptrify[T any](v T) *T {
	return &v
}

var defaultOpenAILLMConfig = &llm.Config{
	MaxOutputTokens:       ptrify(2048),
	SafetyFilterThreshold: llm.BlockLowAndAbove,
}

type openAIModel struct {
	client *openai.Client
	config *llm.Config
	model  string
}

func (o *openAIModel) Name() string {
	return o.model
}

func (g *openAIModel) Close() error {
	return nil
}

var _ provider.LLMClient = (*openAIClient)(nil)

func (g *openAIClient) NewLLM(model string, config *llm.Config) (llm.Model, error) {
	if config == nil {
		config = defaultOpenAILLMConfig
	}

	var _vm = &openAIModel{
		client: g.client,
		config: config,
		model:  model,
	}

	return _vm, nil
}

var _ provider.LLMProvider = Provider

type OpenAIProvider struct {
}

func (OpenAIProvider) NewLLMClient(ctx context.Context, configs ...pconf.Config) (provider.LLMClient, error) {
	return newClient(configs...)
}

const ProviderName = "openai"

var Provider OpenAIProvider

func init() {
	var exists bool
	for _, n := range coord.ListLLMProviders() {
		if n == ProviderName {
			exists = true
			break
		}
	}
	if !exists {
		coord.RegisterLLMProvider(ProviderName, Provider)
	}
}

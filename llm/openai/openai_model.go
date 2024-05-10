package models

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"

	"github.com/lemon-mint/vermittlungsstelle/llm"

	"github.com/sashabaranov/go-openai"
	"github.com/sashabaranov/go-openai/jsonschema"
)

var _ llm.LLM = (*OpenAIModel)(nil)

func convertContextOpenAI(chat *llm.ChatContext) []openai.ChatCompletionMessage {
	contents := make([]openai.ChatCompletionMessage, 0, len(chat.Contents)+1)

L0:
	for i := range chat.Contents {
		var role string
		switch chat.Contents[i].Role {
		case llm.RoleUser:
			role = openai.ChatMessageRoleUser
		case llm.RoleModel:
			role = openai.ChatMessageRoleAssistant
		case llm.RoleFunc:
			role = openai.ChatMessageRoleTool
		default:
			role = openai.ChatMessageRoleUser
		}

		if len(chat.Contents[i].Parts) == 0 {
			continue
		}

		if len(chat.Contents[i].Parts) == 1 {
			switch p := chat.Contents[i].Parts[0].(type) {
			case llm.Text:
				contents = append(contents, openai.ChatCompletionMessage{
					Role:    role,
					Content: string(p),
				})
				continue L0
			}
		}

	L1:
		for _, seg := range chat.Contents[i].Parts {
			var msg openai.ChatCompletionMessage
			msg.Role = role

			switch p := seg.(type) {
			case llm.Text:
				msg.MultiContent = append(msg.MultiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeText,
					Text: string(p),
				})
			case *llm.InlineData:
				msg.MultiContent = append(msg.MultiContent, openai.ChatMessagePart{
					Type: openai.ChatMessagePartTypeImageURL,
					Text: "data:" + p.MIMEType + ";base64," + base64.URLEncoding.EncodeToString(p.Data),
				})
			case *llm.FunctionCall:
				jdata, err := json.Marshal(p.Args)
				if err != nil {
					jdata = []byte("{\"error\": \"RPCError: Failed to marshal the args (HTTP 500)\"}")
				}

				msg.ToolCalls = append(msg.ToolCalls, openai.ToolCall{
					Type: openai.ToolTypeFunction,
					ID:   p.ID,
					Function: openai.FunctionCall{
						Name:      p.Name,
						Arguments: string(jdata),
					},
				})
			case *llm.FunctionResponse:
				jdata, err := json.Marshal(p.Content)
				if err != nil {
					jdata = []byte("{\"error\": \"RPCError: Failed to serialize response (HTTP 500)\"}")
				}

				contents = append(contents, openai.ChatCompletionMessage{
					Role:       role,
					Name:       p.Name,
					Content:    string(jdata),
					ToolCallID: p.ID,
				})
				continue L1
			}

			contents = append(contents, msg)
		}
	}

	return contents
}

func convertFunctionDeclarationOpenAI(f *llm.FunctionDeclaration) *openai.FunctionDefinition {
	return &openai.FunctionDefinition{
		Name:        f.Name,
		Description: f.Description,
		Parameters:  convertSchemaOpenAI(f.Schema, nil),
	}
}

func convTypeOpenAI(t llm.OpenAPIType) jsonschema.DataType {
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

func convertSchemaOpenAI(s *llm.Schema, cache map[*llm.Schema]*jsonschema.Definition) *jsonschema.Definition {
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
		Type:        convTypeOpenAI(s.Type),
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
		schema.Items = convertSchemaOpenAI(s.Items, cache)
	case llm.OpenAPITypeObject:
		schema.Properties = make(map[string]jsonschema.Definition, len(s.Properties))
		for k, v := range s.Properties {
			schema.Properties[k] = *convertSchemaOpenAI(v, cache)
		}
		schema.Required = s.Required
	}

	return schema
}

func convertOpenAIContent(chat openai.ChatCompletionStreamChoiceDelta) (*llm.Content, error) {
	var role llm.Role
	switch chat.Role {
	case openai.ChatMessageRoleUser:
		role = llm.RoleUser
	case openai.ChatMessageRoleAssistant:
		role = llm.RoleModel
	case openai.ChatMessageRoleTool:
		role = llm.RoleFunc
	default:
		role = llm.RoleUser
	}

	var parts []llm.Segment

	if chat.Content != "" {
		parts = append(parts, llm.Text(chat.Content))
	} else if len(chat.ToolCalls) > 0 {
		for _, p := range chat.ToolCalls {
			var args map[string]interface{}
			err := json.Unmarshal([]byte(p.Function.Arguments), &args)
			if err != nil {
				return nil, err
			}

			parts = append(parts, &llm.FunctionCall{
				Name: p.Function.Name,
				ID:   p.ID,
				Args: args,
			})
		}
	}

	return &llm.Content{
		Role:  role,
		Parts: parts,
	}, nil
}

func (g *OpenAIModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	if chat == nil {
		chat = &llm.ChatContext{}
	}

	chat.Contents = append(chat.Contents, input)
	contents := convertContextOpenAI(chat)
	chat.Contents[len(chat.Contents)-1] = nil
	chat.Contents = chat.Contents[:len(chat.Contents)-1]
	var otools []openai.Tool = make([]openai.Tool, len(chat.Tools))
	for i := range chat.Tools {
		otools[i] = openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: convertFunctionDeclarationOpenAI(chat.Tools[i]),
		}
	}

	if g.config.SystemInstruction != "" {
		contents = append([]openai.ChatCompletionMessage{{
			Role:    openai.ChatMessageRoleSystem,
			Content: g.config.SystemInstruction,
		}}, contents...)
	}

	model_request := openai.ChatCompletionRequest{
		Model:    g.model,
		Messages: contents,
		Tools:    otools,
		Stop:     g.config.StopSequences,
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

	go func() {
		defer close(stream)
		for {
			resp, err := iter.Recv()
			if err != nil {
				if err == io.EOF {
					return
				}

				v.Err = err
				return
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

				data, err := convertOpenAIContent(resp.Choices[0].Delta)
				if err != nil {
					v.Err = err
					continue
				}
				v.Content.Role = data.Role
				for i := range data.Parts {
					if text, ok := (data.Parts[i]).(llm.Text); !ok || len(v.Content.Parts) == 0 {
						v.Content.Parts = append(v.Content.Parts, data.Parts[i])
					} else {
						if _, ok := v.Content.Parts[len(v.Content.Parts)-1].(llm.Text); ok {
							v.Content.Parts[len(v.Content.Parts)-1] = v.Content.Parts[len(v.Content.Parts)-1].(llm.Text) + llm.Text(text)
						} else {
							v.Content.Parts = append(v.Content.Parts, data.Parts[i])
						}
					}
				}

				for i := range data.Parts {
					select {
					case stream <- data.Parts[i]:
					case <-ctx.Done():
						return
					}
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

var defaultOpenAIConfig = &llm.Config{
	MaxOutputTokens:       ptrify(2048),
	SafetyFilterThreshold: llm.BlockLowAndAbove,
}

type OpenAIModel struct {
	client *openai.Client
	config *llm.Config
	model  string
}

func (o *OpenAIModel) Name() string {
	return o.model
}

func (g *OpenAIModel) Close() error {
	return nil
}

func NewOpenAIModel(client *openai.Client, model string, config *llm.Config) *OpenAIModel {
	if config == nil {
		config = defaultOpenAIConfig
	}

	return &OpenAIModel{
		client: client,
		model:  model,
		config: config,
	}
}

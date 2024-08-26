package aistudio

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/internal/callid"
	"github.com/lemon-mint/coord/internal/llmutils"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"

	"github.com/google/generative-ai-go/genai"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var _ llm.Model = (*generativeLanguageModel)(nil)

func convTypeGenerativeLanguage(t llm.OpenAPIType) genai.Type {
	switch t {
	case llm.OpenAPITypeString:
		return genai.TypeString
	case llm.OpenAPITypeNumber:
		return genai.TypeNumber
	case llm.OpenAPITypeInteger:
		return genai.TypeInteger
	case llm.OpenAPITypeBoolean:
		return genai.TypeBoolean
	case llm.OpenAPITypeArray:
		return genai.TypeArray
	case llm.OpenAPITypeObject:
		return genai.TypeObject
	}

	return genai.TypeUnspecified
}

func convertSchemaGenerativeLanguage(s *llm.Schema, cache map[*llm.Schema]*genai.Schema) *genai.Schema {
	if s == nil {
		return nil
	}

	if cache == nil {
		cache = make(map[*llm.Schema]*genai.Schema)
	}

	if c, ok := cache[s]; ok {
		return c
	}

	schema := &genai.Schema{
		Type:        convTypeGenerativeLanguage(s.Type),
		Description: s.Description,
		Nullable:    s.Nullable,
		Format:      s.Format,
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
		schema.Items = convertSchemaGenerativeLanguage(s.Items, cache)
	case llm.OpenAPITypeObject:
		schema.Properties = make(map[string]*genai.Schema, len(s.Properties))
		for k, v := range s.Properties {
			schema.Properties[k] = convertSchemaGenerativeLanguage(v, cache)
		}
		schema.Required = s.Required
	}

	return schema
}

func convertFunctionDeclarationGenerativeLanguage(f *llm.FunctionDeclaration) *genai.FunctionDeclaration {
	decl := genai.FunctionDeclaration{
		Name:        f.Name,
		Description: f.Description,
		Parameters:  convertSchemaGenerativeLanguage(f.Schema, nil),
	}

	return &decl
}

func convertContentGenerativeLanguage(s *llm.Content) *genai.Content {
	content := &genai.Content{
		Role: string(s.Role),
	}

	for i := range s.Parts {
		switch p := s.Parts[i].(type) {
		case llm.Text:
			content.Parts = append(content.Parts, genai.Text(p))
		case *llm.InlineData:
			content.Parts = append(content.Parts, genai.Blob{
				MIMEType: p.MIMEType,
				Data:     p.Data,
			})
		case *llm.FileData:
			content.Parts = append(content.Parts, genai.FileData{
				MIMEType: p.MIMEType,
				URI:      p.FileURI,
			})
		case *llm.FunctionCall:
			content.Parts = append(content.Parts, genai.FunctionCall{
				Name: p.Name,
				Args: p.Args,
			})
		case *llm.FunctionResponse:
			jsond, err := json.Marshal(p.Content)
			if err != nil {
				jsond = []byte("{\"error\": \"RPCError: Failed to serialize response (HTTP 500)\"}")
			}

			var data map[string]interface{}
			if err := json.Unmarshal(jsond, &data); err != nil {
				data = map[string]interface{}{
					"error": err.Error(),
				}
			}

			content.Parts = append(content.Parts, genai.FunctionResponse{
				Name:     p.Name,
				Response: data,
			})
		}
	}

	return content
}

func convertContextGenerativeLanguage(c *llm.ChatContext) []*genai.Content {
	var contents []*genai.Content = make([]*genai.Content, len(c.Contents))

	for i := range c.Contents {
		contents[i] = convertContentGenerativeLanguage(c.Contents[i])
	}

	return contents
}

func convertGenerativeLanguageContent(c *genai.Content) *llm.Content {
	lc := &llm.Content{
		Role:  llm.Role(c.Role),
		Parts: make([]llm.Segment, len(c.Parts)),
	}

	for i := range c.Parts {
		switch p := c.Parts[i].(type) {
		case genai.Text:
			lc.Parts[i] = llm.Text(p)
		case genai.Blob:
			lc.Parts[i] = &llm.InlineData{
				MIMEType: p.MIMEType,
				Data:     p.Data,
			}
		case genai.FunctionCall:
			lc.Parts[i] = &llm.FunctionCall{
				Name: p.Name,
				ID:   callid.OpenAICallID(),
				Args: p.Args,
			}
		case genai.FunctionResponse:
			lc.Parts[i] = &llm.FunctionResponse{
				Name:    p.Name,
				ID:      callid.OpenAICallID(),
				Content: p.Response,
			}
		}
	}

	return lc
}

func convertGenerativeLanguageFinishReason(stop_reason genai.FinishReason) llm.FinishReason {
	switch stop_reason {
	case genai.FinishReasonStop:
		return llm.FinishReasonStop
	case genai.FinishReasonMaxTokens:
		return llm.FinishReasonMaxTokens
	case genai.FinishReasonSafety:
		return llm.FinishReasonSafety
	case genai.FinishReasonRecitation:
		return llm.FinishReasonRecitation
	case genai.FinishReasonOther:
		return llm.FinishReasonUnknown
	}

	return llm.FinishReasonUnknown
}

func (g *generativeLanguageModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	if chat == nil {
		chat = &llm.ChatContext{}
	}

	contents := convertContextGenerativeLanguage(chat)
	tools := make([]*genai.FunctionDeclaration, len(chat.Tools))
	for i := range chat.Tools {
		tools[i] = convertFunctionDeclarationGenerativeLanguage(chat.Tools[i])
	}

	model := g.client.GenerativeModel(g.model)

	switch g.config.SafetyFilterThreshold {
	case llm.BlockNone:
		model.SafetySettings = []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockNone,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockNone,
			},
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockNone,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockNone,
			},
		}
	case llm.BlockDefault, llm.BlockLowAndAbove:
		model.SafetySettings = []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockLowAndAbove,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockLowAndAbove,
			},
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockLowAndAbove,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockLowAndAbove,
			},
		}
	case llm.BlockMediumAndAbove:
		model.SafetySettings = []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockMediumAndAbove,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockMediumAndAbove,
			},
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockMediumAndAbove,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockMediumAndAbove,
			},
		}
	case llm.BlockOnlyHigh:
		model.SafetySettings = []*genai.SafetySetting{
			{
				Category:  genai.HarmCategoryHateSpeech,
				Threshold: genai.HarmBlockOnlyHigh,
			},
			{
				Category:  genai.HarmCategoryDangerousContent,
				Threshold: genai.HarmBlockOnlyHigh,
			},
			{
				Category:  genai.HarmCategoryHarassment,
				Threshold: genai.HarmBlockOnlyHigh,
			},
			{
				Category:  genai.HarmCategorySexuallyExplicit,
				Threshold: genai.HarmBlockOnlyHigh,
			},
		}
	}

	if g.config.Temperature != nil {
		model.SetTemperature(*g.config.Temperature)
	}

	if g.config.TopK != nil {
		model.SetTopK(int32(*g.config.TopK))
	}

	if g.config.TopP != nil {
		model.SetTopP(*g.config.TopP)
	}

	if g.config.MaxOutputTokens != nil {
		model.SetMaxOutputTokens(int32(*g.config.MaxOutputTokens))
	}

	model.StopSequences = g.config.StopSequences

	if g.config.SystemInstruction != "" || chat.SystemInstruction != "" {
		model.SystemInstruction = &genai.Content{Parts: []genai.Part{genai.Text(g.config.SystemInstruction + chat.SystemInstruction)}}
	}

	session := model.StartChat()
	session.History = contents

	if len(tools) > 0 {
		model.Tools = []*genai.Tool{
			{
				FunctionDeclarations: tools,
			},
		}
	} else {
		model.Tools = nil
	}

	content := convertContentGenerativeLanguage(input)
	resp := session.SendMessageStream(
		ctx,
		content.Parts...,
	)

	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Content: &llm.Content{},
		Stream:  stream,
	}

	go func() {
		defer close(stream)
		defer func() {
			v.Content.Parts = llmutils.Normalize(v.Content.Parts)
			if v.FinishReason == llm.FinishReasonStop {
				for i := range v.Content.Parts {
					if v.Content.Parts[i].Type() == llm.SegmentTypeFunctionCall {
						v.FinishReason = llm.FinishReasonToolUse
						break
					}
				}
			}
		}()

		for {
			resp, err := resp.Next()
			if err == iterator.Done {
				return
			}
			if err != nil {
				v.Err = err
				return
			}

			if resp.UsageMetadata != nil {
				v.UsageData = &llm.UsageData{
					InputTokens:  int(resp.UsageMetadata.PromptTokenCount),
					OutputTokens: int(resp.UsageMetadata.CandidatesTokenCount),
					TotalTokens:  int(resp.UsageMetadata.TotalTokenCount),
				}
			}

			if len(resp.Candidates) > 0 {
				if resp.Candidates[0].Content == nil {
					v.FinishReason = convertGenerativeLanguageFinishReason(resp.Candidates[0].FinishReason)
					v.Err = llm.ErrNoResponse
					continue
				}

				if resp.Candidates[0].FinishReason != genai.FinishReasonUnspecified {
					v.FinishReason = convertGenerativeLanguageFinishReason(resp.Candidates[0].FinishReason)
				}

				data := convertGenerativeLanguageContent(resp.Candidates[0].Content)
				if v.Content.Role == "" {
					v.Content.Role = data.Role
				}
				v.Content.Parts = append(v.Content.Parts, data.Parts...)
				for i := range data.Parts {
					select {
					case stream <- data.Parts[i]:
					case <-ctx.Done():
						return
					}
				}
			} else {
				v.FinishReason = llm.FinishReasonUnknown
				if resp.PromptFeedback != nil {
					switch resp.PromptFeedback.BlockReason {
					case genai.BlockReasonOther,
						genai.BlockReasonUnspecified:
						v.FinishReason = llm.FinishReasonUnknown
					case genai.BlockReasonSafety:
						v.FinishReason = llm.FinishReasonSafety
					}
				}
				continue
			}

			select {
			case <-ctx.Done(): // context canceled
				return
			default:
			}
		}
	}()

	return v
}

func (g *generativeLanguageModel) Name() string {
	return g.model
}

func (g *generativeLanguageModel) Close() error {
	return nil
}

func ptrify[T any](v T) *T {
	return &v
}

var defaultGenerativeLanguageConfig = &llm.Config{
	Temperature:           ptrify(float32(0.4)),
	MaxOutputTokens:       ptrify(2048),
	SafetyFilterThreshold: llm.BlockOnlyHigh,
}

type generativeLanguageModel struct {
	client *genai.Client
	config *llm.Config
	model  string
}

var _ provider.LLMClient = (*aiStudioClient)(nil)

type aiStudioClient struct {
	client *genai.Client
}

func (g *aiStudioClient) Close() error {
	return g.client.Close()
}

func (g *aiStudioClient) NewLLM(model string, config *llm.Config) (llm.Model, error) {
	if config == nil {
		config = defaultGenerativeLanguageConfig
	}

	var _vm = &generativeLanguageModel{
		client: g.client,
		config: config,
		model:  model,
	}

	return _vm, nil
}

var _ provider.LLMProvider = Provider

type AIStudioProvider struct {
}

var (
	ErrAPIKeyRequired error = errors.New("api key is required")
)

func (AIStudioProvider) newAIStudioClient(ctx context.Context, configs ...pconf.Config) (*aiStudioClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	apiKey := client_config.APIKey
	client_options := client_config.GoogleClientOptions

	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}
	client_options = append(client_options, option.WithAPIKey(apiKey))

	genaiClient, err := genai.NewClient(ctx, client_options...)
	if err != nil {
		return nil, err
	}

	return &aiStudioClient{
		client: genaiClient,
	}, nil
}

func (AIStudioProvider) NewLLMClient(ctx context.Context, configs ...pconf.Config) (provider.LLMClient, error) {
	return (AIStudioProvider).newAIStudioClient(AIStudioProvider{}, ctx, configs...)
}

const ProviderName = "aistudio"

var Provider AIStudioProvider

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

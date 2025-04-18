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

	"google.golang.org/api/iterator"
	"google.golang.org/genai"
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

	nullable := (*bool)(nil)
	if s.Nullable {
		nullable = ptrify(true)
	}

	schema := &genai.Schema{
		Type:        convTypeGenerativeLanguage(s.Type),
		Description: s.Description,
		Nullable:    nullable,
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
			content.Parts = append(content.Parts, &genai.Part{Text: string(p)})
		case *llm.InlineData:
			content.Parts = append(content.Parts, &genai.Part{
				InlineData: &genai.Blob{
					MIMEType: p.MIMEType,
					Data:     p.Data,
				},
			})
		case *llm.FileData:
			content.Parts = append(content.Parts, &genai.Part{
				FileData: &genai.FileData{
					MIMEType: p.MIMEType,
					FileURI:  p.FileURI,
				},
			})
		case *llm.FunctionCall:
			content.Parts = append(content.Parts, &genai.Part{
				FunctionCall: &genai.FunctionCall{
					Name: p.Name,
					Args: p.Args,
				},
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

			content.Parts = append(content.Parts, &genai.Part{
				FunctionResponse: &genai.FunctionResponse{
					Name:     p.Name,
					Response: data,
				},
			})
		}
	}

	return content
}

func convertContextGenerativeLanguage(c *llm.ChatContext) []*genai.Content {
	contents := make([]*genai.Content, len(c.Contents))

	for i := range c.Contents {
		contents[i] = convertContentGenerativeLanguage(c.Contents[i])
	}

	return contents
}

func convertGenerativeLanguageContent(c *genai.Content) *llm.Content {
	lc := &llm.Content{
		Role:  llm.Role(c.Role),
		Parts: make([]llm.Segment, 0, len(c.Parts)),
	}

	for i := range c.Parts {
		if c.Parts[i].Text != "" {
			lc.Parts = append(lc.Parts, llm.Text(c.Parts[i].Text))
		} else if c.Parts[i].InlineData != nil {
			lc.Parts = append(lc.Parts, &llm.InlineData{
				MIMEType: c.Parts[i].InlineData.MIMEType,
				Data:     c.Parts[i].InlineData.Data,
			})
		} else if c.Parts[i].FunctionCall != nil {
			lc.Parts = append(lc.Parts, &llm.FunctionCall{
				Name: c.Parts[i].FunctionCall.Name,
				ID:   callid.OpenAICallID(),
				Args: c.Parts[i].FunctionCall.Args,
			})
		} else if c.Parts[i].FunctionResponse != nil {
			lc.Parts = append(lc.Parts, &llm.FunctionResponse{
				Name:    c.Parts[i].FunctionResponse.Name,
				ID:      callid.OpenAICallID(),
				Content: c.Parts[i].FunctionResponse.Response,
			})
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

	config := &genai.GenerateContentConfig{}
	model := g.model

	if g.config.SafetyFilterThreshold != llm.BlockDefault {
		var threshold genai.HarmBlockThreshold = genai.HarmBlockThresholdUnspecified
		switch g.config.SafetyFilterThreshold {
		case llm.BlockNone:
			threshold = genai.HarmBlockThresholdBlockNone
		case llm.BlockDefault, llm.BlockLowAndAbove:
			threshold = genai.HarmBlockThresholdBlockLowAndAbove
		case llm.BlockMediumAndAbove:
			threshold = genai.HarmBlockThresholdBlockMediumAndAbove
		case llm.BlockOnlyHigh:
			threshold = genai.HarmBlockThresholdBlockOnlyHigh
		}

		config.SafetySettings = []*genai.SafetySetting{
			{Category: genai.HarmCategoryHateSpeech, Threshold: threshold},
			{Category: genai.HarmCategoryDangerousContent, Threshold: threshold},
			{Category: genai.HarmCategoryHarassment, Threshold: threshold},
			{Category: genai.HarmCategorySexuallyExplicit, Threshold: threshold},
		}
	}

	if g.config.Temperature != nil {
		config.Temperature = g.config.Temperature
	}

	if g.config.TopK != nil {
		if g.config.TopK != nil {
			config.TopK = ptrify(float32(*g.config.TopK))
		}
	}

	if g.config.TopP != nil {
		config.TopP = g.config.TopP
	}

	if g.config.MaxOutputTokens != nil {
		max_output_tokens := int32(*g.config.MaxOutputTokens)
		config.MaxOutputTokens = max_output_tokens
	}

	if len(g.config.StopSequences) > 0 {
		config.StopSequences = g.config.StopSequences
	}

	if g.config.SystemInstruction != "" || chat.SystemInstruction != "" {
		config.SystemInstruction = &genai.Content{Parts: []*genai.Part{{Text: g.config.SystemInstruction + chat.SystemInstruction}}}
	}

	if g.config.ThinkingConfig != nil {
		config.ThinkingConfig = &genai.ThinkingConfig{}
		if g.config.ThinkingConfig.IncludeThoughts != nil {
			config.ThinkingConfig.IncludeThoughts = *g.config.ThinkingConfig.IncludeThoughts
		}
		if g.config.ThinkingConfig.ThinkingBudget != nil {
			config.ThinkingConfig.ThinkingBudget = ptrify(int32(*g.config.ThinkingConfig.ThinkingBudget))
		}
	}

	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Content: &llm.Content{},
		Stream:  stream,
	}

	if len(tools) > 0 {
		config.Tools = []*genai.Tool{
			{
				FunctionDeclarations: tools,
			},
		}
	}

	session, err := g.client.Chats.Create(ctx, model, config, contents)
	if err != nil {
		close(stream)
		v.Err = err
		return v
	}

	content := convertContentGenerativeLanguage(input)
	resp := session.SendMessageStream(ctx, unptrSlice(content.Parts)...)

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

		for resp, err := range resp {
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
					case genai.BlockedReasonOther,
						genai.BlockedReasonUnspecified:
						v.FinishReason = llm.FinishReasonUnknown
					case genai.BlockedReasonSafety, genai.BlockedReasonBlocklist, genai.BlockedReasonProhibitedContent:
						v.FinishReason = llm.FinishReasonSafety
					}
				}
				continue
			}

			select {
			case <-ctx.Done():
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

func unptrSlice[T any](v []*T) []T {
	if v == nil {
		return nil
	}

	out := make([]T, len(v))
	for i := range v {
		out[i] = *v[i]
	}

	return out
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
	return nil
}

func (g *aiStudioClient) NewLLM(model string, config *llm.Config) (llm.Model, error) {
	if config == nil {
		config = defaultGenerativeLanguageConfig
	}

	_vm := &generativeLanguageModel{
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
	ErrAPIKeyRequired = errors.New("api key is required")
)

func (AIStudioProvider) newAIStudioClient(ctx context.Context, configs ...pconf.Config) (*aiStudioClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	apiKey := client_config.APIKey
	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}

	genaiClient, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return nil, err
	}

	return &aiStudioClient{client: genaiClient}, nil
}

func (g AIStudioProvider) NewLLMClient(ctx context.Context, configs ...pconf.Config) (provider.LLMClient, error) {
	return g.newAIStudioClient(ctx, configs...)
}

var Provider = AIStudioProvider{}
var _ provider.EmbeddingProvider = Provider
var _ provider.LLMProvider = Provider

const ProviderName = "aistudio"

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

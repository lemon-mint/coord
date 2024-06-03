package ollama

import (
	"context"
	"net"
	"net/http"
	"net/url"
	"strings"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/internal/llmutils"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"

	ollama "github.com/ollama/ollama/api"
)

func convertContextOllama(chat *llm.ChatContext, system string) []ollama.Message {
	messages := make([]ollama.Message, 0, len(chat.Contents)+1)

	if system != "" {
		messages = append(messages, ollama.Message{
			Role:    "system",
			Content: system,
		})
	}

	for i := range chat.Contents {
		var m ollama.Message

		switch chat.Contents[i].Role {
		case llm.RoleUser, llm.RoleFunc:
			m.Role = "user"
		case llm.RoleModel:
			m.Role = "assistant"
		}

		for j := range chat.Contents[i].Parts {
			switch v := chat.Contents[i].Parts[j].(type) {
			case llm.Text:
				m.Content += string(v)
			case *llm.InlineData:
				m.Images = append(m.Images, ollama.ImageData(v.Data))
			case *llm.FunctionCall:
				// TODO: Implement FunctionCall
			case *llm.FunctionResponse:
				// TODO: Implement FunctionResponse
			}
		}

		messages = append(messages, m)
	}

	return messages
}

func (g *ollamaModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	if chat == nil {
		chat = &llm.ChatContext{}
	}

	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Content: &llm.Content{},
		Stream:  stream,
	}

	go func() {
		defer close(stream)
		defer func() {
			v.Content.Parts = llmutils.MergeTexts(v.Content.Parts)
		}()

		err := g.client.Heartbeat(ctx)
		if err != nil {
			v.Err = err
			return
		}

		messages := make([]*llm.Content, len(chat.Contents)+1)
		if len(chat.Contents) > 0 {
			copy(messages, chat.Contents)
		}
		messages[len(messages)-1] = input

		model_request := &ollama.ChatRequest{
			Model: g.model,
			Messages: convertContextOllama(&llm.ChatContext{
				Contents: messages,
				Tools:    chat.Tools,
			}, g.config.SystemInstruction),
			Stream: ptrify(true),
		}

		var sb strings.Builder

		err = g.client.Chat(ctx, model_request, func(cr ollama.ChatResponse) error {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case stream <- llm.Text(cr.Message.Content):
			}

			sb.WriteString(cr.Message.Content)
			return nil
		})
		if err != nil {
			v.Err = err
			return
		}

		v.Content = &llm.Content{
			Role:  llm.RoleModel,
			Parts: []llm.Segment{llm.Text(sb.String())},
		}
		v.FinishReason = llm.FinishReasonStop
		v.UsageData = nil
	}()

	return v
}

func (g *ollamaModel) Name() string {
	return g.model
}

func (g *ollamaModel) Close() error {
	return nil
}

func ptrify[T any](v T) *T {
	return &v
}

var defaultOllamaConfig = &llm.Config{
	Temperature:     ptrify(float32(0.8)),
	MaxOutputTokens: ptrify(2048),
}

var _ llm.LLM = (*ollamaModel)(nil)

type ollamaModel struct {
	client *ollama.Client
	config *llm.Config
	model  string
}

var _ provider.LLMClient = (*OllamaClient)(nil)

type OllamaClient struct {
	client *ollama.Client
}

func (g *OllamaClient) NewModel(model string, config *llm.Config) (llm.LLM, error) {
	if config == nil {
		config = defaultOllamaConfig
	}

	var _vm = &ollamaModel{
		client: g.client,
		model:  model,
		config: config,
	}

	return _vm, nil
}

var _ provider.LLMProvider = Provider

type OllamaProvider struct {
}

func (OllamaProvider) NewClient(ctx context.Context, configs ...pconf.Config) (provider.LLMClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	host, err := ollama.GetOllamaHost()
	if err != nil {
		return nil, err
	}

	//TODO: Apply client_config, baseurl

	return &OllamaClient{
		client: ollama.NewClient(&url.URL{Scheme: host.Scheme, Host: net.JoinHostPort(host.Host, host.Port)}, http.DefaultClient),
	}, nil
}

const ProviderName = "ollama"

var Provider OllamaProvider

func init() {
	var exists bool
	for _, n := range coord.LLMProviders() {
		if n == ProviderName {
			exists = true
			break
		}
	}
	if !exists {
		coord.RegisterLLMProvider(ProviderName, Provider)
	}
}

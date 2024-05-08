package ollama

import (
	"context"
	"strings"

	"github.com/lemon-mint/vermittlungsstelle/llm"
	ollama "github.com/ollama/ollama/api"
)

func ptrify[T any](v T) *T {
	return &v
}

var defaultOllamaConfig = &llm.Config{
	Temperature:     ptrify(float32(0.4)),
	MaxOutputTokens: ptrify(2048),
}

var _ llm.LLM = (*OllamaModel)(nil)

type OllamaModel struct {
	client *ollama.Client
	config *llm.Config
	model  string
}

func NewOllamaModel(client *ollama.Client, model string, config *llm.Config) *OllamaModel {
	if config == nil {
		config = defaultOllamaConfig
	}

	var _vm = &OllamaModel{
		client: client,
		config: config,
		model:  model,
	}

	return _vm
}

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

func (g *OllamaModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Content: &llm.Content{},
		Stream:  stream,
	}

	go func() {
		defer close(stream)

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

func (g *OllamaModel) Name() string {
	return g.model
}

func (g *OllamaModel) Close() error {
	return nil
}

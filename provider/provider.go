package provider

import (
	"context"

	"github.com/lemon-mint/coord/embedding"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/tts"
)

type LLMClient interface {
	NewModel(model string, config *llm.Config) (llm.LLM, error)
}

type LLMProvider interface {
	NewClient(ctx context.Context, configs ...pconf.Config) (LLMClient, error)
}

type TTSClient interface {
	NewModel(model string, config *tts.Config) (tts.TTS, error)
}

type TTSProvider interface {
	NewClient(ctx context.Context, configs ...pconf.Config) (TTSClient, error)
}

type EmbeddingClient interface {
	NewModel(model string, config *tts.Config) (embedding.EmbeddingModel, error)
}

type EmbeddingProvider interface {
	NewClient(ctx context.Context, configs ...pconf.Config) (EmbeddingClient, error)
}

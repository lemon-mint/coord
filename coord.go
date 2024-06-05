package coord

import (
	"context"
	"errors"

	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
)

var ErrNoSuchProvider = errors.New("coord: no such provider")

func NewLLMClient(ctx context.Context, provider string, configs ...pconf.Config) (provider.LLMClient, error) {
	llmProvidersMu.RLock()
	defer llmProvidersMu.RUnlock()

	driver, ok := llmProviders[provider]
	if !ok {
		return nil, ErrNoSuchProvider
	}

	return driver.NewLLMClient(ctx, configs...)
}

func NewEmbeddingClient(ctx context.Context, provider string, configs ...pconf.Config) (provider.EmbeddingClient, error) {
	embeddingProvidersMu.RLock()
	defer embeddingProvidersMu.RUnlock()

	driver, ok := embeddingProviders[provider]
	if !ok {
		return nil, ErrNoSuchProvider
	}

	return driver.NewEmbeddingClient(ctx, configs...)
}

func NewTTSClient(ctx context.Context, provider string, configs ...pconf.Config) (provider.TTSClient, error) {
	ttsProvidersMu.RLock()
	defer ttsProvidersMu.RUnlock()

	driver, ok := ttsProviders[provider]
	if !ok {
		return nil, ErrNoSuchProvider
	}

	return driver.NewTTSClient(ctx, configs...)
}

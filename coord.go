package coord

import (
	"sort"
	"sync"

	"github.com/lemon-mint/coord/provider"
)

var (
	llmProvidersMu sync.RWMutex
	llmProviders   = make(map[string]provider.LLMProvider)

	ttsProvidersMu sync.RWMutex
	ttsProviders   = make(map[string]provider.TTSProvider)

	embeddingProvidersMu sync.RWMutex
	embeddingProviders   = make(map[string]provider.EmbeddingProvider)
)

// LLMProviders returns the names of the registered llm providers.
func LLMProviders() []string {
	llmProvidersMu.RLock()
	defer llmProvidersMu.RUnlock()
	list := make([]string, 0, len(llmProviders))
	for name := range llmProviders {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// RegisterLLMProvider registers a llm provider.
func RegisterLLMProvider(name string, p provider.LLMProvider) {
	llmProvidersMu.Lock()
	defer llmProvidersMu.Unlock()
	llmProviders[name] = p
}

// TTSProviders returns the names of the registered tts providers.
func TTSProviders() []string {
	ttsProvidersMu.RLock()
	defer ttsProvidersMu.RUnlock()
	list := make([]string, 0, len(ttsProviders))
	for name := range ttsProviders {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// RegisterTTSProvider registers a tts provider.
func RegisterTTSProvider(name string, p provider.TTSProvider) {
	ttsProvidersMu.Lock()
	defer ttsProvidersMu.Unlock()
	ttsProviders[name] = p
}

// EmbeddingProviders returns the names of the registered embedding providers.
func EmbeddingProviders() []string {
	embeddingProvidersMu.RLock()
	defer embeddingProvidersMu.RUnlock()
	list := make([]string, 0, len(embeddingProviders))
	for name := range embeddingProviders {
		list = append(list, name)
	}
	sort.Strings(list)
	return list
}

// RegisterEmbeddingProvider registers an embedding provider.
func RegisterEmbeddingProvider(name string, p provider.EmbeddingProvider) {
	embeddingProvidersMu.Lock()
	defer embeddingProvidersMu.Unlock()
	embeddingProviders[name] = p
}

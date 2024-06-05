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

// ListLLMProviders returns the names of the registered llm providers.
func ListLLMProviders() []string {
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

// RemoveLLMProvider removes a llm provider.
func RemoveLLMProvider(name string) {
	llmProvidersMu.Lock()
	defer llmProvidersMu.Unlock()
	delete(llmProviders, name)
}

// ListTTSProviders returns the names of the registered tts providers.
func ListTTSProviders() []string {
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

// RemoveTTSProvider removes a tts provider.
func RemoveTTSProvider(name string) {
	ttsProvidersMu.Lock()
	defer ttsProvidersMu.Unlock()
	delete(ttsProviders, name)
}

// ListEmbeddingProviders returns the names of the registered embedding providers.
func ListEmbeddingProviders() []string {
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

// RemoveEmbeddingProvider removes an embedding provider.
func RemoveEmbeddingProvider(name string) {
	embeddingProvidersMu.Lock()
	defer embeddingProvidersMu.Unlock()
	delete(embeddingProviders, name)
}

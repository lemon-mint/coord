package llmproxy

import (
	"net/http"
	"sync"

	"github.com/lemon-mint/coord/embedding"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/provider"
)

type Server struct {
	rwMu sync.RWMutex

	llmRouter  map[string]llm.Model
	llmClients []provider.LLMClient

	embeddingRouter  map[string]embedding.Model
	embeddingClients []provider.LLMClient

	httpServer http.Server
}

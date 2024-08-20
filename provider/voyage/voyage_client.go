package voyage

type voyageInputType string

const (
	voyageInputNone     voyageInputType = ""
	voyageInputQuery    voyageInputType = "query"
	voyageInputDocument voyageInputType = "document"
)

type voyageEmbeddingRequest struct {
	Input     []string        `json:"input"`
	Model     string          `json:"model"`
	InputType voyageInputType `json:"input_type"`
}

type voyageEmbeddingResponse struct {
	Object string       `json:"object"`
	Data   []voyageData `json:"data"`
	Model  string       `json:"model"`
	Usage  voyageUsage  `json:"usage"`
}

type voyageData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

type voyageUsage struct {
	TotalTokens int `json:"total_tokens"`
}

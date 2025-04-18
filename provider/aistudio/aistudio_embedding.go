package aistudio

import (
	"context"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/embedding"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"google.golang.org/genai"
)

var _ embedding.Model = (*textEmbedding)(nil)

type textEmbedding struct {
	client *genai.Client

	model     string
	outputDim int
}

func (g *textEmbedding) TextEmbedding(ctx context.Context, text string, task embedding.TaskType) ([]float64, error) {
	config := &genai.EmbedContentConfig{}

	var err error

	switch task {
	case embedding.TaskTypeGeneral:
		config.TaskType = ""
	case embedding.TaskTypeSearchQuery:
		config.TaskType = "RETRIEVAL_QUERY"
	case embedding.TaskTypeSearchDocument:
		config.TaskType = "RETRIEVAL_DOCUMENT"
	case embedding.TaskTypeSemanticSimilarity:
		config.TaskType = "SEMANTIC_SIMILARITY"
	case embedding.TaskTypeClassification:
		config.TaskType = "CLASSIFICATION"
	case embedding.TaskTypeClustering:
		config.TaskType = "CLUSTERING"
	case embedding.TaskTypeQA:
		config.TaskType = "QUESTION_ANSWERING"
	case embedding.TaskTypeFactVerification:
		config.TaskType = "FACT_VERIFICATION"
	default:
		return nil, embedding.ErrUnsupported
	}

	response, err := g.client.Models.EmbedContent(ctx, g.model, []*genai.Content{
		&genai.Content{
			Role: genai.RoleUser,
			Parts: []*genai.Part{
				&genai.Part{
					Text: text,
				},
			},
		},
	}, config)
	if err != nil {
		return nil, err
	}

	embeddings := response.Embeddings[0].Values

	if g.outputDim > 0 && len(embeddings) > g.outputDim {
		embeddings = embeddings[:g.outputDim]
	}

	var output []float64 = make([]float64, len(embeddings))
	for i := range embeddings {
		output[i] = float64(embeddings[i])
	}

	return output, nil
}

var _ provider.EmbeddingClient = (*aiStudioClient)(nil)

func (g *aiStudioClient) NewEmbedding(model string, config *embedding.Config) (embedding.Model, error) {
	if config == nil {
		config = &embedding.Config{}
	}

	_em := &textEmbedding{
		client:    g.client,
		model:     model,
		outputDim: config.Dimension,
	}

	return _em, nil
}

func (g AIStudioProvider) NewEmbeddingClient(ctx context.Context, configs ...pconf.Config) (provider.EmbeddingClient, error) {
	return g.newAIStudioClient(ctx, configs...)
}

func init() {
	coord.RegisterEmbeddingProvider(ProviderName, Provider)
}

package aistudio

import (
	"context"

	"github.com/google/generative-ai-go/genai"
	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/embedding"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
)

var _ embedding.EmbeddingModel = (*textEmbedding)(nil)

type textEmbedding struct {
	client *genai.Client

	model     string
	outputDim int
}

func (g *textEmbedding) TextEmbedding(ctx context.Context, text string, task embedding.TaskType) ([]float64, error) {
	model := g.client.EmbeddingModel(g.model)

	var err error

	switch task {
	case embedding.TaskTypeGeneral:
		model.TaskType = genai.TaskTypeUnspecified
	case embedding.TaskTypeSearchQuery:
		model.TaskType = genai.TaskTypeRetrievalQuery
	case embedding.TaskTypeSearchDocument:
		model.TaskType = genai.TaskTypeRetrievalDocument
	case embedding.TaskTypeSemanticSimilarity:
		model.TaskType = genai.TaskTypeSemanticSimilarity
	case embedding.TaskTypeClassification:
		model.TaskType = genai.TaskTypeClassification
	case embedding.TaskTypeClustering:
		model.TaskType = genai.TaskTypeClustering
	case embedding.TaskTypeQA:
		model.TaskType = genai.TaskTypeQuestionAnswering
	case embedding.TaskTypeFactVerification:
		model.TaskType = genai.TaskTypeFactVerification
	default:
		return nil, embedding.ErrUnsupported
	}

	response, err := model.EmbedContent(ctx, genai.Text(text))
	if err != nil {
		return nil, err
	}
	embeddings := response.Embedding.Values

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

func (g *aiStudioClient) NewEmbedding(model string, config *embedding.Config) (embedding.EmbeddingModel, error) {
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

func (AIStudioProvider) NewEmbeddingClient(ctx context.Context, configs ...pconf.Config) (provider.EmbeddingClient, error) {
	return (AIStudioProvider).newAIStudioClient(AIStudioProvider{}, ctx, configs...)
}

func init() {
	var exists bool
	for _, n := range coord.ListEmbeddingProviders() {
		if n == ProviderName {
			exists = true
			break
		}
	}
	if !exists {
		coord.RegisterEmbeddingProvider(ProviderName, Provider)
	}
}

package vertexai

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/embedding"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ embedding.EmbeddingModel = (*textEmbedding)(nil)

type textEmbedding struct {
	client   *aiplatform.PredictionClient
	location string
	project  string

	model     string
	outputDim int
}

func (g *textEmbedding) TextEmbedding(ctx context.Context, text string, task embedding.TaskType) ([]float64, error) {
	base := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models", g.project, g.location)
	url := fmt.Sprintf("%s/%s", base, g.model)

	var err error
	var promptValue *structpb.Value
	var request map[string]interface{}

	switch task {
	case embedding.TaskTypeGeneral:
		request = map[string]interface{}{
			"content": text,
		}
	case embedding.TaskTypeSearchQuery:
		request = map[string]interface{}{
			"task_type": "RETRIEVAL_QUERY",
			"content":   text,
		}
	case embedding.TaskTypeSearchDocument:
		request = map[string]interface{}{
			"task_type": "RETRIEVAL_DOCUMENT",
			"content":   text,
		}
	case embedding.TaskTypeSemanticSimilarity:
		request = map[string]interface{}{
			"task_type": "SEMANTIC_SIMILARITY",
			"content":   text,
		}
	case embedding.TaskTypeClassification:
		request = map[string]interface{}{
			"task_type": "CLASSIFICATION",
			"content":   text,
		}
	case embedding.TaskTypeClustering:
		request = map[string]interface{}{
			"task_type": "CLUSTERING",
			"content":   text,
		}
	case embedding.TaskTypeQA:
		request = map[string]interface{}{
			"task_type": "QUESTION_ANSWERING",
			"content":   text,
		}
	case embedding.TaskTypeFactVerification:
		request = map[string]interface{}{
			"task_type": "FACT_VERIFICATION",
			"content":   text,
		}
	default:
		return nil, embedding.ErrUnsupported
	}

	promptValue, err = structpb.NewValue(request)
	if err != nil {
		return nil, err
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:  url,
		Instances: []*structpb.Value{promptValue},
	}

	resp, err := g.client.Predict(ctx, req)
	if err != nil {
		return nil, err
	}

	if len(resp.Predictions) <= 0 {
		return nil, embedding.ErrNoResult
	}

	r_struct := resp.Predictions[0].GetStructValue()
	if r_struct == nil {
		return nil, embedding.ErrNoResult
	}

	r_map := r_struct.AsMap()
	if r_map == nil {
		return nil, embedding.ErrNoResult
	}

	r_embeddings, ok := r_map["embeddings"]
	if !ok || r_embeddings == nil {
		return nil, embedding.ErrNoResult
	}

	r_embeddings_ifacemap, ok := r_embeddings.(map[string]interface{})
	if !ok || r_embeddings_ifacemap == nil {
		return nil, embedding.ErrNoResult
	}

	r_values, ok := r_embeddings_ifacemap["values"]
	if !ok || r_values == nil {
		return nil, embedding.ErrNoResult
	}

	r_values_list, ok := r_values.([]interface{})
	if !ok || r_values_list == nil || len(r_values_list) <= 0 {
		return nil, embedding.ErrNoResult
	}

	var embeddings []float64 = make([]float64, len(r_values_list))
	for i := range r_values_list {
		switch v := r_values_list[i].(type) {
		case float64:
			embeddings[i] = v
		case float32:
			embeddings[i] = float64(v)
		default:
			return nil, embedding.ErrNoResult
		}
	}

	if g.outputDim > 0 && len(embeddings) > g.outputDim {
		embeddings = embeddings[:g.outputDim]
	}

	return embeddings, nil
}

var _ provider.EmbeddingClient = (*vertexAIClient)(nil)

func (g *vertexAIClient) NewEmbedding(model string, config *embedding.Config) (embedding.EmbeddingModel, error) {
	if config == nil {
		config = &embedding.Config{}
	}

	_em := &textEmbedding{
		client:    g.predictionClient,
		location:  g.location,
		project:   g.projectID,
		model:     model,
		outputDim: config.Dimension,
	}

	return _em, nil
}

func (VertexAIProvider) NewEmbeddingClient(ctx context.Context, configs ...pconf.Config) (provider.EmbeddingClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	projectID := client_config.ProjectID
	location := client_config.Location
	client_options := client_config.GoogleClientOptions

	if projectID == "" {
		return nil, ErrProjectIDRequired
	}

	if location == "" {
		return nil, ErrLocationRequired
	}

	predictionClient, err := aiplatform.NewPredictionClient(ctx, client_options...)
	if err != nil {
		return nil, err
	}

	return &vertexAIClient{
		predictionClient: predictionClient,
		location:         location,
		projectID:        projectID,
	}, nil
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

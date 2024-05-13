package vertexai

import (
	"context"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1"
	"cloud.google.com/go/aiplatform/apiv1/aiplatformpb"
	"github.com/lemon-mint/vermittlungsstelle/embedding"
	"google.golang.org/protobuf/types/known/structpb"
)

var _ embedding.EmbeddingModel = (*TextEmbedding)(nil)

type TextEmbedding struct {
	client   *aiplatform.PredictionClient
	location string
	project  string

	model     string
	outputDim int
}

func NewTextEmbedding(client *aiplatform.PredictionClient, location, project, model string, outputDim int) *TextEmbedding {
	return &TextEmbedding{
		client:    client,
		location:  location,
		project:   project,
		model:     model,
		outputDim: outputDim,
	}
}

func (t *TextEmbedding) TextEmbedding(ctx context.Context, text string, task embedding.TaskType) ([]float64, error) {
	base := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models", t.project, t.location)
	url := fmt.Sprintf("%s/%s", base, t.model)

	var err error
	var promptValue *structpb.Value

	switch task {
	case embedding.TaskTypeGeneral:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"content": text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeSearchQuery:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "RETRIEVAL_QUERY",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeSearchDocument:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "RETRIEVAL_DOCUMENT",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeSemanticSimilarity:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "SEMANTIC_SIMILARITY",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeClassification:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "CLASSIFICATION",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeClustering:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "CLUSTERING",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeQA:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "QUESTION_ANSWERING",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	case embedding.TaskTypeFactVerification:
		promptValue, err = structpb.NewValue(map[string]interface{}{
			"task_type": "FACT_VERIFICATION",
			"content":   text,
		})
		if err != nil {
			return nil, err
		}
	default:
		return nil, embedding.ErrUnsupported
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:  url,
		Instances: []*structpb.Value{promptValue},
	}

	resp, err := t.client.Predict(ctx, req)
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

	return embeddings, nil
}

package embedding

import (
	"context"
	"errors"
)

type EmbeddingModel interface {
	TextEmbedding(ctx context.Context, text string, task TaskType) ([]float64, error) // Returns the embedding of the text.
}

var (
	ErrUnsupported = errors.New("unsupported") // This Error occurs whem the model does not support the content type provided.
	ErrTooLong     = errors.New("too long")    // This Error occurs when the text is too long.
	ErrNoResult    = errors.New("no result")   // This Error occurs when the model does not return any result.
)

//go:generate stringer -type=TaskType
type TaskType uint16

const (
	TaskTypeGeneral TaskType = iota
	TaskTypeSearchQuery
	TaskTypeSearchDocument
	TaskTypeSemanticSimilarity
	TaskTypeClassification
	TaskTypeClustering
	TaskTypeFactVerification
	TaskTypeQA
)

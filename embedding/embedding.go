package embedding

import (
	"context"
	"errors"
)

type Model interface {
	TextEmbedding(ctx context.Context, text string, task TaskType) ([]float64, error) // Returns the embedding of the text.
}

var (
	ErrUnsupported       = errors.New("unsupported")         // This Error occurs whem the model does not support the content type provided.
	ErrMaxLengthExceeded = errors.New("max length exceeded") // This Error occurs when the content exceeds the maximum length.
	ErrNoResult          = errors.New("no result")           // This Error occurs when the model does not return any result.
)

//go:generate go tool stringer -type=TaskType
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

type Config struct {
	Dimension int
}

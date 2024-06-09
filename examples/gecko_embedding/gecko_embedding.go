package main

import (
	"context"
	"fmt"
	"math"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/embedding"
	"github.com/lemon-mint/coord/pconf"
	"gopkg.eu.org/envloader"

	_ "github.com/lemon-mint/coord/provider/vertexai"
)

func cossim(a, b []float64) float64 {
	var dotProduct float64
	for i := range a {
		dotProduct += a[i] * b[i]
	}
	var magA, magB float64
	for i := range a {
		magA += a[i] * a[i]
		magB += b[i] * b[i]
	}
	magA = math.Sqrt(magA)
	magB = math.Sqrt(magB)
	return dotProduct / (magA * magB)
}

func main() {
	type Config struct {
		Location  string `env:"LOCATION,required"`
		ProjectID string `env:"PROJECT_ID,required"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile(".env", c)

	client, err := coord.NewEmbeddingClient(
		context.Background(),
		"vertexai",
		pconf.WithProjectID(c.ProjectID),
		pconf.WithLocation(c.Location),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	model, err := client.NewEmbedding("text-embedding-004", nil)
	if err != nil {
		panic(err)
	}

	texts := []string{"Apple", "Banana", "Cat", "Hamster"}
	embeddings := make([][]float64, len(texts))
	for i := range texts {
		embedding, err := model.TextEmbedding(context.Background(), texts[i], embedding.TaskTypeSemanticSimilarity)
		if err != nil {
			panic(err)
		}
		embeddings[i] = embedding
	}

	// Calculate cosine similarity between the embeddings using a loop
	for i := 0; i < len(embeddings)-1; i++ {
		for j := i + 1; j < len(embeddings); j++ {
			cosSim := cossim(embeddings[i], embeddings[j])
			fmt.Printf("Cosine Similarity between '%s' and '%s': %.4f\n", texts[i], texts[j], cosSim)
		}
	}
}

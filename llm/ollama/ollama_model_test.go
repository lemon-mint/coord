package ollama_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lemon-mint/vermittlungsstelle/llm"
	"github.com/lemon-mint/vermittlungsstelle/llm/ollama"
	"github.com/ollama/ollama/api"
)

var client *api.Client = func() *api.Client {
	c, err := api.ClientFromEnvironment()
	if err != nil {
		panic(err)
	}
	return c
}()

func TestOllamaGenerate(t *testing.T) {
	var model llm.LLM = ollama.NewOllamaModel(client, "llama3:latest", nil)
	defer model.Close()

	output := model.GenerateStream(
		context.Background(),
		&llm.ChatContext{},
		&llm.Content{
			Role:  llm.RoleUser,
			Parts: []llm.Segment{llm.Text("Hello!")},
		},
	)

	for segment := range output.Stream {
		fmt.Print(segment)
	}
	fmt.Println()

	if output.Err != nil {
		t.Error(output.Err)
		return
	}
}

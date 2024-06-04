package ollama_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/provider/ollama"
)

func getClient() provider.LLMClient {
	client, err := ollama.Provider.NewLLMClient(
		context.Background(),
	)
	if err != nil {
		panic(err)
	}

	return client
}

func TestOllamaGenerate(t *testing.T) {
	client := getClient()
	defer client.Close()

	model, err := client.NewLLM("llama3:latest", nil)
	if err != nil {
		panic(err)
	}
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

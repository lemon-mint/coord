package ollama_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/provider/ollama"
)

var client provider.LLMClient = func() provider.LLMClient {
	client, err := ollama.Provider.NewClient(
		context.Background(),
	)
	if err != nil {
		panic(err)
	}

	return client
}()

func TestOllamaGenerate(t *testing.T) {
	model, err := client.NewModel("llama3:latest", nil)
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

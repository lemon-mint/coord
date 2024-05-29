package openai_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/lemon-mint/vermittlungsstelle/llm"
	"github.com/lemon-mint/vermittlungsstelle/llm/openai"
	oai "github.com/sashabaranov/go-openai"
	"gopkg.eu.org/envloader"
)

var client *oai.Client = func() *oai.Client {
	envloader.LoadEnvFile("../../.env")

	return openai.NewClient(os.Getenv("OPENAI_API_KEY"))
}()

func TestOpenAIGenerate(t *testing.T) {
	var model llm.LLM = openai.NewModel(client, "gpt-4o", nil)
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

	if output.UsageData.OutputTokens <= 0 {
		t.Errorf("expected output tokens > 0, got %d\n", output.UsageData.OutputTokens)
		return
	}
}

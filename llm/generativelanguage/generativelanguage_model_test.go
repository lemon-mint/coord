package generativelanguage_test

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/google/generative-ai-go/genai"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/llm/generativelanguage"
	"gopkg.eu.org/envloader"
)

var client *genai.Client = func() *genai.Client {
	envloader.LoadEnvFile("../../.env")

	client, err := generativelanguage.NewClient(
		context.Background(),
		os.Getenv("GEMINI_API_KEY"),
	)
	if err != nil {
		panic(err)
	}
	return client
}()

func TestGenerativeLanguageGenerate(t *testing.T) {
	var model llm.LLM = generativelanguage.NewModel(client, "gemini-pro", nil)
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

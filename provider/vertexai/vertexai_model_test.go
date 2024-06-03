package vertexai_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/provider/vertexai"
	"gopkg.eu.org/envloader"
)

var client provider.LLMClient = func() provider.LLMClient {
	type Config struct {
		Location  string `env:"LOCATION"`
		ProjectID string `env:"PROJECT_ID"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile("../../.env", c)

	client, err := vertexai.Provider.NewLLMClient(
		context.Background(),
		pconf.WithProjectID(c.ProjectID),
		pconf.WithLocation(c.Location),
	)
	if err != nil {
		panic(err)
	}

	return client
}()

func TestVertexAIGenerate(t *testing.T) {
	model, err := client.NewLLM("gemini-1.5-flash-001", nil)
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

	if output.UsageData.OutputTokens <= 0 {
		t.Errorf("expected output tokens > 0, got %d\n", output.UsageData.OutputTokens)
		return
	}
}

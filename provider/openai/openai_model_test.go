package openai_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/provider/openai"
	"gopkg.eu.org/envloader"
)

var client provider.LLMClient = func() provider.LLMClient {
	type Config struct {
		APIKey string `env:"OPENAI_API_KEY"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile("../../.env", c)

	client, err := openai.Provider.NewClient(
		context.Background(),
		pconf.WithAPIKey(c.APIKey),
	)
	if err != nil {
		panic(err)
	}

	return client
}()

func TestAnthropicGenerate(t *testing.T) {
	model, err := client.NewModel("gpt-3.5-turbo", nil)
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

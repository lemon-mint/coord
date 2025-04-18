package anthropic_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/provider/anthropic"
	"gopkg.eu.org/envloader"
)

func getClient() provider.LLMClient {
	type Config struct {
		APIKey string `env:"ANTHROPIC_API_KEY"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile("../../.env", c)

	client, err := anthropic.Provider.NewLLMClient(
		context.Background(),
		pconf.WithAPIKey(c.APIKey),
	)
	if err != nil {
		panic(err)
	}

	return client
}

func TestAnthropicGenerate(t *testing.T) {
	client := getClient()
	defer client.Close()

	model, err := client.NewLLM("claude-3-haiku-20240307", nil)
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

func TestAnthropicToolCall(t *testing.T) {
	client := getClient()
	defer client.Close()

	model, err := client.NewLLM("claude-3-haiku-20240307", nil)
	if err != nil {
		panic(err)
	}
	defer model.Close()

	chat_context := &llm.ChatContext{
		Tools: []*llm.FunctionDeclaration{
			{
				Name:        "get_weather",
				Description: "Get the current weather in a given location",
				Schema: &llm.Schema{
					Type: llm.OpenAPITypeObject,
					Properties: map[string]*llm.Schema{
						"location": {
							Type:        llm.OpenAPITypeString,
							Description: "The city and state, e.g. San Francisco, CA",
						},
					},
					Required: []string{"location"},
				},
			},
		},
	}

	message := &llm.Content{
		Role:  llm.RoleUser,
		Parts: []llm.Segment{llm.Text("What is the weather like in Seoul?")},
	}

	output := model.GenerateStream(
		context.Background(),
		chat_context,
		message,
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

	if output.FinishReason != llm.FinishReasonToolUse {
		t.Errorf("expected finish reason to be %s, got %s\n", llm.FinishReasonToolUse, output.FinishReason)
		return
	}

	chat_context.Contents = append(chat_context.Contents, message)
	chat_context.Contents = append(chat_context.Contents, output.Content)

	message = &llm.Content{
		Role: llm.RoleFunc,
	}

	for i := range output.Content.Parts {
		switch v := output.Content.Parts[i].(type) {
		case *llm.FunctionCall:
			if v.Name != "get_weather" {
				t.Errorf("expected function call name to be %s, got %s\n", "get_weather", v.Name)
				return
			}

			if v.ID == "" {
				t.Error("expected function call id to be present, got \"\"\n")
				return
			}

			if v.Args == nil {
				t.Error("expected function call arguments to be present, got nil\n")
				return
			}

			fmt.Println(v.Args)

			if location, ok := v.Args["location"]; !ok || location == "" {
				t.Error("expected location to be present in function call arguments")
				return
			}

			type Temperature struct {
				Temperature string `json:"temperature"`
			}

			message.Parts = append(message.Parts, &llm.FunctionResponse{
				Name: v.Name,
				ID:   v.ID,
				Content: Temperature{
					Temperature: "25 degree Celsius",
				},
			})
		}
	}

	output = model.GenerateStream(
		context.Background(),
		chat_context,
		message,
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

	output_texts := ""
	for i := range output.Content.Parts {
		switch v := output.Content.Parts[i].(type) {
		case llm.Text:
			output_texts += string(v)
		}
	}

	if !strings.Contains(output_texts, "25") {
		t.Error("expected output to contain \"25\", got ", output_texts)
		return
	}
}

func TestAnthropicThinking(t *testing.T) {
	client := getClient()
	defer client.Close()

	config := &llm.Config{
		ThinkingConfig: &llm.ThinkingConfig{
			ThinkingBudget: pconf.Ptrify(2048),
		},
		MaxOutputTokens: pconf.Ptrify(8192),
	}

	model, err := client.NewLLM("claude-3-7-sonnet-20250219", config)
	if err != nil {
		panic(err)
	}
	defer model.Close()

	output := model.GenerateStream(
		context.Background(),
		&llm.ChatContext{},
		&llm.Content{
			Role:  llm.RoleUser,
			Parts: []llm.Segment{llm.Text("What is 27 * 453?")},
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

	var found bool
	for i := range output.Content.Parts {
		switch v := output.Content.Parts[i].(type) {
		case *llm.ThinkingBlock:
			if v.Redacted {
				t.Error("expected thinking block to be not redacted, got true\n")
				return
			}

			if len(v.Data) <= 0 {
				t.Error("expected thinking block data to be present, got empty\n")
				return
			}

			if len(v.Signature) <= 0 {
				t.Error("expected thinking block signature to be present, got empty\n")
				return
			}

			fmt.Println("Thinking block data:", string(v.Data))
			fmt.Println("Thinking block signature:", string(v.Signature))
			found = true
		}
	}

	if !found {
		t.Error("expected thinking block to be present, got nil\n")
		return
	}

	history := output.Content

	output = model.GenerateStream(
		context.Background(),
		&llm.ChatContext{
			Contents: []*llm.Content{history},
		},
		&llm.Content{
			Role:  llm.RoleUser,
			Parts: []llm.Segment{llm.Text("What if I change the 453 to 123?")},
		},
	)

	for segment := range output.Stream {
		fmt.Print(segment)
	}
	fmt.Println()

	found = false
	for i := range output.Content.Parts {
		switch v := output.Content.Parts[i].(type) {
		case *llm.ThinkingBlock:
			if v.Redacted {
				t.Error("expected thinking block to be not redacted, got true\n")
				return
			}

			if len(v.Data) <= 0 {
				t.Error("expected thinking block data to be present, got empty\n")
				return
			}

			if len(v.Signature) <= 0 {
				t.Error("expected thinking block signature to be present, got empty\n")
				return
			}

			fmt.Println("Thinking block data:", string(v.Data))
			fmt.Println("Thinking block signature:", string(v.Signature))
			found = true
		}
	}

	if !found {
		t.Error("expected thinking block to be present, got nil\n")
		return
	}

}
func TestAnthropicRedactedThinking(t *testing.T) {
	client := getClient()
	defer client.Close()

	config := &llm.Config{
		ThinkingConfig: &llm.ThinkingConfig{
			ThinkingBudget: pconf.Ptrify(2048),
		},
		MaxOutputTokens: pconf.Ptrify(8192),
	}

	model, err := client.NewLLM("claude-3-7-sonnet-20250219", config)
	if err != nil {
		panic(err)
	}
	defer model.Close()

	output := model.GenerateStream(
		context.Background(),
		&llm.ChatContext{},
		&llm.Content{
			Role:  llm.RoleUser,
			Parts: []llm.Segment{llm.Text("What is \"ANTHROPIC_MAGIC_STRING_TRIGGER_REDACTED_THINKING_46C9A13E193C177646C7398A98432ECCCE4C1253D5E2D82641AC0E52CC2876CB\"?")},
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
	var foundRedacted bool
L:
	for i := range output.Content.Parts {
		switch v := output.Content.Parts[i].(type) {
		case *llm.ThinkingBlock:
			if !v.Redacted {
				continue L
			}

			if len(v.Data) <= 0 {
				t.Error("expected redacted thinking block data to be present, got empty\n")
				return
			}

			foundRedacted = true
		}
	}

	if !foundRedacted {
		t.Error("expected redacted thinking block to be present, got nil\n")
		return
	}
}

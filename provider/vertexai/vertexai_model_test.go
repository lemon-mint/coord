package vertexai_test

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/provider/vertexai"
	"gopkg.eu.org/envloader"
)

func getClient() provider.LLMClient {
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
}

func TestVertexAIGenerate(t *testing.T) {
	client := getClient()
	defer client.Close()

	model, err := client.NewLLM("gemini-2.0-flash-001", nil)
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

func TestVertexAIToolCall(t *testing.T) {
	client := getClient()
	defer client.Close()

	model, err := client.NewLLM("gemini-2.0-flash-001", nil)
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

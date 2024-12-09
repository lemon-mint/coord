package main

import (
	"bufio"
	"context"
	"fmt"
	"os"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"gopkg.eu.org/envloader"

	_ "github.com/lemon-mint/coord/provider/vertexai"
)

func main() {
	type Config struct {
		Location  string `env:"LOCATION,required"`
		ProjectID string `env:"PROJECT_ID,required"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile(".env", c)

	client, err := coord.NewLLMClient(
		context.Background(),
		"vertexai",
		pconf.WithProjectID(c.ProjectID),
		pconf.WithLocation(c.Location),
	)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	model, err := client.NewLLM("gemini-1.5-flash-002", nil)
	if err != nil {
		panic(err)
	}
	defer model.Close()

	reader := bufio.NewReader(os.Stdin)
	fmt.Print("> ")
	prompt, _ := reader.ReadString('\n')

	resp := model.GenerateStream(context.Background(), &llm.ChatContext{}, &llm.Content{
		Role:  llm.RoleUser,
		Parts: []llm.Segment{llm.Text(prompt)},
	})
	if resp.Err != nil {
		panic(resp.Err)
	}

	fmt.Println(">>> ")
	for t := range resp.Stream {
		fmt.Print(t)
	}
	fmt.Println()

	fmt.Printf("Total Tokens: %d\n", resp.UsageData.TotalTokens)
	fmt.Printf("Input Tokens: %d\n", resp.UsageData.InputTokens)
	fmt.Printf("Output Tokens: %d\n", resp.UsageData.OutputTokens)
}

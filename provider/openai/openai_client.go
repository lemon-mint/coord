package openai

import (
	"errors"
	"github.com/lemon-mint/coord/pconf"
	"github.com/sashabaranov/go-openai"
)

type openAIClient struct {
	client *openai.Client
}

func (*openAIClient) Close() error {
	return nil
}

var (
	ErrAPIKeyRequired error = errors.New("api key is required")
)

type openaiConfig func(*openAIClient) error

func (openaiConfig) Apply(*pconf.GeneralConfig) error {
	return nil
}

func WithAzureConfig(apiKey, baseURL string) pconf.Config {
	return WithOpenAIConfig(openai.DefaultAzureConfig(apiKey, baseURL))
}

func WithOpenAIConfig(config openai.ClientConfig) pconf.Config {
	return WithOpenAIClient(openai.NewClientWithConfig(config))
}

func WithOpenAIClient(client *openai.Client) pconf.Config {
	return openaiConfig(func(c *openAIClient) error {
		c.client = client
		return nil
	})
}

func newClient(configs ...pconf.Config) (*openAIClient, error) {
	client_config := pconf.GeneralConfig{}
	var openai_client openAIClient
	for i := range configs {
		switch v := configs[i].(type) {
		case openaiConfig:
			if err := v(&openai_client); err != nil {
				return nil, err
			}
		default:
			configs[i].Apply(&client_config)
		}
	}

	if openai_client.client != nil {
		return &openai_client, nil
	}

	if client_config.APIKey == "" {
		return nil, ErrAPIKeyRequired
	}

	openai_config := openai.DefaultConfig(client_config.APIKey)
	if client_config.BaseURL != "" {
		openai_config.BaseURL = client_config.BaseURL
	}

	openai_client.client = openai.NewClientWithConfig(openai_config)
	return &openai_client, nil
}

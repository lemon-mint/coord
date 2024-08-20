package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/internal/llmutils"
	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"

	"github.com/valyala/fastjson"
)

var _ llm.Model = (*anthropicModel)(nil)

type anthropicModel struct {
	client *anthropicAPIClient
	config *llm.Config
	model  string
}

func (g *anthropicModel) Name() string {
	return g.model
}

func (g *anthropicModel) Close() error {
	return nil
}

var (
	_sse_Event = []byte("event: ")
	_sse_Data  = []byte("data: ")
)

func (g *anthropicModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	if chat == nil {
		chat = &llm.ChatContext{}
	}

	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Stream:  stream,
		Content: &llm.Content{},
	}

	msgs := convertContextAnthropic(chat)
	msgs = append(msgs, convertContentAnthropic(input))

	go func() {
		defer close(stream)

		url, err := url.JoinPath(g.client.baseURL, "./messages")
		if err != nil {
			v.Err = err
			return
		}

		model_request := &anthropicCreateMessagesRequest{
			Model:         g.model,
			Messages:      msgs,
			SystemPrompt:  g.config.SystemInstruction + chat.SystemInstruction,
			StopSequences: g.config.StopSequences,
			Tools:         convertToolsAnthropic(chat.Tools),
			Temperature:   g.config.Temperature,
			TopP:          g.config.TopP,
			TopK:          g.config.TopK,
			Stream:        true,
		}

		if g.config.MaxOutputTokens == nil || *g.config.MaxOutputTokens <= 0 {
			model_request.MaxTokens = 2048
		} else {
			model_request.MaxTokens = *g.config.MaxOutputTokens
		}

		payload, err := json.Marshal(model_request)
		if err != nil {
			v.Err = err
			return
		}

		r, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
		if err != nil {
			v.Err = err
			return
		}
		r.Header.Set("Content-Type", "application/json")

		if err := g.client.authHandler(r); err != nil {
			v.Err = err
			return
		}

		resp, err := g.client.httpClient.Do(r)
		if err != nil {
			v.Err = err
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			v.Err = getErrorByStatus(resp.StatusCode)
			return
		}

		br := bufio.NewScanner(resp.Body)
		var parser fastjson.Parser
		var response anthropicCreateMessagesResponse

	L:
		for {
			select {
			case <-ctx.Done():
				v.Err = ctx.Err()
				return
			default:
			}

			if !br.Scan() {
				return
			}

			line := br.Bytes()
			if len(line) == 0 {
				continue L // skip empty lines
			}

			if bytes.HasPrefix(line, _sse_Event) {
				continue L
			} else if bytes.HasPrefix(line, _sse_Data) {
				line = line[len(_sse_Data):]
				if len(line) == 0 {
					continue L
				}

				ae, err := parser.ParseBytes(line)
				if err != nil {
					v.Err = err
					return
				}

				switch string(ae.Get("type").GetStringBytes()) {
				case "ping":
					// {
					// 	"type":"ping"
					// }
				case "message_start":
					// {
					// 	"type":"message_start",
					// 	"message":{
					// 		"id":"msg_1nZdL29xx5MUA1yADyHTEsnR8uuvGzszyY",
					// 		"type":"message",
					// 		"role":"assistant",
					// 		"content":[
					//
					// 		],
					// 		"model":"claude-3-opus-20240229",
					// 		"stop_reason":null,
					// 		"stop_sequence":null,
					// 		"usage":{
					// 			"input_tokens":25,
					// 			"output_tokens":1
					// 		}
					// 	}
					// }

					message := ae.Get("message")
					response.ID = string(message.Get("id").GetStringBytes())
					response.Type = string(message.Get("type").GetStringBytes())
					response.Role = anthropicRole(message.Get("role").GetStringBytes())
					response.Model = string(message.Get("model").GetStringBytes())
					response.StopReason = string(message.Get("stop_reason").GetStringBytes())
					response.StopSequence = string(message.Get("stop_sequence").GetStringBytes())

					if response.Usage == nil {
						response.Usage = new(anthropicUsage)
					}
					response.Usage.InputTokens += message.Get("usage").Get("input_tokens").GetInt()
					response.Usage.OutputTokens += message.Get("usage").Get("output_tokens").GetInt()

					for _, content := range ae.GetArray("content") {
						var c anthropicSegment
						anthropicMapContent(content, &c)
						response.Content = append(response.Content, c)
					}
				case "message_stop":
					// {
					// 	"type":"message_stop"
					// }
					break L
				case "message_delta":
					// {
					// 	"type":"message_delta",
					// 	"delta":{
					// 		"stop_reason":"end_turn",
					// 		"stop_sequence":null
					// 	},
					// 	"usage":{
					// 		"output_tokens":15
					// 	}
					// }

					delta := ae.Get("delta")
					if delta.Get("stop_reason") != nil {
						response.StopReason = string(delta.Get("stop_reason").GetStringBytes())
					}
					if delta.Get("stop_sequence") != nil {
						response.StopSequence = string(delta.Get("stop_sequence").GetStringBytes())
					}

					if response.Usage == nil {
						response.Usage = new(anthropicUsage)
					}
					response.Usage.InputTokens += ae.Get("usage").Get("input_tokens").GetInt()
					response.Usage.OutputTokens += ae.Get("usage").Get("output_tokens").GetInt()

				case "content_block_start":
					// {
					// 	"type":"content_block_start",
					// 	"index":0,
					// 	"content_block":{
					// 		"type":"text",
					// 		"text":""
					// 	}
					// }
					//
					// {
					// 	"type":"content_block_start",
					// 	"index":1,
					// 	"content_block":{
					// 	 	"type":"tool_use",
					// 	 	"id":"toolu_01T1x1fJ34qAmk2tNTrN7Up6",
					// 	 	"name":"get_weather",
					// 	 	"input":{}
					// 	}
					// }

					content_block := ae.Get("content_block")
					var c anthropicSegment
					anthropicMapContent(content_block, &c)

					index, err := ae.Get("index").Int()
					if err != nil {
						v.Err = err
						return
					}
					if index != len(response.Content) {
						v.Err = llm.ErrInvalidResponse
						return
					}
					response.Content = append(response.Content, c)

					if c.Type == anthropicSegmentText && len(c.Text) > 0 {
						select {
						case stream <- llm.Text(c.Text):
						case <-ctx.Done():
							v.Err = ctx.Err()
							return
						}
					}
				case "content_block_delta":
					// {
					// 	"type":"content_block_delta",
					// 	"index":0,
					// 	"delta":{
					// 		"type":"text_delta",
					// 		"text":"Hello"
					// 	}
					// }
					//
					// {
					//    "type":"content_block_delta",
					//    "index":1,
					//    "delta":{
					//       "type":"input_json_delta",
					//       "partial_json":"{\"location\":"
					//    }
					// }

					delta := ae.Get("delta")
					index, err := ae.Get("index").Int()
					if err != nil {
						v.Err = err
						return
					}

					if index < 0 || index >= len(response.Content) {
						v.Err = llm.ErrInvalidResponse
						return
					}

					var c anthropicSegment
					anthropicMapContent(delta, &c)
					switch c.Type {
					case anthropicSegmentTextDelta:
						if len(c.Text) > 0 {
							response.Content[index].Text += c.Text
							select {
							case stream <- llm.Text(c.Text):
							case <-ctx.Done():
								v.Err = ctx.Err()
								return
							}
						}
					case anthropicSegmentInputJSONDelta:
						if len(c.InputJSON) > 0 {
							response.Content[index].InputJSON = append(response.Content[index].InputJSON, c.InputJSON...)
						}
					default:
						response.Content = append(response.Content, c)
					}
				case "content_block_stop":
					// {
					// 	"type":"content_block_stop",
					// 	"index":0
					// }

					index, err := ae.Get("index").Int()
					if err != nil {
						v.Err = err
						return
					}

					if index < 0 || index >= len(response.Content) {
						v.Err = llm.ErrInvalidResponse
						return
					}

					switch response.Content[index].Type {
					case anthropicSegmentToolUse:
						err := json.Unmarshal(response.Content[index].InputJSON, &response.Content[index].Input)
						if err != nil {
							v.Err = err
							return
						}

						select {
						case stream <- &llm.FunctionCall{
							Name: response.Content[index].Name,
							ID:   response.Content[index].ID,
							Args: response.Content[index].Input,
						}:
						case <-ctx.Done():
							v.Err = ctx.Err()
							return
						}
					}
				case "error":
					// {
					// 	"error":{
					// 		"type":"overloaded_error",
					// 		"message":"Overloaded"
					// 	}
					// }

					err_o := ae.Get("error")
					err_t := string(err_o.Get("type").GetStringBytes())
					v.Err = getErrorByType(err_t)
					return
				}
			}
		}

		v.Content = convertAnthropicContent(response)
		v.Content.Parts = llmutils.Normalize(v.Content.Parts)
		v.FinishReason = convertAnthropicFinishReason(response.StopReason)
		if response.Usage != nil {
			v.UsageData = &llm.UsageData{
				InputTokens:  response.Usage.InputTokens,
				OutputTokens: response.Usage.OutputTokens,
				TotalTokens:  response.Usage.InputTokens + response.Usage.OutputTokens,
			}
		}
	}()

	return v
}

func anthropicMapContent(content *fastjson.Value, c *anthropicSegment) {
	c.Type = anthropicSegmentType(content.Get("type").GetStringBytes())
	switch c.Type {
	case anthropicSegmentText:
		c.Text = string(content.Get("text").GetStringBytes())
	case anthropicSegmentTextDelta:
		c.Text = string(content.Get("text").GetStringBytes())
	case anthropicSegmentImage:
		c.Source.Type = string(content.Get("source", "type").GetStringBytes())
		c.Source.MediaType = string(content.Get("source", "media_type").GetStringBytes())
		c.Source.Data = string(content.Get("source", "data").GetStringBytes())
	case anthropicSegmentToolUse:
		c.Name = string(content.Get("name").GetStringBytes())
		c.ID = string(content.Get("id").GetStringBytes())
	case anthropicSegmentInputJSONDelta:
		c.InputJSON = content.Get("partial_json").GetStringBytes()
	}
}

func convertContentAnthropic(s *llm.Content) anthropicMessage {
	var m anthropicMessage

	switch s.Role {
	case llm.RoleUser:
		m.Role = anthropicRoleUser
	case llm.RoleModel:
		m.Role = anthropicRoleAssistant
	case llm.RoleFunc:
		m.Role = anthropicRoleUser
	default:
		m.Role = anthropicRoleUser
	}

	for i := range s.Parts {
		var a anthropicSegment
		switch v := s.Parts[i].(type) {
		case llm.Text:
			a.Type = anthropicSegmentText
			a.Text = string(v)
		case *llm.InlineData:
			a.Type = anthropicSegmentImage
			a.Source = &anthropicFileData{
				Type:      "base64",
				MediaType: v.MIMEType,
				Data:      base64.StdEncoding.EncodeToString(v.Data),
			}
		case *llm.FunctionCall:
			a.Type = anthropicSegmentToolUse
			a.Name = v.Name
			a.ID = v.ID
			a.Input = v.Args
		case *llm.FunctionResponse:
			a.Type = anthropicSegmentToolResult
			a.ToolUseID = v.ID
			a.IsError = v.IsError

			jsond, err := json.Marshal(v.Content)
			if err != nil {
				jsond = []byte("{\"error\": \"RPCError: Failed to serialize response (HTTP 500)\"}")
				a.IsError = true
			}
			a.Content = []anthropicSegment{{Type: anthropicSegmentText, Text: string(jsond)}}
		}

		m.Content = append(m.Content, a)
	}

	return m
}

func convertAnthropicContent(chat anthropicCreateMessagesResponse) *llm.Content {
	var role llm.Role
	switch chat.Role {
	case anthropicRoleUser:
		role = llm.RoleUser
	case anthropicRoleAssistant:
		role = llm.RoleModel
	default:
		role = llm.RoleUser
	}
	var parts []llm.Segment

L:
	for i := range chat.Content {
		var a llm.Segment

		switch chat.Content[i].Type {
		case anthropicSegmentText, anthropicSegmentTextDelta:
			a = llm.Text(chat.Content[i].Text)
		case anthropicSegmentToolUse:
			a = &llm.FunctionCall{
				ID:   chat.Content[i].ID,
				Name: chat.Content[i].Name,
				Args: chat.Content[i].Input,
			}
		default:
			continue L
		}

		parts = append(parts, a)
	}

	return &llm.Content{
		Role:  role,
		Parts: parts,
	}
}

func convertContextAnthropic(c *llm.ChatContext) []anthropicMessage {
	var contents []anthropicMessage = make([]anthropicMessage, len(c.Contents))

	for i := range c.Contents {
		contents[i] = convertContentAnthropic(c.Contents[i])
	}

	return contents
}

func convertToolsAnthropic(c []*llm.FunctionDeclaration) []anthropicTool {
	var tools []anthropicTool = make([]anthropicTool, len(c))

	for i := range c {
		tools[i] = anthropicTool{
			Name:        c[i].Name,
			Description: c[i].Description,
			InputSchema: c[i].Schema,
		}
	}

	return tools
}

func convertAnthropicFinishReason(stop_reason string) llm.FinishReason {
	switch stop_reason {
	case "end_turn":
		return llm.FinishReasonStop
	case "max_tokens":
		return llm.FinishReasonMaxTokens
	case "stop_sequence":
		return llm.FinishReasonStop
	case "tool_use":
		return llm.FinishReasonToolUse
	}

	return llm.FinishReasonUnknown
}

func ptrify[T any](v T) *T {
	return &v
}

var defaultAnthropicConfig = &llm.Config{
	MaxOutputTokens: ptrify(2048),
}

var _ provider.LLMClient = (*anthropicClient)(nil)

type anthropicClient struct {
	client *anthropicAPIClient
}

func (*anthropicClient) Close() error {
	return nil
}

func (g *anthropicClient) NewLLM(model string, config *llm.Config) (llm.Model, error) {
	if config == nil {
		config = defaultAnthropicConfig
	}

	var _vm = &anthropicModel{
		client: g.client,
		model:  model,
		config: config,
	}

	return _vm, nil
}

var _ provider.LLMProvider = Provider

type AnthropicProvider struct {
}

var (
	ErrAPIKeyRequired error = errors.New("api key is required")
)

func (AnthropicProvider) NewLLMClient(ctx context.Context, configs ...pconf.Config) (provider.LLMClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	apiKey := client_config.APIKey

	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}

	_anthropicClient, err := newClient(apiKey)
	if err != nil {
		return nil, err
	}

	if client_config.BaseURL != "" {
		_anthropicClient.baseURL = client_config.BaseURL
	}

	return &anthropicClient{
		client: _anthropicClient,
	}, nil
}

const ProviderName = "anthropic"

var Provider AnthropicProvider

func init() {
	var exists bool
	for _, n := range coord.ListLLMProviders() {
		if n == ProviderName {
			exists = true
			break
		}
	}
	if !exists {
		coord.RegisterLLMProvider(ProviderName, Provider)
	}
}

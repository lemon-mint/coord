package anthropic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/lemon-mint/vermittlungsstelle/llm"

	"github.com/valyala/fastjson"
)

var _ llm.LLM = (*AnthropicModel)(nil)

type AnthropicModel struct {
	client *AnthropicClient
	config *llm.Config
	model  string
}

func (g *AnthropicModel) Name() string {
	return g.model
}

func (g *AnthropicModel) Close() error {
	return nil
}

var (
	_sse_Event = []byte("event: ")
	_sse_Data  = []byte("data: ")
)

func (g *AnthropicModel) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Stream: stream,
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
			SystemPrompt:  g.config.SystemInstruction,
			StopSequences: g.config.StopSequences,
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

		resp, err := anthropicHTTPClient.Do(r)
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
					response.Usage.InputTokens += message.Get("input_tokens").GetInt()
					response.Usage.OutputTokens += message.Get("output_tokens").GetInt()

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
					response.Usage.InputTokens += delta.Get("input_tokens").GetInt()
					response.Usage.OutputTokens += delta.Get("output_tokens").GetInt()
				case "content_block_start":
					// {
					// 	"type":"content_block_start",
					// 	"index":0,
					// 	"content_block":{
					// 		"type":"text",
					// 		"text":""
					// 	}
					// }

					content_block := ae.Get("content_block")
					var c anthropicSegment
					anthropicMapContent(content_block, &c)
					response.Content = append(response.Content, c)
					if c.Type == "text" && len(c.Text) > 0 {
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

					delta := ae.Get("delta")
					var c anthropicSegment
					anthropicMapContent(delta, &c)
					switch c.Type {
					case anthropicSegmentTextDelta:
						if len(response.Content) > 0 && response.Content[len(response.Content)-1].Type == anthropicSegmentText {
							response.Content[len(response.Content)-1].Text += c.Text
						} else {
							response.Content = append(response.Content, c)
						}

						if len(c.Text) > 0 {
							select {
							case stream <- llm.Text(c.Text):
							case <-ctx.Done():
								v.Err = ctx.Err()
								return
							}
						}
					case anthropicSegmentToolUse:
						// TODO: Handle tool_use on streaming
					default:
						response.Content = append(response.Content, c)
					}
				case "content_block_stop":
					// {
					// 	"type":"content_block_stop",
					// 	"index":0
					// }
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
	case anthropicSegmentToolResult:
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
			a.Input = v.Args
			a.ID = v.ID
		case *llm.FunctionResponse:
			a.Type = anthropicSegmentToolResult
			a.Name = v.Name
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
		case anthropicSegmentText:
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

func NewAnthropicModel(client *AnthropicClient, model string, config *llm.Config) *AnthropicModel {
	if config == nil {
		config = defaultAnthropicConfig
	}

	return &AnthropicModel{
		client: client,
		model:  model,
		config: config,
	}
}

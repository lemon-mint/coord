package softcall

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/lemon-mint/coord/internal/callid"
	"github.com/lemon-mint/coord/internal/llmutils"
	"github.com/lemon-mint/coord/llm"

	yaml "github.com/goccy/go-yaml"
)

type SoftCallConfig struct {
	PreserveReasoning bool
}

type YAMLSoftCallLLM struct {
	upstream llm.Model
	config   SoftCallConfig
}

var defaultConfig = &SoftCallConfig{
	PreserveReasoning: false,
}

func NewYAMLSoftCallLLM(upstream llm.Model, config *SoftCallConfig) *YAMLSoftCallLLM {
	if config == nil {
		config = defaultConfig
	}

	return &YAMLSoftCallLLM{upstream: upstream, config: *config}
}

var _ llm.Model = (*YAMLSoftCallLLM)(nil)

const yamlPrompt = `Here are the tools available for you to use in answering the question:

%s
To call a tool, use a <tool_call> block like this:

<tool_call>
name: |-
	function_name
parameters:
  arg0: 42
  arg1: |
    print("Hello, World!")
</tool_call>

You can use one or more <tool_call> blocks to call tools as needed before providing your final answer. Make sure to only call one tool per <tool_call> block.

First, perform any necessary reasoning in a <reasoning> block. If at any point during your reasoning you need to use a tool, call it with a <tool_call> block, carefully following the JSON schema provided for that tool in the <tools> section above.

You must wait for the user to provide <tool_response> before providing a final response.

After you have finished all reasoning and tool usage, provide your final answer to the question for the user. There is no need to use any special formatting for your final answer.

Always use YAML literal style when representing strings in YAML.`

const yamlPromptResonse = `<reasoning>I should follow the instructions above.</reasoning>

I will follow the instructions.`

const yamlTestFunction = `Call test_function0 with apple = "1" arg.`

const yamlTestFunctionResonse = `<reasoning>I should call the test_function0 that the user requested.</reasoning>

<tool_call>
name: |-
	test_function0
parameters:
	apple: |-
		1
</tool_call>`

const yamlTestToolResult = `<tool_response>
exit_code: 0
</tool_response>`

const yamlTestToolResultResonse = `<reasoning>I should return the exit code of 0.</reasoning>

The test_function0 exited with exit code 0.`

func yamlFuncDecl(tools []*llm.FunctionDeclaration) string {
	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("<tools>\n")
	for _, tool := range tools {
		data, err := yaml.MarshalWithOptions(tool,
			yaml.Indent(2),
			yaml.UseLiteralStyleIfMultiline(true),
		)
		if err != nil {
			continue
		}

		sb.WriteString("<tool>\n")
		sb.WriteString(string(data))
		sb.WriteString("</tool>\n")
	}
	sb.WriteString("</tools>\n")

	return sb.String()
}

func convertToYAMLContent(content *llm.Content) *llm.Content {
	if content == nil {
		return nil
	}

	var toolCallExists bool
L:
	for i := range content.Parts {
		switch content.Parts[i].(type) {
		case *llm.FunctionCall:
			toolCallExists = true
			break L
		case *llm.FunctionResponse:
			toolCallExists = true
			break L
		}
	}

	if !toolCallExists {
		return content
	}

	for i := range content.Parts {
		switch v := content.Parts[i].(type) {
		case *llm.FunctionCall:
			var sb strings.Builder

			var call yamlCall
			call.Name = v.Name
			call.Parameters = v.Args

			data, err := yaml.MarshalWithOptions(&call,
				yaml.Indent(2),
				yaml.UseLiteralStyleIfMultiline(true),
			)
			if err != nil {
				data = []byte("name: tool_call_failed\n")
			}

			sb.WriteString("\n\n<tool_call>\n")
			sb.WriteString(string(data))
			sb.WriteString("\n</tool_call>")

			content.Parts[i] = llm.Text(sb.String())
		case *llm.FunctionResponse:
			var sb strings.Builder

			data, err := json.Marshal(v.Content)
			if err != nil {
				data = []byte("error: |-\n  RPCError: Failed to marshal the args (HTTP 500)")
			}

			sb.WriteString("\n\n<tool_response>\n")
			sb.WriteString(string(data))
			sb.WriteString("\n</tool_response>")

			content.Parts[i] = llm.Text(sb.String())
		}
	}

	return content
}

func min(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

type yamlCall struct {
	Name       string                 `yaml:"name"`
	Parameters map[string]interface{} `yaml:"parameters"`
}

func (g *YAMLSoftCallLLM) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	toolsDecl := yamlFuncDecl(chat.Tools)
	if toolsDecl == "" {
		return g.upstream.GenerateStream(ctx, chat, input)
	}

	messages := make([]*llm.Content, 0, len(chat.Contents)+6)

	messages = append(messages, &llm.Content{
		Role: llm.RoleUser,
		Parts: []llm.Segment{
			llm.Text(fmt.Sprintf(yamlPrompt, toolsDecl)),
		},
	})

	messages = append(messages, &llm.Content{
		Role: llm.RoleModel,
		Parts: []llm.Segment{
			llm.Text(yamlPromptResonse),
		},
	})

	messages = append(messages, &llm.Content{
		Role: llm.RoleUser,
		Parts: []llm.Segment{
			llm.Text(yamlTestFunction),
		},
	})

	messages = append(messages, &llm.Content{
		Role: llm.RoleModel,
		Parts: []llm.Segment{
			llm.Text(yamlTestFunctionResonse),
		},
	})

	messages = append(messages, &llm.Content{
		Role: llm.RoleUser,
		Parts: []llm.Segment{
			llm.Text(yamlTestToolResult),
		},
	})

	messages = append(messages, &llm.Content{
		Role: llm.RoleModel,
		Parts: []llm.Segment{
			llm.Text(yamlTestToolResultResonse),
		},
	})

	for i := range chat.Contents {
		content := convertToYAMLContent(chat.Contents[i])
		content.Parts = llmutils.Normalize(content.Parts)
		messages = append(messages, content)
	}

	chat = &llm.ChatContext{
		Contents: messages,
	}
	input = convertToYAMLContent(input)

	stream := make(chan llm.Segment, 128)
	v := &llm.StreamContent{
		Content: &llm.Content{
			Role: llm.RoleModel,
		},
		Stream: stream,
	}

	go func() {
		defer close(stream)
		defer func() {
			v.Content.Parts = llmutils.Normalize(v.Content.Parts)
			if v.FinishReason == llm.FinishReasonStop {
				for i := range v.Content.Parts {
					if v.Content.Parts[i].Type() == llm.SegmentTypeFunctionCall {
						v.FinishReason = llm.FinishReasonToolUse
						break
					}
				}
			}

			var newParts []llm.Segment
			for i := range v.Content.Parts {
				if v.Content.Parts[i].Type() == llm.SegmentTypeText {
					str := string(v.Content.Parts[i].(llm.Text))
					if strings.TrimSpace(str) == "" {
						continue
					}
					newParts = append(newParts, v.Content.Parts[i])
				} else {
					newParts = append(newParts, v.Content.Parts[i])
				}
			}

			v.Content.Parts = newParts
		}()

		var head strings.Builder
		var hold bool
		var reasoning_block bool
		var tool_call_block bool

		resp := g.upstream.GenerateStream(ctx, chat, input)
		for seg := range resp.Stream {
			switch s := seg.(type) {
			case llm.Text:
				head.WriteString(string(s))

				if !hold {
					if strings.Contains(head.String(), "<") {
						hold = true
					} else {
						hold = false
					}
				}

				if hold {
				L:
					for {
						idx := strings.Index(head.String(), "<")
						if idx < 0 {
							hold = false
							break
						}

						if idx > 0 {
							payload := llm.Text(head.String()[:idx])
							v.Content.Parts = append(v.Content.Parts, payload)
							select {
							case stream <- payload:
							case <-ctx.Done():
								return
							}
							new_head := head.String()[idx:]
							head.Reset()
							head.WriteString(new_head)
						}

						switch {
						case !g.config.PreserveReasoning && strings.HasPrefix("<reasoning>", head.String()[:min(len("<reasoning>"), head.Len())]):
							reasoning_block = true
						case strings.HasPrefix("<tool_call>", head.String()[:min(len("<tool_call>"), head.Len())]):
							tool_call_block = true
						default:
							payload := llm.Text(head.String()[:1])
							v.Content.Parts = append(v.Content.Parts, payload)
							select {
							case stream <- payload:
							case <-ctx.Done():
								return
							}
							new_head := head.String()[1:]
							head.Reset()
							head.WriteString(new_head)
							continue L
						}

						if reasoning_block {
							idx := strings.Index(head.String(), "</reasoning>")
							if idx < 0 {
								break
							}
							resoning_process := head.String()[:idx+len("<reasoning>")+1]
							_ = resoning_process
							reasoning_block = false
							new_head := head.String()[idx+len("<reasoning>")+1:]
							head.Reset()
							head.WriteString(new_head)
						}

						if tool_call_block {
							idx := strings.Index(head.String(), "</tool_call>")
							if idx < 0 {
								break
							}
							tool_call_body := head.String()[:idx+len("<tool_call>")+1]
							reasoning_block = false
							new_head := head.String()[idx+len("<tool_call>")+1:]
							head.Reset()
							head.WriteString(new_head)

							var call yamlCall
							err := yaml.Unmarshal(
								[]byte(strings.TrimSuffix(strings.TrimPrefix(tool_call_body, "<tool_call>"), "</tool_call>")),
								&call,
							)
							if err != nil {
								payload := llm.Text(tool_call_body)
								v.Content.Parts = append(v.Content.Parts, payload)
								select {
								case stream <- payload:
								case <-ctx.Done():
									return
								}
								break
							}

							payload := &llm.FunctionCall{
								Name: strings.TrimSpace(call.Name),
								ID:   callid.OpenAICallID(),
								Args: call.Parameters,
							}
							v.Content.Parts = append(v.Content.Parts, payload)
							select {
							case stream <- payload:
							case <-ctx.Done():
								return
							}
						}
					}
				}

				if !hold {
					payload := llm.Text(head.String())
					v.Content.Parts = append(v.Content.Parts, payload)
					select {
					case stream <- payload:
						head.Reset()
					case <-ctx.Done():
						return
					}
				}
			}
		}

		if head.Len() > 0 {
			payload := llm.Text(head.String())
			v.Content.Parts = append(v.Content.Parts, payload)
			select {
			case stream <- payload:
			case <-ctx.Done():
				return
			}
		}

		v.Err = resp.Err
		v.UsageData = resp.UsageData
		v.FinishReason = resp.FinishReason
	}()

	return v
}

func (g *YAMLSoftCallLLM) Close() error {
	return g.upstream.Close()
}

func (g *YAMLSoftCallLLM) Name() string {
	return g.upstream.Name()
}

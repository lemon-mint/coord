package softcall

import (
	"context"
	"fmt"
	"strings"

	"github.com/lemon-mint/coord/llm"

	yaml "github.com/goccy/go-yaml"
)

type YAMLSoftCallLLM struct {
	upstream llm.LLM
}

func NewYAMLSoftCallLLM(upstream llm.LLM) *YAMLSoftCallLLM {
	return &YAMLSoftCallLLM{upstream: upstream}
}

var _ llm.LLM = (*YAMLSoftCallLLM)(nil)

const yamlPrompt = `Here are the tools available for you to use in answering the question:

%s
To call a tool, use a <tool_call> block like this:

<tool_call>
name: 'function_name'
parameters:
  arg0: 42
  arg1: >
    print("Hello, World!")
</tool_call>

You can use one or more <tool_call> blocks to call tools as needed before providing your final answer. Make sure to only call one tool per <tool_call> block.

First, perform any necessary reasoning in a <reasoning> block. If at any point during your reasoning you need to use a tool, call it with a <tool_call> block, carefully following the JSON schema provided for that tool in the <tools> section above.

You must wait for the user to provide <tool_response> before providing a final response.

After you have finished all reasoning and tool usage, provide your final answer to the question for the user. There is no need to use any special formatting for your final answer.`

const yamlPromptResonse = `<reasoning>I should follow the instructions above.</reasoning>

I will follow the instructions.`

const yamlTestFunction = `Call test_function0 with apple = "1" arg.`

const yamlTestFunctionResonse = `<reasoning>I should call the test_function0 that the user requested.</reasoning>

<tool_call>
name: 'test_function0'
parameters:
	apple: "1"
</tool_call>`

const yamlTestToolResult = `<tool_response>
exit_code: 0
</tool_response>`

const yamlTestToolResultResonse = `<reasoning>I should return the exit code of 0.</reasoning>

the test_function0 exited with exit code 0.`

func yamlFuncDecl(tools []*llm.FunctionDeclaration) string {
	if len(tools) == 0 {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("<tools>\n")
	for _, tool := range tools {
		data, err := yaml.Marshal(tool)
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

			data, err := yaml.Marshal(v.Args)
			if err != nil {
				data = []byte("name: tool_call_failed\n")
			}

			sb.WriteString("\n\n<tool_call>\n")
			sb.WriteString(string(data))
			sb.WriteString("\n</tool_call>")

			content.Parts[i] = llm.Text(sb.String())
		case *llm.FunctionResponse:
			var sb strings.Builder

			data, err := yaml.Marshal(v.Content)
			if err != nil {
				data = []byte("error: >\n  RPCError: Failed to marshal the args (HTTP 500)")
			}

			sb.WriteString("\n\n<tool_response>\n")
			sb.WriteString(string(data))
			sb.WriteString("\n</tool_response>")
		}
	}

	return content
}

func (y *YAMLSoftCallLLM) GenerateStream(ctx context.Context, chat *llm.ChatContext, input *llm.Content) *llm.StreamContent {
	toolsDecl := yamlFuncDecl(chat.Tools)
	if toolsDecl == "" {
		return y.upstream.GenerateStream(ctx, chat, input)
	}

	messages := make([]*llm.Content, 0, len(chat.Contents))

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
		messages = append(messages, convertToYAMLContent(chat.Contents[i]))
	}

	chat = &llm.ChatContext{
		Contents: messages,
	}
	input = convertToYAMLContent(input)

	return y.upstream.GenerateStream(ctx, chat, input)
}

func (y *YAMLSoftCallLLM) Close() error {
	return y.upstream.Close()
}

func (y *YAMLSoftCallLLM) Name() string {
	return y.upstream.Name()
}

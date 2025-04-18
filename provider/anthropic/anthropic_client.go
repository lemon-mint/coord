package anthropic

import (
	"net/http"
	"strings"
	"time"

	"github.com/lemon-mint/coord/internal/useragent"
	"github.com/lemon-mint/coord/llm"
)

var UserAgent *string = ptrify(useragent.HTTPUserAgent)

type anthropicRole string

const (
	anthropicRoleAssistant anthropicRole = "assistant"
	anthropicRoleUser      anthropicRole = "user"
)

type anthropicMessage struct {
	Role    anthropicRole      `json:"role"`
	Content []anthropicSegment `json:"content"`
}

type anthropicSegmentType string

const (
	anthropicSegmentText             anthropicSegmentType = "text"
	anthropicSegmentTextDelta        anthropicSegmentType = "text_delta"
	anthropicSegmentThinking         anthropicSegmentType = "thinking"
	anthropicSegmentRedactedThinking anthropicSegmentType = "redacted_thinking"
	anthropicSegmentThinkingDelta    anthropicSegmentType = "thinking_delta"
	anthropicSegmentSignatureDelta   anthropicSegmentType = "signature_delta"
	anthropicSegmentInputJSONDelta   anthropicSegmentType = "input_json_delta"
	anthropicSegmentImage            anthropicSegmentType = "image"
	anthropicSegmentToolUse          anthropicSegmentType = "tool_use"
	anthropicSegmentToolResult       anthropicSegmentType = "tool_result"
)

type anthropicSegment struct {
	Type anthropicSegmentType `json:"type"` // "text", "image", "tool_use", "tool_result"

	Text string `json:"text,omitempty"` // text content for text

	Thinking  string `json:"thinking,omitempty"`  // thinking content for thinking
	Signature string `json:"signature,omitempty"` // signature content for signature

	RedactedThinking string `json:"redacted_thinking,omitempty"` // redacted thinking content for redacted_thinking

	Source *anthropicFileData `json:"source,omitempty"` // file data for image

	ID        string                 `json:"id,omitempty"`    // id for tool_use
	Name      string                 `json:"name,omitempty"`  // name for tool_use
	InputJSON []byte                 `json:"-"`               // raw json input data for tool_use
	Input     map[string]interface{} `json:"input,omitempty"` // input data for tool_use

	ToolUseID string             `json:"tool_use_id,omitempty"` // id for tool_result
	Content   []anthropicSegment `json:"content,omitempty"`     // nested segments for tool_result
	IsError   bool               `json:"is_error,omitempty"`    // true if the file is an error (used for tool_result)
}

type anthropicFileData struct {
	Type      string `json:"type,omitempty"`       // "base64"
	MediaType string `json:"media_type,omitempty"` // "image/jpeg", "image/png", "image/gif", "image/webp"
	Data      string `json:"data,omitempty"`       // base64-encoded image data
}

type anthropicTool struct {
	Name        string      `json:"name"`         // Name of the tool
	Description string      `json:"description"`  // Description of the tool
	InputSchema *llm.Schema `json:"input_schema"` // Input schema for the tool
}

type anthropicThinking struct {
	Type         string `json:"type"`          // "enabled" or "disabled"
	BudgetTokens int    `json:"budget_tokens"` // 16000
}

type anthropicCreateMessagesRequest struct {
	AnthropicVersion string `json:"anthropic_version,omitempty"` // Anthropic API version

	Model     string             `json:"model"`      // Name of the Anthropic model to use
	Messages  []anthropicMessage `json:"messages"`   // List of messages to send to the model
	MaxTokens int                `json:"max_tokens"` // Maximum number of tokens to generate

	SystemPrompt  string                           `json:"system,omitempty"`         // System prompt for the model
	MetaData      *anthropicCreateMessagesMetaData `json:"metadata,omitempty"`       // Metadata for the request
	StopSequences []string                         `json:"stop_sequences,omitempty"` // List of stop sequences for the model

	Thinking *anthropicThinking `json:"thinking,omitempty"` // Thinking configuration
	Tools    []anthropicTool    `json:"tools,omitempty"`    // List of tools to use in the conversation

	Temperature *float32 `json:"temperature,omitempty"` // Temperature parameter for the model
	TopP        *float32 `json:"top_p,omitempty"`       // Top-p parameter for the model
	TopK        *int     `json:"top_k,omitempty"`       // Top-k parameter for the model

	Stream bool `json:"stream"` // Stream responses

}

type anthropicCreateMessagesMetaData struct {
	UserID string `json:"user_id"` // User ID for the request
}

type anthropicStopReason string

const (
	StopEndTurn   anthropicStopReason = "end_turn"
	StopMaxTokens anthropicStopReason = "max_tokens"
	StopSequence  anthropicStopReason = "stop_sequence"
	StopToolUse   anthropicStopReason = "tool_use"
)

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicCreateMessagesResponse struct {
	ID   string        `json:"id"`
	Type string        `json:"type"`
	Role anthropicRole `json:"role"`

	Model        string             `json:"model"`
	StopSequence string             `json:"stop_sequence,omitempty"`
	Content      []anthropicSegment `json:"content"`
	StopReason   string             `json:"stop_reason,omitempty"`
	Usage        *anthropicUsage    `json:"usage"`
	Stream       bool               `json:"stream"`
}

var anthropicHTTPClient *http.Client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:    16,
		IdleConnTimeout: 30 * time.Second,
	},
}

type anthropicAPIClient struct {
	baseURL     string
	authHandler func(r *http.Request) error

	httpClient *http.Client
}

const anthropicBaseURL = "https://api.anthropic.com/v1"

func newClient(apikey string) (*anthropicAPIClient, error) {
	apikey = strings.TrimSpace(apikey)
	return &anthropicAPIClient{
		baseURL: anthropicBaseURL,
		authHandler: func(r *http.Request) error {
			r.Header.Set("X-API-Key", apikey)
			r.Header.Set("Anthropic-Version", "2023-06-01")
			return nil
		},
		httpClient: anthropicHTTPClient,
	}, nil
}

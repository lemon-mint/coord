package llm

import (
	"context"
)

type UsageData struct {
	InputTokens  int
	OutputTokens int
	TotalTokens  int
}

type Role string

const (
	RoleUser  = Role("user")
	RoleModel = Role("model")
	RoleFunc  = Role("function")
)

type Content struct {
	Role  Role      `json:"role"`
	Parts []Segment `json:"parts"`
}

type Segment interface {
	Segment()
}

type Text string

func (Text) Segment() {}

type InlineData struct {
	MIMEType string `json:"mimeType"`
	Data     []byte `json:"data"`
}

func (*InlineData) Segment() {}

type FunctionCall struct {
	Name string                 `json:"name"`
	ID   string                 `json:"id"`
	Args map[string]interface{} `json:"args"`
}

func (*FunctionCall) Segment() {}

type FunctionResponse struct {
	Name    string      `json:"name"`
	ID      string      `json:"id"`
	Content interface{} `json:"content"`
	IsError bool        `json:"-"`
}

func (*FunctionResponse) Segment() {}

type FunctionDeclaration struct {
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Schema      *Schema `json:"schema"`
}

type ChatContext struct {
	Contents []*Content             `json:"contents"`
	Tools    []*FunctionDeclaration `json:"tools"`
}

type FinishReason string

const (
	FinishReasonUnknown    = FinishReason("unknown")
	FinishReasonError      = FinishReason("error")
	FinishReasonSafety     = FinishReason("safety")
	FinishReasonRecitation = FinishReason("recitation")
	FinishReasonStop       = FinishReason("stop")
	FinishReasonMaxTokens  = FinishReason("max_tokens")
	FinishReasonToolUse    = FinishReason("tool_use")
)

type StreamContent struct {
	Err          error        // Only Available after Stream channel is closed
	Content      *Content     // Only Available after Stream channel is closed
	UsageData    *UsageData   // Only Available after Stream channel is closed (Note: UsageData is not available for all LLM providers)
	FinishReason FinishReason // Only Available after Stream channel is closed

	Stream <-chan Segment // Token Stream
}

type LLM interface {
	GenerateStream(ctx context.Context, chat *ChatContext, input *Content) *StreamContent
	Close() error
	Name() string
}

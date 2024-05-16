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

//go:generate stringer -type=SegmentType -linecomment
type SegmentType uint16

const (
	SegmentTypeUnknown          SegmentType = iota // unknown
	SegmentTypeText                                // text
	SegmentTypeInlineData                          // inline_data
	SegmentTypeFileData                            // file_data
	SegmentTypeFunctionCall                        // function_call
	SegmentTypeFunctionResponse                    // function_response
)

type Segment interface {
	Segment()
	Type() SegmentType
}

type Text string

func (Text) Segment()            {}
func (t Text) Type() SegmentType { return SegmentTypeText }

type InlineData struct {
	MIMEType string `json:"mimeType"`
	Data     []byte `json:"data"`
}

func (*InlineData) Segment()          {}
func (*InlineData) Type() SegmentType { return SegmentTypeInlineData }

type FileData struct {
	MIMEType string `json:"mimeType"`
	FileURI  string `json:"fileUri"`
}

func (*FileData) Segment()          {}
func (*FileData) Type() SegmentType { return SegmentTypeFileData }

type FunctionCall struct {
	Name string                 `json:"name"`
	ID   string                 `json:"id"`
	Args map[string]interface{} `json:"args"`
}

func (*FunctionCall) Segment()          {}
func (*FunctionCall) Type() SegmentType { return SegmentTypeFunctionCall }

type FunctionResponse struct {
	Name    string      `json:"name,omitempty"`
	ID      string      `json:"id,omitempty"`
	Content interface{} `json:"content"`
	IsError bool        `json:"-"`
}

func (*FunctionResponse) Segment()          {}
func (*FunctionResponse) Type() SegmentType { return SegmentTypeFunctionResponse }

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
	Err          error        `json:"error"`        // Only Available after Stream channel is closed
	Content      *Content     `json:"content"`      // Only Available after Stream channel is closed
	UsageData    *UsageData   `json:"usageData"`    // Only Available after Stream channel is closed (Note: UsageData is not available for all LLM providers)
	FinishReason FinishReason `json:"finishReason"` // Only Available after Stream channel is closed

	Stream <-chan Segment `json:"-"` // Token Stream
}

type LLM interface {
	GenerateStream(ctx context.Context, chat *ChatContext, input *Content) *StreamContent
	Close() error
	Name() string
}

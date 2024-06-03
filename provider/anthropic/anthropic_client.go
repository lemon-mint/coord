package anthropic

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lemon-mint/coord/internal/useragent"
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
	anthropicSegmentText       anthropicSegmentType = "text"
	anthropicSegmentTextDelta  anthropicSegmentType = "text_delta"
	anthropicSegmentImage      anthropicSegmentType = "image"
	anthropicSegmentToolUse    anthropicSegmentType = "tool_use"
	anthropicSegmentToolResult anthropicSegmentType = "tool_result"
)

type anthropicSegment struct {
	Type anthropicSegmentType `json:"type"` // "text", "image", "tool_use", "tool_result"

	Text string `json:"text,omitempty"` // text content for text

	Source *anthropicFileData `json:"source,omitempty"` // file data for image

	ID    string                 `json:"id,omitempty"`    // id for tool_use
	Name  string                 `json:"name,omitempty"`  // name for tool_use
	Input map[string]interface{} `json:"input,omitempty"` // input data for tool_use

	ToolUseID string             `json:"tool_use_id,omitempty"` // id for tool_result
	Content   []anthropicSegment `json:"content,omitempty"`     // nested segments for tool_result
	IsError   bool               `json:"is_error,omitempty"`    // true if the file is an error (used for tool_result)
}

type anthropicFileData struct {
	Type      string `json:"type,omitempty"`       // "base64"
	MediaType string `json:"media_type,omitempty"` // "image/jpeg", "image/png", "image/gif", "image/webp"
	Data      string `json:"data,omitempty"`       // base64-encoded image data
}

type anthropicCreateMessagesRequest struct {
	AnthropicVersion string `json:"anthropic_version,omitempty"`

	Model     string             `json:"model"`
	Messages  []anthropicMessage `json:"messages"`
	MaxTokens int                `json:"max_tokens"`

	SystemPrompt  string                           `json:"system,omitempty"`
	MetaData      *anthropicCreateMessagesMetaData `json:"metadata,omitempty"`
	StopSequences []string                         `json:"stop_sequences,omitempty"`

	Temperature *float32 `json:"temperature,omitempty"`
	TopP        *float32 `json:"top_p,omitempty"`
	TopK        *int     `json:"top_k,omitempty"`

	Stream bool `json:"stream"`
}

type anthropicCreateMessagesMetaData struct {
	UserID string `json:"user_id"`
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

func (c *anthropicAPIClient) createMessages(req *anthropicCreateMessagesRequest) (*anthropicCreateMessagesResponse, error) {
	url, err := url.JoinPath(c.baseURL, "./messages")
	if err != nil {
		return nil, err
	}

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	r.Header.Set("Content-Type", "application/json")
	if UserAgent != nil {
		r.Header.Set("User-Agent", *UserAgent)
	}

	if err := c.authHandler(r); err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, getErrorByStatus(resp.StatusCode)
	}

	var mres anthropicCreateMessagesResponse
	if err := json.NewDecoder(resp.Body).Decode(&mres); err != nil {
		return nil, err
	}

	return &mres, nil
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

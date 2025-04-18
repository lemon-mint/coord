package llm

type Config struct {
	Temperature     *float32        `json:"temperature,omitempty"`
	TopP            *float32        `json:"top_p,omitempty"`
	TopK            *int            `json:"top_k,omitempty"`
	MaxOutputTokens *int            `json:"max_output_tokens,omitempty"`
	StopSequences   []string        `json:"stop_sequences,omitempty"`
	ThinkingConfig  *ThinkingConfig `json:"thinking_config,omitempty"`

	SystemInstruction     string         `json:"system_instruction,omitempty"`
	SafetyFilterThreshold BlockThreshold `json:"filter_threshold,omitempty"`
}

type ThinkingConfig struct {
	IncludeThoughts *bool `json:"include_thoughts,omitempty"`
	ThinkingBudget  *int  `json:"thinking_budget,omitempty"`
}

type BlockThreshold uint16

const (
	BlockDefault BlockThreshold = iota
	BlockNone
	BlockLowAndAbove
	BlockMediumAndAbove
	BlockOnlyHigh
	BlockOff
)

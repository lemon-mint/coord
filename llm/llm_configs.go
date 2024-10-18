package llm

type Config struct {
	Temperature     *float32 `json:"temperature"`
	TopP            *float32 `json:"top_p"`
	TopK            *int     `json:"top_k"`
	MaxOutputTokens *int     `json:"max_output_tokens"`
	StopSequences   []string `json:"stop_sequences"`

	SystemInstruction     string         `json:"system_instruction"`
	SafetyFilterThreshold BlockThreshold `json:"filter_threshold"`
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

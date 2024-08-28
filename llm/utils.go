package llm

import (
	"strings"
)

func TextContent(role Role, text string) *Content {
	return &Content{
		Role: role,
		Parts: []Segment{
			Text(text),
		},
	}
}

// Text returns the text content of the segment.
// Note: This function must be called after the Stream channel is closed.
func (g *StreamContent) Text() string {
	if g == nil {
		return ""
	}

	var sb strings.Builder

	for i := range g.Content.Parts {
		if g.Content.Parts[i].Type() == SegmentTypeText {
			sb.WriteString(string(g.Content.Parts[i].(Text)))
		}
	}

	return sb.String()
}

package llm

func TextContent(role Role, text string) *Content {
	return &Content{
		Role: role,
		Parts: []Segment{
			Text(text),
		},
	}
}

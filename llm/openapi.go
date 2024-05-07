package llm

type OpenAPIType string

const (
	OpenAPITypeString  OpenAPIType = "string"
	OpenAPITypeNumber  OpenAPIType = "number"
	OpenAPITypeInteger OpenAPIType = "integer"
	OpenAPITypeBoolean OpenAPIType = "boolean"
	OpenAPITypeArray   OpenAPIType = "array"
	OpenAPITypeObject  OpenAPIType = "object"
)

type Schema struct {
	Type  OpenAPIType `json:"type"`
	Title string      `json:"title,omitempty"`

	Description string             `json:"description,omitempty"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Items       *Schema            `json:"items,omitempty"`
	Required    []string           `json:"required,omitempty"`

	Nullable bool          `json:"nullable,omitempty"`
	Format   string        `json:"format,omitempty"`
	Enum     []interface{} `json:"enum,omitempty"`
	Default  interface{}   `json:"default,omitempty"`
}

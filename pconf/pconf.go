package pconf

import "google.golang.org/api/option"

type GeneralConfig struct {
	APIKey  string
	BaseURL string

	ProjectID string
	Location  string

	UseREST bool

	GoogleClientOptions []option.ClientOption
}

func (GeneralConfig) String() string {
	return "<GeneralConfig [REDACTED]>"
}

type Config interface {
	Apply(g *GeneralConfig) error
}

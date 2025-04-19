package pconf

import (
	"cloud.google.com/go/auth"
	"google.golang.org/api/option"
)

type GeneralConfig struct {
	APIKey  string
	BaseURL string

	ProjectID string
	Location  string

	GoogleCredentials   *auth.Credentials
	GoogleClientOptions []option.ClientOption
}

func (GeneralConfig) String() string {
	return "<GeneralConfig [REDACTED]>"
}

type Config interface {
	Apply(g *GeneralConfig) error
}

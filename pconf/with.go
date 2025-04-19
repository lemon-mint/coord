package pconf

import (
	"cloud.google.com/go/auth"
	"google.golang.org/api/option"
)

var _ Config = (*fnConf)(nil)

type fnConf struct {
	Fn func(g *GeneralConfig) error
}

func (a *fnConf) Apply(g *GeneralConfig) error {
	return a.Fn(g)
}

func WithAPIKey(key string) Config {
	return &fnConf{
		func(g *GeneralConfig) error {
			g.APIKey = key
			return nil
		},
	}
}

func WithBaseURL(url string) Config {
	return &fnConf{
		func(g *GeneralConfig) error {
			g.BaseURL = url
			return nil
		},
	}
}

func WithProjectID(id string) Config {
	return &fnConf{
		func(g *GeneralConfig) error {
			g.ProjectID = id
			return nil
		},
	}
}

func WithLocation(location string) Config {
	return &fnConf{
		func(g *GeneralConfig) error {
			g.Location = location
			return nil
		},
	}
}

func WithGoogleCredentials(cred *auth.Credentials) Config {
	return &fnConf{
		func(g *GeneralConfig) error {
			g.GoogleCredentials = cred
			return nil
		},
	}
}

func WithGoogleClientOptions(opts ...option.ClientOption) Config {
	return &fnConf{
		func(g *GeneralConfig) error {
			g.GoogleClientOptions = opts
			return nil
		},
	}
}

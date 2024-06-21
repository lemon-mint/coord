package elevenlabs

import (
	"context"
	"errors"
	"math/rand"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/tts"
)

// =================== Config ==================

var defaultElevenlabsConfig = &tts.Config{
	Stability:       0.5,
	SimilarityBoost: 0.75,
	Style:           0,
	UseSpeakerBoost: true,
}

// =================== Client ===================

var _ provider.TTSClient = (*ElevenlabsClient)(nil)

type ElevenlabsClient struct {
	client *elevenlabsAPIClient
}

func (g *ElevenlabsClient) NewTTS(model string, config *tts.Config) (tts.Model, error) {
	if config == nil {
		return nil, errors.New("config is nil")
	}

	config.Model = model

	_vm := &elevenlabsModel{
		client: g.client,
		config: config,
	}

	return _vm, nil
}

func (*ElevenlabsClient) Close() error {
	return nil
}

func (*ElevenlabsClient) Name() string {
	return ProviderName
}

func (g *ElevenlabsClient) GetVoiceList(ctx context.Context) ([]Voice, error) {
	voices, err := g.client.RequestVoiceList(ctx)
	if err != nil {
		return nil, err
	}
	return voices, nil
}

// =================== Model ===================

var _ tts.Model = (*elevenlabsModel)(nil)

type elevenlabsModel struct {
	client *elevenlabsAPIClient
	config *tts.Config
}

func (g *elevenlabsModel) GenerateSpeech(ctx context.Context, text string) (*tts.AudioFile, error) {
	reqData := ttsRequest{
		ModelID: g.config.Model,
		Text:    text,
		Seed:    g.config.Seed,
		VoiceSettings: TtsVoiceSettings{
			Stability:       g.config.Stability,
			SimilarityBoost: g.config.SimilarityBoost,
			Style:           g.config.Style,
			UseSpeakerBoost: g.config.UseSpeakerBoost,
		},
	}

	if g.config.Stability == 0 {
		reqData.VoiceSettings.Stability = defaultElevenlabsConfig.Stability
	}

	if g.config.SimilarityBoost == 0 {
		reqData.VoiceSettings.SimilarityBoost = defaultElevenlabsConfig.SimilarityBoost
	}

	if g.config.Style == 0 {
		reqData.VoiceSettings.Style = defaultElevenlabsConfig.Style
	}

	if g.config.Seed == 0 {
		reqData.Seed = rand.Int()
	}

	voice, err := g.client.RequestTTS(ctx, g.config.VoiceID, reqData)
	if err != nil {
		return nil, err
	}

	return &tts.AudioFile{
		Format: tts.FormatMP3,
		Data:   voice,
	}, nil
}

// =================== Provider ===================

var _ provider.TTSProvider = Provider

type ElevenlabsProvider struct{}

func (ElevenlabsProvider) NewTTSClient(ctx context.Context, configs ...pconf.Config) (provider.TTSClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	apiKey := client_config.APIKey

	if apiKey == "" {
		return nil, ErrAPIKeyRequired
	}

	_elevenlabsClient, err := newClient(apiKey)
	if err != nil {
		return nil, err
	}

	return &ElevenlabsClient{
		client: _elevenlabsClient,
	}, nil
}

// ===================== Init =====================

const ProviderName = "elevenlabs"

var Provider ElevenlabsProvider

func init() {
	var exists bool
	for _, n := range coord.ListTTSProviders() {
		if n == ProviderName {
			exists = true
			break
		}
	}
	if !exists {
		coord.RegisterTTSProvider(ProviderName, Provider)
	}
}

package openai

import (
	"context"
	"io"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/tts"
	"github.com/sashabaranov/go-openai"
)

type openAITTS struct {
	client *openai.Client

	model openai.SpeechModel
	voice openai.SpeechVoice
	speed float64

	fmt tts.Format
}

var _ tts.Model = (*openAITTS)(nil)

func (g *openAITTS) GenerateSpeech(ctx context.Context, text string) (*tts.AudioFile, error) {
	var encoding openai.SpeechResponseFormat

	switch g.fmt {
	case tts.FormatMP3:
		encoding = openai.SpeechResponseFormatMp3
	case tts.FormatOGG:
		encoding = openai.SpeechResponseFormatOpus
	case tts.FormatAAC:
		encoding = openai.SpeechResponseFormatAac
	case tts.FormatFLAC:
		encoding = openai.SpeechResponseFormatFlac
	case tts.FormatWAV:
		encoding = openai.SpeechResponseFormatWav
	case tts.FormatLINEAR16:
		encoding = openai.SpeechResponseFormatPcm
	default:
		return nil, tts.ErrUnsupportedFileFormat
	}

	resp, err := g.client.CreateSpeech(ctx, openai.CreateSpeechRequest{
		Model:          g.model,
		Voice:          g.voice,
		Speed:          g.speed,
		ResponseFormat: encoding,
		Input:          text,
	})
	if err != nil {
		return nil, err
	}

	file, err := io.ReadAll(resp)
	if err != nil {
		return nil, err
	}

	return &tts.AudioFile{
		Format: g.fmt,
		Data:   file,
	}, nil
}

var defaultOpenAITTSConfig = &tts.Config{
	Model:        string(openai.VoiceNova),
	SpeakingRate: 1.0,
	Format:       tts.FormatMP3,
}

var _ provider.TTSClient = (*openAIClient)(nil)

func (g *openAIClient) NewTTS(model string, config *tts.Config) (tts.Model, error) {
	if config == nil {
		config = defaultOpenAITTSConfig
	}

	_em := &openAITTS{
		client: g.client,
		voice:  openai.SpeechVoice(config.Model),
		speed:  config.SpeakingRate,
		fmt:    config.Format,
	}

	if _em.fmt == "" {
		_em.fmt = tts.FormatMP3
	}

	return _em, nil
}

var _ provider.TTSProvider = Provider

func (OpenAIProvider) NewTTSClient(ctx context.Context, configs ...pconf.Config) (provider.TTSClient, error) {
	client_config := pconf.GeneralConfig{}
	var openai_config *openai.ClientConfig
	for i := range configs {
		switch v := configs[i].(type) {
		case *openaiConfig:
			openai_config = &v.c
		default:
			configs[i].Apply(&client_config)
		}
	}

	if openai_config == nil {
		if client_config.APIKey == "" {
			return nil, ErrAPIKeyRequired
		}
		v := openai.DefaultConfig(client_config.APIKey)
		openai_config = &v
	}

	if client_config.BaseURL != "" {
		openai_config.BaseURL = client_config.BaseURL
	}

	return &openAIClient{
		client: openai.NewClientWithConfig(*openai_config),
	}, nil
}

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

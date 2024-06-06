package vertexai

import (
	"context"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/tts"
)

type textToSpeechModel struct {
	client *texttospeech.Client

	fmt tts.Format

	language string
	name     string

	speaking_rate float64
	pitch         float64

	sample_rate int32
}

var _ tts.TTS = (*textToSpeechModel)(nil)

func (g *textToSpeechModel) GenerateSpeech(ctx context.Context, text string) (*tts.AudioFile, error) {
	var encoding texttospeechpb.AudioEncoding

	switch g.fmt {
	case tts.FormatLINEAR16:
		encoding = texttospeechpb.AudioEncoding_LINEAR16
	case tts.FormatMP3:
		encoding = texttospeechpb.AudioEncoding_MP3
	case tts.FormatOGG:
		encoding = texttospeechpb.AudioEncoding_OGG_OPUS
	case tts.FormatALAW:
		encoding = texttospeechpb.AudioEncoding_ALAW
	case tts.FormatMULAW:
		encoding = texttospeechpb.AudioEncoding_MULAW
	default:
		return nil, tts.ErrUnsupportedFileFormat
	}

	req := texttospeechpb.SynthesizeSpeechRequest{
		Input: &texttospeechpb.SynthesisInput{
			InputSource: &texttospeechpb.SynthesisInput_Text{Text: text},
		},

		Voice: &texttospeechpb.VoiceSelectionParams{
			LanguageCode: g.language,
			Name:         g.name,
		},

		AudioConfig: &texttospeechpb.AudioConfig{
			AudioEncoding:   encoding,
			SpeakingRate:    g.speaking_rate,
			Pitch:           g.pitch,
			SampleRateHertz: g.sample_rate,
		},
	}

	resp, err := g.client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return nil, err
	}

	return &tts.AudioFile{
		Format: g.fmt,
		Data:   resp.AudioContent,
	}, nil
}

var defaultTextToSpeechConfig = &tts.Config{
	Language:     "en-US",
	Model:        "en-US-Journey-F",
	SpeakingRate: 1.0,
	Pitch:        0,
	Format:       tts.FormatMP3,
}

type textToSpeechClient struct {
	client *texttospeech.Client
}

var _ provider.TTSClient = (*textToSpeechClient)(nil)

func (g *textToSpeechClient) NewTTS(model string, config *tts.Config) (tts.TTS, error) {
	if config == nil {
		config = defaultTextToSpeechConfig
	}

	_em := &textToSpeechModel{
		client:        g.client,
		language:      config.Language,
		name:          config.Model,
		speaking_rate: config.SpeakingRate,
		pitch:         config.Pitch,
		sample_rate:   int32(config.SampleRate),
		fmt:           config.Format,
	}

	if _em.fmt == "" {
		_em.fmt = tts.FormatMP3
	}

	return _em, nil
}

func (g *textToSpeechClient) Close() error {
	return g.client.Close()
}

var _ provider.TTSProvider = (*VertexAIProvider)(nil)

func (VertexAIProvider) NewTTSClient(ctx context.Context, configs ...pconf.Config) (provider.TTSClient, error) {
	client_config := pconf.GeneralConfig{}
	for i := range configs {
		configs[i].Apply(&client_config)
	}

	client_options := client_config.GoogleClientOptions

	client, err := texttospeech.NewClient(ctx, client_options...)
	if err != nil {
		return nil, err
	}

	return &textToSpeechClient{
		client: client,
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

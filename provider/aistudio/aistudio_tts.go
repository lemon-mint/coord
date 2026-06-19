package aistudio

import (
	"context"
	"encoding/binary"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider"
	"github.com/lemon-mint/coord/tts"
	"google.golang.org/genai"
)

const (
	defaultGeminiTTSModel = "gemini-3.1-flash-tts-preview"
	defaultGeminiTTSVoice = "Kore"

	geminiTTSSampleRate    = 24000
	geminiTTSChannels      = 1
	geminiTTSBitsPerSample = 16
	wavHeaderLen           = 44
)

var defaultGeminiTTSConfig = &tts.Config{
	Model:   defaultGeminiTTSModel,
	VoiceID: defaultGeminiTTSVoice,
	Format:  tts.FormatWAV,
}

type geminiTTSModel struct {
	client *genai.Client
	model  string
	voice  string
	lang   string
	fmt    tts.Format
}

var _ tts.Model = (*geminiTTSModel)(nil)

func (g *geminiTTSModel) GenerateSpeech(ctx context.Context, text string) (*tts.AudioFile, error) {
	resp, err := g.client.Models.GenerateContent(ctx, g.model, genai.Text(text), &genai.GenerateContentConfig{
		ResponseModalities: []string{"AUDIO"},
		SpeechConfig: &genai.SpeechConfig{
			VoiceConfig: &genai.VoiceConfig{
				PrebuiltVoiceConfig: &genai.PrebuiltVoiceConfig{VoiceName: g.voice},
			},
			LanguageCode: g.lang,
		},
	})
	if err != nil {
		return nil, err
	}

	pcm, err := geminiTTSAudioData(resp)
	if err != nil {
		return nil, err
	}

	if g.fmt == tts.FormatLINEAR16 {
		return &tts.AudioFile{Format: tts.FormatLINEAR16, Data: pcm}, nil
	}

	return &tts.AudioFile{Format: tts.FormatWAV, Data: wrapGeminiPCMAsWAV(pcm)}, nil
}

var _ provider.TTSClient = (*aiStudioClient)(nil)

func (g *aiStudioClient) NewTTS(model string, config *tts.Config) (tts.Model, error) {
	if config == nil {
		config = defaultGeminiTTSConfig
	}

	if model == "" {
		model = config.Model
	}
	if model == "" {
		model = defaultGeminiTTSModel
	}

	voice := config.VoiceID
	if voice == "" {
		voice = defaultGeminiTTSVoice
	}

	format := config.Format
	if format == "" {
		format = tts.FormatWAV
	}
	if format != tts.FormatWAV && format != tts.FormatLINEAR16 {
		return nil, tts.ErrUnsupportedFileFormat
	}

	return &geminiTTSModel{
		client: g.client,
		model:  model,
		voice:  voice,
		lang:   config.Language,
		fmt:    format,
	}, nil
}

func (g AIStudioProvider) NewTTSClient(ctx context.Context, configs ...pconf.Config) (provider.TTSClient, error) {
	return g.newAIStudioClient(ctx, configs...)
}

var _ provider.TTSProvider = Provider

func geminiTTSAudioData(resp *genai.GenerateContentResponse) ([]byte, error) {
	if resp == nil || len(resp.Candidates) == 0 || resp.Candidates[0].Content == nil || len(resp.Candidates[0].Content.Parts) == 0 {
		return nil, tts.ErrNoResponse
	}

	data := resp.Candidates[0].Content.Parts[0].InlineData
	if data == nil || len(data.Data) == 0 {
		return nil, tts.ErrInvalidResponse
	}

	return data.Data, nil
}

func wrapGeminiPCMAsWAV(pcm []byte) []byte {
	wav := make([]byte, wavHeaderLen+len(pcm))
	copy(wav, "RIFF")
	binary.LittleEndian.PutUint32(wav[4:8], uint32(36+len(pcm)))
	copy(wav[8:], "WAVE")
	copy(wav[12:], "fmt ")
	binary.LittleEndian.PutUint32(wav[16:20], 16)
	binary.LittleEndian.PutUint16(wav[20:22], 1)
	binary.LittleEndian.PutUint16(wav[22:24], geminiTTSChannels)
	binary.LittleEndian.PutUint32(wav[24:28], geminiTTSSampleRate)
	binary.LittleEndian.PutUint32(wav[28:32], geminiTTSSampleRate*geminiTTSChannels*geminiTTSBitsPerSample/8)
	binary.LittleEndian.PutUint16(wav[32:34], geminiTTSChannels*geminiTTSBitsPerSample/8)
	binary.LittleEndian.PutUint16(wav[34:36], geminiTTSBitsPerSample)
	copy(wav[36:], "data")
	binary.LittleEndian.PutUint32(wav[40:44], uint32(len(pcm)))
	copy(wav[wavHeaderLen:], pcm)
	return wav
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

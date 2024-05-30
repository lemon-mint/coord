package openai

import (
	"context"
	"io"

	"github.com/lemon-mint/coord/tts"
	"github.com/sashabaranov/go-openai"
)

type OpenAITextToSpeech struct {
	client *openai.Client

	model openai.SpeechModel
	voice openai.SpeechVoice
	speed float64
}

var _ tts.TTSModel = (*OpenAITextToSpeech)(nil)

func (g *OpenAITextToSpeech) GenerateSpeech(ctx context.Context, text string, fmt tts.Format) (*tts.AudioFile, error) {
	var encoding openai.SpeechResponseFormat

	switch fmt {
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
		Format: fmt,
		Data:   file,
	}, nil
}

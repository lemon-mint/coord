package google

import (
	"context"

	texttospeech "cloud.google.com/go/texttospeech/apiv1"
	"cloud.google.com/go/texttospeech/apiv1/texttospeechpb"
	"github.com/lemon-mint/coord/tts"
)

type TextToSpeechModel struct {
	client *texttospeech.Client

	language string
	name     string

	speaking_rate float64
	pitch         float64

	sample_rate int32
}

var _ tts.TTS = (*TextToSpeechModel)(nil)

func (g *TextToSpeechModel) GenerateSpeech(ctx context.Context, text string, fmt tts.Format) (*tts.AudioFile, error) {
	var encoding texttospeechpb.AudioEncoding

	switch fmt {
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
			AudioEncoding: encoding,
			SpeakingRate:  g.speaking_rate,
			Pitch:         g.pitch,
		},
	}

	resp, err := g.client.SynthesizeSpeech(ctx, &req)
	if err != nil {
		return nil, err
	}

	return &tts.AudioFile{
		Format: fmt,
		Data:   resp.AudioContent,
	}, nil
}

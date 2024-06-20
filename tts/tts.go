package tts

import (
	"context"
)

type Format string

const (
	FormatLINEAR16 Format = "audio/l16"
	FormatMP3      Format = "audio/mpeg"
	FormatOGG      Format = "audio/ogg"
	FormatALAW     Format = "audio/alaw"
	FormatMULAW    Format = "audio/mulaw"
	FormatAAC      Format = "audio/aac"
	FormatFLAC     Format = "audio/flac"
	FormatWAV      Format = "audio/wav"
)

type AudioFile struct {
	Format Format `json:"mime"`
	Data   []byte `json:"data"`
}

type Config struct {
	Language string
	Model    string

	SpeakingRate float64
	Pitch        float64
	SampleRate   int

	Format Format

	VoiceID         string
	Stability       float64
	SimilarityBoost float64
	Style           int
	UseSpeakerBoost bool
	Seed            int

	PronunciationDictLocs []struct {
		PronunciationDictionaryID string
		VersionID                 string
	}
}

type Model interface {
	GenerateSpeech(ctx context.Context, text string) (*AudioFile, error)
}

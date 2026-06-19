package aistudio

import (
	"bytes"
	"encoding/binary"
	"errors"
	"testing"

	"github.com/lemon-mint/coord/tts"
	"google.golang.org/genai"
)

func TestNewGeminiTTSDefaults(t *testing.T) {
	model, err := (&aiStudioClient{}).NewTTS("", nil)
	if err != nil {
		t.Fatal(err)
	}

	gemini, ok := model.(*geminiTTSModel)
	if !ok {
		t.Fatalf("model type = %T, want *geminiTTSModel", model)
	}
	if gemini.model != defaultGeminiTTSModel {
		t.Fatalf("model = %q, want %q", gemini.model, defaultGeminiTTSModel)
	}
	if gemini.voice != defaultGeminiTTSVoice {
		t.Fatalf("voice = %q, want %q", gemini.voice, defaultGeminiTTSVoice)
	}
	if gemini.fmt != tts.FormatWAV {
		t.Fatalf("format = %q, want %q", gemini.fmt, tts.FormatWAV)
	}
}

func TestNewGeminiTTSAcceptsLinear16(t *testing.T) {
	model, err := (&aiStudioClient{}).NewTTS("gemini-3.1-flash-tts-preview", &tts.Config{
		VoiceID:  "Puck",
		Language: "ko-KR",
		Format:   tts.FormatLINEAR16,
	})
	if err != nil {
		t.Fatal(err)
	}

	gemini := model.(*geminiTTSModel)
	if gemini.voice != "Puck" {
		t.Fatalf("voice = %q, want Puck", gemini.voice)
	}
	if gemini.lang != "ko-KR" {
		t.Fatalf("language = %q, want ko-KR", gemini.lang)
	}
	if gemini.fmt != tts.FormatLINEAR16 {
		t.Fatalf("format = %q, want %q", gemini.fmt, tts.FormatLINEAR16)
	}
}

func TestNewGeminiTTSRejectsUnsupportedFormat(t *testing.T) {
	_, err := (&aiStudioClient{}).NewTTS("", &tts.Config{Format: tts.FormatMP3})
	if !errors.Is(err, tts.ErrUnsupportedFileFormat) {
		t.Fatalf("err = %v, want %v", err, tts.ErrUnsupportedFileFormat)
	}
}

func TestGeminiTTSAudioData(t *testing.T) {
	pcm := []byte{1, 2, 3, 4}
	got, err := geminiTTSAudioData(&genai.GenerateContentResponse{
		Candidates: []*genai.Candidate{{
			Content: &genai.Content{Parts: []*genai.Part{{
				InlineData: &genai.Blob{Data: pcm},
			}}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if &got[0] != &pcm[0] {
		t.Fatal("audio data was copied")
	}
}

func TestWrapGeminiPCMAsWAV(t *testing.T) {
	pcm := []byte{0x01, 0x02, 0x03, 0x04}
	wav := wrapGeminiPCMAsWAV(pcm)

	if len(wav) != wavHeaderLen+len(pcm) {
		t.Fatalf("len = %d, want %d", len(wav), wavHeaderLen+len(pcm))
	}
	if !bytes.Equal(wav[0:4], []byte("RIFF")) {
		t.Fatalf("missing RIFF header")
	}
	if binary.LittleEndian.Uint32(wav[4:8]) != uint32(36+len(pcm)) {
		t.Fatalf("RIFF size = %d", binary.LittleEndian.Uint32(wav[4:8]))
	}
	if !bytes.Equal(wav[8:12], []byte("WAVE")) {
		t.Fatalf("missing WAVE header")
	}
	if binary.LittleEndian.Uint16(wav[20:22]) != 1 {
		t.Fatalf("audio format = %d", binary.LittleEndian.Uint16(wav[20:22]))
	}
	if binary.LittleEndian.Uint16(wav[22:24]) != geminiTTSChannels {
		t.Fatalf("channels = %d", binary.LittleEndian.Uint16(wav[22:24]))
	}
	if binary.LittleEndian.Uint32(wav[24:28]) != geminiTTSSampleRate {
		t.Fatalf("sample rate = %d", binary.LittleEndian.Uint32(wav[24:28]))
	}
	if binary.LittleEndian.Uint16(wav[34:36]) != geminiTTSBitsPerSample {
		t.Fatalf("bits per sample = %d", binary.LittleEndian.Uint16(wav[34:36]))
	}
	if !bytes.Equal(wav[wavHeaderLen:], pcm) {
		t.Fatalf("pcm payload mismatch")
	}
}

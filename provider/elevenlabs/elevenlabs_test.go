package elevenlabs_test

import (
	"context"
	"testing"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider/elevenlabs"
	"github.com/lemon-mint/coord/tts"
	"gopkg.eu.org/envloader"
)

func getClient() *elevenlabs.ElevenlabsClient {
	type Config struct {
		APIKey string `env:"ELEVENLABS_API_KEY,required"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile("../../.env", c)

	_client, err := coord.NewTTSClient(context.Background(), "elevenlabs", pconf.WithAPIKey(c.APIKey))
	if err != nil {
		panic(err)
	}

	client, ok := _client.(*elevenlabs.ElevenlabsClient)
	if !ok {
		panic("unexpected type")
	}

	return client
}

func TestGetVoiceList(t *testing.T) {
	client := getClient()

	voices, err := client.GetVoiceList(context.Background())
	if err != nil {
		panic(err)
	}

	if len(voices) == 0 {
		t.Error("no voices found")
	}
}

func TestGenerateSpeech(t *testing.T) {
	client := getClient()

	model, err := client.NewTTS("eleven_monolingual_v1", &tts.Config{
		VoiceID: "21m00Tcm4TlvDq8ikWAM", // elevenlabs built-in voice

		Stability:       0.73,
		SimilarityBoost: 0.21,
		Style:           1,
		UseSpeakerBoost: false,
	})
	defer client.Close()
	if err != nil {
		panic(err)
	}

	testText := "It hath been said that love in dreams is but a dream: I will affirm as true in this, that it appears but so: For never heart like mine was given to a green girl's dream, Nor ever kept awake for such a trifle so low."

	audio, err := model.GenerateSpeech(context.Background(), testText)
	if err != nil {
		panic(err)
	}

	if len(audio.Data) == 0 {
		t.Error("no audio data")
	}
}

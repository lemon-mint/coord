package main

import (
	"context"
	"fmt"
	"os"

	"github.com/lemon-mint/coord"
	"github.com/lemon-mint/coord/pconf"
	"github.com/lemon-mint/coord/provider/elevenlabs"
	"github.com/lemon-mint/coord/tts"
	"gopkg.eu.org/envloader"
)

func main() {
	type Config struct {
		APIKey string `env:"ELEVENLABS_API_KEY,required"`
	}
	c := &Config{}

	envloader.LoadAndBindEnvFile(".env", c)

	_client, err := coord.NewTTSClient(context.Background(), "elevenlabs", pconf.WithAPIKey(c.APIKey))
	if err != nil {
		panic(err)
	}

	client, ok := _client.(*elevenlabs.ElevenlabsClient)
	if !ok {
		panic("unexpected type")
	}

	voices, err := client.GetVoiceList(context.Background())
	if err != nil {
		panic(err)
	}

	fmt.Printf("Voice List: %+v\n", voices)

	model, err := client.NewTTS("eleven_monolingual_v1", &tts.Config{
		VoiceID: voices[1].ID,

		Stability:       0.73,
		SimilarityBoost: 0.21,
		Style:           1,
		UseSpeakerBoost: false,
	})
	defer client.Close()
	if err != nil {
		panic(err)
	}

	audio, err := model.GenerateSpeech(context.Background(), "It hath been said that love in dreams is but a dream: I will affirm as true in this, that it appears but so: For never heart like mine was given to a green girl's dream, Nor ever kept awake for such a trifle so low.")
	if err != nil {
		panic(err)
	}
	os.WriteFile("output.mp3", audio.Data, 0o644)
}

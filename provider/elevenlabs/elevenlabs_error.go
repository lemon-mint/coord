package elevenlabs

import (
	"errors"
	"fmt"

	"github.com/lemon-mint/coord/llm"
	"github.com/lemon-mint/coord/tts"
)

var (
	ErrAPIKeyRequired    error = errors.New("api key is required")
	ErrModelNameRequired error = errors.New("model name is required")
	ErrTextRequired      error = errors.New("text is required")
	ErrVoiceIdRequired   error = errors.New("voice id is required")
)

func getErrorByStatus(err_c int) error {
	switch err_c {
	case 422:
		return tts.ErrUnprocessableContent
	}

	fmt.Printf("Unknown error code: %d\n", err_c)
	return llm.ErrUnknown
}

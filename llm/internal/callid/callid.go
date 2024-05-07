package callid

import (
	"crypto/rand"
)

const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"

const OpenAIPrefix = "call_"

func OpenAICallID() string {
	b := make([]byte, 24)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	var sb [5 + 24]byte
	copy(sb[:5], "call_")

	for i := 0; i < 24; i++ {
		sb[5+i] = chars[b[i]%byte(len(chars))]
	}

	return string(sb[:])
}

const AnthropicPrefix = "toolu_"

func AnthropicCallID() string {
	// toolu_01D7FLrfh4GYq7yT1ULFeyMV
	// toolu_01A09q90qw90lq917835lq9
	b := make([]byte, 21)
	_, err := rand.Read(b)
	if err != nil {
		panic(err)
	}

	var sb [8 + 21]byte
	for i := 0; i < 21; i++ {
		sb[8+i] = chars[b[i]%byte(len(chars))]
	}
	copy(sb[:8], "toolu_01")

	return string(sb[:])
}

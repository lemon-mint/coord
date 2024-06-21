package elevenlabs

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type ttsRequest struct {
	Text                            string                               `json:"text"`
	ModelID                         string                               `json:"model_id"`
	VoiceSettings                   TtsVoiceSettings                     `json:"voice_settings"`
	PronunciationDictionaryLocators []TtsPronunciationDictionaryLocators `json:"pronunciation_dictionary_locators"`
	Seed                            int                                  `json:"seed"`
	PreviousText                    string                               `json:"previous_text"`
	NextText                        string                               `json:"next_text"`
	PreviousRequestIds              []string                             `json:"previous_request_ids"`
	NextRequestIds                  []string                             `json:"next_request_ids"`
}

type TtsVoiceSettings struct {
	Stability       float64 `json:"stability"`
	SimilarityBoost float64 `json:"similarity_boost"`
	Style           int     `json:"style"`
	UseSpeakerBoost bool    `json:"use_speaker_boost"`
}

type TtsPronunciationDictionaryLocators struct {
	PronunciationDictionaryID string `json:"pronunciation_dictionary_id"`
	VersionID                 string `json:"version_id"`
}

type VoiceListResponse struct {
	Voices []struct {
		VoiceID    string      `json:"voice_id"`
		Name       string      `json:"name"`
		Samples    interface{} `json:"samples"`
		Category   string      `json:"category"`
		FineTuning struct {
			IsAllowedToFineTune         bool          `json:"is_allowed_to_fine_tune"`
			FinetuningState             string        `json:"finetuning_state"`
			VerificationFailures        []interface{} `json:"verification_failures"`
			VerificationAttemptsCount   int           `json:"verification_attempts_count"`
			ManualVerificationRequested bool          `json:"manual_verification_requested"`
			Language                    interface{}   `json:"language"`
			FinetuningProgress          struct{}      `json:"finetuning_progress"`
			Message                     interface{}   `json:"message"`
			DatasetDurationSeconds      interface{}   `json:"dataset_duration_seconds"`
			VerificationAttempts        interface{}   `json:"verification_attempts"`
			SliceIds                    interface{}   `json:"slice_ids"`
			ManualVerification          interface{}   `json:"manual_verification"`
		} `json:"fine_tuning"`
		Labels struct {
			Description string `json:"description"`
			UseCase     string `json:"use case"`
			Accent      string `json:"accent"`
			Gender      string `json:"gender"`
			Age         string `json:"age"`
		} `json:"labels"`
		Description             interface{}   `json:"description"`
		PreviewURL              string        `json:"preview_url"`
		AvailableForTiers       []interface{} `json:"available_for_tiers"`
		Settings                interface{}   `json:"settings"`
		Sharing                 interface{}   `json:"sharing"`
		HighQualityBaseModelIds []interface{} `json:"high_quality_base_model_ids"`
		SafetyControl           interface{}   `json:"safety_control"`
		VoiceVerification       struct {
			RequiresVerification      bool          `json:"requires_verification"`
			IsVerified                bool          `json:"is_verified"`
			VerificationFailures      []interface{} `json:"verification_failures"`
			VerificationAttemptsCount int           `json:"verification_attempts_count"`
			Language                  interface{}   `json:"language"`
			VerificationAttempts      interface{}   `json:"verification_attempts"`
		} `json:"voice_verification"`
		OwnerID              interface{} `json:"owner_id"`
		PermissionOnResource interface{} `json:"permission_on_resource"`
	} `json:"voices"`
}

type Voice struct {
	ID   string
	Name string
}

// =================== API Client ===================

type elevenlabsAPIClient struct {
	baseURL     string
	authHandler func(r *http.Request) error

	httpClient *http.Client
}

const elevenlabsBaseURL = "https://api.elevenlabs.io/v1"

func (c *elevenlabsAPIClient) RequestTTS(ctx context.Context, voiceid string, req ttsRequest) ([]byte, error) {
	url, err := url.JoinPath(c.baseURL, "/text-to-speech/"+voiceid)
	if err != nil {
		return nil, err
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	r, _ := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))

	if err := c.authHandler(r); err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, getErrorByStatus(resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	return body, nil
}

func (c *elevenlabsAPIClient) RequestVoiceList(ctx context.Context) ([]Voice, error) {
	url, err := url.JoinPath(c.baseURL, "/voices")
	if err != nil {
		return nil, err
	}

	r, _ := http.NewRequestWithContext(ctx, "GET", url, nil)

	if err := c.authHandler(r); err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, getErrorByStatus(resp.StatusCode)
	}

	body, _ := io.ReadAll(resp.Body)

	var voiceList VoiceListResponse

	if err := json.Unmarshal(body, &voiceList); err != nil {
		return nil, err
	}

	var voices []Voice

	for _, v := range voiceList.Voices {
		voices = append(voices, Voice{
			ID:   v.VoiceID,
			Name: v.Name,
		})
	}

	return voices, nil
}

// ================================================

var elevenlabsHTTPClient *http.Client = &http.Client{
	Transport: &http.Transport{
		MaxIdleConns:    16,
		IdleConnTimeout: 30 * time.Second,
	},
}

func newClient(apikey string) (*elevenlabsAPIClient, error) {
	apikey = strings.TrimSpace(apikey)
	return &elevenlabsAPIClient{
		baseURL: elevenlabsBaseURL,
		authHandler: func(r *http.Request) error {
			r.Header.Set("Content-Type", "application/json")
			r.Header.Set("X-API-Key", apikey)
			return nil
		},
		httpClient: elevenlabsHTTPClient,
	}, nil
}

// ttsRequest{
// 	Text:    text,
// 	ModelID: req., // eleven_monolingual_v1
// 	VoiceSettings: ttsVoiceSettings{
// 		Stability:       0,
// 		SimilarityBoost: 0,
// 		Style:           0,
// 		UseSpeakerBoost: false,
// 	},
// 	PronunciationDictionaryLocators: []ttsPronunciationDictionaryLocators{
// 		{
// 			PronunciationDictionaryID: "",
// 			VersionID:                 "",
// 		},
// 	},
// 	Seed:               rand.Int(),
// 	PreviousText:       "",
// 	NextText:           "",
// 	PreviousRequestIds: []string{""},
// 	NextRequestIds:     []string{""},
// }

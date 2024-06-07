package elevenlabs

type ttsRequest struct {
	Text                            string                               `json:"text"`
	ModelID                         string                               `json:"model_id"`
	VoiceSettings                   ttsVoiceSettings                     `json:"voice_settings"`
	PronunciationDictionaryLocators []ttsPronunciationDictionaryLocators `json:"pronunciation_dictionary_locators"`
	Seed                            int                                  `json:"seed"`
	PreviousText                    string                               `json:"previous_text"`
	NextText                        string                               `json:"next_text"`
	PreviousRequestIds              []string                             `json:"previous_request_ids"`
	NextRequestIds                  []string                             `json:"next_request_ids"`
}

type ttsVoiceSettings struct {
	Stability       int  `json:"stability"`
	SimilarityBoost int  `json:"similarity_boost"`
	Style           int  `json:"style"`
	UseSpeakerBoost bool `json:"use_speaker_boost"`
}

type ttsPronunciationDictionaryLocators struct {
	PronunciationDictionaryID string `json:"pronunciation_dictionary_id"`
	VersionID                 string `json:"version_id"`
}

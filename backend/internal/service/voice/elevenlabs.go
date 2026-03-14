package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
)

type Service struct {
	provider string
	apiKey   string
	voiceID  string
	ttsModel string
	sttModel string
	baseURL  string
	client   *http.Client
}

type STTResult struct {
	Text string `json:"text"`
}

func NewService(cfg *config.Config) *Service {
	provider := "elevenlabs"
	apiKey := ""
	voiceID := "EXAVITQu4vr4xnSDxMaL"
	ttsModel := "eleven_multilingual_v2"
	sttModel := "scribe_v1"

	if cfg != nil {
		provider = strings.ToLower(strings.TrimSpace(cfg.VoiceProvider))
		apiKey = strings.TrimSpace(cfg.ElevenLabsAPIKey)
		voiceID = strings.TrimSpace(cfg.ElevenLabsVoiceID)
		ttsModel = strings.TrimSpace(cfg.ElevenLabsTTSModel)
		sttModel = strings.TrimSpace(cfg.ElevenLabsSTTModel)
	}

	if provider == "" {
		provider = "elevenlabs"
	}
	if voiceID == "" {
		voiceID = "EXAVITQu4vr4xnSDxMaL"
	}
	if ttsModel == "" {
		ttsModel = "eleven_multilingual_v2"
	}
	if sttModel == "" {
		sttModel = "scribe_v1"
	}

	return &Service{
		provider: provider,
		apiKey:   apiKey,
		voiceID:  voiceID,
		ttsModel: ttsModel,
		sttModel: sttModel,
		baseURL:  "https://api.elevenlabs.io/v1",
		client:   &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *Service) IsReady() bool {
	return s != nil && s.provider == "elevenlabs" && s.apiKey != ""
}

func (s *Service) TextToSpeech(text string) ([]byte, error) {
	if !s.IsReady() {
		return nil, fmt.Errorf("elevenlabs is not configured")
	}

	payload := map[string]any{
		"text":     text,
		"model_id": s.ttsModel,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/text-to-speech/%s", s.baseURL, s.voiceID)
	request, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "audio/mpeg")
	request.Header.Set("xi-api-key", s.apiKey)

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("elevenlabs tts error: %s", string(responseBody))
	}

	return responseBody, nil
}

func (s *Service) SpeechToText(audio []byte, fileName string) (*STTResult, error) {
	if !s.IsReady() {
		return nil, fmt.Errorf("elevenlabs is not configured")
	}

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("model_id", s.sttModel); err != nil {
		return nil, err
	}

	part, err := writer.CreateFormFile("file", sanitizeFilename(fileName))
	if err != nil {
		return nil, err
	}
	if _, err := part.Write(audio); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	request, err := http.NewRequest(http.MethodPost, s.baseURL+"/speech-to-text", body)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set("xi-api-key", s.apiKey)

	response, err := s.client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return nil, fmt.Errorf("elevenlabs stt error: %s", string(responseBody))
	}

	var parsed STTResult
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return nil, err
	}

	return &parsed, nil
}

func sanitizeFilename(value string) string {
	cleaned := strings.TrimSpace(value)
	if cleaned == "" {
		return "audio.webm"
	}
	cleaned = strings.ReplaceAll(cleaned, "\n", "")
	cleaned = strings.ReplaceAll(cleaned, "\r", "")
	return cleaned
}

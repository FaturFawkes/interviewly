package voice

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
)

type Service struct {
	provider       string
	apiKey         string
	voiceID        string
	ttsModel       string
	sttModel       string
	agentID        string
	branchID       string
	reviewAgentID  string
	reviewBranchID string
	baseURL        string
	client         *http.Client
}

type STTResult struct {
	Text string `json:"text"`
}

type AgentSignedURLResult struct {
	SignedURL      string `json:"signed_url"`
	ConversationID string `json:"conversation_id,omitempty"`
}

func NewService(cfg *config.Config) *Service {
	provider := "elevenlabs"
	apiKey := ""
	voiceID := "EXAVITQu4vr4xnSDxMaL"
	ttsModel := "eleven_multilingual_v2"
	sttModel := "scribe_v1"
	agentID := ""
	branchID := ""
	reviewAgentID := ""
	reviewBranchID := ""

	if cfg != nil {
		provider = strings.ToLower(strings.TrimSpace(cfg.VoiceProvider))
		apiKey = strings.TrimSpace(cfg.ElevenLabsAPIKey)
		voiceID = strings.TrimSpace(cfg.ElevenLabsVoiceID)
		ttsModel = strings.TrimSpace(cfg.ElevenLabsTTSModel)
		sttModel = strings.TrimSpace(cfg.ElevenLabsSTTModel)
		agentID = strings.TrimSpace(cfg.ElevenLabsAgentID)
		branchID = strings.TrimSpace(cfg.ElevenLabsAgentBranchID)
		reviewAgentID = strings.TrimSpace(cfg.ElevenLabsReviewAgentID)
		reviewBranchID = strings.TrimSpace(cfg.ElevenLabsReviewAgentBranchID)
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
		provider:       provider,
		apiKey:         apiKey,
		voiceID:        voiceID,
		ttsModel:       ttsModel,
		sttModel:       sttModel,
		agentID:        agentID,
		branchID:       branchID,
		reviewAgentID:  reviewAgentID,
		reviewBranchID: reviewBranchID,
		baseURL:        "https://api.elevenlabs.io/v1",
		client:         &http.Client{Timeout: 45 * time.Second},
	}
}

func (s *Service) IsReady() bool {
	return s != nil && s.provider == "elevenlabs" && s.apiKey != ""
}

func (s *Service) AgentIsReady() bool {
	return s.IsReady() && strings.TrimSpace(s.agentID) != ""
}

func (s *Service) ReviewAgentIsReady() bool {
	return s.IsReady() && strings.TrimSpace(s.reviewAgentID) != ""
}

func (s *Service) GetAgentSignedURL(includeConversationID bool) (*AgentSignedURLResult, error) {
	if !s.AgentIsReady() {
		return nil, fmt.Errorf("elevenlabs agent is not configured")
	}

	return s.getAgentSignedURL(s.agentID, s.branchID, includeConversationID)
}

func (s *Service) GetReviewAgentSignedURL(includeConversationID bool) (*AgentSignedURLResult, error) {
	if !s.ReviewAgentIsReady() {
		return nil, fmt.Errorf("elevenlabs review agent is not configured")
	}

	return s.getAgentSignedURL(s.reviewAgentID, s.reviewBranchID, includeConversationID)
}

func (s *Service) getAgentSignedURL(agentID, branchID string, includeConversationID bool) (*AgentSignedURLResult, error) {
	queryValues := url.Values{}
	queryValues.Set("agent_id", strings.TrimSpace(agentID))
	if includeConversationID {
		queryValues.Set("include_conversation_id", "true")
	}
	if strings.TrimSpace(branchID) != "" {
		queryValues.Set("branch_id", branchID)
	}

	requestURL := fmt.Sprintf("%s/convai/conversation/get-signed-url?%s", s.baseURL, queryValues.Encode())
	request, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, err
	}

	request.Header.Set("Accept", "application/json")
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
		return nil, fmt.Errorf("elevenlabs agent signed-url error: %s", strings.TrimSpace(string(responseBody)))
	}

	var parsed AgentSignedURLResult
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return nil, err
	}

	if strings.TrimSpace(parsed.SignedURL) == "" {
		return nil, fmt.Errorf("elevenlabs agent signed-url response is empty")
	}

	return &parsed, nil
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

	requestURL := fmt.Sprintf("%s/text-to-speech/%s", s.baseURL, s.voiceID)
	request, err := http.NewRequest(http.MethodPost, requestURL, bytes.NewReader(body))
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

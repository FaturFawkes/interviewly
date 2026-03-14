package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/domain"
)

// Service is a lightweight AI abstraction layer that can later be swapped with real providers.
type Service struct {
	provider   string
	model      string
	apiBaseURL string
	apiKey     string
	httpClient *http.Client
}

func NewService(cfg *config.Config) domain.AIService {
	provider := "local"
	model := "gpt-4o-mini"
	apiBaseURL := "https://api.openai.com/v1"
	apiKey := ""

	if cfg != nil {
		provider = strings.ToLower(strings.TrimSpace(cfg.AIProvider))
		model = strings.TrimSpace(cfg.AIModel)
		apiBaseURL = strings.TrimRight(strings.TrimSpace(cfg.AIAPIBaseURL), "/")
		apiKey = strings.TrimSpace(cfg.AIAPIKey)
	}

	if provider == "" {
		provider = "local"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if apiBaseURL == "" {
		apiBaseURL = "https://api.openai.com/v1"
	}

	return &Service{
		provider:   provider,
		model:      model,
		apiBaseURL: apiBaseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) ParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
	if s.useRemoteProvider() {
		if remote, err := s.remoteParseJobDescription(jobDescription); err == nil {
			return remote, nil
		}
	}

	normalized := strings.ToLower(jobDescription)
	tokens := tokenize(normalized)

	skills := detectSkills(normalized)
	keywords := topKeywords(tokens, 10)
	themes := detectThemes(normalized)
	seniority := detectSeniority(normalized)

	return &domain.JobInsights{
		Skills:    skills,
		Keywords:  keywords,
		Themes:    themes,
		Seniority: seniority,
	}, nil
}

func (s *Service) GenerateQuestions(resumeText, jobDescription string) ([]domain.GeneratedQuestion, error) {
	if s.useRemoteProvider() {
		if remote, err := s.remoteGenerateQuestions(resumeText, jobDescription); err == nil {
			return remote, nil
		}
	}

	resumeTokens := topKeywords(tokenize(strings.ToLower(resumeText)), 6)
	jobTokens := topKeywords(tokenize(strings.ToLower(jobDescription)), 6)

	primaryResume := pickOrDefault(resumeTokens, 0, "your background")
	secondaryResume := pickOrDefault(resumeTokens, 1, "a key project")
	primaryJob := pickOrDefault(jobTokens, 0, "this role")
	secondaryJob := pickOrDefault(jobTokens, 1, "the main responsibilities")

	behavioral := []domain.GeneratedQuestion{
		{Type: "behavioral", Question: "Tell me about a time you delivered impact related to " + primaryJob + "."},
		{Type: "behavioral", Question: "Describe a challenge you faced while working on " + secondaryResume + " and how you solved it."},
		{Type: "behavioral", Question: "Share an example of collaborating with others to achieve a difficult goal."},
		{Type: "behavioral", Question: "How do you prioritize tasks when deadlines are tight and priorities change?"},
		{Type: "behavioral", Question: "Describe a situation where you received critical feedback and what actions you took."},
	}

	technical := []domain.GeneratedQuestion{
		{Type: "technical", Question: "Walk through a system or feature you built involving " + primaryResume + "."},
		{Type: "technical", Question: "How would you design an API workflow for " + primaryJob + "?"},
		{Type: "technical", Question: "What trade-offs would you consider when scaling a service handling " + secondaryJob + "?"},
		{Type: "technical", Question: "Explain how you debug production issues in distributed systems."},
		{Type: "technical", Question: "How do you ensure reliability, observability, and performance in backend services?"},
	}

	questions := make([]domain.GeneratedQuestion, 0, len(behavioral)+len(technical))
	questions = append(questions, behavioral...)
	questions = append(questions, technical...)

	return questions, nil
}

func (s *Service) AnalyzeAnswer(question, answer string) (*domain.AnswerAnalysis, error) {
	if s.useRemoteProvider() {
		if remote, err := s.remoteAnalyzeAnswer(question, answer); err == nil {
			return remote, nil
		}
	}

	normalizedAnswer := strings.TrimSpace(answer)
	wordCount := len(strings.Fields(normalizedAnswer))

	score := 45
	strengths := make([]string, 0)
	weaknesses := make([]string, 0)
	improvements := make([]string, 0)

	if wordCount >= 40 {
		score += 20
		strengths = append(strengths, "sufficient detail")
	} else {
		weaknesses = append(weaknesses, "answer is too brief")
		improvements = append(improvements, "add more context and concrete details")
	}

	if containsAny(normalizedAnswer, []string{"I ", "i "}) {
		score += 10
		strengths = append(strengths, "clear ownership of actions")
	} else {
		weaknesses = append(weaknesses, "ownership is unclear")
		improvements = append(improvements, "describe your specific actions")
	}

	if containsAny(strings.ToLower(normalizedAnswer), []string{"result", "impact", "%", "improved", "reduced", "increased"}) {
		score += 15
		strengths = append(strengths, "mentions outcome or impact")
	} else {
		weaknesses = append(weaknesses, "missing measurable outcomes")
		improvements = append(improvements, "include measurable results when possible")
	}

	questionTokens := topKeywords(tokenize(strings.ToLower(question)), 5)
	answerTokens := tokenize(strings.ToLower(normalizedAnswer))
	matchCount := 0
	for _, token := range questionTokens {
		if containsAny(strings.Join(answerTokens, " "), []string{token}) {
			matchCount++
		}
	}
	if matchCount >= 2 {
		score += 10
		strengths = append(strengths, "answer is relevant to the question")
	} else {
		weaknesses = append(weaknesses, "answer drifts from the main question")
		improvements = append(improvements, "focus directly on the asked topic")
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	if len(strengths) == 0 {
		strengths = append(strengths, "response provided")
	}
	if len(weaknesses) == 0 {
		weaknesses = append(weaknesses, "minor clarity improvements possible")
	}
	if len(improvements) == 0 {
		improvements = append(improvements, "add clearer structure and concise takeaway")
	}

	starFeedback := "Use STAR framing: Situation and Task in 1-2 sentences, then specific Actions you took, and end with measurable Result."

	return &domain.AnswerAnalysis{
		Score:        score,
		Strengths:    strengths,
		Weaknesses:   weaknesses,
		Improvements: improvements,
		STARFeedback: starFeedback,
	}, nil
}

func tokenize(input string) []string {
	r := strings.NewReplacer(
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"/", " ",
		"\\n", " ",
	)
	cleaned := r.Replace(input)
	parts := strings.Fields(cleaned)

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if isStopWord(part) {
			continue
		}
		filtered = append(filtered, part)
	}

	return filtered
}

func isStopWord(word string) bool {
	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {},
		"you": {}, "your": {}, "are": {}, "our": {}, "have": {}, "will": {}, "all": {},
		"can": {}, "has": {}, "not": {}, "but": {}, "job": {}, "role": {}, "team": {},
	}
	_, found := stopWords[word]
	return found
}

func topKeywords(tokens []string, limit int) []string {
	freq := make(map[string]int)
	for _, token := range tokens {
		freq[token]++
	}

	type entry struct {
		word  string
		count int
	}

	list := make([]entry, 0, len(freq))
	for word, count := range freq {
		list = append(list, entry{word: word, count: count})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].word < list[j].word
		}
		return list[i].count > list[j].count
	})

	if len(list) > limit {
		list = list[:limit]
	}

	result := make([]string, 0, len(list))
	for _, item := range list {
		result = append(result, item.word)
	}

	return result
}

func detectSkills(input string) []string {
	catalog := []string{
		"golang", "go", "python", "java", "javascript", "typescript", "react", "next.js", "node.js",
		"postgresql", "redis", "docker", "kubernetes", "aws", "gcp", "azure", "gin", "gorm",
		"microservices", "rest", "grpc", "sql", "nosql", "graphql", "ci/cd",
	}

	found := make([]string, 0)
	for _, skill := range catalog {
		if strings.Contains(input, strings.ToLower(skill)) {
			found = append(found, skill)
		}
	}

	if len(found) == 0 {
		return []string{"general-software-engineering"}
	}

	return found
}

func detectThemes(input string) []string {
	themes := make([]string, 0)
	if strings.Contains(input, "backend") || strings.Contains(input, "api") {
		themes = append(themes, "backend-development")
	}
	if strings.Contains(input, "frontend") || strings.Contains(input, "ui") {
		themes = append(themes, "frontend-development")
	}
	if strings.Contains(input, "cloud") || strings.Contains(input, "deploy") || strings.Contains(input, "infrastructure") {
		themes = append(themes, "cloud-infrastructure")
	}
	if strings.Contains(input, "data") || strings.Contains(input, "analytics") {
		themes = append(themes, "data-and-analytics")
	}

	if len(themes) == 0 {
		themes = append(themes, "general-engineering")
	}

	return themes
}

func detectSeniority(input string) string {
	if strings.Contains(input, "principal") || strings.Contains(input, "staff") {
		return "staff"
	}
	if strings.Contains(input, "senior") || strings.Contains(input, "lead") {
		return "senior"
	}
	if strings.Contains(input, "mid") || strings.Contains(input, "intermediate") {
		return "mid"
	}
	if strings.Contains(input, "junior") || strings.Contains(input, "entry") || strings.Contains(input, "fresh graduate") {
		return "junior"
	}
	return "unspecified"
}

func pickOrDefault(values []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(values) && strings.TrimSpace(values[idx]) != "" {
		return values[idx]
	}
	return fallback
}

func containsAny(input string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.Contains(input, candidate) {
			return true
		}
	}
	return false
}

func (s *Service) useRemoteProvider() bool {
	if s == nil {
		return false
	}
	if s.provider != "openai" {
		return false
	}
	return strings.TrimSpace(s.apiKey) != ""
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

func (s *Service) chatCompletion(systemPrompt, userPrompt string) (string, error) {
	requestPayload := openAIChatRequest{
		Model: s.model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   700,
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, s.apiBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.apiKey)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("ai provider error: %s", string(responseBody))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai provider returned no choices")
	}

	return parsed.Choices[0].Message.Content, nil
}

func (s *Service) remoteParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
	systemPrompt := "You are an interview analysis assistant. Return only strict JSON."
	userPrompt := "Parse the following job description into JSON with keys skills (array of strings), keywords (array of strings), themes (array of strings), seniority (string). Keep each list concise.\n\nJob Description:\n" + jobDescription

	raw, err := s.chatCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result domain.JobInsights
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	if len(result.Skills) == 0 {
		result.Skills = []string{"general-software-engineering"}
	}
	if len(result.Keywords) == 0 {
		result.Keywords = []string{"general"}
	}
	if len(result.Themes) == 0 {
		result.Themes = []string{"general-engineering"}
	}
	if strings.TrimSpace(result.Seniority) == "" {
		result.Seniority = "unspecified"
	}

	return &result, nil
}

func (s *Service) remoteGenerateQuestions(resumeText, jobDescription string) ([]domain.GeneratedQuestion, error) {
	systemPrompt := "You are an interview coach. Return only strict JSON."
	userPrompt := "Generate 10 interview questions in JSON array with each item keys type and question. Include 5 behavioral and 5 technical questions tailored to the candidate CV and job description.\n\nCV:\n" + resumeText + "\n\nJob Description:\n" + jobDescription

	raw, err := s.chatCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result []domain.GeneratedQuestion
	if err := extractJSONArray(raw, &result); err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no generated questions")
	}

	return result, nil
}

func (s *Service) remoteAnalyzeAnswer(question, answer string) (*domain.AnswerAnalysis, error) {
	systemPrompt := "You are an interview evaluator. Return only strict JSON."
	userPrompt := "Evaluate the candidate answer and return JSON with keys score (0-100 int), strengths (array), weaknesses (array), improvements (array), star_feedback (string).\n\nQuestion:\n" + question + "\n\nAnswer:\n" + answer

	raw, err := s.chatCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result domain.AnswerAnalysis
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}
	if len(result.Strengths) == 0 {
		result.Strengths = []string{"response provided"}
	}
	if len(result.Weaknesses) == 0 {
		result.Weaknesses = []string{"minor clarity improvements possible"}
	}
	if len(result.Improvements) == 0 {
		result.Improvements = []string{"add clearer structure and concise takeaway"}
	}
	if strings.TrimSpace(result.STARFeedback) == "" {
		result.STARFeedback = "Use STAR framing: Situation, Task, Action, Result with measurable impact."
	}

	return &result, nil
}

func extractJSONObject(raw string, target any) error {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return fmt.Errorf("no json object found")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), target)
}

func extractJSONArray(raw string, target any) error {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return fmt.Errorf("no json array found")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), target)
}

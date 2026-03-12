package ai

import (
	"sort"
	"strings"

	"github.com/interview_app/backend/internal/domain"
)

// Service is a lightweight AI abstraction layer that can later be swapped with real providers.
type Service struct{}

func NewService() domain.AIService {
	return &Service{}
}

func (s *Service) ParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
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
	return &domain.AnswerAnalysis{
		Score:        70,
		Strengths:    []string{"clear communication"},
		Weaknesses:   []string{"could add more detail"},
		Improvements: []string{"include measurable impact"},
		STARFeedback: "Structure your answer with Situation, Task, Action, and Result.",
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

package ai

import "github.com/interview_app/backend/internal/domain"

// Service is a lightweight AI abstraction layer that can later be swapped with real providers.
type Service struct{}

func NewService() domain.AIService {
	return &Service{}
}

func (s *Service) ParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
	return &domain.JobInsights{
		Skills:    []string{},
		Keywords:  []string{},
		Themes:    []string{},
		Seniority: "unknown",
	}, nil
}

func (s *Service) GenerateQuestions(resumeText, jobDescription string) ([]domain.GeneratedQuestion, error) {
	return []domain.GeneratedQuestion{
		{Type: "behavioral", Question: "Tell me about yourself and your most relevant experience."},
		{Type: "technical", Question: "Describe one technical problem you solved recently and your approach."},
	}, nil
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

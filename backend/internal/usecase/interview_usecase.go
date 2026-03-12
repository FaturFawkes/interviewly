package usecase

import (
	"errors"
	"strings"

	"github.com/interview_app/backend/internal/domain"
)

type interviewUseCase struct {
	aiService domain.AIService
	repo      domain.InterviewRepository
}

// NewInterviewUseCase creates a usecase for interview business workflows.
func NewInterviewUseCase(aiService domain.AIService, repo domain.InterviewRepository) domain.InterviewUseCase {
	return &interviewUseCase{
		aiService: aiService,
		repo:      repo,
	}
}

func (uc *interviewUseCase) ParseJobDescription(userID, rawDescription string) (*domain.ParsedJobDescription, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(rawDescription) == "" {
		return nil, errors.New("job description is required")
	}

	insights, err := uc.aiService.ParseJobDescription(rawDescription)
	if err != nil {
		return nil, err
	}

	return uc.repo.SaveParsedJob(userID, rawDescription, insights)
}

func (uc *interviewUseCase) SaveResume(userID, content string) (*domain.ResumeRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("resume content is required")
	}

	return uc.repo.SaveResume(userID, content)
}

func (uc *interviewUseCase) GenerateQuestions(userID, resumeText, jobDescription string) ([]domain.StoredQuestion, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(resumeText) == "" {
		return nil, errors.New("resume text is required")
	}
	if strings.TrimSpace(jobDescription) == "" {
		return nil, errors.New("job description is required")
	}

	resume, err := uc.repo.SaveResume(userID, resumeText)
	if err != nil {
		return nil, err
	}

	insights, err := uc.aiService.ParseJobDescription(jobDescription)
	if err != nil {
		return nil, err
	}

	parsedJob, err := uc.repo.SaveParsedJob(userID, jobDescription, insights)
	if err != nil {
		return nil, err
	}

	generated, err := uc.aiService.GenerateQuestions(resumeText, jobDescription)
	if err != nil {
		return nil, err
	}

	return uc.repo.SaveGeneratedQuestions(userID, resume.ID, parsedJob.ID, generated)
}

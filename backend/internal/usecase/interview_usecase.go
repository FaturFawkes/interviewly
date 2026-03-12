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

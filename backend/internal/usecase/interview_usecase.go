package usecase

import (
	"errors"
	"math"
	"sort"
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

func (uc *interviewUseCase) CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string) (*domain.PracticeSession, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(resumeID) == "" {
		return nil, errors.New("resume id is required")
	}
	if strings.TrimSpace(jobParseID) == "" {
		return nil, errors.New("job parse id is required")
	}
	if len(questionIDs) == 0 {
		return nil, errors.New("question ids are required")
	}

	return uc.repo.CreatePracticeSession(userID, resumeID, jobParseID, questionIDs)
}

func (uc *interviewUseCase) ListPracticeSessions(userID string) ([]domain.PracticeSession, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	return uc.repo.ListPracticeSessions(userID)
}

func (uc *interviewUseCase) SubmitSessionAnswer(userID, sessionID, questionID, answer string) (*domain.SessionAnswer, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session id is required")
	}
	if strings.TrimSpace(questionID) == "" {
		return nil, errors.New("question id is required")
	}
	if strings.TrimSpace(answer) == "" {
		return nil, errors.New("answer is required")
	}

	return uc.repo.SaveSessionAnswer(userID, sessionID, questionID, answer)
}

func (uc *interviewUseCase) GenerateFeedback(userID, sessionID, questionID, question, answer string) (*domain.FeedbackRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session id is required")
	}
	if strings.TrimSpace(questionID) == "" {
		return nil, errors.New("question id is required")
	}
	if strings.TrimSpace(question) == "" {
		return nil, errors.New("question is required")
	}
	if strings.TrimSpace(answer) == "" {
		return nil, errors.New("answer is required")
	}

	analysis, err := uc.aiService.AnalyzeAnswer(question, answer)
	if err != nil {
		return nil, err
	}

	feedback, err := uc.repo.SaveFeedback(userID, sessionID, questionID, question, answer, analysis)
	if err != nil {
		return nil, err
	}

	if _, aggregateErr := uc.AggregateProgress(userID); aggregateErr != nil {
		return nil, aggregateErr
	}

	return feedback, nil
}

func (uc *interviewUseCase) AggregateProgress(userID string) (*domain.ProgressMetrics, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	feedbackItems, err := uc.repo.ListFeedbackByUser(userID)
	if err != nil {
		return nil, err
	}

	if len(feedbackItems) == 0 {
		return uc.repo.SaveProgressMetrics(userID, 0, []string{}, 0)
	}

	sum := 0
	sessionSet := make(map[string]struct{})
	weakCount := make(map[string]int)

	for _, item := range feedbackItems {
		sum += item.Score
		sessionSet[item.SessionID] = struct{}{}
		for _, weak := range item.Weaknesses {
			key := strings.TrimSpace(strings.ToLower(weak))
			if key == "" {
				continue
			}
			weakCount[key]++
		}
	}

	average := float64(sum) / float64(len(feedbackItems))
	average = math.Round(average*100) / 100

	weakAreas := topWeakAreas(weakCount, 3)
	sessionsCompleted := len(sessionSet)

	return uc.repo.SaveProgressMetrics(userID, average, weakAreas, sessionsCompleted)
}

func (uc *interviewUseCase) GetProgress(userID string) (*domain.ProgressMetrics, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	return uc.repo.GetProgressMetrics(userID)
}

func topWeakAreas(freq map[string]int, limit int) []string {
	type entry struct {
		value string
		count int
	}

	list := make([]entry, 0, len(freq))
	for value, count := range freq {
		list = append(list, entry{value: value, count: count})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].value < list[j].value
		}
		return list[i].count > list[j].count
	})

	if len(list) > limit {
		list = list[:limit]
	}

	result := make([]string, 0, len(list))
	for _, item := range list {
		result = append(result, item.value)
	}
	return result
}

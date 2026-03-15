package usecase

import (
	"errors"
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/interview_app/backend/internal/domain"
)

type interviewUseCase struct {
	aiService     domain.AIService
	repo          domain.InterviewRepository
	resumeStorage domain.ResumeFileStorage
}

// NewInterviewUseCase creates a usecase for interview business workflows.
func NewInterviewUseCase(aiService domain.AIService, repo domain.InterviewRepository, resumeStorage domain.ResumeFileStorage) domain.InterviewUseCase {
	return &interviewUseCase{
		aiService:     aiService,
		repo:          repo,
		resumeStorage: resumeStorage,
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

func (uc *interviewUseCase) SaveResume(userID string, upload domain.ResumeUpload) (*domain.ResumeRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	content := strings.TrimSpace(upload.Content)
	if content == "" {
		return nil, errors.New("resume content is required")
	}

	existingResume, err := uc.repo.GetLatestResume(userID)
	if err != nil {
		return nil, err
	}

	oldMinIOPath := ""
	if existingResume != nil {
		oldMinIOPath = strings.TrimSpace(existingResume.MinIOPath)
	}

	minIOPath := oldMinIOPath
	uploadedNewFile := false
	if uc.resumeStorage != nil && len(upload.FileData) > 0 {
		path, uploadErr := uc.resumeStorage.UploadResume(userID, upload.FileName, upload.ContentType, upload.FileData)
		if uploadErr != nil {
			return nil, uploadErr
		}
		minIOPath = path
		uploadedNewFile = true
	}

	resume, err := uc.repo.SaveResume(userID, content, minIOPath)
	if err != nil {
		if uploadedNewFile && uc.resumeStorage != nil {
			_ = uc.resumeStorage.DeleteResume(minIOPath)
		}
		return nil, err
	}

	if uploadedNewFile && uc.resumeStorage != nil && oldMinIOPath != "" && oldMinIOPath != minIOPath {
		_ = uc.resumeStorage.DeleteResume(oldMinIOPath)
	}

	return resume, nil
}

func (uc *interviewUseCase) GetLatestResume(userID string) (*domain.ResumeRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	return uc.repo.GetLatestResume(userID)
}

func (uc *interviewUseCase) AnalyzeResume(userID string, upload domain.ResumeUpload) (*domain.ResumeAnalysisResult, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	if len(upload.FileData) == 0 && strings.TrimSpace(upload.Content) == "" {
		latestResume, err := uc.repo.GetLatestResume(userID)
		if err != nil {
			return nil, err
		}
		if latestResume == nil || strings.TrimSpace(latestResume.Content) == "" {
			return nil, errors.New("resume not found, please upload your cv first")
		}

		analysis, err := uc.aiService.AnalyzeResume(latestResume.Content)
		if err != nil {
			return nil, err
		}

		return &domain.ResumeAnalysisResult{
			Resume:   latestResume,
			Analysis: analysis,
		}, nil
	}

	resume, err := uc.SaveResume(userID, upload)
	if err != nil {
		return nil, err
	}

	analysis, err := uc.aiService.AnalyzeResume(resume.Content)
	if err != nil {
		return nil, err
	}

	return &domain.ResumeAnalysisResult{
		Resume:   resume,
		Analysis: analysis,
	}, nil
}

func (uc *interviewUseCase) DownloadLatestResume(userID string) (*domain.ResumeFile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	if uc.resumeStorage == nil {
		return nil, errors.New("resume file storage is not configured")
	}

	resume, err := uc.repo.GetLatestResume(userID)
	if err != nil {
		return nil, err
	}

	if resume == nil || strings.TrimSpace(resume.MinIOPath) == "" {
		return nil, errors.New("no uploaded cv file found for download")
	}

	return uc.resumeStorage.DownloadResume(resume.MinIOPath)
}

func (uc *interviewUseCase) GenerateQuestions(
	userID,
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
) ([]domain.StoredQuestion, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(jobDescription) == "" {
		return nil, errors.New("job description is required")
	}
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	interviewMode = domain.NormalizeInterviewMode(string(interviewMode))
	interviewDifficulty = domain.NormalizeInterviewDifficulty(string(interviewDifficulty))

	var resume *domain.ResumeRecord
	var err error
	if strings.TrimSpace(resumeText) == "" {
		resume, err = uc.repo.GetLatestResume(userID)
		if err != nil {
			return nil, err
		}
		if resume == nil || strings.TrimSpace(resume.Content) == "" {
			return nil, errors.New("resume not found, please upload and analyze your cv first")
		}
		resumeText = resume.Content
	} else {
		resume, err = uc.repo.SaveResume(userID, resumeText, "")
		if err != nil {
			return nil, err
		}
	}

	insights, err := uc.aiService.ParseJobDescription(jobDescription)
	if err != nil {
		return nil, err
	}

	parsedJob, err := uc.repo.SaveParsedJob(userID, jobDescription, insights)
	if err != nil {
		return nil, err
	}

	generated, err := uc.aiService.GenerateQuestions(
		resumeText,
		jobDescription,
		interviewLanguage,
		interviewMode,
		interviewDifficulty,
	)
	if err != nil {
		return nil, err
	}

	return uc.repo.SaveGeneratedQuestions(userID, resume.ID, parsedJob.ID, generated)
}

func (uc *interviewUseCase) CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string, metadata domain.SessionMetadata) (*domain.PracticeSession, error) {
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

	mode := strings.TrimSpace(strings.ToLower(metadata.InterviewMode))
	if mode == "" {
		mode = "text"
	}
	if mode != "text" && mode != "voice" {
		return nil, errors.New("interview mode must be text or voice")
	}
	metadata.InterviewMode = mode
	metadata.InterviewLanguage = domain.NormalizeInterviewLanguage(string(metadata.InterviewLanguage))
	metadata.TargetRole = strings.TrimSpace(metadata.TargetRole)
	metadata.TargetCompany = strings.TrimSpace(metadata.TargetCompany)

	return uc.repo.CreatePracticeSession(userID, resumeID, jobParseID, questionIDs, metadata)
}

func (uc *interviewUseCase) ListPracticeSessions(userID string) ([]domain.PracticeSession, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	return uc.repo.ListPracticeSessions(userID)
}

func (uc *interviewUseCase) CompletePracticeSession(userID, sessionID string) (*domain.PracticeSession, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session id is required")
	}

	session, err := uc.repo.CompletePracticeSession(userID, sessionID)
	if err != nil {
		return nil, err
	}

	if _, aggregateErr := uc.AggregateProgress(userID); aggregateErr != nil {
		return nil, aggregateErr
	}

	return session, nil
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

func (uc *interviewUseCase) GenerateFeedback(userID, sessionID, questionID, question, answer string, interviewLanguage domain.InterviewLanguage) (*domain.FeedbackRecord, error) {
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
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))

	analysis, err := uc.aiService.AnalyzeAnswer(question, answer, interviewLanguage)
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

func (uc *interviewUseCase) GetAnalyticsOverview(userID string) (*domain.AnalyticsOverview, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	progress, err := uc.AggregateProgress(userID)
	if err != nil {
		return nil, err
	}

	sessions, err := uc.repo.ListPracticeSessions(userID)
	if err != nil {
		return nil, err
	}
	completedSessions := filterCompletedSessions(sessions)

	readiness := int(math.Min(100, math.Round(progress.AverageScore*0.78+float64(progress.SessionsCompleted)*2.4)))
	practiceHours := math.Round((float64(progress.SessionsCompleted)*18.0/60.0)*10) / 10

	history := make([]domain.AnalyticsPoint, 0)
	for index, session := range completedSessions {
		if index >= 8 {
			break
		}
		history = append(history, domain.AnalyticsPoint{
			Label: "S" + strconv.Itoa(index+1),
			Score: session.Score,
		})
	}

	avgScoreTrend := 0
	if len(completedSessions) >= 2 {
		latest := completedSessions[0].Score
		oldest := completedSessions[len(completedSessions)-1].Score
		avgScoreTrend = latest - oldest
	}

	recommendations := make([]string, 0)
	if len(progress.WeakAreas) > 0 {
		for _, weak := range progress.WeakAreas {
			recommendations = append(recommendations, "Practice one STAR answer focused on "+strings.ToLower(weak)+".")
		}
	} else {
		recommendations = append(recommendations, "No personalized recommendations yet. Complete a practice session first.")
	}

	streak := computePracticeStreakDays(completedSessions)

	recentSessions := completedSessions
	if len(recentSessions) > 5 {
		recentSessions = recentSessions[:5]
	}

	return &domain.AnalyticsOverview{
		InterviewReadiness: readiness,
		AverageScore:       progress.AverageScore,
		AvgScoreTrend:      avgScoreTrend,
		TotalSessions:      progress.SessionsCompleted,
		PracticeHours:      practiceHours,
		PracticeStreakDays: streak,
		WeakAreas:          append([]string{}, progress.WeakAreas...),
		Recommendations:    recommendations,
		RecentSessions:     append([]domain.PracticeSession{}, recentSessions...),
		ScoreHistory:       history,
	}, nil
}

func filterCompletedSessions(sessions []domain.PracticeSession) []domain.PracticeSession {
	filtered := make([]domain.PracticeSession, 0, len(sessions))
	for _, session := range sessions {
		if session.Status == domain.SessionStatusCompleted {
			filtered = append(filtered, session)
		}
	}
	return filtered
}

func computePracticeStreakDays(sessions []domain.PracticeSession) int {
	if len(sessions) == 0 {
		return 0
	}

	seen := make(map[string]struct{})
	days := make([]time.Time, 0)
	for _, session := range sessions {
		timestamp := session.CreatedAt
		if session.CompletedAt != nil {
			timestamp = *session.CompletedAt
		}
		day := time.Date(timestamp.Year(), timestamp.Month(), timestamp.Day(), 0, 0, 0, 0, time.UTC)
		key := day.Format("2006-01-02")
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		days = append(days, day)
	}

	if len(days) == 0 {
		return 0
	}

	sort.Slice(days, func(i, j int) bool {
		return days[i].After(days[j])
	})

	streak := 1
	for i := 1; i < len(days); i++ {
		diff := days[i-1].Sub(days[i]).Hours() / 24
		if diff == 1 {
			streak++
			continue
		}
		break
	}

	return streak
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

package usecase

import (
	"errors"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/service/subscription"
)

type interviewUseCase struct {
	aiService           domain.AIService
	repo                domain.InterviewRepository
	subscriptionService *subscription.Service
}

// NewInterviewUseCase creates a usecase for interview business workflows.
func NewInterviewUseCase(aiService domain.AIService, repo domain.InterviewRepository, subscriptionService *subscription.Service) domain.InterviewUseCase {
	return &interviewUseCase{
		aiService:           aiService,
		repo:                repo,
		subscriptionService: subscriptionService,
	}
}

type aiModelOverrideService interface {
	ParseJobDescriptionWithModel(jobDescription, modelOverride string) (*domain.JobInsights, error)
	GenerateQuestionsWithModel(
		resumeText,
		jobDescription string,
		interviewLanguage domain.InterviewLanguage,
		interviewMode domain.InterviewMode,
		interviewDifficulty domain.InterviewDifficulty,
		modelOverride string,
	) ([]domain.GeneratedQuestion, error)
	AnalyzeAnswerWithModel(
		question,
		answer string,
		interviewLanguage domain.InterviewLanguage,
		modelOverride string,
	) (*domain.AnswerAnalysis, error)
	AnalyzeResumeWithModel(resumeText, modelOverride string) (*domain.ResumeAIAnalysis, error)
}

func (uc *interviewUseCase) ParseJobDescription(userID, rawDescription string) (*domain.ParsedJobDescription, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(rawDescription) == "" {
		return nil, errors.New("job description is required")
	}

	if err := uc.consumeJDParse(userID, "parse_job_description"); err != nil {
		return nil, err
	}

	decision, err := uc.consumeTextRequest(userID, "parse_job_description")
	if err != nil {
		return nil, err
	}

	modelOverride := uc.applyTextFUPDecision(decision)
	insights, err := uc.parseJobDescriptionWithModel(rawDescription, modelOverride)
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
	if content == "" && len(upload.FileData) > 0 {
		content = strings.TrimSpace(string(upload.FileData))
	}
	if strings.TrimSpace(content) == "" {
		return nil, errors.New("resume content is required")
	}

	return uc.repo.SaveResume(userID, content)
}

func (uc *interviewUseCase) GetLatestResume(userID string) (*domain.ResumeRecord, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	return uc.repo.GetLatestResume(userID)
}

func (uc *interviewUseCase) AnalyzeResume(userID string, upload domain.ResumeUpload) (*domain.ResumeAIAnalysis, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	content := strings.TrimSpace(upload.Content)
	if content == "" && len(upload.FileData) > 0 {
		content = strings.TrimSpace(string(upload.FileData))
	}
	if content == "" {
		latest, err := uc.repo.GetLatestResume(userID)
		if err != nil {
			return nil, err
		}
		if latest == nil || strings.TrimSpace(latest.Content) == "" {
			return nil, errors.New("resume content is required")
		}
		content = latest.Content
	}

	decision, err := uc.consumeTextRequest(userID, "analyze_resume")
	if err != nil {
		return nil, err
	}

	modelOverride := uc.applyTextFUPDecision(decision)
	return uc.analyzeResumeWithModel(content, modelOverride)
}

func (uc *interviewUseCase) DownloadLatestResume(userID string) (*domain.ResumeFile, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	latest, err := uc.repo.GetLatestResume(userID)
	if err != nil {
		return nil, err
	}
	if latest == nil || strings.TrimSpace(latest.Content) == "" {
		return nil, errors.New("no uploaded cv")
	}

	return &domain.ResumeFile{
		FileName:    "resume.txt",
		ContentType: "text/plain",
		Data:        []byte(latest.Content),
	}, nil
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
		resume, err = uc.repo.SaveResume(userID, resumeText)
		if err != nil {
			return nil, err
		}
	}

	if err := uc.consumeJDParse(userID, "generate_questions"); err != nil {
		return nil, err
	}

	parseDecision, err := uc.consumeTextRequest(userID, "generate_questions_parse")
	if err != nil {
		return nil, err
	}

	parseModelOverride := uc.applyTextFUPDecision(parseDecision)
	insights, err := uc.parseJobDescriptionWithModel(jobDescription, parseModelOverride)
	if err != nil {
		return nil, err
	}

	parsedJob, err := uc.repo.SaveParsedJob(userID, jobDescription, insights)
	if err != nil {
		return nil, err
	}

	generateDecision, err := uc.consumeTextRequest(userID, "generate_questions")
	if err != nil {
		return nil, err
	}

	generateModelOverride := uc.applyTextFUPDecision(generateDecision)
	generated, err := uc.generateQuestionsWithModel(
		resumeText,
		jobDescription,
		interviewLanguage,
		interviewMode,
		interviewDifficulty,
		generateModelOverride,
	)
	if err != nil {
		return nil, err
	}

	return uc.repo.SaveGeneratedQuestions(userID, resume.ID, parsedJob.ID, generated)
}

func (uc *interviewUseCase) CreatePracticeSession(
	userID,
	resumeID,
	jobParseID string,
	questionIDs []string,
	metadata domain.SessionMetadata,
) (*domain.PracticeSession, error) {
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

	return uc.repo.CreatePracticeSession(userID, resumeID, jobParseID, questionIDs, metadata)
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

	decision, err := uc.consumeTextRequest(userID, "generate_feedback")
	if err != nil {
		return nil, err
	}

	modelOverride := uc.applyTextFUPDecision(decision)
	analysis, err := uc.analyzeAnswerWithModel(question, answer, domain.InterviewLanguageEnglish, modelOverride)
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

func (uc *interviewUseCase) SubmitAgentFeedback(userID, sessionID, questionID, question, answer string, analysis *domain.AnswerAnalysis) (*domain.FeedbackRecord, error) {
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
	if analysis == nil {
		return nil, errors.New("analysis is required")
	}

	if analysis.Score < 0 {
		analysis.Score = 0
	}
	if analysis.Score > 100 {
		analysis.Score = 100
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

	progress, err := uc.GetProgress(userID)
	if err != nil {
		return nil, err
	}

	sessions, err := uc.ListPracticeSessions(userID)
	if err != nil {
		return nil, err
	}

	recent := sessions
	if len(recent) > 10 {
		recent = recent[:10]
	}

	return &domain.AnalyticsOverview{
		AverageScore:      progress.AverageScore,
		SessionsCompleted: progress.SessionsCompleted,
		WeakAreas:         progress.WeakAreas,
		RecentSessions:    recent,
	}, nil
}

func (uc *interviewUseCase) TouchSessionActivity(userID, sessionID string) error {
	if strings.TrimSpace(userID) == "" {
		return errors.New("user id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return errors.New("session id is required")
	}

	return uc.repo.TouchSessionActivity(userID, sessionID)
}

func (uc *interviewUseCase) AbandonIdleSessions(idleFor time.Duration) (int64, error) {
	if idleFor <= 0 {
		return 0, errors.New("idle duration must be greater than zero")
	}

	return uc.repo.AbandonIdleSessions(idleFor)
}

func (uc *interviewUseCase) consumeJDParse(userID, source string) error {
	if uc.subscriptionService == nil {
		return nil
	}

	_, err := uc.subscriptionService.ConsumeJDParse(userID, source)
	return err
}

func (uc *interviewUseCase) consumeTextRequest(userID, source string) (*subscription.TextUsageDecision, error) {
	if uc.subscriptionService == nil {
		return nil, nil
	}

	return uc.subscriptionService.ConsumeTextRequest(userID, source)
}

func (uc *interviewUseCase) applyTextFUPDecision(decision *subscription.TextUsageDecision) string {
	if uc.subscriptionService == nil || decision == nil {
		return ""
	}

	if decision.ShouldDelayResponse {
		time.Sleep(uc.subscriptionService.TextFUPDelay())
	}

	if !decision.FUPExceeded {
		return ""
	}

	if model := strings.TrimSpace(decision.SuggestedModel); model != "" {
		return model
	}

	return uc.subscriptionService.TextFUPDowngradeModel()
}

func (uc *interviewUseCase) parseJobDescriptionWithModel(jobDescription, modelOverride string) (*domain.JobInsights, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return uc.aiService.ParseJobDescription(jobDescription)
	}

	overrideService, ok := uc.aiService.(aiModelOverrideService)
	if !ok {
		return uc.aiService.ParseJobDescription(jobDescription)
	}

	return overrideService.ParseJobDescriptionWithModel(jobDescription, modelOverride)
}

func (uc *interviewUseCase) generateQuestionsWithModel(
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
	modelOverride string,
) ([]domain.GeneratedQuestion, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return uc.aiService.GenerateQuestions(resumeText, jobDescription, interviewLanguage, interviewMode, interviewDifficulty)
	}

	overrideService, ok := uc.aiService.(aiModelOverrideService)
	if !ok {
		return uc.aiService.GenerateQuestions(resumeText, jobDescription, interviewLanguage, interviewMode, interviewDifficulty)
	}

	return overrideService.GenerateQuestionsWithModel(
		resumeText,
		jobDescription,
		interviewLanguage,
		interviewMode,
		interviewDifficulty,
		modelOverride,
	)
}

func (uc *interviewUseCase) analyzeAnswerWithModel(
	question,
	answer string,
	interviewLanguage domain.InterviewLanguage,
	modelOverride string,
) (*domain.AnswerAnalysis, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return uc.aiService.AnalyzeAnswer(question, answer, interviewLanguage)
	}

	overrideService, ok := uc.aiService.(aiModelOverrideService)
	if !ok {
		return uc.aiService.AnalyzeAnswer(question, answer, interviewLanguage)
	}

	return overrideService.AnalyzeAnswerWithModel(question, answer, interviewLanguage, modelOverride)
}

func (uc *interviewUseCase) analyzeResumeWithModel(resumeText, modelOverride string) (*domain.ResumeAIAnalysis, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return uc.aiService.AnalyzeResume(resumeText)
	}

	overrideService, ok := uc.aiService.(aiModelOverrideService)
	if !ok {
		return uc.aiService.AnalyzeResume(resumeText)
	}

	return overrideService.AnalyzeResumeWithModel(resumeText, modelOverride)
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

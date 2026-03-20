package usecase

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	resumeStorage       domain.ResumeFileStorage
	subscriptionService *subscription.Service
}

// NewInterviewUseCase creates a usecase for interview business workflows.
func NewInterviewUseCase(
	aiService domain.AIService,
	repo domain.InterviewRepository,
	resumeStorage domain.ResumeFileStorage,
	subscriptionService *subscription.Service,
) domain.InterviewUseCase {
	return &interviewUseCase{
		aiService:           aiService,
		repo:                repo,
		resumeStorage:       resumeStorage,
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

	storagePath := ""
	if len(upload.FileData) > 0 {
		if uc.resumeStorage == nil {
			return nil, errors.New("resume storage is not configured")
		}

		uploadedPath, err := uc.resumeStorage.UploadResume(userID, upload.FileName, upload.ContentType, upload.FileData)
		if err != nil {
			return nil, err
		}
		storagePath = uploadedPath
	}

	return uc.repo.SaveResumeWithFilePath(userID, content, storagePath)
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

	var latestResume *domain.ResumeRecord
	content := strings.TrimSpace(upload.Content)
	if content == "" && len(upload.FileData) > 0 {
		content = strings.TrimSpace(string(upload.FileData))
	}

	if len(upload.FileData) > 0 || content != "" {
		savedResume, err := uc.SaveResume(userID, upload)
		if err != nil {
			return nil, err
		}
		latestResume = savedResume
		content = strings.TrimSpace(savedResume.Content)
	} else {
		latest, err := uc.repo.GetLatestResume(userID)
		if err != nil {
			return nil, err
		}
		if latest == nil || strings.TrimSpace(latest.Content) == "" {
			return nil, errors.New("resume content is required")
		}
		latestResume = latest
		content = latest.Content
	}

	decision, err := uc.consumeTextRequest(userID, "analyze_resume")
	if err != nil {
		return nil, err
	}

	modelOverride := uc.applyTextFUPDecision(decision)
	analysisLanguage := normalizeResumeAnalysisLanguage(upload.Language)
	modelKey := normalizeAnalysisModelKey(modelOverride, analysisLanguage)
	contentHash := hashResumeContent(content)

	cached, err := uc.repo.FindResumeAnalysisByHash(userID, contentHash, modelKey)
	if err != nil {
		return nil, err
	}
	if cached != nil {
		return &domain.ResumeAIAnalysis{
			Summary:         cached.Summary,
			Response:        cached.Response,
			Highlights:      append([]string{}, cached.Highlights...),
			Recommendations: append([]string{}, cached.Recommendations...),
		}, nil
	}

	analysisInput := buildResumeAnalysisInput(content, analysisLanguage)
	analysis, err := uc.analyzeResumeWithModel(analysisInput, modelOverride)
	if err != nil {
		return nil, err
	}

	analysis = uc.ensureResumeAnalysisLanguage(analysis, analysisLanguage)

	resumeID := ""
	if latestResume != nil {
		resumeID = latestResume.ID
	}

	if _, err := uc.repo.SaveResumeAnalysis(userID, resumeID, contentHash, modelKey, analysis); err != nil {
		return nil, err
	}

	return analysis, nil
}

func (uc *interviewUseCase) GetLatestResumeAnalysis(userID, language string) (*domain.ResumeAIAnalysis, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	analysisLanguage := normalizeResumeAnalysisLanguage(language)
	record, err := uc.repo.GetLatestResumeAnalysisByLanguage(userID, analysisLanguage)
	if err != nil {
		return nil, err
	}
	if record == nil {
		latestRecord, latestErr := uc.repo.GetLatestResumeAnalysis(userID)
		if latestErr != nil {
			return nil, latestErr
		}
		if latestRecord == nil {
			return nil, nil
		}

		translated := resumeAnalysisFromRecord(latestRecord)
		translated = uc.ensureResumeAnalysisLanguage(translated, analysisLanguage)
		if translated == nil {
			return nil, nil
		}

		_, _ = uc.repo.SaveResumeAnalysis(
			userID,
			latestRecord.ResumeID,
			latestRecord.ContentHash,
			modelKeyWithLanguage(latestRecord.Model, analysisLanguage),
			translated,
		)

		return translated, nil
	}

	analysis := resumeAnalysisFromRecord(record)
	adjusted := uc.ensureResumeAnalysisLanguage(analysis, analysisLanguage)
	if adjusted != nil && !sameResumeAnalysisContent(analysis, adjusted) {
		_, _ = uc.repo.SaveResumeAnalysis(
			userID,
			record.ResumeID,
			record.ContentHash,
			modelKeyWithLanguage(record.Model, analysisLanguage),
			adjusted,
		)
		return adjusted, nil
	}

	return analysis, nil
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

	if strings.TrimSpace(latest.MinIOPath) != "" {
		if uc.resumeStorage == nil {
			return nil, errors.New("resume storage is not configured")
		}

		file, err := uc.resumeStorage.DownloadResume(latest.MinIOPath)
		if err != nil {
			return nil, err
		}

		return file, nil
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

func (uc *interviewUseCase) StartReviewSession(userID string, input domain.ReviewStartInput) (*domain.ReviewSession, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	input.SessionType = normalizeReviewSessionType(input.SessionType)
	input.InputMode = normalizeReviewInputMode(input.InputMode)

	if strings.TrimSpace(input.InputText) == "" && strings.TrimSpace(input.TranscriptText) == "" {
		return nil, errors.New("input text or transcript is required")
	}

	memory, err := uc.repo.GetCoachingMemory(userID)
	if err != nil {
		return nil, err
	}

	session, err := uc.repo.CreateReviewSession(userID, input)
	if err != nil {
		return nil, err
	}

	userInput := strings.TrimSpace(input.InputText)
	if userInput == "" {
		userInput = strings.TrimSpace(input.TranscriptText)
	}

	feedback, scoreReady := uc.analyzeReviewTurn(domain.ReviewAIInput{
		SessionType:     input.SessionType,
		InputMode:       input.InputMode,
		UserInput:       userInput,
		InterviewPrompt: input.InterviewPrompt,
		TargetRole:      input.TargetRole,
		TargetCompany:   input.TargetCompany,
		Memory:          *memory,
	})

	updated, err := uc.repo.UpdateReviewSessionFeedback(userID, session.ID, feedback, userInput)
	if err != nil {
		return nil, err
	}

	if scoreReady {
		_, _ = uc.repo.SaveProgressTrackingPoint(userID, updated.ID, domain.ProgressTrackingPoint{
			Communication: updated.Feedback.Communication,
			StructureSTAR: updated.Feedback.StructureSTAR,
			Confidence:    updated.Feedback.Confidence,
			OverallScore:  updated.Feedback.Score,
			Notes:         updated.Feedback.Insight,
		})
	}

	return updated, nil
}

func (uc *interviewUseCase) RespondReviewSession(userID string, input domain.ReviewRespondInput) (*domain.ReviewSession, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(input.SessionID) == "" {
		return nil, errors.New("session id is required")
	}

	session, err := uc.repo.GetReviewSession(userID, input.SessionID)
	if err != nil {
		return nil, err
	}
	if session.Status != domain.SessionStatusActive {
		return nil, errors.New("review session is not active")
	}

	memory, err := uc.repo.GetCoachingMemory(userID)
	if err != nil {
		return nil, err
	}

	userInput := strings.TrimSpace(input.InputText)
	if userInput == "" {
		userInput = strings.TrimSpace(input.TranscriptText)
	}
	if userInput == "" {
		return nil, errors.New("input text or transcript is required")
	}

	feedback, scoreReady := uc.analyzeReviewTurn(domain.ReviewAIInput{
		SessionType:     session.SessionType,
		InputMode:       session.InputMode,
		UserInput:       userInput,
		InterviewPrompt: input.InterviewPrompt,
		TargetRole:      session.RoleTarget,
		TargetCompany:   session.CompanyTarget,
		Memory:          *memory,
	})

	updated, err := uc.repo.UpdateReviewSessionFeedback(userID, session.ID, feedback, userInput)
	if err != nil {
		return nil, err
	}

	if scoreReady {
		_, _ = uc.repo.SaveProgressTrackingPoint(userID, updated.ID, domain.ProgressTrackingPoint{
			Communication: updated.Feedback.Communication,
			StructureSTAR: updated.Feedback.StructureSTAR,
			Confidence:    updated.Feedback.Confidence,
			OverallScore:  updated.Feedback.Score,
			Notes:         updated.Feedback.Insight,
		})
	}

	return updated, nil
}

func (uc *interviewUseCase) EndReviewSession(userID, sessionID string) (*domain.ReviewEndResult, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}
	if strings.TrimSpace(sessionID) == "" {
		return nil, errors.New("session id is required")
	}

	session, err := uc.repo.GetReviewSession(userID, sessionID)
	if err != nil {
		return nil, err
	}

	history, err := uc.repo.ListRecentReviewSessions(userID, 15)
	if err != nil {
		return nil, err
	}

	memory, err := uc.repo.GetCoachingMemory(userID)
	if err != nil {
		return nil, err
	}

	plan, err := uc.aiService.GenerateImprovementPlan(history, *memory)
	if err != nil || plan == nil {
		plan = &domain.ImprovementPlan{
			FocusAreas:       []string{"strengthen STAR structure", "improve clarity in key points"},
			PracticePlan:     []string{"practice 2 mock answers with STAR", "record 1 voice reflection and self-review"},
			WeeklyTarget:     "complete 2 focused review sessions this week",
			NextSessionFocus: "answer depth and measurable outcomes",
		}
	}

	coachingSummary := strings.TrimSpace(session.Feedback.Insight)
	if coachingSummary == "" {
		coachingSummary = "You have clear potential. Focus on structure and measurable impact in your next answers."
	}

	completed, err := uc.repo.CompleteReviewSession(userID, sessionID, plan, coachingSummary)
	if err != nil {
		return nil, err
	}

	updatedMemory := domain.CoachingMemory{
		UserID:            userID,
		TargetRole:        firstNonEmpty(completed.RoleTarget, memory.TargetRole),
		Strengths:         uniqueStrings(completed.Feedback.Strengths, memory.Strengths),
		Weaknesses:        uniqueStrings(completed.Feedback.Weaknesses, memory.Weaknesses),
		PreferredLanguage: firstNonEmpty(memory.PreferredLanguage, "en"),
		LastSummary:       coachingSummary,
		FocusAreas:        append([]string{}, plan.FocusAreas...),
		NextActions:       append([]string{}, plan.PracticePlan...),
	}
	_, _ = uc.repo.UpsertCoachingMemory(updatedMemory)

	return &domain.ReviewEndResult{
		SessionID:       completed.ID,
		Feedback:        completed.Feedback,
		ImprovementPlan: *plan,
		CoachingSummary: coachingSummary,
	}, nil
}

func (uc *interviewUseCase) GetReviewProgress(userID string) (*domain.ReviewProgress, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	points, err := uc.repo.ListProgressTracking(userID, 30)
	if err != nil {
		return nil, err
	}

	if len(points) == 0 {
		return &domain.ReviewProgress{
			UserID:              userID,
			CommunicationTrend:  []domain.ProgressTrackingPoint{},
			StructureTrend:      []domain.ProgressTrackingPoint{},
			ConfidenceTrend:     []domain.ProgressTrackingPoint{},
			LatestOverallScore:  0,
			AverageOverallScore: 0,
		}, nil
	}

	communication := make([]domain.ProgressTrackingPoint, 0, len(points))
	structure := make([]domain.ProgressTrackingPoint, 0, len(points))
	confidence := make([]domain.ProgressTrackingPoint, 0, len(points))
	total := 0

	for _, point := range points {
		communication = append(communication, point)
		structure = append(structure, point)
		confidence = append(confidence, point)
		total += point.OverallScore
	}

	latest := points[0].OverallScore
	avg := float64(total) / float64(len(points))
	avg = math.Round(avg*100) / 100

	return &domain.ReviewProgress{
		UserID:              userID,
		CommunicationTrend:  communication,
		StructureTrend:      structure,
		ConfidenceTrend:     confidence,
		LatestOverallScore:  latest,
		AverageOverallScore: avg,
	}, nil
}

func (uc *interviewUseCase) GetCoachingSummary(userID string) (*domain.ReviewEndResult, error) {
	if strings.TrimSpace(userID) == "" {
		return nil, errors.New("user id is required")
	}

	memory, err := uc.repo.GetCoachingMemory(userID)
	if err != nil {
		return nil, err
	}
	sessions, err := uc.repo.ListRecentReviewSessions(userID, 1)
	if err != nil {
		return nil, err
	}

	feedback := domain.ReviewAIFeedback{
		Strengths:   append([]string{}, memory.Strengths...),
		Weaknesses:  append([]string{}, memory.Weaknesses...),
		Suggestions: append([]string{}, memory.NextActions...),
		Insight:     memory.LastSummary,
	}

	sessionID := ""
	if len(sessions) > 0 {
		sessionID = sessions[0].ID
		feedback = sessions[0].Feedback
	}

	plan := domain.ImprovementPlan{
		FocusAreas:       append([]string{}, memory.FocusAreas...),
		PracticePlan:     append([]string{}, memory.NextActions...),
		WeeklyTarget:     "complete focused review sessions this week",
		NextSessionFocus: firstItem(memory.FocusAreas),
	}

	return &domain.ReviewEndResult{
		SessionID:       sessionID,
		Feedback:        feedback,
		ImprovementPlan: plan,
		CoachingSummary: memory.LastSummary,
	}, nil
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

func normalizeReviewSessionType(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == domain.ReviewSessionTypeRecovery {
		return domain.ReviewSessionTypeRecovery
	}
	return domain.ReviewSessionTypeStandard
}

func hashResumeContent(content string) string {
	normalized := strings.TrimSpace(strings.ToLower(content))
	sum := sha256.Sum256([]byte(normalized))
	return hex.EncodeToString(sum[:])
}

func normalizeAnalysisModelKey(modelOverride, language string) string {
	trimmed := strings.TrimSpace(modelOverride)
	if trimmed == "" {
		trimmed = "default"
	}
	return trimmed + "|lang:" + normalizeResumeAnalysisLanguage(language)
}

func normalizeResumeAnalysisLanguage(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == string(domain.InterviewLanguageIndonesian) {
		return string(domain.InterviewLanguageIndonesian)
	}
	return string(domain.InterviewLanguageEnglish)
}

func buildResumeAnalysisInput(content, language string) string {
	language = normalizeResumeAnalysisLanguage(language)
	if language == string(domain.InterviewLanguageIndonesian) {
		return "Please return all output fields in Bahasa Indonesia.\n\n" + content
	}

	return "Please return all output fields in English.\n\n" + content
}

func buildResumeAnalysisTranslationInput(analysis *domain.ResumeAIAnalysis, language string) string {
	language = normalizeResumeAnalysisLanguage(language)
	encoded, err := json.Marshal(analysis)
	if err != nil {
		encoded = []byte(`{"summary":"","response":"","highlights":[],"recommendations":[]}`)
	}

	if language == string(domain.InterviewLanguageIndonesian) {
		return "Translate this resume analysis JSON into Bahasa Indonesia. Keep the same meaning, keep it concise, and return all output fields in Bahasa Indonesia.\n\nResume analysis JSON:\n" + string(encoded)
	}

	return "Translate this resume analysis JSON into English. Keep the same meaning, keep it concise, and return all output fields in English.\n\nResume analysis JSON:\n" + string(encoded)
}

func (uc *interviewUseCase) translateResumeAnalysisToLanguage(analysis *domain.ResumeAIAnalysis, language string) (*domain.ResumeAIAnalysis, error) {
	if analysis == nil {
		return nil, errors.New("analysis is required")
	}

	translationInput := buildResumeAnalysisTranslationInput(analysis, language)
	return uc.analyzeResumeWithModel(translationInput, "")
}

func (uc *interviewUseCase) ensureResumeAnalysisLanguage(analysis *domain.ResumeAIAnalysis, language string) *domain.ResumeAIAnalysis {
	if analysis == nil {
		return nil
	}

	language = normalizeResumeAnalysisLanguage(language)
	if language == string(domain.InterviewLanguageIndonesian) {
		if looksLikeIndonesianResumeAnalysis(analysis) {
			return analysis
		}

		translated, err := uc.translateResumeAnalysisToLanguage(analysis, language)
		if err == nil && translated != nil {
			return translated
		}
	}

	if language == string(domain.InterviewLanguageEnglish) {
		if looksLikeEnglishResumeAnalysis(analysis) {
			return analysis
		}

		translated, err := uc.translateResumeAnalysisToLanguage(analysis, language)
		if err == nil && translated != nil {
			return translated
		}
	}

	return analysis
}

func looksLikeEnglishResumeAnalysis(analysis *domain.ResumeAIAnalysis) bool {
	if analysis == nil {
		return false
	}

	blob := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		analysis.Summary,
		analysis.Response,
		strings.Join(analysis.Highlights, " "),
		strings.Join(analysis.Recommendations, " "),
	}, " ")))

	if blob == "" {
		return false
	}

	indicators := []string{
		"the cv indicates",
		"overall, the profile",
		"highlight measurable impact",
		"tailor headline",
		"add 2-3 quantified achievements",
		"leadership outcomes",
	}

	for _, marker := range indicators {
		if strings.Contains(blob, marker) {
			return true
		}
	}

	return false
}

func looksLikeIndonesianResumeAnalysis(analysis *domain.ResumeAIAnalysis) bool {
	if analysis == nil {
		return false
	}

	blob := strings.ToLower(strings.TrimSpace(strings.Join([]string{
		analysis.Summary,
		analysis.Response,
		strings.Join(analysis.Highlights, " "),
		strings.Join(analysis.Recommendations, " "),
	}, " ")))

	if blob == "" {
		return false
	}

	indicators := []string{
		"cv ini",
		"secara keseluruhan",
		"dampak",
		"pencapaian",
		"kepemimpinan",
		"rekomendasi",
	}

	for _, marker := range indicators {
		if strings.Contains(blob, marker) {
			return true
		}
	}

	return false
}

func sameResumeAnalysisContent(a, b *domain.ResumeAIAnalysis) bool {
	if a == nil || b == nil {
		return a == b
	}

	if strings.TrimSpace(a.Summary) != strings.TrimSpace(b.Summary) {
		return false
	}
	if strings.TrimSpace(a.Response) != strings.TrimSpace(b.Response) {
		return false
	}
	if len(a.Highlights) != len(b.Highlights) || len(a.Recommendations) != len(b.Recommendations) {
		return false
	}

	for index := range a.Highlights {
		if strings.TrimSpace(a.Highlights[index]) != strings.TrimSpace(b.Highlights[index]) {
			return false
		}
	}
	for index := range a.Recommendations {
		if strings.TrimSpace(a.Recommendations[index]) != strings.TrimSpace(b.Recommendations[index]) {
			return false
		}
	}

	return true
}

func modelKeyWithLanguage(model, language string) string {
	language = normalizeResumeAnalysisLanguage(language)
	base := strings.TrimSpace(model)
	if base == "" {
		base = "default"
	}

	lower := strings.ToLower(base)
	if marker := strings.LastIndex(lower, "|lang:"); marker >= 0 {
		base = strings.TrimSpace(base[:marker])
	}

	if base == "" {
		base = "default"
	}

	return base + "|lang:" + language
}

func resumeAnalysisFromRecord(record *domain.ResumeAnalysisRecord) *domain.ResumeAIAnalysis {
	if record == nil {
		return nil
	}

	return &domain.ResumeAIAnalysis{
		Summary:         record.Summary,
		Response:        record.Response,
		Highlights:      append([]string{}, record.Highlights...),
		Recommendations: append([]string{}, record.Recommendations...),
	}
}

func normalizeReviewInputMode(value string) string {
	trimmed := strings.TrimSpace(strings.ToLower(value))
	if trimmed == string(domain.InterviewModeVoice) {
		return string(domain.InterviewModeVoice)
	}
	return string(domain.InterviewModeText)
}

func (uc *interviewUseCase) analyzeReviewTurn(input domain.ReviewAIInput) (domain.ReviewAIFeedback, bool) {
	feedback, err := uc.aiService.AnalyzeReview(input)
	if err == nil && feedback != nil {
		return *feedback, true
	}

	return domain.ReviewAIFeedback{
		Score:            0,
		Communication:    0,
		StructureSTAR:    0,
		Confidence:       0,
		Strengths:        []string{"you reflected honestly on your experience"},
		Weaknesses:       []string{"detailed scoring is temporarily unavailable"},
		Suggestions:      []string{"retry once to get full scoring", "share a STAR-formatted answer for deeper analysis"},
		Insight:          "Scoring could not be generated right now, but your reflection is still useful for coaching.",
		FollowUpQuestion: "What exact question did the interviewer ask, and how did you structure your answer?",
	}, false
}

func uniqueStrings(primary []string, fallback []string) []string {
	set := make(map[string]struct{})
	result := make([]string, 0)

	for _, value := range append(primary, fallback...) {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		if _, exists := set[clean]; exists {
			continue
		}
		set[clean] = struct{}{}
		result = append(result, clean)
	}

	return result
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean != "" {
			return clean
		}
	}
	return ""
}

func firstItem(values []string) string {
	if len(values) == 0 {
		return ""
	}
	return values[0]
}

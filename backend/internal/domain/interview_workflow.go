package domain

import "time"

const (
	SessionStatusActive                             = "active"
	SessionStatusCompleted                          = "completed"
	SessionStatusAbandoned                          = "abandoned"
	ReviewSessionTypeStandard                       = "review"
	ReviewSessionTypeRecovery                       = "recovery"
	InterviewLanguageEnglish    InterviewLanguage   = "en"
	InterviewLanguageIndonesian InterviewLanguage   = "id"
	InterviewModeText           InterviewMode       = "text"
	InterviewModeVoice          InterviewMode       = "voice"
	InterviewDifficultyEasy     InterviewDifficulty = "easy"
	InterviewDifficultyMedium   InterviewDifficulty = "medium"
	InterviewDifficultyHard     InterviewDifficulty = "hard"
)

type InterviewLanguage string
type InterviewMode string
type InterviewDifficulty string

type SessionMetadata struct {
	InterviewMode       string
	InterviewLanguage   InterviewLanguage
	InterviewDifficulty InterviewDifficulty
	TargetRole          string
	TargetCompany       string
}

func NormalizeInterviewLanguage(value string) InterviewLanguage {
	switch InterviewLanguage(value) {
	case InterviewLanguageIndonesian:
		return InterviewLanguageIndonesian
	default:
		return InterviewLanguageEnglish
	}
}

func NormalizeInterviewMode(value string) InterviewMode {
	switch InterviewMode(value) {
	case InterviewModeVoice:
		return InterviewModeVoice
	default:
		return InterviewModeText
	}
}

func NormalizeInterviewDifficulty(value string) InterviewDifficulty {
	switch InterviewDifficulty(value) {
	case InterviewDifficultyEasy:
		return InterviewDifficultyEasy
	case InterviewDifficultyHard:
		return InterviewDifficultyHard
	default:
		return InterviewDifficultyMedium
	}
}

// ParsedJobDescription stores one parsed job description and its extracted insights.
type ParsedJobDescription struct {
	ID             string      `json:"id"`
	UserID         string      `json:"user_id"`
	RawDescription string      `json:"raw_description"`
	Insights       JobInsights `json:"insights"`
	CreatedAt      time.Time   `json:"created_at"`
}

// ResumeRecord stores raw resume input from a user.
type ResumeRecord struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	Content   string    `json:"content"`
	MinIOPath string    `json:"minio_path,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type ResumeUpload struct {
	Content     string
	FileName    string
	ContentType string
	FileData    []byte
	Language    string
}

type ResumeFile struct {
	FileName    string
	ContentType string
	Data        []byte
}

type ResumeFileStorage interface {
	UploadResume(userID, fileName, contentType string, data []byte) (string, error)
	DownloadResume(minIOPath string) (*ResumeFile, error)
	DeleteResume(minIOPath string) error
}

type AnalyticsOverview struct {
	AverageScore      float64           `json:"average_score"`
	SessionsCompleted int               `json:"sessions_completed"`
	WeakAreas         []string          `json:"weak_areas"`
	RecentSessions    []PracticeSession `json:"recent_sessions"`
}

// StoredQuestion represents one generated question persisted by backend.
type StoredQuestion struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	ResumeID   string    `json:"resume_id"`
	JobParseID string    `json:"job_parse_id"`
	Type       string    `json:"type"`
	Question   string    `json:"question"`
	CreatedAt  time.Time `json:"created_at"`
}

// PracticeSession represents one interview practice lifecycle.
type PracticeSession struct {
	ID                  string     `json:"id"`
	UserID              string     `json:"user_id"`
	ResumeID            string     `json:"resume_id"`
	JobParseID          string     `json:"job_parse_id"`
	QuestionIDs         []string   `json:"question_ids"`
	InterviewMode       string     `json:"interview_mode"`
	InterviewLanguage   string     `json:"interview_language"`
	InterviewDifficulty string     `json:"interview_difficulty"`
	TargetRole          string     `json:"target_role,omitempty"`
	TargetCompany       string     `json:"target_company,omitempty"`
	Status              string     `json:"status"`
	Score               int        `json:"score"`
	CreatedAt           time.Time  `json:"created_at"`
	LastActivityAt      time.Time  `json:"last_activity_at"`
	CompletedAt         *time.Time `json:"completed_at,omitempty"`
}

// SessionAnswer represents one submitted answer in an interview session.
type SessionAnswer struct {
	ID         string    `json:"id"`
	SessionID  string    `json:"session_id"`
	QuestionID string    `json:"question_id"`
	UserID     string    `json:"user_id"`
	Answer     string    `json:"answer"`
	CreatedAt  time.Time `json:"created_at"`
}

// FeedbackRecord stores generated AI feedback for an answer.
type FeedbackRecord struct {
	ID           string    `json:"id"`
	UserID       string    `json:"user_id"`
	SessionID    string    `json:"session_id"`
	QuestionID   string    `json:"question_id"`
	Question     string    `json:"question"`
	Answer       string    `json:"answer"`
	Score        int       `json:"score"`
	Strengths    []string  `json:"strengths"`
	Weaknesses   []string  `json:"weaknesses"`
	Improvements []string  `json:"improvements"`
	STARFeedback string    `json:"star_feedback"`
	CreatedAt    time.Time `json:"created_at"`
}

// ProgressMetrics stores aggregated user analytics for dashboard queries.
type ProgressMetrics struct {
	UserID            string    `json:"user_id"`
	AverageScore      float64   `json:"average_score"`
	WeakAreas         []string  `json:"weak_areas"`
	SessionsCompleted int       `json:"sessions_completed"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ReviewSession struct {
	ID              string           `json:"id"`
	UserID          string           `json:"user_id"`
	SessionType     string           `json:"session_type"`
	InputMode       string           `json:"input_mode"`
	InputText       string           `json:"input_text,omitempty"`
	VoiceURL        string           `json:"voice_url,omitempty"`
	TranscriptText  string           `json:"transcript_text,omitempty"`
	RoleTarget      string           `json:"role_target,omitempty"`
	CompanyTarget   string           `json:"company_target,omitempty"`
	Status          string           `json:"status"`
	Feedback        ReviewAIFeedback `json:"feedback"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
	CompletedAt     *time.Time       `json:"completed_at,omitempty"`
	ImprovementPlan *ImprovementPlan `json:"improvement_plan,omitempty"`
}

type CoachingMemory struct {
	UserID            string    `json:"user_id"`
	TargetRole        string    `json:"target_role,omitempty"`
	Strengths         []string  `json:"strengths"`
	Weaknesses        []string  `json:"weaknesses"`
	PreferredLanguage string    `json:"preferred_language"`
	LastSummary       string    `json:"last_summary"`
	FocusAreas        []string  `json:"focus_areas"`
	NextActions       []string  `json:"next_actions"`
	UpdatedAt         time.Time `json:"updated_at"`
}

type ProgressTrackingPoint struct {
	ReviewSessionID string    `json:"review_session_id,omitempty"`
	Communication   int       `json:"communication"`
	StructureSTAR   int       `json:"structure_star"`
	Confidence      int       `json:"confidence"`
	OverallScore    int       `json:"overall_score"`
	Notes           string    `json:"notes,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

type ReviewProgress struct {
	UserID              string                  `json:"user_id"`
	CommunicationTrend  []ProgressTrackingPoint `json:"communication_trend"`
	StructureTrend      []ProgressTrackingPoint `json:"structure_trend"`
	ConfidenceTrend     []ProgressTrackingPoint `json:"confidence_trend"`
	LatestOverallScore  int                     `json:"latest_overall_score"`
	AverageOverallScore float64                 `json:"average_overall_score"`
}

type ReviewStartInput struct {
	SessionType     string `json:"session_type"`
	InputMode       string `json:"input_mode"`
	InputText       string `json:"input_text,omitempty"`
	VoiceURL        string `json:"voice_url,omitempty"`
	TranscriptText  string `json:"transcript_text,omitempty"`
	InterviewPrompt string `json:"interview_prompt,omitempty"`
	TargetRole      string `json:"target_role,omitempty"`
	TargetCompany   string `json:"target_company,omitempty"`
}

type ReviewRespondInput struct {
	SessionID       string `json:"session_id"`
	InputText       string `json:"input_text,omitempty"`
	VoiceURL        string `json:"voice_url,omitempty"`
	TranscriptText  string `json:"transcript_text,omitempty"`
	InterviewPrompt string `json:"interview_prompt,omitempty"`
}

type ReviewEndResult struct {
	SessionID       string           `json:"session_id"`
	Feedback        ReviewAIFeedback `json:"feedback"`
	ImprovementPlan ImprovementPlan  `json:"improvement_plan"`
	CoachingSummary string           `json:"coaching_summary"`
}

// AgentFeedbackInput represents feedback generated externally (e.g. conversational agent)
// and submitted to backend as the source of truth for scoring.
type AgentFeedbackInput struct {
	QuestionID string         `json:"question_id"`
	Question   string         `json:"question"`
	Answer     string         `json:"answer"`
	Analysis   AnswerAnalysis `json:"analysis"`
	SessionID  string         `json:"session_id"`
	UserID     string         `json:"user_id"`
}

// InterviewRepository defines data persistence required by interview workflows.
type InterviewRepository interface {
	SaveParsedJob(userID, rawDescription string, insights *JobInsights) (*ParsedJobDescription, error)
	SaveResume(userID, content string) (*ResumeRecord, error)
	SaveResumeWithFilePath(userID, content, filePath string) (*ResumeRecord, error)
	GetLatestResume(userID string) (*ResumeRecord, error)
	SaveResumeAnalysis(userID, resumeID, contentHash, model string, analysis *ResumeAIAnalysis) (*ResumeAnalysisRecord, error)
	FindResumeAnalysisByHash(userID, contentHash, model string) (*ResumeAnalysisRecord, error)
	GetLatestResumeAnalysis(userID string) (*ResumeAnalysisRecord, error)
	GetLatestResumeAnalysisByLanguage(userID, language string) (*ResumeAnalysisRecord, error)
	SaveGeneratedQuestions(userID, resumeID, jobParseID string, questions []GeneratedQuestion) ([]StoredQuestion, error)
	CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string, metadata SessionMetadata) (*PracticeSession, error)
	CompletePracticeSession(userID, sessionID string) (*PracticeSession, error)
	ListPracticeSessions(userID string) ([]PracticeSession, error)
	SaveSessionAnswer(userID, sessionID, questionID, answer string) (*SessionAnswer, error)
	SaveFeedback(userID, sessionID, questionID, question, answer string, analysis *AnswerAnalysis) (*FeedbackRecord, error)
	ListFeedbackByUser(userID string) ([]FeedbackRecord, error)
	SaveProgressMetrics(userID string, averageScore float64, weakAreas []string, sessionsCompleted int) (*ProgressMetrics, error)
	GetProgressMetrics(userID string) (*ProgressMetrics, error)
	TouchSessionActivity(userID, sessionID string) error
	AbandonIdleSessions(idleFor time.Duration) (int64, error)
	CreateReviewSession(userID string, input ReviewStartInput) (*ReviewSession, error)
	GetReviewSession(userID, sessionID string) (*ReviewSession, error)
	UpdateReviewSessionFeedback(userID, sessionID string, feedback ReviewAIFeedback, appendInput string) (*ReviewSession, error)
	CompleteReviewSession(userID, sessionID string, plan *ImprovementPlan, summary string) (*ReviewSession, error)
	ListRecentReviewSessions(userID string, limit int) ([]ReviewSession, error)
	GetCoachingMemory(userID string) (*CoachingMemory, error)
	UpsertCoachingMemory(memory CoachingMemory) (*CoachingMemory, error)
	SaveProgressTrackingPoint(userID, reviewSessionID string, point ProgressTrackingPoint) (*ProgressTrackingPoint, error)
	ListProgressTracking(userID string, limit int) ([]ProgressTrackingPoint, error)
}

// InterviewUseCase defines interview workflows.
type InterviewUseCase interface {
	ParseJobDescription(userID, rawDescription string) (*ParsedJobDescription, error)
	SaveResume(userID string, upload ResumeUpload) (*ResumeRecord, error)
	GetLatestResume(userID string) (*ResumeRecord, error)
	AnalyzeResume(userID string, upload ResumeUpload) (*ResumeAIAnalysis, error)
	GetLatestResumeAnalysis(userID, language string) (*ResumeAIAnalysis, error)
	DownloadLatestResume(userID string) (*ResumeFile, error)
	GenerateQuestions(userID, resumeText, jobDescription string, interviewLanguage InterviewLanguage, interviewMode InterviewMode, interviewDifficulty InterviewDifficulty) ([]StoredQuestion, error)
	CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string, metadata SessionMetadata) (*PracticeSession, error)
	CompletePracticeSession(userID, sessionID string) (*PracticeSession, error)
	ListPracticeSessions(userID string) ([]PracticeSession, error)
	SubmitSessionAnswer(userID, sessionID, questionID, answer string) (*SessionAnswer, error)
	GenerateFeedback(userID, sessionID, questionID, question, answer string) (*FeedbackRecord, error)
	SubmitAgentFeedback(userID, sessionID, questionID, question, answer string, analysis *AnswerAnalysis) (*FeedbackRecord, error)
	AggregateProgress(userID string) (*ProgressMetrics, error)
	GetProgress(userID string) (*ProgressMetrics, error)
	GetAnalyticsOverview(userID string) (*AnalyticsOverview, error)
	TouchSessionActivity(userID, sessionID string) error
	AbandonIdleSessions(idleFor time.Duration) (int64, error)
	StartReviewSession(userID string, input ReviewStartInput) (*ReviewSession, error)
	RespondReviewSession(userID string, input ReviewRespondInput) (*ReviewSession, error)
	EndReviewSession(userID, sessionID string) (*ReviewEndResult, error)
	GetReviewProgress(userID string) (*ReviewProgress, error)
	GetCoachingSummary(userID string) (*ReviewEndResult, error)
}

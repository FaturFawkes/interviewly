package domain

import (
	"strings"
	"time"
)

const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
	SessionStatusAbandoned = "abandoned"
)

type InterviewLanguage string

const (
	InterviewLanguageEnglish    InterviewLanguage = "en"
	InterviewLanguageIndonesian InterviewLanguage = "id"
)

type InterviewMode string

const (
	InterviewModeText  InterviewMode = "text"
	InterviewModeVoice InterviewMode = "voice"
)

type InterviewDifficulty string

const (
	InterviewDifficultyEasy   InterviewDifficulty = "easy"
	InterviewDifficultyMedium InterviewDifficulty = "medium"
	InterviewDifficultyHard   InterviewDifficulty = "hard"
)

func NormalizeInterviewLanguage(raw string) InterviewLanguage {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch InterviewLanguage(normalized) {
	case InterviewLanguageIndonesian:
		return InterviewLanguageIndonesian
	default:
		return InterviewLanguageEnglish
	}
}

func NormalizeInterviewMode(raw string) InterviewMode {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch InterviewMode(normalized) {
	case InterviewModeVoice:
		return InterviewModeVoice
	default:
		return InterviewModeText
	}
}

func NormalizeInterviewDifficulty(raw string) InterviewDifficulty {
	normalized := strings.ToLower(strings.TrimSpace(raw))
	switch InterviewDifficulty(normalized) {
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

// ResumeUpload stores parsed resume text and optional original file payload.
type ResumeUpload struct {
	Content     string
	FileName    string
	ContentType string
	FileData    []byte
}

// ResumeFile contains downloadable CV object data.
type ResumeFile struct {
	FileName    string
	ContentType string
	Data        []byte
}

// ResumeAnalysisResult combines persisted resume record and AI analysis output.
type ResumeAnalysisResult struct {
	Resume   *ResumeRecord     `json:"resume"`
	Analysis *ResumeAIAnalysis `json:"analysis"`
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
	ID                string            `json:"id"`
	UserID            string            `json:"user_id"`
	ResumeID          string            `json:"resume_id"`
	JobParseID        string            `json:"job_parse_id"`
	InterviewMode     string            `json:"interview_mode"`
	InterviewLanguage InterviewLanguage `json:"interview_language"`
	TargetRole        string            `json:"target_role,omitempty"`
	TargetCompany     string            `json:"target_company,omitempty"`
	QuestionIDs       []string          `json:"question_ids"`
	Status            string            `json:"status"`
	Score             int               `json:"score"`
	CreatedAt         time.Time         `json:"created_at"`
	CompletedAt       *time.Time        `json:"completed_at,omitempty"`
}

type SessionMetadata struct {
	InterviewMode     string            `json:"interview_mode"`
	InterviewLanguage InterviewLanguage `json:"interview_language"`
	TargetRole        string            `json:"target_role"`
	TargetCompany     string            `json:"target_company"`
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

type AnalyticsOverview struct {
	InterviewReadiness int               `json:"interview_readiness"`
	AverageScore       float64           `json:"average_score"`
	AvgScoreTrend      int               `json:"avg_score_trend"`
	TotalSessions      int               `json:"total_sessions"`
	PracticeHours      float64           `json:"practice_hours"`
	PracticeStreakDays int               `json:"practice_streak_days"`
	WeakAreas          []string          `json:"weak_areas"`
	Recommendations    []string          `json:"recommendations"`
	RecentSessions     []PracticeSession `json:"recent_sessions"`
	ScoreHistory       []AnalyticsPoint  `json:"score_history"`
}

type AnalyticsPoint struct {
	Label string `json:"label"`
	Score int    `json:"score"`
}

// InterviewRepository defines data persistence required by interview workflows.
type InterviewRepository interface {
	SaveParsedJob(userID, rawDescription string, insights *JobInsights) (*ParsedJobDescription, error)
	SaveResume(userID, content, minIOPath string) (*ResumeRecord, error)
	GetLatestResume(userID string) (*ResumeRecord, error)
	SaveGeneratedQuestions(userID, resumeID, jobParseID string, questions []GeneratedQuestion) ([]StoredQuestion, error)
	CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string, metadata SessionMetadata) (*PracticeSession, error)
	ListPracticeSessions(userID string) ([]PracticeSession, error)
	CompletePracticeSession(userID, sessionID string) (*PracticeSession, error)
	SaveSessionAnswer(userID, sessionID, questionID, answer string) (*SessionAnswer, error)
	SaveFeedback(userID, sessionID, questionID, question, answer string, analysis *AnswerAnalysis) (*FeedbackRecord, error)
	ListFeedbackByUser(userID string) ([]FeedbackRecord, error)
	SaveProgressMetrics(userID string, averageScore float64, weakAreas []string, sessionsCompleted int) (*ProgressMetrics, error)
	GetProgressMetrics(userID string) (*ProgressMetrics, error)
}

// InterviewUseCase defines interview workflows.
type InterviewUseCase interface {
	ParseJobDescription(userID, rawDescription string) (*ParsedJobDescription, error)
	SaveResume(userID string, upload ResumeUpload) (*ResumeRecord, error)
	GetLatestResume(userID string) (*ResumeRecord, error)
	AnalyzeResume(userID string, upload ResumeUpload) (*ResumeAnalysisResult, error)
	DownloadLatestResume(userID string) (*ResumeFile, error)
	GenerateQuestions(
		userID,
		resumeText,
		jobDescription string,
		interviewLanguage InterviewLanguage,
		interviewMode InterviewMode,
		interviewDifficulty InterviewDifficulty,
	) ([]StoredQuestion, error)
	CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string, metadata SessionMetadata) (*PracticeSession, error)
	ListPracticeSessions(userID string) ([]PracticeSession, error)
	CompletePracticeSession(userID, sessionID string) (*PracticeSession, error)
	SubmitSessionAnswer(userID, sessionID, questionID, answer string) (*SessionAnswer, error)
	GenerateFeedback(userID, sessionID, questionID, question, answer string, interviewLanguage InterviewLanguage) (*FeedbackRecord, error)
	AggregateProgress(userID string) (*ProgressMetrics, error)
	GetProgress(userID string) (*ProgressMetrics, error)
	GetAnalyticsOverview(userID string) (*AnalyticsOverview, error)
}

// ResumeFileStorage defines object storage for original CV files.
type ResumeFileStorage interface {
	UploadResume(userID, fileName, contentType string, data []byte) (string, error)
	DownloadResume(minIOPath string) (*ResumeFile, error)
	DeleteResume(minIOPath string) error
}

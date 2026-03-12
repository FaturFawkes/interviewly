package domain

import "time"

const (
	SessionStatusActive    = "active"
	SessionStatusCompleted = "completed"
	SessionStatusAbandoned = "abandoned"
)

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
	CreatedAt time.Time `json:"created_at"`
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
	ID          string     `json:"id"`
	UserID      string     `json:"user_id"`
	ResumeID    string     `json:"resume_id"`
	JobParseID  string     `json:"job_parse_id"`
	QuestionIDs []string   `json:"question_ids"`
	Status      string     `json:"status"`
	Score       int        `json:"score"`
	CreatedAt   time.Time  `json:"created_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
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

// InterviewRepository defines data persistence required by interview workflows.
type InterviewRepository interface {
	SaveParsedJob(userID, rawDescription string, insights *JobInsights) (*ParsedJobDescription, error)
	SaveResume(userID, content string) (*ResumeRecord, error)
	SaveGeneratedQuestions(userID, resumeID, jobParseID string, questions []GeneratedQuestion) ([]StoredQuestion, error)
	CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string) (*PracticeSession, error)
	ListPracticeSessions(userID string) ([]PracticeSession, error)
	SaveSessionAnswer(userID, sessionID, questionID, answer string) (*SessionAnswer, error)
	SaveFeedback(userID, sessionID, questionID, question, answer string, analysis *AnswerAnalysis) (*FeedbackRecord, error)
	ListFeedbackByUser(userID string) ([]FeedbackRecord, error)
	SaveProgressMetrics(userID string, averageScore float64, weakAreas []string, sessionsCompleted int) (*ProgressMetrics, error)
	GetProgressMetrics(userID string) (*ProgressMetrics, error)
}

// InterviewUseCase defines interview workflows.
type InterviewUseCase interface {
	ParseJobDescription(userID, rawDescription string) (*ParsedJobDescription, error)
	SaveResume(userID, content string) (*ResumeRecord, error)
	GenerateQuestions(userID, resumeText, jobDescription string) ([]StoredQuestion, error)
	CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string) (*PracticeSession, error)
	ListPracticeSessions(userID string) ([]PracticeSession, error)
	SubmitSessionAnswer(userID, sessionID, questionID, answer string) (*SessionAnswer, error)
	GenerateFeedback(userID, sessionID, questionID, question, answer string) (*FeedbackRecord, error)
	AggregateProgress(userID string) (*ProgressMetrics, error)
	GetProgress(userID string) (*ProgressMetrics, error)
}

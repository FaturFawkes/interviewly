package domain

import "time"

// JobInsights represents structured parsing output from a raw job description.
type JobInsights struct {
	Skills    []string `json:"skills"`
	Keywords  []string `json:"keywords"`
	Themes    []string `json:"themes"`
	Seniority string   `json:"seniority"`
}

// GeneratedQuestion represents one interview question produced by AI logic.
type GeneratedQuestion struct {
	Type     string `json:"type"`
	Question string `json:"question"`
}

// AnswerAnalysis represents structured feedback for one interview answer.
type AnswerAnalysis struct {
	Score        int      `json:"score"`
	Strengths    []string `json:"strengths"`
	Weaknesses   []string `json:"weaknesses"`
	Improvements []string `json:"improvements"`
	STARFeedback string   `json:"star_feedback"`
}

// ResumeAIAnalysis represents AI summary output for uploaded CV content.
type ResumeAIAnalysis struct {
	Summary         string   `json:"summary"`
	Response        string   `json:"response"`
	Highlights      []string `json:"highlights"`
	Recommendations []string `json:"recommendations"`
}

type ResumeAnalysisRecord struct {
	ID              string    `json:"id"`
	UserID          string    `json:"user_id"`
	ResumeID        string    `json:"resume_id,omitempty"`
	ContentHash     string    `json:"content_hash"`
	Model           string    `json:"model"`
	Summary         string    `json:"summary"`
	Response        string    `json:"response"`
	Highlights      []string  `json:"highlights"`
	Recommendations []string  `json:"recommendations"`
	RawResponse     string    `json:"raw_response,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ReviewAIInput represents one coaching turn context for review mode.
type ReviewAIInput struct {
	SessionType     string         `json:"session_type"`
	InputMode       string         `json:"input_mode"`
	UserInput       string         `json:"user_input"`
	InterviewPrompt string         `json:"interview_prompt,omitempty"`
	TargetRole      string         `json:"target_role,omitempty"`
	TargetCompany   string         `json:"target_company,omitempty"`
	Memory          CoachingMemory `json:"memory"`
}

// ReviewAIFeedback is structured coaching output for review mode.
type ReviewAIFeedback struct {
	Score              int      `json:"score"`
	Communication      int      `json:"communication"`
	StructureSTAR      int      `json:"structure_star"`
	Confidence         int      `json:"confidence"`
	Strengths          []string `json:"strengths"`
	Weaknesses         []string `json:"weaknesses"`
	Suggestions        []string `json:"suggestions"`
	BetterAnswer       string   `json:"better_answer"`
	Insight            string   `json:"insight"`
	FollowUpQuestion   string   `json:"follow_up_question"`
	RecoverySimulation string   `json:"recovery_simulation,omitempty"`
}

// ImprovementPlan contains personalized actions generated from user history.
type ImprovementPlan struct {
	FocusAreas       []string `json:"focus_areas"`
	PracticePlan     []string `json:"practice_plan"`
	WeeklyTarget     string   `json:"weekly_target"`
	NextSessionFocus string   `json:"next_session_focus"`
}

// AIService defines all AI-related business capabilities required by backend workflows.
type AIService interface {
	ParseJobDescription(jobDescription string) (*JobInsights, error)
	GenerateQuestions(
		resumeText,
		jobDescription string,
		interviewLanguage InterviewLanguage,
		interviewMode InterviewMode,
		interviewDifficulty InterviewDifficulty,
	) ([]GeneratedQuestion, error)
	AnalyzeAnswer(question, answer string, interviewLanguage InterviewLanguage) (*AnswerAnalysis, error)
	AnalyzeResume(resumeText string) (*ResumeAIAnalysis, error)
	AnalyzeReview(input ReviewAIInput) (*ReviewAIFeedback, error)
	GenerateImprovementPlan(history []ReviewSession, memory CoachingMemory) (*ImprovementPlan, error)
}

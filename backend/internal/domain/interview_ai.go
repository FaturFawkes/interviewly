package domain

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
}

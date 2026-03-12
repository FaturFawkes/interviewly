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

// AIService defines all AI-related business capabilities required by backend workflows.
type AIService interface {
	ParseJobDescription(jobDescription string) (*JobInsights, error)
	GenerateQuestions(resumeText, jobDescription string) ([]GeneratedQuestion, error)
	AnalyzeAnswer(question, answer string) (*AnswerAnalysis, error)
}

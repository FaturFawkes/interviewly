package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/interview_app/backend/config"
	"github.com/interview_app/backend/internal/domain"
)

// Service is a lightweight AI abstraction layer that can later be swapped with real providers.
type Service struct {
	provider   string
	model      string
	apiBaseURL string
	apiKey     string
	httpClient *http.Client
}

func NewService(cfg *config.Config) domain.AIService {
	provider := "local"
	model := "gpt-4o-mini"
	apiBaseURL := "https://api.openai.com/v1"
	apiKey := ""

	if cfg != nil {
		provider = strings.ToLower(strings.TrimSpace(cfg.AIProvider))
		model = strings.TrimSpace(cfg.AIModel)
		apiBaseURL = strings.TrimRight(strings.TrimSpace(cfg.AIAPIBaseURL), "/")
		apiKey = strings.TrimSpace(cfg.AIAPIKey)
	}

	if provider == "" {
		provider = "local"
	}
	if model == "" {
		model = "gpt-4o-mini"
	}
	if apiBaseURL == "" {
		apiBaseURL = "https://api.openai.com/v1"
	}

	return &Service{
		provider:   provider,
		model:      model,
		apiBaseURL: apiBaseURL,
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (s *Service) ParseJobDescriptionWithModel(jobDescription, modelOverride string) (*domain.JobInsights, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return s.ParseJobDescription(jobDescription)
	}

	if s.useRemoteProvider() {
		if remote, err := s.remoteParseJobDescriptionWithModel(jobDescription, modelOverride); err == nil {
			return remote, nil
		}
	}

	return s.ParseJobDescription(jobDescription)
}

func (s *Service) GenerateQuestionsWithModel(
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
	modelOverride string,
) ([]domain.GeneratedQuestion, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return s.GenerateQuestions(resumeText, jobDescription, interviewLanguage, interviewMode, interviewDifficulty)
	}

	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	interviewMode = domain.NormalizeInterviewMode(string(interviewMode))
	interviewDifficulty = domain.NormalizeInterviewDifficulty(string(interviewDifficulty))

	if s.useRemoteProvider() {
		if remote, err := s.remoteGenerateQuestionsWithModel(
			resumeText,
			jobDescription,
			interviewLanguage,
			interviewMode,
			interviewDifficulty,
			modelOverride,
		); err == nil {
			return sanitizeGeneratedQuestionsByMode(remote, interviewMode), nil
		}
	}

	return s.GenerateQuestions(resumeText, jobDescription, interviewLanguage, interviewMode, interviewDifficulty)
}

func (s *Service) AnalyzeAnswerWithModel(
	question,
	answer string,
	interviewLanguage domain.InterviewLanguage,
	modelOverride string,
) (*domain.AnswerAnalysis, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return s.AnalyzeAnswer(question, answer, interviewLanguage)
	}

	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))

	if s.useRemoteProvider() {
		if remote, err := s.remoteAnalyzeAnswerWithModel(question, answer, interviewLanguage, modelOverride); err == nil {
			return remote, nil
		}
	}

	return s.AnalyzeAnswer(question, answer, interviewLanguage)
}

func (s *Service) AnalyzeResumeWithModel(resumeText, modelOverride string) (*domain.ResumeAIAnalysis, error) {
	if strings.TrimSpace(modelOverride) == "" {
		return s.AnalyzeResume(resumeText)
	}

	if s.useRemoteProvider() {
		if remote, err := s.remoteAnalyzeResumeWithModel(resumeText, modelOverride); err == nil {
			return remote, nil
		}
	}

	return s.AnalyzeResume(resumeText)
}

func (s *Service) ParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
	if s.useRemoteProvider() {
		if remote, err := s.remoteParseJobDescription(jobDescription); err == nil {
			return remote, nil
		}
	}

	normalized := strings.ToLower(jobDescription)
	tokens := tokenize(normalized)

	skills := detectSkills(normalized)
	keywords := topKeywords(tokens, 10)
	themes := detectThemes(normalized)
	seniority := detectSeniority(normalized)

	return &domain.JobInsights{
		Skills:    skills,
		Keywords:  keywords,
		Themes:    themes,
		Seniority: seniority,
	}, nil
}

func (s *Service) GenerateQuestions(
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
) ([]domain.GeneratedQuestion, error) {
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	interviewMode = domain.NormalizeInterviewMode(string(interviewMode))
	interviewDifficulty = domain.NormalizeInterviewDifficulty(string(interviewDifficulty))

	if s.useRemoteProvider() {
		if remote, err := s.remoteGenerateQuestions(
			resumeText,
			jobDescription,
			interviewLanguage,
			interviewMode,
			interviewDifficulty,
		); err == nil {
			return sanitizeGeneratedQuestionsByMode(remote, interviewMode), nil
		}
	}

	totalQuestions := questionCountByMode(interviewMode)
	behavioralCount := totalQuestions / 2
	technicalCount := totalQuestions - behavioralCount

	normalizedResume := strings.ToLower(resumeText)
	normalizedJobDescription := strings.ToLower(jobDescription)

	resumeTokens := topKeywords(tokenize(normalizedResume), 6)
	jobTokens := topKeywords(tokenize(normalizedJobDescription), 8)
	jobSkills := detectSkills(normalizedJobDescription)
	jobThemes := detectThemes(normalizedJobDescription)

	primaryResume := pickOrDefault(resumeTokens, 0, "your relevant project background")
	secondaryResume := pickOrDefault(resumeTokens, 1, "a key project")
	primaryJobSkill := pickOrDefault(jobSkills, 0, pickOrDefault(jobTokens, 0, "the core role requirements"))
	secondaryJobSkill := pickOrDefault(jobSkills, 1, pickOrDefault(jobTokens, 1, "the required technical stack"))
	jobTheme := pickOrDefault(jobThemes, 0, pickOrDefault(jobTokens, 2, "the main responsibilities"))

	if interviewLanguage == domain.InterviewLanguageIndonesian {
		behavioralTemplates := []string{
			"Ceritakan pengalaman paling relevan Anda yang secara langsung mendukung kebutuhan %s pada job description ini.",
			"Jelaskan situasi ketika Anda menghadapi tantangan pada area %s dan bagaimana Anda menyelesaikannya.",
			"Berikan contoh kolaborasi lintas tim untuk mencapai target yang terkait dengan %s.",
			"Dalam konteks tanggung jawab di JD ini, bagaimana Anda memprioritaskan pekerjaan saat deadline ketat?",
			"Ceritakan feedback kritis yang pernah Anda terima saat mengerjakan %s, serta tindakan perbaikannya.",
			"Ceritakan keputusan sulit yang pernah Anda ambil terkait prioritas pekerjaan pada area %s.",
			"Bagaimana Anda menangani konflik ekspektasi stakeholder saat mengerjakan inisiatif %s?",
			"Berikan contoh inisiatif proaktif Anda yang berdampak langsung pada objective tim di area %s.",
		}

		technicalTemplates := []string{
			"Jelaskan desain sistem/fitur yang pernah Anda bangun dan relevansinya dengan kebutuhan %s pada role ini.",
			"Jika diminta mengimplementasikan solusi untuk %s sesuai JD ini, apa pendekatan teknis Anda dari awal sampai deployment?",
			"Trade-off apa yang Anda pertimbangkan untuk scaling layanan yang menangani %s?",
			"Bagaimana strategi Anda melakukan debugging insiden produksi untuk layanan yang kritikal terhadap objective role ini?",
			"Bagaimana Anda memastikan reliability, observability, dan performance pada stack yang relevan dengan %s?",
			"Bagaimana Anda menyusun test strategy (unit/integration/e2e) untuk fitur yang terkait %s?",
			"Jelaskan pendekatan monitoring dan incident response Anda untuk sistem dengan kebutuhan %s.",
			"Apa pertimbangan security utama ketika membangun layanan yang memproses %s?",
		}

		behavioral := buildQuestionBatch(
			behavioralTemplates,
			[]string{primaryJobSkill, secondaryJobSkill, jobTheme, secondaryResume},
			"behavioral",
			behavioralCount,
			interviewLanguage,
			interviewDifficulty,
		)
		technical := buildQuestionBatch(
			technicalTemplates,
			[]string{primaryJobSkill, secondaryJobSkill, jobTheme},
			"technical",
			technicalCount,
			interviewLanguage,
			interviewDifficulty,
		)

		questions := make([]domain.GeneratedQuestion, 0, len(behavioral)+len(technical))
		questions = append(questions, behavioral...)
		questions = append(questions, technical...)
		return sanitizeGeneratedQuestionsByMode(questions, interviewMode), nil
	}

	behavioralTemplates := []string{
		"Tell me about an achievement that directly aligns with the job requirement around %s.",
		"Describe a challenge you faced in %s and how you resolved it.",
		"Share an example of cross-functional collaboration to deliver outcomes related to %s.",
		"Given the responsibilities in this JD, how do you prioritize when deadlines are tight and priorities shift?",
		"Describe critical feedback you received while working on %s and what changed afterward.",
		"Tell me about a difficult trade-off decision you made in an initiative involving %s.",
		"How do you handle conflicting stakeholder expectations while owning delivery in %s?",
		"Share a proactive improvement you drove that materially impacted goals tied to %s.",
	}

	technicalTemplates := []string{
		"Walk through a system or feature you built in %s and map it to this role's requirement in %s.",
		"How would you design an implementation plan for %s based on this JD?",
		"What trade-offs would you consider when scaling services that support %s?",
		"How do you approach debugging production issues in systems similar to this role's technical scope?",
		"How do you ensure reliability, observability, and performance for workloads tied to %s?",
		"How would you define a practical unit/integration/e2e testing strategy for work involving %s?",
		"What is your monitoring and incident-response approach for systems serving %s?",
		"What security controls would you prioritize for services handling %s?",
	}

	behavioral := buildQuestionBatch(
		behavioralTemplates,
		[]string{primaryJobSkill, secondaryJobSkill, jobTheme, secondaryResume},
		"behavioral",
		behavioralCount,
		interviewLanguage,
		interviewDifficulty,
	)
	technical := buildQuestionBatch(
		technicalTemplates,
		[]string{primaryResume, primaryJobSkill, secondaryJobSkill, jobTheme},
		"technical",
		technicalCount,
		interviewLanguage,
		interviewDifficulty,
	)

	questions := make([]domain.GeneratedQuestion, 0, len(behavioral)+len(technical))
	questions = append(questions, behavioral...)
	questions = append(questions, technical...)

	return sanitizeGeneratedQuestionsByMode(questions, interviewMode), nil
}

func sanitizeGeneratedQuestionsByMode(questions []domain.GeneratedQuestion, interviewMode domain.InterviewMode) []domain.GeneratedQuestion {
	if domain.NormalizeInterviewMode(string(interviewMode)) != domain.InterviewModeText {
		return questions
	}

	result := make([]domain.GeneratedQuestion, 0, len(questions))
	for _, question := range questions {
		cleaned := trimLeadingBracketExpressions(question.Question)
		if strings.TrimSpace(cleaned) == "" {
			cleaned = strings.TrimSpace(question.Question)
		}

		question.Question = cleaned
		result = append(result, question)
	}

	return result
}

func trimLeadingBracketExpressions(value string) string {
	cleaned := strings.TrimSpace(value)

	for strings.HasPrefix(cleaned, "[") {
		closingIndex := strings.Index(cleaned, "]")
		if closingIndex <= 1 {
			break
		}

		cleaned = strings.TrimSpace(cleaned[closingIndex+1:])
	}

	return cleaned
}

func (s *Service) AnalyzeAnswer(question, answer string, interviewLanguage domain.InterviewLanguage) (*domain.AnswerAnalysis, error) {
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))

	if s.useRemoteProvider() {
		if remote, err := s.remoteAnalyzeAnswer(question, answer, interviewLanguage); err == nil {
			return remote, nil
		}
	}

	normalizedAnswer := strings.TrimSpace(answer)
	lowerAnswer := strings.ToLower(normalizedAnswer)
	answerTokens := tokenize(lowerAnswer)
	wordCount := len(strings.Fields(normalizedAnswer))

	score := 45
	strengths := make([]string, 0)
	weaknesses := make([]string, 0)
	improvements := make([]string, 0)
	indonesian := interviewLanguage == domain.InterviewLanguageIndonesian

	if wordCount >= 40 {
		score += 20
		if indonesian {
			strengths = append(strengths, "detail jawaban cukup")
		} else {
			strengths = append(strengths, "sufficient detail")
		}
	} else {
		if indonesian {
			weaknesses = append(weaknesses, "jawaban terlalu singkat")
			improvements = append(improvements, "tambahkan konteks dan detail konkret")
		} else {
			weaknesses = append(weaknesses, "answer is too brief")
			improvements = append(improvements, "add more context and concrete details")
		}
	}

	if containsAnyToken(answerTokens, []string{"i", "saya", "aku", "kami", "we"}) {
		score += 10
		if indonesian {
			strengths = append(strengths, "kepemilikan tindakan jelas")
		} else {
			strengths = append(strengths, "clear ownership of actions")
		}
	} else {
		if indonesian {
			weaknesses = append(weaknesses, "kepemilikan tindakan kurang jelas")
			improvements = append(improvements, "jelaskan aksi spesifik yang Anda lakukan")
		} else {
			weaknesses = append(weaknesses, "ownership is unclear")
			improvements = append(improvements, "describe your specific actions")
		}
	}

	if containsAny(lowerAnswer, []string{"result", "impact", "%", "improved", "reduced", "increased", "hasil", "dampak", "meningkat", "menurunkan"}) {
		score += 15
		if indonesian {
			strengths = append(strengths, "menyebutkan hasil atau dampak")
		} else {
			strengths = append(strengths, "mentions outcome or impact")
		}
	} else {
		if indonesian {
			weaknesses = append(weaknesses, "belum ada hasil terukur")
			improvements = append(improvements, "sertakan hasil yang terukur jika memungkinkan")
		} else {
			weaknesses = append(weaknesses, "missing measurable outcomes")
			improvements = append(improvements, "include measurable results when possible")
		}
	}

	questionTokens := topKeywords(tokenize(strings.ToLower(question)), 5)
	answerTokenSet := make(map[string]struct{}, len(answerTokens))
	for _, token := range answerTokens {
		answerTokenSet[token] = struct{}{}
	}
	matchCount := 0
	for _, token := range questionTokens {
		if _, exists := answerTokenSet[token]; exists {
			matchCount++
		}
	}
	if matchCount >= 2 {
		score += 10
		if indonesian {
			strengths = append(strengths, "jawaban relevan dengan pertanyaan")
		} else {
			strengths = append(strengths, "answer is relevant to the question")
		}
	} else {
		if indonesian {
			weaknesses = append(weaknesses, "jawaban kurang fokus pada inti pertanyaan")
			improvements = append(improvements, "fokuskan jawaban langsung ke topik yang ditanya")
		} else {
			weaknesses = append(weaknesses, "answer drifts from the main question")
			improvements = append(improvements, "focus directly on the asked topic")
		}
	}

	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	if len(strengths) == 0 {
		if indonesian {
			strengths = append(strengths, "jawaban sudah diberikan")
		} else {
			strengths = append(strengths, "response provided")
		}
	}
	if len(weaknesses) == 0 {
		if indonesian {
			weaknesses = append(weaknesses, "masih ada ruang peningkatan pada kejelasan")
		} else {
			weaknesses = append(weaknesses, "minor clarity improvements possible")
		}
	}
	if len(improvements) == 0 {
		if indonesian {
			improvements = append(improvements, "gunakan struktur yang lebih jelas dengan kesimpulan singkat")
		} else {
			improvements = append(improvements, "add clearer structure and concise takeaway")
		}
	}

	starFeedback := "Use STAR framing: Situation and Task in 1-2 sentences, then specific Actions you took, and end with measurable Result."
	if indonesian {
		starFeedback = "Gunakan format STAR: jelaskan Situation dan Task secara singkat, lalu Actions spesifik yang Anda lakukan, dan akhiri dengan Result yang terukur."
	}

	return &domain.AnswerAnalysis{
		Score:        score,
		Strengths:    strengths,
		Weaknesses:   weaknesses,
		Improvements: improvements,
		STARFeedback: starFeedback,
	}, nil
}

func (s *Service) AnalyzeResume(resumeText string) (*domain.ResumeAIAnalysis, error) {
	if strings.TrimSpace(resumeText) == "" {
		return nil, fmt.Errorf("resume content is required")
	}

	if translationRequest, targetLanguage, sourceAnalysis := parseResumeAnalysisTranslationRequest(resumeText); translationRequest {
		if sourceAnalysis == nil {
			return nil, fmt.Errorf("resume analysis translation payload is invalid")
		}

		if s.useRemoteProvider() {
			if remote, err := s.remoteTranslateResumeAnalysis(sourceAnalysis, targetLanguage); err == nil {
				return ensureTranslationQuality(sourceAnalysis, remote, targetLanguage), nil
			}
		}

		return translateResumeAnalysisLocally(sourceAnalysis, targetLanguage), nil
	}

	if s.useRemoteProvider() {
		if remote, err := s.remoteAnalyzeResume(resumeText); err == nil {
			return remote, nil
		}
	}

	normalized := strings.ToLower(strings.TrimSpace(resumeText))
	keywords := topKeywords(tokenize(normalized), 8)

	primary := pickOrDefault(keywords, 0, "software development")
	secondary := pickOrDefault(keywords, 1, "problem solving")
	tertiary := pickOrDefault(keywords, 2, "delivery")

	highlights := make([]string, 0, 5)
	for index, keyword := range keywords {
		if index >= 5 {
			break
		}
		highlights = append(highlights, keyword)
	}
	if len(highlights) == 0 {
		highlights = []string{"general engineering", "collaboration", "execution"}
	}

	recommendations := []string{
		"Add 2-3 quantified achievements for key projects.",
		"Highlight measurable impact using metrics (%, time, revenue, scale).",
		"Tailor headline and recent experience toward the target role.",
	}

	if containsAny(normalized, []string{"lead", "mentoring", "ownership", "stakeholder"}) {
		recommendations = append([]string{"Emphasize leadership outcomes and scope ownership."}, recommendations...)
	}

	summary := fmt.Sprintf(
		"The CV indicates a profile focused on %s, %s, and %s with practical engineering exposure.",
		primary,
		secondary,
		tertiary,
	)

	response := "Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence."

	return &domain.ResumeAIAnalysis{
		Summary:         summary,
		Response:        response,
		Highlights:      highlights,
		Recommendations: recommendations,
	}, nil
}

func (s *Service) AnalyzeReview(input domain.ReviewAIInput) (*domain.ReviewAIFeedback, error) {
	if strings.TrimSpace(input.UserInput) == "" {
		return nil, fmt.Errorf("review input is required")
	}
	reviewLanguage := resolveReviewLanguage(input.InterviewLanguage, input.Memory.PreferredLanguage)
	input.InterviewLanguage = reviewLanguage
	isIndonesian := reviewLanguage == domain.InterviewLanguageIndonesian

	if s.useRemoteProvider() {
		if remote, err := s.remoteAnalyzeReview(input); err == nil {
			return remote, nil
		}
	}

	normalized := strings.ToLower(strings.TrimSpace(input.UserInput))
	score := 55
	communication := 55
	structure := 50
	confidence := 50

	strengths := []string{"you are self-aware about the interview outcome"}
	weaknesses := []string{}
	suggestions := []string{}
	if isIndonesian {
		strengths = []string{"Anda cukup sadar diri terhadap hasil interview Anda"}
	}

	if containsAny(normalized, []string{"result", "impact", "%", "improved", "increased", "decreased", "dampak", "hasil"}) {
		score += 12
		structure += 8
		if isIndonesian {
			strengths = append(strengths, "Anda menyebutkan hasil, bukan hanya aktivitas")
		} else {
			strengths = append(strengths, "you mentioned outcomes, not only activities")
		}
	} else {
		if isIndonesian {
			weaknesses = append(weaknesses, "cerita Anda belum memiliki hasil yang terukur")
			suggestions = append(suggestions, "tambahkan satu metrik konkret untuk setiap contoh utama")
		} else {
			weaknesses = append(weaknesses, "your story lacks measurable outcomes")
			suggestions = append(suggestions, "add one concrete metric for each key example")
		}
	}

	if containsAny(normalized, []string{"situation", "task", "action", "result", "star"}) {
		score += 14
		structure += 18
		if isIndonesian {
			strengths = append(strengths, "jawaban Anda memiliki struktur STAR yang jelas")
		} else {
			strengths = append(strengths, "your answer has recognizable STAR structure")
		}
	} else {
		if isIndonesian {
			weaknesses = append(weaknesses, "struktur jawaban masih kurang jelas")
			suggestions = append(suggestions, "gunakan urutan STAR: konteks, aksi, hasil terukur")
		} else {
			weaknesses = append(weaknesses, "answer structure is unclear")
			suggestions = append(suggestions, "use STAR order: context, action, measurable result")
		}
	}

	if containsAny(normalized, []string{"umm", "maybe", "not sure", "kayaknya", "mungkin", "gak yakin"}) {
		confidence -= 12
		if isIndonesian {
			weaknesses = append(weaknesses, "sinyal percaya diri terdengar ragu-ragu")
			suggestions = append(suggestions, "ganti kata-kata ragu dengan kalimat yang lebih tegas")
		} else {
			weaknesses = append(weaknesses, "confidence signal sounds hesitant")
			suggestions = append(suggestions, "replace hedging words with decisive phrasing")
		}
	} else {
		confidence += 10
		if isIndonesian {
			strengths = append(strengths, "tone jawaban Anda terdengar cukup percaya diri")
		} else {
			strengths = append(strengths, "your tone reads reasonably confident")
		}
	}

	if containsAny(normalized, []string{"because", "therefore", "so that", "karena", "sehingga"}) {
		communication += 12
		if isIndonesian {
			strengths = append(strengths, "alur alasan di jawaban Anda terlihat jelas")
		} else {
			strengths = append(strengths, "your reasoning chain is visible")
		}
	} else {
		if isIndonesian {
			weaknesses = append(weaknesses, "alasan di balik keputusan belum dijelaskan dengan cukup")
			suggestions = append(suggestions, "jelaskan secara eksplisit kenapa Anda memilih tiap aksi")
		} else {
			weaknesses = append(weaknesses, "reasoning behind decisions is under-explained")
			suggestions = append(suggestions, "explicitly explain why you chose each action")
		}
	}

	if len(weaknesses) == 0 {
		if isIndonesian {
			weaknesses = append(weaknesses, "masih ada ruang untuk meningkatkan kejelasan dan keringkasan")
		} else {
			weaknesses = append(weaknesses, "there is room to sharpen clarity and brevity")
		}
	}
	if len(suggestions) == 0 {
		if isIndonesian {
			suggestions = append(suggestions, "rapikan pembuka dalam 2 kalimat sebelum masuk detail")
		} else {
			suggestions = append(suggestions, "tighten your opening in 2 sentences before deep details")
		}
	}

	score = clampScore(score)
	communication = clampScore(communication)
	structure = clampScore(structure)
	confidence = clampScore(confidence)

	betterAnswer := "Use this pattern: 'In my previous role, I faced [Situation]. My goal was [Task]. I took [Action 1, 2]. As a result, we achieved [quantified Result]. If repeated, I would improve by [next step].'"
	insight := "Your main opportunity is to improve structure and quantified impact so interviewers can trust your execution level faster."
	followUpQuestion := "Which part of your original answer felt weakest to you: context, action depth, or measurable result?"
	if isIndonesian {
		betterAnswer = "Gunakan pola ini: 'Di peran sebelumnya, saya menghadapi [Situasi]. Target saya adalah [Tugas]. Saya melakukan [Aksi 1, 2]. Hasilnya, kami mencapai [Result terukur]. Jika diulang, saya akan meningkatkan [langkah berikutnya].'"
		insight = "Peluang terbesar Anda adalah memperkuat struktur dan dampak terukur agar interviewer lebih cepat percaya pada level eksekusi Anda."
		followUpQuestion = "Bagian mana dari jawaban Anda yang paling lemah: konteks, kedalaman aksi, atau hasil terukur?"
	}
	if strings.TrimSpace(input.InterviewPrompt) != "" {
		if isIndonesian {
			betterAnswer = fmt.Sprintf("Untuk pertanyaan '%s', mulai dengan konteks dalam satu kalimat, lalu jelaskan 2-3 aksi konkret yang Anda lakukan sendiri, dan tutup dengan hasil terukur serta pelajaran yang Anda dapat.", strings.TrimSpace(input.InterviewPrompt))
		} else {
			betterAnswer = fmt.Sprintf("For the question '%s', start with context in one sentence, then 2-3 concrete actions you personally took, and close with a measurable result and lesson learned.", strings.TrimSpace(input.InterviewPrompt))
		}
	}

	return &domain.ReviewAIFeedback{
		Score:            score,
		Communication:    communication,
		StructureSTAR:    structure,
		Confidence:       confidence,
		Strengths:        strengths,
		Weaknesses:       weaknesses,
		Suggestions:      suggestions,
		BetterAnswer:     betterAnswer,
		Insight:          insight,
		FollowUpQuestion: followUpQuestion,
	}, nil
}

func (s *Service) GenerateImprovementPlan(history []domain.ReviewSession, memory domain.CoachingMemory) (*domain.ImprovementPlan, error) {
	if s.useRemoteProvider() {
		if remote, err := s.remoteGenerateImprovementPlan(history, memory); err == nil {
			return remote, nil
		}
	}

	focusAreas := []string{"STAR structure consistency", "clearer confidence language", "stronger measurable outcomes"}
	if len(memory.FocusAreas) > 0 {
		focusAreas = append([]string{}, memory.FocusAreas...)
	}

	practicePlan := []string{
		"Record 1 voice reflection using STAR for a failed interview question",
		"Rewrite 2 past answers with quantified results",
		"Do 1 mock response where you answer in under 90 seconds with clear structure",
	}

	if len(history) > 0 {
		latest := history[0]
		if latest.Feedback.StructureSTAR < 60 {
			focusAreas[0] = "STAR structure and narrative flow"
		}
		if latest.Feedback.Confidence < 60 {
			focusAreas[1] = "confidence and concise delivery"
		}
	}

	return &domain.ImprovementPlan{
		FocusAreas:       focusAreas,
		PracticePlan:     practicePlan,
		WeeklyTarget:     "Complete at least 2 review sessions and 1 recovery simulation this week",
		NextSessionFocus: "Answer relevance and stronger action/result details",
	}, nil
}

// TranslateText translates short pieces of text into targetLanguage ("id" or "en").
// TranslateText removed during rollback; prefer client-side Google Translate widget.

func tokenize(input string) []string {
	r := strings.NewReplacer(
		",", " ",
		".", " ",
		";", " ",
		":", " ",
		"(", " ",
		")", " ",
		"/", " ",
		"\\n", " ",
	)
	cleaned := r.Replace(input)
	parts := strings.Fields(cleaned)

	filtered := make([]string, 0, len(parts))
	for _, part := range parts {
		if len(part) < 3 {
			continue
		}
		if isStopWord(part) {
			continue
		}
		filtered = append(filtered, part)
	}

	return filtered
}

func isStopWord(word string) bool {
	stopWords := map[string]struct{}{
		"the": {}, "and": {}, "for": {}, "with": {}, "that": {}, "this": {}, "from": {},
		"you": {}, "your": {}, "are": {}, "our": {}, "have": {}, "will": {}, "all": {},
		"can": {}, "has": {}, "not": {}, "but": {}, "job": {}, "role": {}, "team": {},
		"dan": {}, "yang": {}, "untuk": {}, "dengan": {}, "pada": {}, "atau": {}, "dari": {},
		"kami": {}, "anda": {}, "dalam": {}, "ini": {}, "itu": {}, "akan": {}, "sebagai": {},
	}
	_, found := stopWords[word]
	return found
}

func topKeywords(tokens []string, limit int) []string {
	freq := make(map[string]int)
	for _, token := range tokens {
		freq[token]++
	}

	type entry struct {
		word  string
		count int
	}

	list := make([]entry, 0, len(freq))
	for word, count := range freq {
		list = append(list, entry{word: word, count: count})
	}

	sort.Slice(list, func(i, j int) bool {
		if list[i].count == list[j].count {
			return list[i].word < list[j].word
		}
		return list[i].count > list[j].count
	})

	if len(list) > limit {
		list = list[:limit]
	}

	result := make([]string, 0, len(list))
	for _, item := range list {
		result = append(result, item.word)
	}

	return result
}

func detectSkills(input string) []string {
	catalog := []string{
		"golang", "go", "python", "java", "javascript", "typescript", "react", "next.js", "node.js",
		"postgresql", "redis", "docker", "kubernetes", "aws", "gcp", "azure", "gin", "gorm",
		"microservices", "rest", "grpc", "sql", "nosql", "graphql", "ci/cd",
	}

	found := make([]string, 0)
	for _, skill := range catalog {
		if strings.Contains(input, strings.ToLower(skill)) {
			found = append(found, skill)
		}
	}

	if len(found) == 0 {
		return []string{"general-software-engineering"}
	}

	return found
}

func detectThemes(input string) []string {
	themes := make([]string, 0)
	if strings.Contains(input, "backend") || strings.Contains(input, "api") {
		themes = append(themes, "backend-development")
	}
	if strings.Contains(input, "frontend") || strings.Contains(input, "ui") {
		themes = append(themes, "frontend-development")
	}
	if strings.Contains(input, "cloud") || strings.Contains(input, "deploy") || strings.Contains(input, "infrastructure") {
		themes = append(themes, "cloud-infrastructure")
	}
	if strings.Contains(input, "data") || strings.Contains(input, "analytics") {
		themes = append(themes, "data-and-analytics")
	}

	if len(themes) == 0 {
		themes = append(themes, "general-engineering")
	}

	return themes
}

func detectSeniority(input string) string {
	if strings.Contains(input, "principal") || strings.Contains(input, "staff") {
		return "staff"
	}
	if strings.Contains(input, "senior") || strings.Contains(input, "lead") {
		return "senior"
	}
	if strings.Contains(input, "mid") || strings.Contains(input, "intermediate") {
		return "mid"
	}
	if strings.Contains(input, "junior") || strings.Contains(input, "entry") || strings.Contains(input, "fresh graduate") {
		return "junior"
	}
	return "unspecified"
}

func pickOrDefault(values []string, idx int, fallback string) string {
	if idx >= 0 && idx < len(values) && strings.TrimSpace(values[idx]) != "" {
		return values[idx]
	}
	return fallback
}

func questionCountByMode(mode domain.InterviewMode) int {
	if domain.NormalizeInterviewMode(string(mode)) == domain.InterviewModeVoice {
		return 15
	}

	return 10
}

func difficultyBadge(interviewLanguage domain.InterviewLanguage, difficulty domain.InterviewDifficulty) string {
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	difficulty = domain.NormalizeInterviewDifficulty(string(difficulty))

	if interviewLanguage == domain.InterviewLanguageIndonesian {
		switch difficulty {
		case domain.InterviewDifficultyEasy:
			return "Mudah"
		case domain.InterviewDifficultyHard:
			return "Sulit"
		default:
			return "Sedang"
		}
	}

	switch difficulty {
	case domain.InterviewDifficultyEasy:
		return "Easy"
	case domain.InterviewDifficultyHard:
		return "Hard"
	default:
		return "Medium"
	}
}

func difficultyGuidance(interviewLanguage domain.InterviewLanguage, difficulty domain.InterviewDifficulty) string {
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	difficulty = domain.NormalizeInterviewDifficulty(string(difficulty))

	if interviewLanguage == domain.InterviewLanguageIndonesian {
		switch difficulty {
		case domain.InterviewDifficultyEasy:
			return "Jawaban boleh sederhana, fokus pada langkah inti."
		case domain.InterviewDifficultyHard:
			return "Bahas trade-off, edge case, serta dampak terukur."
		default:
			return "Sertakan alasan keputusan dan satu contoh konkret."
		}
	}

	switch difficulty {
	case domain.InterviewDifficultyEasy:
		return "Keep the answer practical and straightforward."
	case domain.InterviewDifficultyHard:
		return "Cover trade-offs, edge cases, and measurable impact."
	default:
		return "Include decision rationale and one concrete example."
	}
}

func fillTemplate(template string, placeholders []string) string {
	result := template
	if len(placeholders) == 0 {
		placeholders = []string{"the role requirements"}
	}

	index := 0
	for strings.Contains(result, "%s") {
		replacement := pickOrDefault(placeholders, index%len(placeholders), "the role requirements")
		result = strings.Replace(result, "%s", replacement, 1)
		index++
	}

	return result
}

func buildQuestionBatch(
	templates []string,
	placeholders []string,
	questionType string,
	count int,
	interviewLanguage domain.InterviewLanguage,
	interviewDifficulty domain.InterviewDifficulty,
) []domain.GeneratedQuestion {
	if count <= 0 || len(templates) == 0 {
		return []domain.GeneratedQuestion{}
	}

	questions := make([]domain.GeneratedQuestion, 0, count)
	guidance := difficultyGuidance(interviewLanguage, interviewDifficulty)
	badge := difficultyBadge(interviewLanguage, interviewDifficulty)

	for index := 0; index < count; index++ {
		template := templates[index%len(templates)]
		questionText := strings.TrimSpace(fillTemplate(template, placeholders))
		if strings.TrimSpace(guidance) != "" {
			questionText = strings.TrimSpace(questionText + " " + guidance)
		}

		questions = append(questions, domain.GeneratedQuestion{
			Type:     questionType,
			Question: fmt.Sprintf("[%s] %s", badge, questionText),
		})
	}

	return questions
}

func containsAny(input string, candidates []string) bool {
	for _, candidate := range candidates {
		if strings.Contains(input, candidate) {
			return true
		}
	}
	return false
}

func containsAnyToken(tokens []string, candidates []string) bool {
	set := make(map[string]struct{}, len(tokens))
	for _, token := range tokens {
		set[token] = struct{}{}
	}

	for _, candidate := range candidates {
		if _, exists := set[strings.ToLower(strings.TrimSpace(candidate))]; exists {
			return true
		}
	}

	return false
}

func clampScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func (s *Service) useRemoteProvider() bool {
	if s == nil {
		return false
	}
	if s.provider != "openai" {
		return false
	}
	return strings.TrimSpace(s.apiKey) != ""
}

type openAIChatRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIChatMessage `json:"messages"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

type openAIChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openAIChatResponse struct {
	Choices []struct {
		Message openAIChatMessage `json:"message"`
	} `json:"choices"`
}

func (s *Service) chatCompletion(systemPrompt, userPrompt string) (string, error) {
	return s.chatCompletionWithModel(systemPrompt, userPrompt, "")
}

func (s *Service) chatCompletionWithModel(systemPrompt, userPrompt, modelOverride string) (string, error) {
	model := strings.TrimSpace(modelOverride)
	if model == "" {
		model = s.model
	}

	requestPayload := openAIChatRequest{
		Model: model,
		Messages: []openAIChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
		Temperature: 0.2,
		MaxTokens:   700,
	}

	body, err := json.Marshal(requestPayload)
	if err != nil {
		return "", err
	}

	request, err := http.NewRequest(http.MethodPost, s.apiBaseURL+"/chat/completions", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Authorization", "Bearer "+s.apiKey)

	response, err := s.httpClient.Do(request)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	responseBody, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return "", fmt.Errorf("ai provider error: %s", string(responseBody))
	}

	var parsed openAIChatResponse
	if err := json.Unmarshal(responseBody, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("ai provider returned no choices")
	}

	return parsed.Choices[0].Message.Content, nil
}

func (s *Service) remoteParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
	return s.remoteParseJobDescriptionWithModel(jobDescription, "")
}

func (s *Service) remoteParseJobDescriptionWithModel(jobDescription, modelOverride string) (*domain.JobInsights, error) {
	systemPrompt := "You are an interview analysis assistant. Return only strict JSON."
	userPrompt := "Parse the following job description into JSON with keys skills (array of strings), keywords (array of strings), themes (array of strings), seniority (string). Keep each list concise.\n\nJob Description:\n" + jobDescription

	raw, err := s.chatCompletionWithModel(systemPrompt, userPrompt, modelOverride)
	if err != nil {
		return nil, err
	}

	var result domain.JobInsights
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	if len(result.Skills) == 0 {
		result.Skills = []string{"general-software-engineering"}
	}
	if len(result.Keywords) == 0 {
		result.Keywords = []string{"general"}
	}
	if len(result.Themes) == 0 {
		result.Themes = []string{"general-engineering"}
	}
	if strings.TrimSpace(result.Seniority) == "" {
		result.Seniority = "unspecified"
	}

	return &result, nil
}

func (s *Service) remoteGenerateQuestions(
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
) ([]domain.GeneratedQuestion, error) {
	return s.remoteGenerateQuestionsWithModel(
		resumeText,
		jobDescription,
		interviewLanguage,
		interviewMode,
		interviewDifficulty,
		"",
	)
}

func (s *Service) remoteGenerateQuestionsWithModel(
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
	modelOverride string,
) ([]domain.GeneratedQuestion, error) {
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	interviewMode = domain.NormalizeInterviewMode(string(interviewMode))
	interviewDifficulty = domain.NormalizeInterviewDifficulty(string(interviewDifficulty))

	totalQuestions := questionCountByMode(interviewMode)
	behavioralCount := totalQuestions / 2
	technicalCount := totalQuestions - behavioralCount
	modeLabel := "text interview"
	if interviewMode == domain.InterviewModeVoice {
		modeLabel = "voice interview"
	}
	difficultyLabel := strings.ToUpper(string(interviewDifficulty))
	if difficultyLabel == "" {
		difficultyLabel = "MEDIUM"
	}

	targetLanguage := "English"
	if interviewLanguage == domain.InterviewLanguageIndonesian {
		targetLanguage = "Bahasa Indonesia"
	}

	systemPrompt := "You are an interview coach. Return only strict JSON."
	userPrompt := fmt.Sprintf(
		"Generate exactly %d interview questions in JSON array with each item containing keys type and question. Include exactly %d behavioral and %d technical questions. Context is %s. Difficulty level is %s. Every question must be clearly grounded in the provided job description responsibilities, stack, and seniority requirements. Questions must be written in %s. Return only JSON array with no markdown.",
		totalQuestions,
		behavioralCount,
		technicalCount,
		modeLabel,
		difficultyLabel,
		targetLanguage,
	) + "\n\nCV:\n" + resumeText + "\n\nJob Description:\n" + jobDescription

	raw, err := s.chatCompletionWithModel(systemPrompt, userPrompt, modelOverride)
	if err != nil {
		return nil, err
	}

	var result []domain.GeneratedQuestion
	if err := extractJSONArray(raw, &result); err != nil {
		return nil, err
	}
	if len(result) != totalQuestions {
		return nil, fmt.Errorf("expected %d generated questions, got %d", totalQuestions, len(result))
	}

	return result, nil
}

func (s *Service) remoteAnalyzeAnswer(question, answer string, interviewLanguage domain.InterviewLanguage) (*domain.AnswerAnalysis, error) {
	return s.remoteAnalyzeAnswerWithModel(question, answer, interviewLanguage, "")
}

func (s *Service) remoteAnalyzeAnswerWithModel(question, answer string, interviewLanguage domain.InterviewLanguage, modelOverride string) (*domain.AnswerAnalysis, error) {
	interviewLanguage = domain.NormalizeInterviewLanguage(string(interviewLanguage))
	targetLanguage := "English"
	if interviewLanguage == domain.InterviewLanguageIndonesian {
		targetLanguage = "Bahasa Indonesia"
	}

	systemPrompt := "You are an interview evaluator. Return only strict JSON."
	userPrompt := fmt.Sprintf(
		"Evaluate the candidate answer and return JSON with keys score (0-100 int), strengths (array), weaknesses (array), improvements (array), star_feedback (string). Ensure feedback language is %s and keep it concise, concrete, and tied to relevance with the interview question. Return only JSON.",
		targetLanguage,
	) + "\n\nQuestion:\n" + question + "\n\nAnswer:\n" + answer

	raw, err := s.chatCompletionWithModel(systemPrompt, userPrompt, modelOverride)
	if err != nil {
		return nil, err
	}

	var result domain.AnswerAnalysis
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	if result.Score < 0 {
		result.Score = 0
	}
	if result.Score > 100 {
		result.Score = 100
	}
	if len(result.Strengths) == 0 {
		result.Strengths = []string{"response provided"}
	}
	if len(result.Weaknesses) == 0 {
		result.Weaknesses = []string{"minor clarity improvements possible"}
	}
	if len(result.Improvements) == 0 {
		result.Improvements = []string{"add clearer structure and concise takeaway"}
	}
	if strings.TrimSpace(result.STARFeedback) == "" {
		result.STARFeedback = "Use STAR framing: Situation, Task, Action, Result with measurable impact."
	}

	return &result, nil
}

func (s *Service) remoteAnalyzeResume(resumeText string) (*domain.ResumeAIAnalysis, error) {
	return s.remoteAnalyzeResumeWithModel(resumeText, "")
}

func (s *Service) remoteAnalyzeResumeWithModel(resumeText, modelOverride string) (*domain.ResumeAIAnalysis, error) {
	systemPrompt := "You are an interview coaching assistant. Return only strict JSON."
	userPrompt := "Analyze the following CV and return JSON with keys summary (string), response (string), highlights (array of concise strings), recommendations (array of actionable strings). Keep content concise and practical.\n\nCV:\n" + resumeText

	raw, err := s.chatCompletionWithModel(systemPrompt, userPrompt, modelOverride)
	if err != nil {
		return nil, err
	}

	var result domain.ResumeAIAnalysis
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = "CV analysis is available but summary was not generated."
	}
	if strings.TrimSpace(result.Response) == "" {
		result.Response = "Please refine impact metrics and role-specific examples for stronger interview performance."
	}
	if len(result.Highlights) == 0 {
		result.Highlights = []string{"general engineering profile"}
	}
	if len(result.Recommendations) == 0 {
		result.Recommendations = []string{"Add measurable outcomes and tailor CV to target role."}
	}

	return &result, nil
}

func (s *Service) remoteTranslateResumeAnalysis(analysis *domain.ResumeAIAnalysis, targetLanguage string) (*domain.ResumeAIAnalysis, error) {
	targetLanguage = normalizeResumeAnalysisLanguage(targetLanguage)
	payload, err := json.Marshal(analysis)
	if err != nil {
		return nil, err
	}

	languageLabel := "English"
	if targetLanguage == "id" {
		languageLabel = "Bahasa Indonesia"
	}

	systemPrompt := "You are a translation assistant for interview coaching outputs. Return only strict JSON."
	userPrompt := fmt.Sprintf(
		"Translate this resume analysis JSON into %s while preserving meaning and structure. Return JSON with keys summary (string), response (string), highlights (array of strings), recommendations (array of strings).\n\nResume analysis JSON:\n%s",
		languageLabel,
		string(payload),
	)

	raw, err := s.chatCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result domain.ResumeAIAnalysis
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	if strings.TrimSpace(result.Summary) == "" {
		result.Summary = analysis.Summary
	}
	if strings.TrimSpace(result.Response) == "" {
		result.Response = analysis.Response
	}
	if len(result.Highlights) == 0 {
		result.Highlights = append([]string{}, analysis.Highlights...)
	}
	if len(result.Recommendations) == 0 {
		result.Recommendations = append([]string{}, analysis.Recommendations...)
	}

	return ensureTranslationQuality(analysis, &result, targetLanguage), nil
}

func parseResumeAnalysisTranslationRequest(input string) (bool, string, *domain.ResumeAIAnalysis) {
	trimmed := strings.TrimSpace(input)
	lower := strings.ToLower(trimmed)
	if !strings.Contains(lower, "resume analysis json") {
		return false, "", nil
	}

	targetLanguage := "en"
	if strings.Contains(lower, "bahasa indonesia") {
		targetLanguage = "id"
	}

	markerIndex := strings.Index(lower, "resume analysis json")
	if markerIndex < 0 {
		return true, targetLanguage, nil
	}

	payloadSection := strings.TrimSpace(trimmed[markerIndex+len("resume analysis json"):])
	payloadSection = strings.TrimLeft(payloadSection, ": \n\t")

	jsonStart := strings.Index(payloadSection, "{")
	jsonEnd := strings.LastIndex(payloadSection, "}")
	if jsonStart < 0 || jsonEnd < jsonStart {
		return true, targetLanguage, nil
	}

	jsonPayload := payloadSection[jsonStart : jsonEnd+1]

	var analysis domain.ResumeAIAnalysis
	if err := json.Unmarshal([]byte(jsonPayload), &analysis); err != nil {
		return true, targetLanguage, nil
	}

	return true, targetLanguage, &analysis
}

func translateResumeAnalysisLocally(analysis *domain.ResumeAIAnalysis, targetLanguage string) *domain.ResumeAIAnalysis {
	targetLanguage = normalizeResumeAnalysisLanguage(targetLanguage)
	if analysis == nil {
		return &domain.ResumeAIAnalysis{}
	}

	if targetLanguage == "id" {
		highlights := make([]string, 0, len(analysis.Highlights))
		for _, item := range analysis.Highlights {
			highlights = append(highlights, translateTextToIndonesian(item))
		}

		recommendations := make([]string, 0, len(analysis.Recommendations))
		for _, item := range analysis.Recommendations {
			recommendations = append(recommendations, translateTextToIndonesian(item))
		}

		return &domain.ResumeAIAnalysis{
			Summary:         translateTextToIndonesian(analysis.Summary),
			Response:        translateTextToIndonesian(analysis.Response),
			Highlights:      highlights,
			Recommendations: recommendations,
		}
	}

	highlights := make([]string, 0, len(analysis.Highlights))
	for _, item := range analysis.Highlights {
		highlights = append(highlights, translateTextToEnglish(item))
	}

	recommendations := make([]string, 0, len(analysis.Recommendations))
	for _, item := range analysis.Recommendations {
		recommendations = append(recommendations, translateTextToEnglish(item))
	}

	return &domain.ResumeAIAnalysis{
		Summary:         translateTextToEnglish(analysis.Summary),
		Response:        translateTextToEnglish(analysis.Response),
		Highlights:      highlights,
		Recommendations: recommendations,
	}
}

func normalizeResumeAnalysisLanguage(language string) string {
	trimmed := strings.TrimSpace(strings.ToLower(language))
	if trimmed == "id" {
		return "id"
	}
	return "en"
}

func translateTextToIndonesian(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	replacements := []struct {
		old string
		new string
	}{
		{"The CV indicates a profile focused on ", "CV ini menunjukkan profil yang berfokus pada "},
		{" with practical engineering exposure.", " dengan pengalaman engineering praktis."},
		{"Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence.", "Secara keseluruhan, profil ini relevan untuk persiapan interview. Prioritaskan narasi dampak yang lebih jelas dan positioning yang lebih spesifik sesuai role agar meningkatkan kepercayaan recruiter dan interviewer."},
		{"Emphasize leadership outcomes and scope ownership.", "Tekankan hasil kepemimpinan dan kepemilikan ruang lingkup pekerjaan."},
		{"Add 2-3 quantified achievements for key projects.", "Tambahkan 2-3 pencapaian terukur untuk proyek utama."},
		{"Highlight measurable impact using metrics (%, time, revenue, scale).", "Tonjolkan dampak terukur dengan metrik (%, waktu, pendapatan, skala)."},
		{"Tailor headline and recent experience toward the target role.", "Sesuaikan headline dan pengalaman terbaru ke role yang dituju."},
		{"analysis", "analisis"},
		{"impact", "dampak"},
		{"leadership", "kepemimpinan"},
		{"ownership", "kepemilikan"},
		{"response", "respons"},
	}

	translated := trimmed
	for _, item := range replacements {
		translated = strings.ReplaceAll(translated, item.old, item.new)
	}

	if translated == trimmed {
		return trimmed
	}

	return translated
}

func translateTextToEnglish(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}

	// No special prefix expected from the translator; work with the raw trimmed text

	replacements := []struct {
		old string
		new string
	}{
		{"CV ini menunjukkan profil yang berfokus pada ", "The CV indicates a profile focused on "},
		{" dengan pengalaman engineering praktis.", " with practical engineering exposure."},
		{"Secara keseluruhan, profil ini relevan untuk persiapan interview. Prioritaskan narasi dampak yang lebih jelas dan positioning yang lebih spesifik sesuai role agar meningkatkan kepercayaan recruiter dan interviewer.", "Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence."},
		{"Tekankan hasil kepemimpinan dan kepemilikan ruang lingkup pekerjaan.", "Emphasize leadership outcomes and scope ownership."},
		{"Tambahkan 2-3 pencapaian terukur untuk proyek utama.", "Add 2-3 quantified achievements for key projects."},
		{"Tonjolkan dampak terukur dengan metrik (%, waktu, pendapatan, skala).", "Highlight measurable impact using metrics (%, time, revenue, scale)."},
		{"Sesuaikan headline dan pengalaman terbaru ke role yang dituju.", "Tailor headline and recent experience toward the target role."},
		{"analisis", "analysis"},
		{"dampak", "impact"},
		{"kepemimpinan", "leadership"},
		{"kepemilikan", "ownership"},
		{"respons", "response"},
	}

	translated := trimmed
	for _, item := range replacements {
		translated = strings.ReplaceAll(translated, item.old, item.new)
	}

	if translated == trimmed {
		lower := strings.ToLower(trimmed)
		if strings.Contains(lower, "secara keseluruhan") || strings.Contains(lower, "cv ini") || strings.Contains(lower, "dampak") || strings.Contains(lower, "pencapaian") {
			return "English translation: " + trimmed
		}
	}

	return translated
}

func ensureTranslationQuality(source, translated *domain.ResumeAIAnalysis, targetLanguage string) *domain.ResumeAIAnalysis {
	targetLanguage = normalizeResumeAnalysisLanguage(targetLanguage)
	if translated == nil {
		return translateResumeAnalysisLocally(source, targetLanguage)
	}

	if targetLanguage == "id" && !looksLikeIndonesianResumeAnalysis(translated) {
		return translateResumeAnalysisLocally(source, targetLanguage)
	}

	if targetLanguage == "en" && !looksLikeEnglishResumeAnalysis(translated) {
		return translateResumeAnalysisLocally(source, targetLanguage)
	}

	return translated
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
		// removed to avoid false-positive detection of an artificial marker
	}

	for _, marker := range indicators {
		if strings.Contains(blob, marker) {
			return true
		}
	}

	return false
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

func (s *Service) remoteAnalyzeReview(input domain.ReviewAIInput) (*domain.ReviewAIFeedback, error) {
	reviewLanguage := resolveReviewLanguage(input.InterviewLanguage, input.Memory.PreferredLanguage)
	input.InterviewLanguage = reviewLanguage

	systemPrompt := "You are a senior AI Career Coach in Review Mode. You must be specific, actionable, and non-generic. Always produce strict JSON only. If information is missing, ask one follow-up question."
	languageInstruction := "Return every text field in English."
	if reviewLanguage == domain.InterviewLanguageIndonesian {
		languageInstruction = "Kembalikan semua field teks dalam Bahasa Indonesia."
	}
	userPrompt := fmt.Sprintf(
		"Analyze this interview reflection and return JSON with keys: score (0-100 int), communication (0-100 int), structure_star (0-100 int), confidence (0-100 int), strengths (array), weaknesses (array), suggestions (array), better_answer (string), insight (string), follow_up_question (string), recovery_simulation (string optional). Focus on why the candidate likely failed and what to improve next. %s Session type: %s. Input mode: %s. Interview language: %s. Target role: %s. Target company: %s. Memory weaknesses: %v. Memory focus areas: %v. Interview prompt/context: %s. Candidate reflection: %s",
		languageInstruction,
		input.SessionType,
		input.InputMode,
		string(reviewLanguage),
		input.TargetRole,
		input.TargetCompany,
		input.Memory.Weaknesses,
		input.Memory.FocusAreas,
		input.InterviewPrompt,
		input.UserInput,
	)

	raw, err := s.chatCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result domain.ReviewAIFeedback
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}

	result.Score = clampScore(result.Score)
	result.Communication = clampScore(result.Communication)
	result.StructureSTAR = clampScore(result.StructureSTAR)
	result.Confidence = clampScore(result.Confidence)
	if len(result.Strengths) == 0 {
		result.Strengths = []string{"reflection provided"}
	}
	if len(result.Weaknesses) == 0 {
		result.Weaknesses = []string{"needs clearer STAR flow"}
	}
	if len(result.Suggestions) == 0 {
		result.Suggestions = []string{"add one measurable result and explicit action ownership"}
	}
	if strings.TrimSpace(result.FollowUpQuestion) == "" {
		if reviewLanguage == domain.InterviewLanguageIndonesian {
			result.FollowUpQuestion = "Pertanyaan interviewer yang mana yang ingin Anda jawab ulang sekarang?"
		} else {
			result.FollowUpQuestion = "What exact interviewer question do you want to re-answer now?"
		}
	}

	return &result, nil
}

func resolveReviewLanguage(selected domain.InterviewLanguage, preferred string) domain.InterviewLanguage {
	if strings.TrimSpace(string(selected)) != "" {
		return domain.NormalizeInterviewLanguage(string(selected))
	}
	if strings.TrimSpace(preferred) != "" {
		return domain.NormalizeInterviewLanguage(preferred)
	}
	return domain.InterviewLanguageEnglish
}

func (s *Service) remoteGenerateImprovementPlan(history []domain.ReviewSession, memory domain.CoachingMemory) (*domain.ImprovementPlan, error) {
	systemPrompt := "You are a senior AI Career Coach. Return only strict JSON."
	historyBytes, _ := json.Marshal(history)
	memoryBytes, _ := json.Marshal(memory)
	userPrompt := fmt.Sprintf(
		"Build a personalized improvement plan from this review history and coaching memory. Return JSON with keys: focus_areas (2-3 items), practice_plan (2-4 items), weekly_target (string), next_session_focus (string). Be specific and actionable. History: %s. Memory: %s",
		string(historyBytes),
		string(memoryBytes),
	)

	raw, err := s.chatCompletion(systemPrompt, userPrompt)
	if err != nil {
		return nil, err
	}

	var result domain.ImprovementPlan
	if err := extractJSONObject(raw, &result); err != nil {
		return nil, err
	}
	if len(result.FocusAreas) == 0 {
		result.FocusAreas = []string{"STAR structure", "confidence", "clarity"}
	}
	if len(result.PracticePlan) == 0 {
		result.PracticePlan = []string{"Run one recovery simulation on last failed interview question"}
	}
	if strings.TrimSpace(result.WeeklyTarget) == "" {
		result.WeeklyTarget = "Complete 2 focused review sessions this week"
	}
	if strings.TrimSpace(result.NextSessionFocus) == "" {
		result.NextSessionFocus = "answer relevance and measurable impact"
	}

	return &result, nil
}

func extractJSONObject(raw string, target any) error {
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start < 0 || end <= start {
		return fmt.Errorf("no json object found")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), target)
}

func extractJSONArray(raw string, target any) error {
	start := strings.Index(raw, "[")
	end := strings.LastIndex(raw, "]")
	if start < 0 || end <= start {
		return fmt.Errorf("no json array found")
	}
	return json.Unmarshal([]byte(raw[start:end+1]), target)
}

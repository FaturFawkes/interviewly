package usecase

import (
	"strings"
	"testing"

	"github.com/interview_app/backend/internal/domain"
	"github.com/interview_app/backend/internal/repository"
)

type fakeResumeAIService struct{}

func (f *fakeResumeAIService) ParseJobDescription(jobDescription string) (*domain.JobInsights, error) {
	return &domain.JobInsights{}, nil
}

func (f *fakeResumeAIService) GenerateQuestions(
	resumeText,
	jobDescription string,
	interviewLanguage domain.InterviewLanguage,
	interviewMode domain.InterviewMode,
	interviewDifficulty domain.InterviewDifficulty,
) ([]domain.GeneratedQuestion, error) {
	return []domain.GeneratedQuestion{}, nil
}

func (f *fakeResumeAIService) AnalyzeAnswer(question, answer string, interviewLanguage domain.InterviewLanguage) (*domain.AnswerAnalysis, error) {
	return &domain.AnswerAnalysis{}, nil
}

func (f *fakeResumeAIService) AnalyzeResume(resumeText string) (*domain.ResumeAIAnalysis, error) {
	if strings.Contains(resumeText, "Bahasa Indonesia") {
		return &domain.ResumeAIAnalysis{
			Summary:         "Ringkasan CV dalam Bahasa Indonesia",
			Response:        "Respons analisis dalam Bahasa Indonesia",
			Highlights:      []string{"pengalaman backend", "komunikasi"},
			Recommendations: []string{"tambahkan metrik dampak"},
		}, nil
	}

	return &domain.ResumeAIAnalysis{
		Summary:         "English summary",
		Response:        "English response",
		Highlights:      []string{"backend experience"},
		Recommendations: []string{"add impact metrics"},
	}, nil
}

func (f *fakeResumeAIService) AnalyzeReview(input domain.ReviewAIInput) (*domain.ReviewAIFeedback, error) {
	return &domain.ReviewAIFeedback{}, nil
}

func (f *fakeResumeAIService) GenerateImprovementPlan(history []domain.ReviewSession, memory domain.CoachingMemory) (*domain.ImprovementPlan, error) {
	return &domain.ImprovementPlan{}, nil
}

func TestGetLatestResumeAnalysis_TranslatesAndCachesRequestedLanguage(t *testing.T) {
	repo := repository.NewInterviewRepository(nil)
	userID := "user-123"
	contentHash := "hash-abc"

	_, err := repo.SaveResumeAnalysis(userID, "", contentHash, "default|lang:en", &domain.ResumeAIAnalysis{
		Summary:         "English summary",
		Response:        "English response",
		Highlights:      []string{"backend experience"},
		Recommendations: []string{"add impact metrics"},
	})
	if err != nil {
		t.Fatalf("failed seeding english analysis: %v", err)
	}

	uc := NewInterviewUseCase(&fakeResumeAIService{}, repo, nil, nil)

	analysis, err := uc.GetLatestResumeAnalysis(userID, "id")
	if err != nil {
		t.Fatalf("GetLatestResumeAnalysis returned error: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected analysis, got nil")
	}
	if !strings.Contains(strings.ToLower(analysis.Summary), "bahasa indonesia") {
		t.Fatalf("expected Indonesian summary, got: %q", analysis.Summary)
	}

	cachedID, err := repo.GetLatestResumeAnalysisByLanguage(userID, "id")
	if err != nil {
		t.Fatalf("failed loading cached Indonesian analysis: %v", err)
	}
	if cachedID == nil {
		t.Fatal("expected cached Indonesian analysis, got nil")
	}
	if !strings.Contains(strings.ToLower(cachedID.Summary), "bahasa indonesia") {
		t.Fatalf("expected cached Indonesian summary, got: %q", cachedID.Summary)
	}
}

func TestGetLatestResumeAnalysis_FixesEnglishContentInIndonesianCache(t *testing.T) {
	repo := repository.NewInterviewRepository(nil)
	userID := "user-456"
	contentHash := "hash-id-english"

	_, err := repo.SaveResumeAnalysis(userID, "", contentHash, "default|lang:id", &domain.ResumeAIAnalysis{
		Summary:         "The CV indicates a profile focused on backend and impact.",
		Response:        "Overall, the profile is relevant for interview preparation.",
		Highlights:      []string{"analysis", "impact"},
		Recommendations: []string{"Add 2-3 quantified achievements for key projects."},
	})
	if err != nil {
		t.Fatalf("failed seeding invalid Indonesian cache: %v", err)
	}

	uc := NewInterviewUseCase(&fakeResumeAIService{}, repo, nil, nil)

	analysis, err := uc.GetLatestResumeAnalysis(userID, "id")
	if err != nil {
		t.Fatalf("GetLatestResumeAnalysis returned error: %v", err)
	}
	if analysis == nil {
		t.Fatal("expected analysis, got nil")
	}

	if strings.Contains(strings.ToLower(analysis.Summary), "the cv indicates") {
		t.Fatalf("expected Indonesian summary after correction, got: %q", analysis.Summary)
	}
	if !strings.Contains(strings.ToLower(analysis.Summary), "bahasa indonesia") {
		t.Fatalf("expected corrected Indonesian summary, got: %q", analysis.Summary)
	}
}

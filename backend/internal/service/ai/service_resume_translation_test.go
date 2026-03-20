package ai

import (
	"strings"
	"testing"

	"github.com/interview_app/backend/internal/domain"
)

func TestAnalyzeResume_TranslationRequestToIndonesian_LocalFallback(t *testing.T) {
	svc := &Service{provider: "local"}

	input := `Translate this resume analysis JSON into Bahasa Indonesia. Keep the same meaning, keep it concise, and return all output fields in Bahasa Indonesia.

Resume analysis JSON:
{"summary":"The CV indicates a profile focused on analysis, bahasa, and impact with practical engineering exposure.","response":"Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence.","highlights":["analysis","bahasa","impact","indonesia","json"],"recommendations":["Emphasize leadership outcomes and scope ownership.","Add 2-3 quantified achievements for key projects.","Highlight measurable impact using metrics (%, time, revenue, scale).","Tailor headline and recent experience toward the target role."]}`

	result, err := svc.AnalyzeResume(input)
	if err != nil {
		t.Fatalf("AnalyzeResume returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if strings.Contains(strings.ToLower(result.Summary), "the cv indicates") {
		t.Fatalf("expected Indonesian summary, got: %q", result.Summary)
	}
	if !strings.Contains(strings.ToLower(result.Summary), "cv ini") {
		t.Fatalf("expected translated Indonesian summary, got: %q", result.Summary)
	}
	if strings.Contains(strings.ToLower(result.Response), "overall, the profile is relevant") {
		t.Fatalf("expected Indonesian response, got: %q", result.Response)
	}
	if len(result.Recommendations) == 0 {
		t.Fatal("expected recommendations")
	}
	if strings.Contains(strings.ToLower(result.Recommendations[0]), "emphasize leadership") {
		t.Fatalf("expected Indonesian recommendation, got: %q", result.Recommendations[0])
	}
}

func TestEnsureTranslationQuality_FallsBackWhenRemoteStillEnglish(t *testing.T) {
	source := englishResumeAnalysisFixture()
	remoteEnglish := englishResumeAnalysisFixture()

	result := ensureTranslationQuality(source, remoteEnglish, "id")
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if strings.Contains(strings.ToLower(result.Summary), "the cv indicates") {
		t.Fatalf("expected Indonesian fallback summary, got: %q", result.Summary)
	}
	if !strings.Contains(strings.ToLower(result.Summary), "cv ini") {
		t.Fatalf("expected Indonesian fallback summary marker, got: %q", result.Summary)
	}
}

func TestAnalyzeResume_TranslationRequestToEnglish_LocalFallback(t *testing.T) {
	svc := &Service{provider: "local"}

	input := `Translate this resume analysis JSON into English. Keep the same meaning, keep it concise, and return all output fields in English.

Resume analysis JSON:
{"summary":"CV ini menunjukkan profil yang berfokus pada analisis dan dampak dengan pengalaman engineering praktis.","response":"Secara keseluruhan, profil ini relevan untuk persiapan interview. Prioritaskan narasi dampak yang lebih jelas.","highlights":["analisis","dampak"],"recommendations":["Tambahkan 2-3 pencapaian terukur untuk proyek utama."]}`

	result, err := svc.AnalyzeResume(input)
	if err != nil {
		t.Fatalf("AnalyzeResume returned error: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}

	if strings.Contains(strings.ToLower(result.Summary), "cv ini") {
		t.Fatalf("expected English summary, got: %q", result.Summary)
	}
	if !strings.Contains(strings.ToLower(result.Summary), "the cv indicates") {
		t.Fatalf("expected translated English summary, got: %q", result.Summary)
	}
	if len(result.Highlights) == 0 || strings.Contains(strings.ToLower(result.Highlights[0]), "analisis") {
		t.Fatalf("expected English highlight, got: %v", result.Highlights)
	}
}

func englishResumeAnalysisFixture() *domain.ResumeAIAnalysis {
	return &domain.ResumeAIAnalysis{
		Summary:         "The CV indicates a profile focused on analysis, bahasa, and impact with practical engineering exposure.",
		Response:        "Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence.",
		Highlights:      []string{"analysis", "bahasa", "impact", "indonesia", "json"},
		Recommendations: []string{"Emphasize leadership outcomes and scope ownership.", "Add 2-3 quantified achievements for key projects.", "Highlight measurable impact using metrics (%, time, revenue, scale).", "Tailor headline and recent experience toward the target role."},
	}
}

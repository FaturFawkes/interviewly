"use client";

import { ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Tag } from "@/components/ui/Tag";
import { api } from "@/lib/api/endpoints";
import type { ResumeAIAnalysis } from "@/lib/api/types";
import { extractTextFromResumeFile, getAllowedResumeExtensionsLabel } from "@/lib/resume-parser";
import { pickLocaleText } from "@/lib/i18n";

const RESUME_ACCEPT = ".pdf,.docx,.txt,.md,.rtf,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,text/markdown,application/rtf,text/rtf";

function localizeUploadError(locale: "id" | "en", message: string): string {
  const normalized = message.toLowerCase();

  if (normalized.includes("resume storage is not configured")) {
    return pickLocaleText(locale, "Penyimpanan file CV belum terkonfigurasi.", "CV file storage is not configured.");
  }
  if (normalized.includes("resume content is required")) {
    return pickLocaleText(locale, "Konten CV wajib diisi.", "Resume content is required.");
  }
  if (normalized.includes("resume analysis not found")) {
    return pickLocaleText(locale, "Analisis CV belum tersedia.", "Resume analysis is not available yet.");
  }
  if (normalized.includes("resume not found")) {
    return pickLocaleText(locale, "CV belum tersedia.", "Resume is not available yet.");
  }

  return message;
}

function localizeAnalysisText(locale: "id" | "en", text: string): string {
  const trimmed = text.trim();
  if (!trimmed) {
    return text;
  }

  const working = trimmed;

  const replacements: Array<[string, string]> = locale === "id"
    ? [
      ["The CV indicates a profile focused on ", "CV ini menunjukkan profil yang berfokus pada "],
      [" with practical engineering exposure.", " dengan pengalaman engineering praktis."],
      [
        "Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence.",
        "Secara keseluruhan, profil ini relevan untuk persiapan interview. Prioritaskan narasi dampak yang lebih jelas dan positioning yang lebih spesifik sesuai role agar meningkatkan kepercayaan recruiter dan interviewer.",
      ],
      ["Emphasize leadership outcomes and scope ownership.", "Tekankan hasil kepemimpinan dan kepemilikan ruang lingkup pekerjaan."],
      ["Add 2-3 quantified achievements for key projects.", "Tambahkan 2-3 pencapaian terukur untuk proyek utama."],
      ["Highlight measurable impact using metrics (%, time, revenue, scale).", "Tonjolkan dampak terukur dengan metrik (%, waktu, pendapatan, skala)."],
      ["Tailor headline and recent experience toward the target role.", "Sesuaikan headline dan pengalaman terbaru ke role yang dituju."],
      ["analysis", "analisis"],
      ["impact", "dampak"],
      ["leadership", "kepemimpinan"],
      ["ownership", "kepemilikan"],
    ]
    : [
      ["CV ini menunjukkan profil yang berfokus pada ", "The CV indicates a profile focused on "],
      [" dengan pengalaman engineering praktis.", " with practical engineering exposure."],
      [
        "Secara keseluruhan, profil ini relevan untuk persiapan interview. Prioritaskan narasi dampak yang lebih jelas dan positioning yang lebih spesifik sesuai role agar meningkatkan kepercayaan recruiter dan interviewer.",
        "Overall, the profile is relevant for interview preparation. Prioritize clearer impact storytelling and role-specific positioning to improve recruiter and interviewer confidence.",
      ],
      ["Tekankan hasil kepemimpinan dan kepemilikan ruang lingkup pekerjaan.", "Emphasize leadership outcomes and scope ownership."],
      ["Tambahkan 2-3 pencapaian terukur untuk proyek utama.", "Add 2-3 quantified achievements for key projects."],
      ["Tonjolkan dampak terukur dengan metrik (%, waktu, pendapatan, skala).", "Highlight measurable impact using metrics (%, time, revenue, scale)."],
      ["Sesuaikan headline dan pengalaman terbaru ke role yang dituju.", "Tailor headline and recent experience toward the target role."],
      ["analisis", "analysis"],
      ["dampak", "impact"],
      ["kepemimpinan", "leadership"],
      ["kepemilikan", "ownership"],
      ["respons", "response"],
    ];

  let localized = working;
  for (const [from, to] of replacements) {
    localized = localized.replaceAll(from, to);
  }

  if (localized === trimmed) {
    return trimmed;
  }

  return localized;
}

function normalizeAnalysisForLocale(locale: "id" | "en", analysis: ResumeAIAnalysis): ResumeAIAnalysis {
  if (locale !== "id") {
    return analysis;
  }

  return {
    ...analysis,
    summary: localizeAnalysisText(locale, analysis.summary ?? ""),
    response: localizeAnalysisText(locale, analysis.response ?? ""),
    highlights: Array.isArray(analysis.highlights)
      ? analysis.highlights.map((item) => localizeAnalysisText(locale, item))
      : [],
    recommendations: Array.isArray(analysis.recommendations)
      ? analysis.recommendations.map((item) => localizeAnalysisText(locale, item))
      : [],
  };
}

export default function UploadPage() {
  const { locale } = useLanguage();
  const [resumeText, setResumeText] = useState("");
  const [selectedFile, setSelectedFile] = useState<File | null>(null);
  const [selectedFileName, setSelectedFileName] = useState<string | null>(null);
  const [latestResumePreview, setLatestResumePreview] = useState<string>("");
  const [analysisSummary, setAnalysisSummary] = useState<string>("");
  const [analysisResponse, setAnalysisResponse] = useState<string>("");
  const [analysisHighlights, setAnalysisHighlights] = useState<string[]>([]);
  const [analysisRecommendations, setAnalysisRecommendations] = useState<string[]>([]);
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingLatest, setLoadingLatest] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const effectiveResume = resumeText.trim();

  const cvStats = useMemo(() => {
    const cleaned = effectiveResume;
    if (!cleaned) {
      return { words: 0, chars: 0, lines: 0 };
    }

    return {
      words: cleaned.split(/\s+/).length,
      chars: cleaned.length,
      lines: cleaned.split(/\n+/).length,
    };
  }, [effectiveResume]);

  const loadLatestResumeAndAnalysis = useCallback(async () => {
    setLoadingLatest(true);
    setError(null);

    try {
      const latestResume = await api.getLatestResume();
      const normalizedResume = latestResume.content.trim();
      setResumeText(normalizedResume);
      setLatestResumePreview(normalizedResume.slice(0, 2200));

      const analysis = normalizeAnalysisForLocale(locale, await api.getLatestResumeAnalysis(locale));
      setAnalysisSummary(analysis.summary ?? "");
      setAnalysisResponse(analysis.response ?? "");
      setAnalysisHighlights(Array.isArray(analysis.highlights) ? analysis.highlights : []);
      setAnalysisRecommendations(Array.isArray(analysis.recommendations) ? analysis.recommendations : []);
      setSaved(true);
    } catch (err) {
      const rawMessage = err instanceof Error ? err.message : pickLocaleText(locale, "Gagal memuat CV terbaru.", "Failed to load latest CV.");
      const message = localizeUploadError(locale, rawMessage);
      const lowerMessage = message.toLowerCase();

      if (lowerMessage.includes("resume analysis not found")) {
        setAnalysisSummary("");
        setAnalysisResponse("");
        setAnalysisHighlights([]);
        setAnalysisRecommendations([]);
      } else if (lowerMessage.includes("resume not found") || lowerMessage.includes("not found")) {
        setSaved(false);
        setLatestResumePreview("");
        setAnalysisSummary("");
        setAnalysisResponse("");
        setAnalysisHighlights([]);
        setAnalysisRecommendations([]);
      } else {
        setError(message);
      }
    } finally {
      setLoadingLatest(false);
    }
  }, [locale]);

  useEffect(() => {
    void loadLatestResumeAndAnalysis();
  }, [loadLatestResumeAndAnalysis]);

  async function handleAnalyze() {
    if (!effectiveResume) {
      setError(pickLocaleText(locale, "Unggah file CV terlebih dahulu sebelum dianalisis.", "Upload CV file first before analysis."));
      return;
    }

    setLoading(true);
    setError(null);

    try {
      if (selectedFile) {
        await api.saveResumeUpload(selectedFile, effectiveResume, locale);
      } else {
        await api.saveResume(effectiveResume, locale);
      }

      const analysis = normalizeAnalysisForLocale(locale, await api.analyzeResume(undefined, locale));
      setAnalysisSummary(analysis.summary ?? "");
      setAnalysisResponse(analysis.response ?? "");
      setAnalysisHighlights(Array.isArray(analysis.highlights) ? analysis.highlights : []);
      setAnalysisRecommendations(Array.isArray(analysis.recommendations) ? analysis.recommendations : []);
      setLatestResumePreview(effectiveResume.slice(0, 2200));
      setSaved(true);
    } catch (err) {
      const rawMessage = err instanceof Error ? err.message : pickLocaleText(locale, "Analisis gagal.", "Analysis failed.");
      setError(localizeUploadError(locale, rawMessage));
    } finally {
      setLoading(false);
    }
  }

  async function handleFileUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }

    setLoading(true);
    setError(null);
    setSelectedFileName(file.name);
    setSelectedFile(file);
    setSaved(false);
    setAnalysisSummary("");
    setAnalysisResponse("");
    setAnalysisHighlights([]);
    setAnalysisRecommendations([]);

    try {
      const content = await extractTextFromResumeFile(file);
      if (!content.trim()) {
        throw new Error(pickLocaleText(locale, "Konten CV tidak dapat dibaca. Coba file lain.", "CV content could not be parsed. Please try another file."));
      }

      setResumeText(content);
    } catch (err) {
      setResumeText("");
      setLatestResumePreview("");
      setSelectedFile(null);
      setSelectedFileName(null);
      const rawMessage = err instanceof Error ? err.message : pickLocaleText(locale, "Gagal membaca file CV.", "Failed to read CV file.");
      setError(localizeUploadError(locale, rawMessage));
    } finally {
      setLoading(false);
    }
  }

  return (
    <AppShell title={pickLocaleText(locale, "Unggah Resume (CV)", "Upload Resume (CV)")} subtitle={pickLocaleText(locale, "Unggah file CV lalu analisis sebelum mulai interview.", "Upload CV file and analyze before starting interview.")}>
      <div className="grid gap-4 xl:grid-cols-2">
        <Card className="space-y-4 xl:col-span-2">
          <div>
            <h2 className="text-base font-semibold text-white">{pickLocaleText(locale, "Unggah CV", "Upload CV")}</h2>
            <p className="mt-1 text-sm text-[var(--color-text-muted)]">
              {pickLocaleText(locale, "Pengguna hanya bisa mengunggah file CV. Format didukung:", "Only CV files are supported. Allowed formats:")} {getAllowedResumeExtensionsLabel()}.
            </p>
          </div>

          <Input type="file" accept={RESUME_ACCEPT} onChange={handleFileUpload} className="cursor-pointer" />

          {selectedFileName && (
            <p className="text-xs text-cyan-200/85">
              {pickLocaleText(locale, "File terpilih", "Selected file")}: {selectedFileName}
            </p>
          )}

          <Button onClick={handleAnalyze} disabled={loading}>
            {loading ? pickLocaleText(locale, "Memproses...", "Processing...") : pickLocaleText(locale, "Simpan & Analisis CV", "Save & Analyze CV")}
          </Button>

          {error && <p className="text-sm text-red-300">{error}</p>}
          {saved && (
            <p className="text-sm text-cyan-200">
              {pickLocaleText(locale, "CV berhasil disimpan dan dianalisis. Kamu bisa lanjut ke Practice untuk interview.", "CV has been saved and analyzed. You can continue to Practice for interview.")}
            </p>
          )}
        </Card>

        <Card className="xl:col-span-2">
          <div className="flex flex-wrap items-start justify-between gap-3">
            <div>
              <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "CV Terbaru", "Latest CV")}</h3>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Pratinjau CV terbaru yang tersimpan di backend.", "Preview latest CV saved on backend.")}</p>
            </div>
          </div>

          <div className="mt-4 rounded-[16px] border border-white/10 bg-white/5 p-4 text-sm text-white/85">
            {loadingLatest ? (
              <p className="text-[var(--color-text-muted)]">{pickLocaleText(locale, "Memuat CV terbaru...", "Loading latest CV...")}</p>
            ) : latestResumePreview ? (
              <p className="whitespace-pre-wrap leading-relaxed">{latestResumePreview}</p>
            ) : (
              <p className="text-[var(--color-text-muted)]">{pickLocaleText(locale, "Belum ada CV tersimpan.", "No CV saved yet.")}</p>
            )}
          </div>
        </Card>

        <Card className="xl:col-span-2">
          <div>
            <h3 className="text-base font-semibold text-white">{pickLocaleText(locale, "Ringkasan analisis CV", "CV analysis summary")}</h3>
            <p className="mt-1 text-sm text-[var(--color-text-muted)]">{pickLocaleText(locale, "Hasil analisis AI berdasarkan CV terbaru.", "AI analysis results based on latest CV.")}</p>
          </div>

          <div className="mt-4 grid gap-4 lg:grid-cols-3">
            <InfoColumn
              title={pickLocaleText(locale, "Jumlah kata", "Word count")}
              items={cvStats.words ? [cvStats.words.toString()] : []}
              empty={pickLocaleText(locale, "Belum ada konten CV.", "No CV content yet.")}
            />
            <InfoColumn
              title={pickLocaleText(locale, "Jumlah karakter", "Character count")}
              items={cvStats.chars ? [cvStats.chars.toString()] : []}
              empty={pickLocaleText(locale, "Belum ada konten CV.", "No CV content yet.")}
            />
            <InfoColumn
              title={pickLocaleText(locale, "Jumlah baris", "Line count")}
              items={cvStats.lines ? [cvStats.lines.toString()] : []}
              empty={pickLocaleText(locale, "Belum ada konten CV.", "No CV content yet.")}
            />
          </div>

          <div className="mt-5 space-y-4">
            <div className="rounded-[16px] border border-white/10 bg-white/5 p-4">
              <p className="text-sm font-semibold text-white">{pickLocaleText(locale, "Ringkasan", "Summary")}</p>
              <p className="mt-2 text-sm text-white/85">{analysisSummary || pickLocaleText(locale, "Belum ada ringkasan analisis.", "No analysis summary yet.")}</p>
            </div>

            <div className="rounded-[16px] border border-white/10 bg-white/5 p-4">
              <p className="text-sm font-semibold text-white">{pickLocaleText(locale, "Respons", "Response")}</p>
              <p className="mt-2 text-sm text-white/85">{analysisResponse || pickLocaleText(locale, "Belum ada respons analisis.", "No analysis response yet.")}</p>
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <InfoColumn
                title={pickLocaleText(locale, "Sorotan", "Highlights")}
                items={analysisHighlights}
                empty={pickLocaleText(locale, "Belum ada highlight analisis.", "No analysis highlights yet.")}
              />
              <InfoColumn
                title={pickLocaleText(locale, "Rekomendasi", "Recommendations")}
                items={analysisRecommendations}
                empty={pickLocaleText(locale, "Belum ada rekomendasi analisis.", "No analysis recommendations yet.")}
              />
            </div>
          </div>

          <div className="mt-5">
            <p className="mb-2 text-sm font-semibold text-white">{pickLocaleText(locale, "Status analisis", "Analysis status")}</p>
            <div className="flex flex-wrap gap-2">
              <Tag>{saved ? pickLocaleText(locale, "CV siap untuk interview", "CV ready for interview") : pickLocaleText(locale, "Unggah dan analisis CV terlebih dahulu", "Upload and analyze your CV first")}</Tag>
            </div>
          </div>
        </Card>
      </div>
    </AppShell>
  );
}

// translation via backend removed; frontend uses Google Translate widget for dynamic client-side translation

function InfoColumn({ title, items, empty }: { title: string; items: string[]; empty: string }) {
  return (
    <div className="rounded-[16px] border border-white/10 bg-white/5 p-4">
      <p className="text-sm font-semibold text-white">{title}</p>
      <div className="mt-2 space-y-1 text-sm text-white/90">
        {items.length > 0 ? items.map((item) => <p key={item}>• {item}</p>) : <p className="text-[var(--color-text-muted)]">{empty}</p>}
      </div>
    </div>
  );
}

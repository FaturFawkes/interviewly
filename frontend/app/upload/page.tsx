"use client";

import { ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Tag } from "@/components/ui/Tag";
import { api } from "@/lib/api/endpoints";
import { extractTextFromResumeFile, getAllowedResumeExtensionsLabel } from "@/lib/resume-parser";
import { pickLocaleText } from "@/lib/i18n";

const RESUME_ACCEPT = ".pdf,.docx,.txt,.md,.rtf,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,text/markdown,application/rtf,text/rtf";

export default function UploadPage() {
  const { locale } = useLanguage();
  const [resumeText, setResumeText] = useState("");
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

      const analysis = await api.analyzeResume(normalizedResume);
      setAnalysisSummary(analysis.summary ?? "");
      setAnalysisResponse(analysis.response ?? "");
      setAnalysisHighlights(Array.isArray(analysis.highlights) ? analysis.highlights : []);
      setAnalysisRecommendations(Array.isArray(analysis.recommendations) ? analysis.recommendations : []);
      setSaved(true);
    } catch (err) {
      const message = err instanceof Error ? err.message : pickLocaleText(locale, "Gagal memuat CV terbaru.", "Failed to load latest CV.");
      const lowerMessage = message.toLowerCase();

      if (lowerMessage.includes("resume not found") || lowerMessage.includes("not found")) {
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
  }, []);

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
      await api.saveResume(effectiveResume);
      const analysis = await api.analyzeResume(effectiveResume);
      setAnalysisSummary(analysis.summary ?? "");
      setAnalysisResponse(analysis.response ?? "");
      setAnalysisHighlights(Array.isArray(analysis.highlights) ? analysis.highlights : []);
      setAnalysisRecommendations(Array.isArray(analysis.recommendations) ? analysis.recommendations : []);
      setLatestResumePreview(effectiveResume.slice(0, 2200));
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : pickLocaleText(locale, "Analisis gagal.", "Analysis failed."));
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
    setSaved(false);
    setAnalysisSummary("");
    setAnalysisResponse("");
    setAnalysisHighlights([]);
    setAnalysisRecommendations([]);

    try {
      const content = await extractTextFromResumeFile(file);
      if (!content.trim()) {
        throw new Error("Konten CV tidak dapat dibaca. Coba file lain.");
      }

      setResumeText(content);
    } catch (err) {
      setResumeText("");
      setLatestResumePreview("");
      setError(err instanceof Error ? err.message : pickLocaleText(locale, "Gagal membaca file CV.", "Failed to read CV file."));
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
              CV berhasil disimpan dan dianalisis. Kamu bisa lanjut ke Practice untuk interview.
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
                title="Highlights"
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

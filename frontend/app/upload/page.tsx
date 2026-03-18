"use client";

import { ChangeEvent, useCallback, useEffect, useMemo, useState } from "react";

import { AppShell } from "@/components/layout/AppShell";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input } from "@/components/ui/Input";
import { Tag } from "@/components/ui/Tag";
import { api } from "@/lib/api/endpoints";
import { extractTextFromResumeFile, getAllowedResumeExtensionsLabel } from "@/lib/resume-parser";

const RESUME_ACCEPT = ".pdf,.docx,.txt,.md,.rtf,application/pdf,application/vnd.openxmlformats-officedocument.wordprocessingml.document,text/plain,text/markdown,application/rtf,text/rtf";

export default function UploadPage() {
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
      const message = err instanceof Error ? err.message : "Failed to load latest CV.";
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
      setError("Upload file CV terlebih dahulu sebelum dianalisis.");
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
      setError(err instanceof Error ? err.message : "Analysis failed.");
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
      setError(err instanceof Error ? err.message : "Gagal membaca file CV.");
    } finally {
      setLoading(false);
    }
  }

  return (
    <AppShell title="Upload Resume (CV)" subtitle="Upload file CV lalu analisis sebelum mulai interview.">
      <div className="grid gap-4 xl:grid-cols-2">
        <Card className="space-y-4 xl:col-span-2">
          <div>
            <h2 className="text-base font-semibold text-white">Upload CV</h2>
            <p className="mt-1 text-sm text-[var(--color-text-muted)]">
              User hanya bisa upload file CV. Format didukung: {getAllowedResumeExtensionsLabel()}.
            </p>
          </div>

          <Input type="file" accept={RESUME_ACCEPT} onChange={handleFileUpload} className="cursor-pointer" />

          {selectedFileName && (
            <p className="text-xs text-cyan-200/85">
              Selected file: {selectedFileName}
            </p>
          )}

          <Button onClick={handleAnalyze} disabled={loading}>
            {loading ? "Processing..." : "Simpan & Analyze CV"}
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
              <h3 className="text-base font-semibold text-white">Latest CV</h3>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">Preview CV terbaru yang tersimpan di backend.</p>
            </div>
          </div>

          <div className="mt-4 rounded-[16px] border border-white/10 bg-white/5 p-4 text-sm text-white/85">
            {loadingLatest ? (
              <p className="text-[var(--color-text-muted)]">Memuat latest CV...</p>
            ) : latestResumePreview ? (
              <p className="whitespace-pre-wrap leading-relaxed">{latestResumePreview}</p>
            ) : (
              <p className="text-[var(--color-text-muted)]">Belum ada CV tersimpan.</p>
            )}
          </div>
        </Card>

        <Card className="xl:col-span-2">
          <div>
            <h3 className="text-base font-semibold text-white">CV analysis summary</h3>
            <p className="mt-1 text-sm text-[var(--color-text-muted)]">Hasil analisis AI berdasarkan CV terbaru.</p>
          </div>

          <div className="mt-4 grid gap-4 lg:grid-cols-3">
            <InfoColumn
              title="Word count"
              items={cvStats.words ? [cvStats.words.toString()] : []}
              empty="No CV content yet."
            />
            <InfoColumn
              title="Character count"
              items={cvStats.chars ? [cvStats.chars.toString()] : []}
              empty="No CV content yet."
            />
            <InfoColumn
              title="Line count"
              items={cvStats.lines ? [cvStats.lines.toString()] : []}
              empty="No CV content yet."
            />
          </div>

          <div className="mt-5 space-y-4">
            <div className="rounded-[16px] border border-white/10 bg-white/5 p-4">
              <p className="text-sm font-semibold text-white">Summary</p>
              <p className="mt-2 text-sm text-white/85">{analysisSummary || "Belum ada summary analysis."}</p>
            </div>

            <div className="rounded-[16px] border border-white/10 bg-white/5 p-4">
              <p className="text-sm font-semibold text-white">Response</p>
              <p className="mt-2 text-sm text-white/85">{analysisResponse || "Belum ada response analysis."}</p>
            </div>

            <div className="grid gap-4 lg:grid-cols-2">
              <InfoColumn
                title="Highlights"
                items={analysisHighlights}
                empty="Belum ada highlights analysis."
              />
              <InfoColumn
                title="Recommendations"
                items={analysisRecommendations}
                empty="Belum ada recommendations analysis."
              />
            </div>
          </div>

          <div className="mt-5">
            <p className="mb-2 text-sm font-semibold text-white">Analysis status</p>
            <div className="flex flex-wrap gap-2">
              <Tag>{saved ? "CV ready for interview" : "Upload and analyze your CV first"}</Tag>
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

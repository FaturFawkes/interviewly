"use client";

import { ChangeEvent, useEffect, useMemo, useState } from "react";
import { Brain, CheckCircle, Code, Download, FileText, Sparkles, Users } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { Button } from "@/components/ui/Button";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { Input, TextArea } from "@/components/ui/Input";
import { api } from "@/lib/api/endpoints";
import type { ResumeAIAnalysis } from "@/lib/api/types";

const CV_UPLOAD_ACCEPT = ".docx,.pdf,.txt,application/vnd.openxmlformats-officedocument.wordprocessingml.document,application/pdf,text/plain";

export default function UploadPage() {
  const [resumeText, setResumeText] = useState("");
  const [existingResumeText, setExistingResumeText] = useState("");
  const [resumeFile, setResumeFile] = useState<File | null>(null);
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(false);
  const [loadingExisting, setLoadingExisting] = useState(true);
  const [downloading, setDownloading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [analysis, setAnalysis] = useState<ResumeAIAnalysis | null>(null);

  useEffect(() => {
    let mounted = true;

    async function loadExistingResume() {
      try {
        const latestResume = await api.getLatestResume();
        if (!mounted) {
          return;
        }

        setExistingResumeText((latestResume.content ?? "").trim());
      } catch (err) {
        if (!mounted) {
          return;
        }

        const message = err instanceof Error ? err.message.toLowerCase() : "";
        if (message.includes("resume not found") || message.includes("not found")) {
          setExistingResumeText("");
          return;
        }

        setError(err instanceof Error ? err.message : "Failed to load existing CV.");
      } finally {
        if (mounted) {
          setLoadingExisting(false);
        }
      }
    }

    void loadExistingResume();

    return () => {
      mounted = false;
    };
  }, []);

  const displayedResumeText = useMemo(() => {
    const current = resumeText.trim();
    if (current) {
      return current;
    }

    return existingResumeText.trim();
  }, [existingResumeText, resumeText]);

  const cvStats = useMemo(() => {
    const cleaned = displayedResumeText.trim();
    if (!cleaned) {
      return { words: 0, chars: 0, lines: 0 };
    }

    return {
      words: cleaned.split(/\s+/).length,
      chars: cleaned.length,
      lines: cleaned.split(/\n+/).length,
    };
  }, [displayedResumeText]);

  async function handleAnalyze() {
    const hasUploadedReplacement = Boolean(resumeFile) && Boolean(resumeText.trim());
    const hasLatestResume = Boolean(existingResumeText.trim());

    if (!hasUploadedReplacement && !hasLatestResume) {
      setError("No CV found. Please upload your CV first.");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const result = hasUploadedReplacement
        ? await api.analyzeResume(resumeText, resumeFile ?? undefined)
        : await api.analyzeResume("");

      setAnalysis(result.analysis);
      setExistingResumeText(result.resume.content?.trim() ?? "");
      setResumeText("");
      setResumeFile(null);
      setSaved(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Analysis failed.");
    } finally {
      setLoading(false);
    }
  }

  async function handleDownloadCV() {
    setDownloading(true);
    setError(null);

    try {
      await api.downloadLatestResume();
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to download CV.");
    } finally {
      setDownloading(false);
    }
  }

  async function handleFileUpload(event: ChangeEvent<HTMLInputElement>) {
    const file = event.target.files?.[0];
    if (!file) {
      return;
    }

    setError(null);
    setSaved(false);
    setAnalysis(null);
    setExistingResumeText("");
    setResumeFile(file);

    try {
      const content = await extractResumeText(file);
      const cleanedContent = content.trim();

      if (!cleanedContent) {
        setResumeText("");
        setResumeFile(null);
        setError("No readable text found in the uploaded file.");
        return;
      }

      setResumeText(cleanedContent);
    } catch (err) {
      setResumeText("");
      setResumeFile(null);
      setError(err instanceof Error ? err.message : "Failed to read uploaded CV file.");
    } finally {
      event.target.value = "";
    }
  }

  return (
    <AppShell title="Upload CV" subtitle="Analyze and save your CV profile for interview generation.">
      <div className="grid gap-4 xl:grid-cols-2">
        <GlassCard className="space-y-4 p-6">
          <div className="flex items-center gap-2">
            <FileText className="h-5 w-5 text-purple-400" />
            <h2 className="text-base font-semibold text-white">CV Input</h2>
          </div>
          <p className="text-sm text-[var(--color-text-muted)]">Upload your CV file (.docx, .pdf, or .txt) for analysis.</p>
          <Input type="file" accept={CV_UPLOAD_ACCEPT} onChange={handleFileUpload} className="cursor-pointer" />

          <div className="space-y-2">
            <p className="text-xs uppercase tracking-wide text-white/45">
              {loadingExisting ? "Loading existing CV..." : "Existing CV placeholder"}
            </p>
            <TextArea
              value={displayedResumeText}
              readOnly
              placeholder="Upload your latest CV file to replace existing CV data."
              className="min-h-40"
            />
            <p className="text-xs text-white/35">
              {resumeText.trim()
                ? "Preview of the uploaded replacement CV file."
                : existingResumeText.trim()
                  ? "Using your latest saved CV. You can analyze directly or upload a new file to replace it."
                  : "No saved CV found yet. Upload one to start."}
            </p>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button onClick={handleAnalyze} disabled={loading} className={loading ? "opacity-80 pointer-events-none" : ""}>
              {loading
                ? "Analyzing..."
                : resumeText.trim() && resumeFile
                  ? "Analyze Uploaded CV"
                  : existingResumeText.trim()
                    ? "Analyze Latest CV"
                    : "Analyze CV"}
            </Button>
            <Button variant="secondary" onClick={() => void handleDownloadCV()} disabled={downloading}>
              <Download className="mr-2 h-4 w-4" />
              {downloading ? "Downloading..." : "Download CV"}
            </Button>
          </div>
          {error && <p className="text-sm text-red-300">{error}</p>}
          {saved && (
            <p className="text-sm text-cyan-200 flex items-center gap-2">
              <CheckCircle className="h-4 w-4" />
              CV analyzed with AI and saved. Continue to Interview page and input JD only.
            </p>
          )}
        </GlassCard>

        <GradientBorderCard className="xl:col-span-1">
          <div className="p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 className="text-base font-semibold text-white">CV analysis summary</h3>
                <p className="mt-1 text-sm text-[var(--color-text-muted)]">OpenAI-generated summary and response from your uploaded CV.</p>
              </div>
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

            {analysis ? (
              <div className="mt-5 rounded-[16px] border border-cyan-300/20 bg-cyan-400/10 p-4 text-sm text-cyan-100 space-y-3">
                <div>
                  <p className="text-xs uppercase tracking-wide text-cyan-200/80">Summary</p>
                  <p className="mt-1 text-sm text-cyan-100">{analysis.summary}</p>
                </div>
                <div>
                  <p className="text-xs uppercase tracking-wide text-cyan-200/80">Response</p>
                  <p className="mt-1 text-sm text-cyan-100">{analysis.response}</p>
                </div>
                {analysis.highlights.length > 0 && (
                  <div>
                    <p className="text-xs uppercase tracking-wide text-cyan-200/80">Highlights</p>
                    <p className="mt-1 text-sm text-cyan-100">{analysis.highlights.join(", ")}</p>
                  </div>
                )}
                {analysis.recommendations.length > 0 && (
                  <div>
                    <p className="text-xs uppercase tracking-wide text-cyan-200/80">Recommendations</p>
                    <ul className="mt-1 list-disc list-inside space-y-1 text-sm text-cyan-100">
                      {analysis.recommendations.map((item) => (
                        <li key={item}>{item}</li>
                      ))}
                    </ul>
                  </div>
                )}
              </div>
            ) : (
              <div className="mt-5 rounded-[16px] border border-cyan-300/20 bg-cyan-400/10 p-4 text-sm text-cyan-100">
                {saved ? "CV ready for interview generation." : "Upload and analyze your CV first."}
              </div>
            )}
          </div>
        </GradientBorderCard>

        <GlassCard className="xl:col-span-2 p-6">
          <div className="grid gap-4 md:grid-cols-3">
            <InsightBlock title="Technical Skills" icon={<Code className="h-5 w-5 text-purple-400" />} values={extractKeywords(displayedResumeText, "technical")} color="purple" />
            <InsightBlock title="Behavioral Skills" icon={<Users className="h-5 w-5 text-cyan-400" />} values={extractKeywords(displayedResumeText, "behavioral")} color="cyan" />
            <InsightBlock title="Key Signals" icon={<Brain className="h-5 w-5 text-blue-400" />} values={extractKeywords(displayedResumeText, "signal")} color="blue" />
          </div>

          {!displayedResumeText.trim() && (
            <div className="mt-6 flex flex-col items-center justify-center min-h-[160px] rounded-[16px] border border-white/[0.06] bg-white/[0.02]">
              <div className="w-14 h-14 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-3">
                <Sparkles className="w-6 h-6 text-white/15" />
              </div>
              <p className="text-white/30 text-sm text-center">
                Upload your CV and click analyze<br />
                to see extracted highlights
              </p>
            </div>
          )}
        </GlassCard>
      </div>
    </AppShell>
  );
}

function fileExtension(fileName: string): string {
  const dotIndex = fileName.lastIndexOf(".");
  if (dotIndex < 0) {
    return "";
  }

  return fileName.slice(dotIndex).toLowerCase();
}

async function extractResumeText(file: File): Promise<string> {
  const extension = fileExtension(file.name);

  if (extension === ".txt") {
    return file.text();
  }

  if (extension === ".docx") {
    return extractDocxText(file);
  }

  if (extension === ".pdf") {
    return extractPdfText(file);
  }

  throw new Error("Unsupported file format. Please upload .docx, .pdf, or .txt.");
}

type MammothExtractRawText = (input: { arrayBuffer: ArrayBuffer }) => Promise<{ value: string }>;

async function extractDocxText(file: File): Promise<string> {
  const mammothModule = await import("mammoth");
  const extractRawText =
    (mammothModule as { extractRawText?: MammothExtractRawText }).extractRawText ??
    (mammothModule as { default?: { extractRawText?: MammothExtractRawText } }).default?.extractRawText;

  if (!extractRawText) {
    throw new Error("Failed to initialize DOCX parser.");
  }

  const result = await extractRawText({ arrayBuffer: await file.arrayBuffer() });
  return result.value ?? "";
}

async function extractPdfText(file: File): Promise<string> {
  const pdfjs = await import("pdfjs-dist");
  pdfjs.GlobalWorkerOptions.workerSrc = new URL("pdfjs-dist/build/pdf.worker.min.mjs", import.meta.url).toString();

  const loadingTask = pdfjs.getDocument({ data: new Uint8Array(await file.arrayBuffer()) });
  const pdfDocument = await loadingTask.promise;

  const pageTexts: string[] = [];
  for (let pageNumber = 1; pageNumber <= pdfDocument.numPages; pageNumber += 1) {
    const page = await pdfDocument.getPage(pageNumber);
    const textContent = await page.getTextContent();
    const pageText = textContent.items
      .map((item) => ("str" in item ? item.str : ""))
      .join(" ")
      .trim();

    if (pageText) {
      pageTexts.push(pageText);
    }
  }

  await pdfDocument.destroy();
  return pageTexts.join("\n");
}

function extractKeywords(text: string, kind: "technical" | "behavioral" | "signal") {
  const content = text.toLowerCase();

  const libraries = {
    technical: ["react", "typescript", "node", "go", "sql", "docker", "aws", "kubernetes"],
    behavioral: ["lead", "team", "stakeholder", "communication", "mentoring", "collaboration", "ownership"],
    signal: ["years", "senior", "impact", "scale", "optimize", "architect", "deliver"],
  }[kind];

  const detected = libraries.filter((item) => content.includes(item));
  if (detected.length > 0) {
    return detected.slice(0, 8);
  }

  return kind === "technical"
    ? ["React", "TypeScript", "Node.js"]
    : kind === "behavioral"
      ? ["Communication", "Leadership", "Teamwork"]
      : ["Impact", "Ownership", "Delivery"];
}

function InsightBlock({ title, icon, values, color }: { title: string; icon: React.ReactNode; values: string[]; color: "purple" | "cyan" | "blue" }) {
  const colorClass = color === "purple"
    ? "bg-purple-500/10 border-purple-500/20 text-purple-300"
    : color === "cyan"
      ? "bg-cyan-500/10 border-cyan-500/20 text-cyan-300"
      : "bg-blue-500/10 border-blue-500/20 text-blue-300";

  return (
    <GlassCard className="p-5">
      <div className="flex items-center gap-2 mb-4">
        {icon}
        <h3 className="text-white">{title}</h3>
      </div>
      <div className="flex flex-wrap gap-2">
        {values.map((kw) => (
          <span key={kw} className={`px-3 py-1.5 rounded-lg border text-sm ${colorClass}`}>
            {kw}
          </span>
        ))}
      </div>
    </GlassCard>
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

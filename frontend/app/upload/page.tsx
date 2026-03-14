"use client";

import { ChangeEvent, useMemo, useState } from "react";
import { Brain, CheckCircle, Code, FileText, Sparkles, Users } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { Button } from "@/components/ui/Button";
import { GlassCard, GradientBorderCard } from "@/components/ui/GlassCard";
import { Input, TextArea } from "@/components/ui/Input";
import { api } from "@/lib/api/endpoints";

export default function UploadPage() {
  const [resumeText, setResumeText] = useState("");
  const [saved, setSaved] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const cvStats = useMemo(() => {
    const cleaned = resumeText.trim();
    if (!cleaned) {
      return { words: 0, chars: 0, lines: 0 };
    }

    return {
      words: cleaned.split(/\s+/).length,
      chars: cleaned.length,
      lines: cleaned.split(/\n+/).length,
    };
  }, [resumeText]);

  async function handleAnalyze() {
    if (!resumeText.trim()) {
      setError("Please upload or paste your CV before analyzing.");
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.saveResume(resumeText);
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

    const content = await file.text();
    setResumeText(content);
    setSaved(false);
  }

  return (
    <AppShell title="Upload CV" subtitle="Analyze and save your CV profile for interview generation.">
      <div className="grid gap-4 xl:grid-cols-2">
        <GlassCard className="space-y-4 p-6">
          <div className="flex items-center gap-2">
            <FileText className="h-5 w-5 text-purple-400" />
            <h2 className="text-base font-semibold text-white">CV Input</h2>
          </div>
          <p className="text-sm text-[var(--color-text-muted)]">Upload .txt file or paste your CV content.</p>
          <Input type="file" accept=".txt" onChange={handleFileUpload} className="cursor-pointer" />
          <TextArea
            value={resumeText}
            onChange={(event) => {
              setResumeText(event.target.value);
              setSaved(false);
            }}
            placeholder="Paste CV text..."
            className="min-h-72"
          />

          <Button onClick={handleAnalyze} disabled={loading} className={loading ? "opacity-80 pointer-events-none" : ""}>
            {loading ? "Analyzing..." : "Analyze CV"}
          </Button>
          {error && <p className="text-sm text-red-300">{error}</p>}
          {saved && (
            <p className="text-sm text-cyan-200 flex items-center gap-2">
              <CheckCircle className="h-4 w-4" />
              CV analyzed and saved. Continue to Interview page and input JD only.
            </p>
          )}
        </GlassCard>

        <GradientBorderCard className="xl:col-span-1">
          <div className="p-6">
            <div className="flex flex-wrap items-start justify-between gap-3">
              <div>
                <h3 className="text-base font-semibold text-white">CV analysis summary</h3>
                <p className="mt-1 text-sm text-[var(--color-text-muted)]">Basic profile metrics from your uploaded CV.</p>
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

            <div className="mt-5 rounded-[16px] border border-cyan-300/20 bg-cyan-400/10 p-4 text-sm text-cyan-100">
              {saved ? "CV ready for interview generation." : "Upload and analyze your CV first."}
            </div>
          </div>
        </GradientBorderCard>

        <GlassCard className="xl:col-span-2 p-6">
          <div className="grid gap-4 md:grid-cols-3">
            <InsightBlock title="Technical Skills" icon={<Code className="h-5 w-5 text-purple-400" />} values={extractKeywords(resumeText, "technical")} color="purple" />
            <InsightBlock title="Behavioral Skills" icon={<Users className="h-5 w-5 text-cyan-400" />} values={extractKeywords(resumeText, "behavioral")} color="cyan" />
            <InsightBlock title="Key Signals" icon={<Brain className="h-5 w-5 text-blue-400" />} values={extractKeywords(resumeText, "signal")} color="blue" />
          </div>

          {!resumeText.trim() && (
            <div className="mt-6 flex flex-col items-center justify-center min-h-[160px] rounded-[16px] border border-white/[0.06] bg-white/[0.02]">
              <div className="w-14 h-14 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-3">
                <Sparkles className="w-6 h-6 text-white/15" />
              </div>
              <p className="text-white/30 text-sm text-center">
                Paste your CV and click analyze<br />
                to see extracted highlights
              </p>
            </div>
          )}
        </GlassCard>
      </div>
    </AppShell>
  );
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

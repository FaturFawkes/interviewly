"use client";

import { ChangeEvent, useMemo, useState } from "react";

import { AppShell } from "@/components/layout/AppShell";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input, TextArea } from "@/components/ui/Input";
import { Tag } from "@/components/ui/Tag";
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
    <AppShell title="CV Upload" subtitle="Analyze and save your CV profile for upcoming interview sessions.">
      <div className="grid gap-4 xl:grid-cols-2">
        <Card className="space-y-4 xl:col-span-2">
          <div>
            <h2 className="text-base font-semibold text-white">CV input</h2>
            <p className="mt-1 text-sm text-[var(--color-text-muted)]">Upload .txt file or paste your CV content.</p>
          </div>
          <Input type="file" accept=".txt" onChange={handleFileUpload} className="cursor-pointer" />
          <TextArea
            value={resumeText}
            onChange={(event) => {
              setResumeText(event.target.value);
              setSaved(false);
            }}
            placeholder="Paste CV text..."
            className="min-h-52"
          />
          <Button onClick={handleAnalyze} disabled={loading}>
            {loading ? "Analyzing..." : "Analyze CV"}
          </Button>
          {error && <p className="text-sm text-red-300">{error}</p>}
          {saved && <p className="text-sm text-cyan-200">CV analyzed and saved. You can continue to Interview page and input JD only.</p>}
        </Card>

        <Card className="xl:col-span-2">
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

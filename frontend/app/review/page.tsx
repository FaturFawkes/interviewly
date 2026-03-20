"use client";

import { useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { AlertCircle, Bot, Mic, MessageSquare, RefreshCcw, Send, Sparkles, Target } from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input, TextArea } from "@/components/ui/Input";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";
import type {
  InterviewLanguage,
  InterviewMode,
  ReviewEndResponse,
  ReviewProgress,
  ReviewSession,
  ReviewStartPayload,
} from "@/lib/api/types";

const modeOptions: Array<{ value: InterviewMode; icon: typeof MessageSquare }> = [
  { value: "text", icon: MessageSquare },
  { value: "voice", icon: Mic },
];

export default function ReviewPage() {
  const { locale } = useLanguage();
  const router = useRouter();

  const [sessionType, setSessionType] = useState<"review" | "recovery">("review");
  const [inputMode, setInputMode] = useState<InterviewMode>("text");
  const [interviewLanguage, setInterviewLanguage] = useState<InterviewLanguage>("id");
  const [targetRole, setTargetRole] = useState("");
  const [targetCompany, setTargetCompany] = useState("");
  const [interviewPrompt, setInterviewPrompt] = useState("");
  const [startInput, setStartInput] = useState("");
  const [responseInput, setResponseInput] = useState("");

  const [activeSession, setActiveSession] = useState<ReviewSession | null>(null);
  const [coachingSummary, setCoachingSummary] = useState<ReviewEndResponse | null>(null);
  const [reviewProgress, setReviewProgress] = useState<ReviewProgress | null>(null);

  const [loadingStart, setLoadingStart] = useState(false);
  const [loadingRespond, setLoadingRespond] = useState(false);
  const [loadingEnd, setLoadingEnd] = useState(false);
  const [loadingSummary, setLoadingSummary] = useState(false);
  const [error, setError] = useState<string | null>(null);

  async function loadSummary(): Promise<void> {
    setLoadingSummary(true);
    try {
      const [summary, progress] = await Promise.all([api.getCoachingSummary(), api.getProgress()]);
      setCoachingSummary(summary);
      setReviewProgress(progress.review_progress);
    } catch {
      // Summary is optional for first-time users.
    } finally {
      setLoadingSummary(false);
    }
  }

  useEffect(() => {
    void loadSummary();
  }, []);

  async function handleStart(): Promise<void> {
    if (inputMode === "voice") {
      const params = new URLSearchParams({
        session_type: sessionType,
        interview_language: interviewLanguage,
        target_role: targetRole,
        target_company: targetCompany,
        interview_prompt: interviewPrompt,
      });
      router.push(`/review/voice/call?${params.toString()}`);
      return;
    }

    if (!startInput.trim()) {
      setError(pickLocaleText(locale, "Isi cerita interview Anda dulu.", "Please share your interview story first."));
      return;
    }

    setError(null);
    setLoadingStart(true);

    try {
      const payload: ReviewStartPayload = {
        session_type: sessionType,
        input_mode: inputMode,
        interview_language: interviewLanguage,
        interview_prompt: interviewPrompt,
        target_role: targetRole,
        target_company: targetCompany,
      };

      payload.input_text = startInput;

      const result = await api.startReview(payload);
      setActiveSession(result.session);
      setResponseInput("");
      setStartInput("");
      await loadSummary();
    } catch (err) {
      setError(err instanceof Error ? err.message : pickLocaleText(locale, "Gagal memulai review.", "Failed to start review."));
    } finally {
      setLoadingStart(false);
    }
  }

  async function handleRespond(): Promise<void> {
    if (!activeSession) {
      return;
    }
    if (!responseInput.trim()) {
      setError(pickLocaleText(locale, "Isi respon lanjutan dulu.", "Please provide your follow-up response first."));
      return;
    }

    setError(null);
    setLoadingRespond(true);
    try {
      const activeLanguage = activeSession.interview_language || interviewLanguage;
      const result = await api.respondReview({
        session_id: activeSession.id,
        interview_language: activeLanguage,
        interview_prompt: interviewPrompt,
        ...(inputMode === "voice" ? { transcript_text: responseInput } : { input_text: responseInput }),
      });
      setActiveSession(result.session);
      setResponseInput("");
      await loadSummary();
    } catch (err) {
      setError(err instanceof Error ? err.message : pickLocaleText(locale, "Gagal mengirim respon.", "Failed to send response."));
    } finally {
      setLoadingRespond(false);
    }
  }

  async function handleEnd(): Promise<void> {
    if (!activeSession) {
      return;
    }

    setError(null);
    setLoadingEnd(true);
    try {
      const result = await api.endReview(activeSession.id);
      setCoachingSummary(result);
      setActiveSession(null);
      setResponseInput("");
      await loadSummary();
    } catch (err) {
      setError(err instanceof Error ? err.message : pickLocaleText(locale, "Gagal mengakhiri sesi.", "Failed to end session."));
    } finally {
      setLoadingEnd(false);
    }
  }

  const latestFeedback = activeSession?.feedback ?? coachingSummary?.feedback;

  const progressItems = useMemo(() => {
    if (!reviewProgress) {
      return [];
    }

    return [
      {
        label: pickLocaleText(locale, "Skor review terbaru", "Latest review score"),
        value: `${reviewProgress.latest_overall_score}`,
      },
      {
        label: pickLocaleText(locale, "Rata-rata review", "Average review score"),
        value: `${Math.round(reviewProgress.average_overall_score)}`,
      },
      {
        label: pickLocaleText(locale, "Data trend", "Trend points"),
        value: `${reviewProgress.communication_trend.length}`,
      },
    ];
  }, [locale, reviewProgress]);

  return (
    <AppShell
      title={pickLocaleText(locale, "AI Career Coach (Review Mode)", "AI Career Coach (Review Mode)")}
      subtitle={pickLocaleText(locale, "Review pengalaman interview nyata Anda (chat/voice) dan dapatkan coaching personal.", "Review your real interview experience (chat/voice) and get personalized coaching.")}
    >
      <div className="space-y-5">
        <Card className="space-y-4 p-5">
          <div className="flex items-center gap-2 text-cyan-200">
            <Target className="h-4 w-4" />
            <h2 className="text-sm font-semibold text-white">{pickLocaleText(locale, "Mulai sesi review", "Start review session")}</h2>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <label className="space-y-1">
              <span className="text-sm text-white/70">{pickLocaleText(locale, "Tipe sesi", "Session type")}</span>
              <select
                className="input-base"
                value={sessionType}
                onChange={(event) => setSessionType(event.target.value as "review" | "recovery")}
              >
                <option value="review">{pickLocaleText(locale, "Review biasa", "Standard review")}</option>
                <option value="recovery">{pickLocaleText(locale, "Post-interview recovery", "Post-interview recovery")}</option>
              </select>
            </label>

            <label className="space-y-1">
              <span className="text-sm text-white/70">{pickLocaleText(locale, "Mode input", "Input mode")}</span>
              <div className="grid grid-cols-2 gap-2">
                {modeOptions.map((option) => {
                  const Icon = option.icon;
                  const active = inputMode === option.value;
                  return (
                    <button
                      key={option.value}
                      type="button"
                      onClick={() => setInputMode(option.value)}
                      className={`rounded-xl border px-3 py-2 text-sm ${active ? "border-cyan-300/45 bg-cyan-400/15 text-cyan-100" : "border-white/10 bg-white/5 text-white/70"}`}
                    >
                      <span className="inline-flex items-center gap-2">
                        <Icon className="h-4 w-4" />
                        {option.value === "text" ? pickLocaleText(locale, "Teks", "Text") : pickLocaleText(locale, "Voice", "Voice")}
                      </span>
                    </button>
                  );
                })}
              </div>
            </label>
          </div>

          <div className="space-y-1">
            <span className="text-sm text-white/70">{pickLocaleText(locale, "Bahasa review", "Review language")}</span>
            <div className="grid grid-cols-2 gap-2">
              <button
                type="button"
                onClick={() => setInterviewLanguage("id")}
                disabled={Boolean(activeSession)}
                className={`rounded-xl border px-3 py-2 text-sm ${interviewLanguage === "id" ? "border-cyan-300/45 bg-cyan-400/15 text-cyan-100" : "border-white/10 bg-white/5 text-white/70"} ${activeSession ? "cursor-not-allowed opacity-60" : ""}`}
              >
                Bahasa Indonesia
              </button>
              <button
                type="button"
                onClick={() => setInterviewLanguage("en")}
                disabled={Boolean(activeSession)}
                className={`rounded-xl border px-3 py-2 text-sm ${interviewLanguage === "en" ? "border-cyan-300/45 bg-cyan-400/15 text-cyan-100" : "border-white/10 bg-white/5 text-white/70"} ${activeSession ? "cursor-not-allowed opacity-60" : ""}`}
              >
                English
              </button>
            </div>
          </div>

          <div className="grid gap-3 md:grid-cols-2">
            <Input
              value={targetRole}
              onChange={(event) => setTargetRole(event.target.value)}
              placeholder={pickLocaleText(locale, "Target role (opsional)", "Target role (optional)")}
            />
            <Input
              value={targetCompany}
              onChange={(event) => setTargetCompany(event.target.value)}
              placeholder={pickLocaleText(locale, "Target company (opsional)", "Target company (optional)")}
            />
          </div>

          <Input
            value={interviewPrompt}
            onChange={(event) => setInterviewPrompt(event.target.value)}
            placeholder={pickLocaleText(locale, "Pertanyaan interview yang ditanya (opsional)", "Interview question asked (optional)")}
          />

          <TextArea
            value={startInput}
            onChange={(event) => setStartInput(event.target.value)}
            className="min-h-40"
            placeholder={pickLocaleText(
              locale,
              "Ceritakan pengalaman interview nyata Anda...",
              "Share your real interview experience...",
            )}
            disabled={inputMode === "voice"}
          />

          {inputMode === "voice" && (
            <p className="text-xs text-white/60">
              {pickLocaleText(
                locale,
                "Mode voice menggunakan alur panggilan seperti Practice Voice dan agen coach khusus review.",
                "Voice mode uses a call flow similar to Practice Voice with a dedicated review coach agent.",
              )}
            </p>
          )}

          <div className="flex flex-wrap gap-2">
            <Button type="button" onClick={() => void handleStart()} disabled={loadingStart || Boolean(activeSession)}>
              <Sparkles className="mr-2 h-4 w-4" />
              {loadingStart
                ? pickLocaleText(locale, "Memulai...", "Starting...")
                : inputMode === "voice"
                  ? pickLocaleText(locale, "Mulai Voice Review", "Start Voice Review")
                  : pickLocaleText(locale, "Mulai Review", "Start Review")}
            </Button>
            <Button type="button" variant="secondary" onClick={() => void loadSummary()} disabled={loadingSummary}>
              <RefreshCcw className="mr-2 h-4 w-4" />
              {pickLocaleText(locale, "Refresh ringkasan", "Refresh summary")}
            </Button>
          </div>

          {error && (
            <p className="inline-flex items-center gap-2 rounded-full border border-red-400/30 bg-red-500/10 px-3 py-1 text-xs text-red-300">
              <AlertCircle className="h-3.5 w-3.5" />
              {error}
            </p>
          )}
        </Card>

        <div className="grid gap-4 xl:grid-cols-3">
          <Card className="xl:col-span-2 space-y-4 p-5">
            <div className="flex items-center gap-2">
              <Bot className="h-4 w-4 text-purple-300" />
              <h3 className="text-sm font-semibold text-white">{pickLocaleText(locale, "Smart coaching conversation", "Smart coaching conversation")}</h3>
            </div>

            {activeSession ? (
              <>
                <div className="rounded-xl border border-white/10 bg-white/5 p-3 text-sm text-white/85">
                  <p className="font-medium text-cyan-100">{pickLocaleText(locale, "Follow-up question", "Follow-up question")}</p>
                  <p className="mt-1">{activeSession.feedback.follow_up_question || "-"}</p>
                </div>

                <TextArea
                  value={responseInput}
                  onChange={(event) => setResponseInput(event.target.value)}
                  className="min-h-35"
                  placeholder={pickLocaleText(locale, "Jawab follow-up dari coach...", "Answer coach follow-up...")}
                />

                <div className="flex flex-wrap gap-2">
                  <Button type="button" onClick={() => void handleRespond()} disabled={loadingRespond || !responseInput.trim()}>
                    <Send className="mr-2 h-4 w-4" />
                    {loadingRespond
                      ? pickLocaleText(locale, "Mengirim...", "Sending...")
                      : pickLocaleText(locale, "Kirim Respon", "Send Response")}
                  </Button>
                  <Button type="button" variant="secondary" onClick={() => void handleEnd()} disabled={loadingEnd}>
                    {loadingEnd ? pickLocaleText(locale, "Mengakhiri...", "Ending...") : pickLocaleText(locale, "Akhiri Sesi", "End Session")}
                  </Button>
                </div>
              </>
            ) : (
              <p className="text-sm text-white/60">
                {pickLocaleText(locale, "Belum ada sesi aktif. Mulai sesi review terlebih dahulu.", "No active session yet. Start a review session first.")}
              </p>
            )}
          </Card>

          <Card className="space-y-3 p-5">
            <h3 className="text-sm font-semibold text-white">{pickLocaleText(locale, "Progress intelligence", "Progress intelligence")}</h3>
            {progressItems.length > 0 ? (
              progressItems.map((item) => (
                <div key={item.label} className="rounded-xl border border-white/10 bg-white/5 px-3 py-2">
                  <p className="text-xs text-white/60">{item.label}</p>
                  <p className="text-lg font-semibold text-white">{item.value}</p>
                </div>
              ))
            ) : (
              <p className="text-sm text-white/60">{pickLocaleText(locale, "Belum ada data progress review.", "No review progress data yet.")}</p>
            )}
          </Card>
        </div>

        <Card className="space-y-3 p-5">
          <h3 className="text-sm font-semibold text-white">{pickLocaleText(locale, "Feedback terbaru", "Latest feedback")}</h3>
          {latestFeedback ? (
            <>
              <div className="grid gap-2 md:grid-cols-4">
                <Metric title={pickLocaleText(locale, "Overall", "Overall")} value={latestFeedback.score} />
                <Metric title={pickLocaleText(locale, "Communication", "Communication")} value={latestFeedback.communication} />
                <Metric title={pickLocaleText(locale, "Structure (STAR)", "Structure (STAR)")} value={latestFeedback.structure_star} />
                <Metric title={pickLocaleText(locale, "Confidence", "Confidence")} value={latestFeedback.confidence} />
              </div>

              <div className="grid gap-3 md:grid-cols-3">
                <FeedbackList
                  title={pickLocaleText(locale, "Strengths", "Strengths")}
                  items={latestFeedback.strengths}
                  emptyText={pickLocaleText(locale, "Belum ada", "None")}
                />
                <FeedbackList
                  title={pickLocaleText(locale, "Weaknesses", "Weaknesses")}
                  items={latestFeedback.weaknesses}
                  emptyText={pickLocaleText(locale, "Belum ada", "None")}
                />
                <FeedbackList
                  title={pickLocaleText(locale, "Suggestions", "Suggestions")}
                  items={latestFeedback.suggestions}
                  emptyText={pickLocaleText(locale, "Belum ada", "None")}
                />
              </div>

              <div className="rounded-xl border border-cyan-300/30 bg-cyan-400/10 px-4 py-3 text-sm text-cyan-100">
                <p className="font-medium">{pickLocaleText(locale, "Insight", "Insight")}</p>
                <p className="mt-1">{latestFeedback.insight || "-"}</p>
              </div>

              {coachingSummary?.improvement_plan && (
                <div className="rounded-xl border border-purple-300/30 bg-purple-500/10 px-4 py-3 text-sm text-purple-100">
                  <p className="font-medium">{pickLocaleText(locale, "Improvement plan", "Improvement plan")}</p>
                  <ul className="mt-2 list-disc space-y-1 pl-4">
                    {coachingSummary.improvement_plan.practice_plan.map((item) => (
                      <li key={item}>{item}</li>
                    ))}
                  </ul>
                </div>
              )}
            </>
          ) : (
            <p className="text-sm text-white/60">{pickLocaleText(locale, "Belum ada feedback coaching.", "No coaching feedback yet.")}</p>
          )}
        </Card>
      </div>
    </AppShell>
  );
}

function Metric({ title, value }: { title: string; value: number }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 px-3 py-2">
      <p className="text-xs text-white/60">{title}</p>
      <p className="text-lg font-semibold text-white">{value}</p>
    </div>
  );
}

function FeedbackList({ title, items, emptyText }: { title: string; items: string[]; emptyText: string }) {
  return (
    <div className="rounded-xl border border-white/10 bg-white/5 px-3 py-2">
      <p className="text-xs font-medium text-white/80">{title}</p>
      {items.length > 0 ? (
        <ul className="mt-2 list-disc space-y-1 pl-4 text-sm text-white/80">
          {items.map((item) => (
            <li key={item}>{item}</li>
          ))}
        </ul>
      ) : (
        <p className="mt-2 text-sm text-white/55">{emptyText}</p>
      )}
    </div>
  );
}

"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import {
  AlertCircle,
  ArrowRight,
  Bot,
  CheckCircle,
  ChevronRight,
  Lightbulb,
  MessageSquare,
  Send,
  Sparkles,
  User,
} from "lucide-react";

import { AppShell } from "@/components/layout/AppShell";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { FeedbackPanel } from "@/components/interview/FeedbackPanel";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input, TextArea } from "@/components/ui/Input";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";
import type { Locale } from "@/lib/i18n";
import type { InterviewDifficulty, InterviewLanguage, InterviewMode } from "@/lib/api/types";

export default function PracticePage() {
  const { locale } = useLanguage();
  const router = useRouter();
  const [showVoiceStartModal, setShowVoiceStartModal] = useState(false);
  const [connectingDotCount, setConnectingDotCount] = useState(1);
  const [startInterviewMode, setStartInterviewMode] = useState<InterviewMode>("text");
  const [showBackExitModal, setShowBackExitModal] = useState(false);
  const [backExitAction, setBackExitAction] = useState<"idle" | "ending">("idle");
  const allowBackNavigationRef = useRef(false);
  const {
    session,
    questions,
    currentQuestion,
    currentIndex,
    answer,
    setAnswer,
    feedback,
    lastScore,
    loading,
    error,
    initializeInterview,
    submitCurrentAnswer,
    completeSession,
    resetInterviewFlow,
    goToNextQuestion,
    sessionCompleted,
  } = useInterviewFlow();

  const isLastQuestion = currentIndex >= questions.length - 1;
  const hasActiveTextSession = Boolean(
    currentQuestion
    && session?.interview_mode === "text"
    && session?.status === "active"
    && !sessionCompleted,
  );
  const hasActiveVoiceSession = Boolean(
    currentQuestion
    && session?.interview_mode === "voice"
    && session?.status === "active"
    && !sessionCompleted,
  );

  useEffect(() => {
    if (!showVoiceStartModal) {
      return;
    }

    const intervalId = window.setInterval(() => {
      setConnectingDotCount((prev) => (prev >= 3 ? 1 : prev + 1));
    }, 350);

    return () => {
      window.clearInterval(intervalId);
    };
  }, [showVoiceStartModal]);

  useEffect(() => {
    if (typeof window === "undefined" || !hasActiveTextSession) {
      return;
    }

    const guardState = { interviewBackGuard: "practice-text" };
    window.history.pushState(guardState, "", window.location.href);

    const handlePopState = () => {
      if (allowBackNavigationRef.current) {
        return;
      }

      setShowBackExitModal(true);
      window.history.pushState(guardState, "", window.location.href);
    };

    window.addEventListener("popstate", handlePopState);
    return () => {
      window.removeEventListener("popstate", handlePopState);
    };
  }, [hasActiveTextSession]);

  useEffect(() => {
    if (!hasActiveTextSession || !session?.id) {
      return;
    }

    const sendHeartbeat = async () => {
      try {
        await api.touchSessionActivity(session.id);
      } catch {
        return;
      }
    };

    void sendHeartbeat();
    const intervalID = window.setInterval(() => {
      void sendHeartbeat();
    }, 25000);

    return () => {
      window.clearInterval(intervalID);
    };
  }, [hasActiveTextSession, session?.id]);

  const navigateBackWithoutPrompt = useCallback(() => {
    if (typeof window === "undefined") {
      return;
    }

    allowBackNavigationRef.current = true;
    setShowBackExitModal(false);
    window.history.back();

    window.setTimeout(() => {
      allowBackNavigationRef.current = false;
    }, 300);
  }, []);

  async function handleBackEndSession(): Promise<void> {
    if (!session || backExitAction !== "idle") {
      return;
    }

    setBackExitAction("ending");

    const completed = sessionCompleted ? true : await completeSession();
    if (!completed) {
      setBackExitAction("idle");
      return;
    }

    resetInterviewFlow();
    navigateBackWithoutPrompt();
  }

  async function handleStartInterview(payload: {
    jobDescription: string;
    interviewMode: InterviewMode;
    interviewLanguage: InterviewLanguage;
    interviewDifficulty: InterviewDifficulty;
    targetRole: string;
    targetCompany: string;
  }): Promise<boolean> {
    const isVoiceInterview = payload.interviewMode === "voice";
    setStartInterviewMode(payload.interviewMode);
    setConnectingDotCount(1);
    setShowVoiceStartModal(true);

    const initialized = await initializeInterview(payload);
    if (initialized && isVoiceInterview) {
      router.push("/practice/voice/call");
    }

    if (!initialized || !isVoiceInterview) {
      setShowVoiceStartModal(false);
      setConnectingDotCount(1);
    }

    return initialized;
  }

  return (
    <AppShell title={pickLocaleText(locale, "Latihan Interview", "Interview Practice")} subtitle={pickLocaleText(locale, "Pilih mode, bahasa, dan tingkat kesulitan sebelum memulai.", "Choose mode, language, and difficulty before starting.")}>
      <div className="space-y-6">
        <Card className="space-y-4 p-6">
          <h2 className="text-base font-semibold text-white">{pickLocaleText(locale, "Pengaturan interview", "Interview setup")}</h2>
          <p className="text-sm text-white/40">{pickLocaleText(locale, "Pilih mode, bahasa, dan tingkat kesulitan sebelum memulai interview.", "Select mode, language, and difficulty before starting interview.")}</p>
          <SetupForm onStart={handleStartInterview} loading={loading} locale={locale} />
          {error && <p className="text-sm text-red-300">{error}</p>}
        </Card>

        {hasActiveVoiceSession && (
          <div className="rounded-[20px] bg-linear-to-r from-purple-500/35 via-cyan-500/30 to-purple-500/35 p-px">
            <Card className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between">
              <div>
                <h3 className="text-white text-base font-semibold">{pickLocaleText(locale, "Sesi interview suara sedang aktif", "Voice interview session is active")}</h3>
                <p className="mt-1 text-sm text-white/45">
                  {pickLocaleText(locale, "Lanjutkan sesi interview suara Anda bersama interviewer AI dan transkrip langsung.", "Continue your ongoing voice interview with AI interviewer and live transcript.")}
                </p>
              </div>

              <Button type="button" onClick={() => router.push("/practice/voice/call")}>
                {pickLocaleText(locale, "Lanjutkan sesi suara", "Continue voice session")}
                <ChevronRight className="ml-2 h-4 w-4" />
              </Button>
            </Card>
          </div>
        )}

        {currentQuestion && session?.interview_mode === "text" && (
          <>
            <div className="flex flex-col md:flex-row md:items-center justify-between gap-4">
              <div>
                <h2 className="text-2xl text-white tracking-tight">Interview Session</h2>
                <p className="text-white/40 mt-1">{pickLocaleText(locale, "Pertanyaan", "Question")} {currentIndex + 1} {pickLocaleText(locale, "dari", "of")} {questions.length}</p>
              </div>
              <div className="flex items-center gap-1.5">
                {questions.map((item, index) => (
                  <div
                    key={item.id}
                    className={`w-8 h-1.5 rounded-full transition-all ${
                      index === currentIndex
                        ? "bg-gradient-to-r from-purple-500 to-cyan-400"
                        : index < currentIndex
                          ? "bg-purple-500/30"
                          : "bg-white/[0.08]"
                    }`}
                  />
                ))}
              </div>
            </div>

            <div className="grid lg:grid-cols-5 gap-6">
              <div className="lg:col-span-3 space-y-5">
                <div className="rounded-[20px] bg-gradient-to-r from-purple-500/35 via-cyan-500/30 to-purple-500/35 p-[1px]">
                  <Card className="p-6">
                    <div className="flex items-start gap-4">
                      <div className="w-10 h-10 rounded-xl bg-gradient-to-br from-purple-500 to-blue-600 flex items-center justify-center shrink-0">
                        <Bot className="w-5 h-5 text-white" />
                      </div>
                      <div className="flex-1">
                        <div className="flex items-center gap-2 mb-3">
                          <span className="text-purple-400 text-sm">{pickLocaleText(locale, "Interviewer AI", "AI Interviewer")}</span>
                          <span className="px-2 py-0.5 rounded-full bg-white/[0.06] text-white/40 text-xs capitalize">
                            {currentQuestion.type}
                          </span>
                          <span className="px-2 py-0.5 rounded-full bg-cyan-500/10 text-cyan-300 text-xs capitalize">
                            {session.interview_difficulty}
                          </span>
                        </div>
                        <p className="text-white/85 leading-relaxed text-sm">{currentQuestion.question}</p>
                      </div>
                    </div>
                  </Card>
                </div>

                <Card className="p-6">
                  <div className="flex items-center gap-3 mb-4">
                    <div className="w-10 h-10 rounded-xl bg-cyan-500/10 flex items-center justify-center">
                      <User className="w-5 h-5 text-cyan-400" />
                    </div>
                    <span className="text-cyan-400 text-sm">{pickLocaleText(locale, "Jawaban Anda", "Your Answer")}</span>
                  </div>

                  <TextArea
                    value={answer}
                    onChange={(event) => setAnswer(event.target.value)}
                    placeholder={pickLocaleText(locale, "Ketik jawaban Anda di sini...", "Type your answer here...")}
                    className="min-h-[200px]"
                  />

                  <div className="flex items-center justify-between mt-4">
                    <p className="text-white/20 text-xs">{answer.length} {pickLocaleText(locale, "karakter", "characters")}</p>
                    <div className="flex flex-wrap gap-2 justify-end">
                      <Button onClick={() => void submitCurrentAnswer()} disabled={loading || !answer.trim()}>
                        <Send className="mr-2 h-4 w-4" />
                        {loading ? pickLocaleText(locale, "Mengirim...", "Submitting...") : pickLocaleText(locale, "Kirim Jawaban", "Submit Answer")}
                      </Button>
                      <Button variant="secondary" onClick={goToNextQuestion} disabled={currentIndex >= questions.length - 1}>
                        {pickLocaleText(locale, "Pertanyaan Berikutnya", "Next Question")}
                        <ChevronRight className="ml-2 h-4 w-4" />
                      </Button>
                      {isLastQuestion && !sessionCompleted && (
                        <Button variant="secondary" onClick={() => void completeSession()} disabled={loading}>
                          <CheckCircle className="mr-2 h-4 w-4" />
                          {pickLocaleText(locale, "Selesaikan Sesi", "Finish Session")}
                        </Button>
                      )}
                    </div>
                  </div>
                </Card>
              </div>

              <div className="lg:col-span-2">
                {feedback ? (
                  <div className="space-y-4">
                    <FeedbackPanel
                      score={feedback.score}
                      strengths={feedback.strengths}
                      weaknesses={feedback.weaknesses}
                      improvements={feedback.improvements}
                      starFeedback={feedback.star_feedback}
                    />
                    <Card className="p-5">
                      <div className="flex items-center gap-2 mb-3">
                        <Lightbulb className="w-4 h-4 text-yellow-400" />
                        <h4 className="text-white text-sm">{pickLocaleText(locale, "Ringkasan Skor", "Score Summary")}</h4>
                      </div>
                      <p className="text-white/70 text-sm">{pickLocaleText(locale, "Skor terbaru", "Latest score")}: {lastScore}/100</p>
                    </Card>
                  </div>
                ) : loading ? (
                  <div className="flex flex-col items-center justify-center min-h-[320px]">
                    <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-purple-500/20 to-cyan-500/20 flex items-center justify-center mb-4 animate-pulse">
                      <Sparkles className="w-6 h-6 text-purple-400" />
                    </div>
                    <p className="text-white/40 text-sm">{pickLocaleText(locale, "Menganalisis jawaban Anda...", "Analyzing your answer...")}</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center min-h-[320px]">
                    <div className="w-14 h-14 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-4">
                      <MessageSquare className="w-6 h-6 text-white/15" />
                    </div>
                    <p className="text-white/30 text-sm text-center">
                      {pickLocaleText(locale, "Kirim jawaban untuk mendapatkan", "Submit your answer to receive")}
                      <br />
                      {pickLocaleText(locale, "feedback dan skoring berbasis AI", "AI-powered feedback and scoring")}
                    </p>
                    {error && (
                      <div className="mt-4 inline-flex items-center gap-2 rounded-full border border-red-400/25 bg-red-500/10 px-3 py-1 text-xs text-red-300">
                        <AlertCircle className="w-3.5 h-3.5" />
                        {error}
                      </div>
                    )}
                  </div>
                )}
              </div>
            </div>
          </>
        )}

        {showBackExitModal && (
          <div className="fixed inset-0 z-60 flex items-center justify-center bg-slate-950/75 px-4 backdrop-blur-sm">
            <Card className="w-full max-w-md border border-white/15 bg-[rgba(10,15,26,0.96)] p-6">
              <h3 className="text-base font-semibold text-white">
                {pickLocaleText(locale, "Keluar dari sesi interview?", "Leave interview session?")}
              </h3>
              <p className="mt-2 text-sm text-white/70">
                {pickLocaleText(
                  locale,
                  "Pilih `Akhiri sesi` untuk menutup interview sekarang, atau `Lanjutkan nanti` untuk kembali tanpa menyelesaikan sesi.",
                  "Choose `End session` to finish now, or `Continue later` to go back without completing the session.",
                )}
              </p>

              <div className="mt-5 flex flex-wrap justify-end gap-2">
                <Button
                  type="button"
                  variant="secondary"
                  onClick={() => setShowBackExitModal(false)}
                  disabled={backExitAction !== "idle"}
                >
                  {pickLocaleText(locale, "Tetap di sesi", "Stay in session")}
                </Button>
                <Button
                  type="button"
                  variant="secondary"
                  onClick={navigateBackWithoutPrompt}
                  disabled={backExitAction !== "idle"}
                >
                  {pickLocaleText(locale, "Lanjutkan nanti", "Continue later")}
                </Button>
                <Button
                  type="button"
                  onClick={() => void handleBackEndSession()}
                  className="border-red-400/50 text-red-200 hover:border-red-300/70"
                  disabled={backExitAction !== "idle"}
                >
                  {backExitAction === "ending"
                    ? pickLocaleText(locale, "Mengakhiri sesi...", "Ending session...")
                    : pickLocaleText(locale, "Akhiri sesi", "End session")}
                </Button>
              </div>
            </Card>
          </div>
        )}

        {showVoiceStartModal && (
          <div
            className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/75 px-4 backdrop-blur-sm transition-opacity duration-300 ease-out opacity-100"
          >
            <div
              className="relative w-full max-w-md overflow-hidden rounded-[22px] border border-white/10 bg-slate-900/95 p-6 shadow-[0_24px_80px_rgba(0,0,0,0.45)] transition-all duration-300 ease-out translate-y-0 scale-100 opacity-100"
            >
              <div aria-hidden className="pointer-events-none absolute -top-16 left-1/2 h-40 w-40 -translate-x-1/2 rounded-full bg-cyan-400/20 blur-3xl animate-pulse" />
              <div className="relative mx-auto mb-4 flex h-20 w-20 items-center justify-center">
                <div className="absolute inset-0 rounded-full border border-cyan-100/20 animate-pulse" />
                <div className="absolute inset-2 rounded-full border border-cyan-200/20" />
                <div className="h-10 w-10 animate-spin rounded-full border-[3px] border-cyan-100/25 border-t-cyan-100 drop-shadow-[0_0_18px_rgba(103,232,249,0.6)]" />
              </div>
              <h3 className="relative text-center text-base font-semibold text-white">
                {startInterviewMode === "voice"
                  ? pickLocaleText(locale, "Menyiapkan interview suara", "Preparing voice interview")
                  : pickLocaleText(locale, "Menyiapkan interview teks", "Preparing text interview")}
              </h3>
              <p className="relative mt-2 text-center text-sm text-white/60">
                {startInterviewMode === "voice"
                  ? pickLocaleText(locale, "Mohon tunggu sebentar, kami sedang menghubungkan Anda ke interviewer AI.", "Please wait a moment while we connect you to the AI interviewer.")
                  : pickLocaleText(locale, "Mohon tunggu sebentar, kami sedang menyiapkan pertanyaan interview untuk Anda.", "Please wait a moment while we prepare interview questions for you.")}
              </p>
              <p className="relative mt-1 text-center text-xs font-medium tracking-wide text-cyan-200/90">
                {startInterviewMode === "voice"
                  ? pickLocaleText(locale, "Menghubungkan", "Connecting")
                  : pickLocaleText(locale, "Menyiapkan", "Preparing")}
                <span className="inline-block min-w-6 text-left">{".".repeat(connectingDotCount)}</span>
              </p>
            </div>
          </div>
        )}
      </div>
    </AppShell>
  );
}

function SetupForm({
  onStart,
  loading,
  locale,
}: {
  onStart: (payload: {
    jobDescription: string;
    interviewMode: InterviewMode;
    interviewLanguage: InterviewLanguage;
    interviewDifficulty: InterviewDifficulty;
    targetRole: string;
    targetCompany: string;
  }) => Promise<boolean>;
  loading: boolean;
  locale: Locale;
}) {
  const [jobDescription, setJobDescription] = useState("");
  const [targetRole, setTargetRole] = useState("");
  const [targetCompany, setTargetCompany] = useState("");
  const [interviewMode, setInterviewMode] = useState<InterviewMode>("text");
  const [interviewLanguage, setInterviewLanguage] = useState<InterviewLanguage>("en");
  const [interviewDifficulty, setInterviewDifficulty] = useState<InterviewDifficulty>("medium");

  return (
    <div className="space-y-3">
      <div className="space-y-2">
        <p className="text-xs uppercase tracking-wide text-white/45">{pickLocaleText(locale, "Mode interview", "Interview mode")}</p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant={interviewMode === "text" ? "primary" : "secondary"} onClick={() => setInterviewMode("text")}>
            {pickLocaleText(locale, "Interview teks", "Text interview")}
          </Button>
          <Button type="button" variant={interviewMode === "voice" ? "primary" : "secondary"} onClick={() => setInterviewMode("voice")}>
            {pickLocaleText(locale, "Interview suara", "Voice interview")}
          </Button>
        </div>
        {interviewMode === "voice" && (
          <p className="text-xs text-cyan-200/85">
            {pickLocaleText(locale, "Mode suara menggunakan interviewer AI realtime, transkrip otomatis, dan skoring backend per jawaban.", "Voice mode uses realtime AI interviewer, automatic transcript, and backend scoring per answer.")}
          </p>
        )}
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <Input value={targetRole} onChange={(event) => setTargetRole(event.target.value)} placeholder={pickLocaleText(locale, "Target role (opsional)", "Target role (optional)")} />
        <Input value={targetCompany} onChange={(event) => setTargetCompany(event.target.value)} placeholder={pickLocaleText(locale, "Target perusahaan (opsional)", "Target company (optional)")} />
      </div>

      <TextArea
        value={jobDescription}
        onChange={(event) => setJobDescription(event.target.value)}
        placeholder={pickLocaleText(locale, "Tempel job description...", "Paste job description...")}
        className="min-h-28"
      />

      <div className="space-y-2">
        <p className="text-xs uppercase tracking-wide text-white/45">{pickLocaleText(locale, "Bahasa interview", "Interview language")}</p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant={interviewLanguage === "id" ? "primary" : "secondary"} onClick={() => setInterviewLanguage("id")}>
            Bahasa Indonesia
          </Button>
          <Button type="button" variant={interviewLanguage === "en" ? "primary" : "secondary"} onClick={() => setInterviewLanguage("en")}>
            English
          </Button>
        </div>
      </div>

      <div className="space-y-2">
        <p className="text-xs uppercase tracking-wide text-white/45">{pickLocaleText(locale, "Tingkat kesulitan interview", "Interview difficulty")}</p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant={interviewDifficulty === "easy" ? "primary" : "secondary"} onClick={() => setInterviewDifficulty("easy")}>
            Easy
          </Button>
          <Button type="button" variant={interviewDifficulty === "medium" ? "primary" : "secondary"} onClick={() => setInterviewDifficulty("medium")}>
            Medium
          </Button>
          <Button type="button" variant={interviewDifficulty === "hard" ? "primary" : "secondary"} onClick={() => setInterviewDifficulty("hard")}>
            Hard
          </Button>
        </div>
      </div>

      <Button
        onClick={() =>
          void onStart({
            jobDescription,
            interviewMode,
            interviewLanguage,
            interviewDifficulty,
            targetRole,
            targetCompany,
          })
        }
        disabled={loading || !jobDescription.trim()}
      >
        {loading ? pickLocaleText(locale, "Menyiapkan interview...", "Preparing interview...") : pickLocaleText(locale, "Mulai interview", "Start interview")}
        {!loading && <ArrowRight className="ml-2 h-4 w-4" />}
      </Button>
    </div>
  );
}

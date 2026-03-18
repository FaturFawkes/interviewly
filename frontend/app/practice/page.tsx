"use client";

import { useState } from "react";
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
import { FeedbackPanel } from "@/components/interview/FeedbackPanel";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { Input, TextArea } from "@/components/ui/Input";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";
import type { InterviewDifficulty, InterviewLanguage, InterviewMode } from "@/lib/api/types";

export default function PracticePage() {
  const router = useRouter();
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
    goToNextQuestion,
    sessionCompleted,
  } = useInterviewFlow();

  const isLastQuestion = currentIndex >= questions.length - 1;
  const hasActiveVoiceSession = Boolean(
    currentQuestion
    && session?.interview_mode === "voice"
    && session?.status === "active"
    && !sessionCompleted,
  );

  async function handleStartInterview(payload: {
    jobDescription: string;
    interviewMode: InterviewMode;
    interviewLanguage: InterviewLanguage;
    interviewDifficulty: InterviewDifficulty;
    targetRole: string;
    targetCompany: string;
  }): Promise<boolean> {
    const initialized = await initializeInterview(payload);
    if (initialized && payload.interviewMode === "voice") {
      router.push("/practice/voice/call");
    }
    return initialized;
  }

  return (
    <AppShell title="Interview Practice" subtitle="Choose mode, language, and difficulty before starting.">
      <div className="space-y-6">
        <Card className="space-y-4 p-6">
          <h2 className="text-base font-semibold text-white">Interview setup</h2>
          <p className="text-sm text-white/40">Select mode, language, and difficulty before starting interview.</p>
          <SetupForm onStart={handleStartInterview} loading={loading} />
          {error && <p className="text-sm text-red-300">{error}</p>}
        </Card>

        {hasActiveVoiceSession && (
          <div className="rounded-[20px] bg-linear-to-r from-purple-500/35 via-cyan-500/30 to-purple-500/35 p-px">
            <Card className="flex flex-col gap-4 p-6 md:flex-row md:items-center md:justify-between">
              <div>
                <h3 className="text-white text-base font-semibold">Voice interview session is active</h3>
                <p className="mt-1 text-sm text-white/45">
                  Continue your ongoing voice interview with AI interviewer and live transcript.
                </p>
              </div>

              <Button type="button" onClick={() => router.push("/practice/voice/call")}>
                Continue voice session
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
                <p className="text-white/40 mt-1">Question {currentIndex + 1} of {questions.length}</p>
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
                          <span className="text-purple-400 text-sm">AI Interviewer</span>
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
                    <span className="text-cyan-400 text-sm">Your Answer</span>
                  </div>

                  <TextArea
                    value={answer}
                    onChange={(event) => setAnswer(event.target.value)}
                    placeholder="Type your answer here..."
                    className="min-h-[200px]"
                  />

                  <div className="flex items-center justify-between mt-4">
                    <p className="text-white/20 text-xs">{answer.length} characters</p>
                    <div className="flex flex-wrap gap-2 justify-end">
                      <Button onClick={() => void submitCurrentAnswer()} disabled={loading || !answer.trim()}>
                        <Send className="mr-2 h-4 w-4" />
                        {loading ? "Submitting..." : "Submit Answer"}
                      </Button>
                      <Button variant="secondary" onClick={goToNextQuestion} disabled={currentIndex >= questions.length - 1}>
                        Next Question
                        <ChevronRight className="ml-2 h-4 w-4" />
                      </Button>
                      {isLastQuestion && !sessionCompleted && (
                        <Button variant="secondary" onClick={() => void completeSession()} disabled={loading}>
                          <CheckCircle className="mr-2 h-4 w-4" />
                          Finish Session
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
                        <h4 className="text-white text-sm">Score Summary</h4>
                      </div>
                      <p className="text-white/70 text-sm">Latest score: {lastScore}/100</p>
                    </Card>
                  </div>
                ) : loading ? (
                  <div className="flex flex-col items-center justify-center min-h-[320px]">
                    <div className="w-14 h-14 rounded-2xl bg-gradient-to-br from-purple-500/20 to-cyan-500/20 flex items-center justify-center mb-4 animate-pulse">
                      <Sparkles className="w-6 h-6 text-purple-400" />
                    </div>
                    <p className="text-white/40 text-sm">Analyzing your answer...</p>
                  </div>
                ) : (
                  <div className="flex flex-col items-center justify-center min-h-[320px]">
                    <div className="w-14 h-14 rounded-2xl bg-white/[0.03] border border-white/[0.06] flex items-center justify-center mb-4">
                      <MessageSquare className="w-6 h-6 text-white/15" />
                    </div>
                    <p className="text-white/30 text-sm text-center">
                      Submit your answer to receive
                      <br />
                      AI-powered feedback and scoring
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
      </div>
    </AppShell>
  );
}

function SetupForm({
  onStart,
  loading,
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
        <p className="text-xs uppercase tracking-wide text-white/45">Interview mode</p>
        <div className="flex flex-wrap gap-2">
          <Button type="button" variant={interviewMode === "text" ? "primary" : "secondary"} onClick={() => setInterviewMode("text")}>
            Text interview
          </Button>
          <Button type="button" variant={interviewMode === "voice" ? "primary" : "secondary"} onClick={() => setInterviewMode("voice")}>
            Voice interview
          </Button>
        </div>
        {interviewMode === "voice" && (
          <p className="text-xs text-cyan-200/85">
            Voice mode uses realtime AI interviewer, automatic transcript, and backend scoring per answer.
          </p>
        )}
      </div>

      <div className="grid gap-3 md:grid-cols-2">
        <Input value={targetRole} onChange={(event) => setTargetRole(event.target.value)} placeholder="Target role (optional)" />
        <Input value={targetCompany} onChange={(event) => setTargetCompany(event.target.value)} placeholder="Target company (optional)" />
      </div>

      <TextArea
        value={jobDescription}
        onChange={(event) => setJobDescription(event.target.value)}
        placeholder="Paste job description..."
        className="min-h-28"
      />

      <div className="space-y-2">
        <p className="text-xs uppercase tracking-wide text-white/45">Interview language</p>
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
        <p className="text-xs uppercase tracking-wide text-white/45">Interview difficulty</p>
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
        {loading ? "Preparing interview..." : "Start interview"}
        {!loading && <ArrowRight className="ml-2 h-4 w-4" />}
      </Button>
    </div>
  );
}

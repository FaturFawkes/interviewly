"use client";

import { useEffect, useMemo, useState } from "react";

import { api } from "@/lib/api/endpoints";
import type {
  FeedbackRecord,
  InterviewDifficulty,
  InterviewLanguage,
  InterviewMode,
  PracticeSession,
  StoredQuestion,
} from "@/lib/api/types";

type SetupPayload = {
  jobDescription: string;
  interviewMode?: InterviewMode;
  interviewLanguage?: InterviewLanguage;
  interviewDifficulty?: InterviewDifficulty;
  targetRole?: string;
  targetCompany?: string;
};

type InterviewSetupContext = {
  jobDescription: string;
  interviewMode: InterviewMode;
  interviewLanguage: InterviewLanguage;
  interviewDifficulty: InterviewDifficulty;
  targetRole: string;
  targetCompany: string;
};

type UseInterviewFlowOptions = {
  storageKey?: string;
};

type PersistedFlowState = {
  questions: StoredQuestion[];
  session: PracticeSession | null;
  currentIndex: number;
  answer: string;
  feedback: FeedbackRecord | null;
  lastScore: number;
  timerSeconds: number;
  setupContext: InterviewSetupContext | null;
};

export function useInterviewFlow(options?: UseInterviewFlowOptions) {
  const storageKey = options?.storageKey ?? "interview-flow";
  const [questions, setQuestions] = useState<StoredQuestion[]>([]);
  const [session, setSession] = useState<PracticeSession | null>(null);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answer, setAnswer] = useState("");
  const [feedback, setFeedback] = useState<FeedbackRecord | null>(null);
  const [lastScore, setLastScore] = useState(0);
  const [setupContext, setSetupContext] = useState<InterviewSetupContext | null>(null);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [timerSeconds, setTimerSeconds] = useState(0);
  const sessionCompleted = session?.status === "completed";

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const raw = window.sessionStorage.getItem(storageKey);
    if (!raw) {
      return;
    }

    try {
      const parsed = JSON.parse(raw) as PersistedFlowState;
      if (Array.isArray(parsed.questions)) {
        setQuestions(parsed.questions);
      }
      setSession(parsed.session ?? null);
      setCurrentIndex(typeof parsed.currentIndex === "number" ? parsed.currentIndex : 0);
      setAnswer(typeof parsed.answer === "string" ? parsed.answer : "");
      setFeedback(parsed.feedback ?? null);
      setLastScore(typeof parsed.lastScore === "number" ? parsed.lastScore : 0);
      setTimerSeconds(typeof parsed.timerSeconds === "number" ? parsed.timerSeconds : 0);
      setSetupContext(parsed.setupContext ?? null);
    } catch {
      window.sessionStorage.removeItem(storageKey);
    }
  }, [storageKey]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    const payload: PersistedFlowState = {
      questions,
      session,
      currentIndex,
      answer,
      feedback,
      lastScore,
      timerSeconds,
      setupContext,
    };

    window.sessionStorage.setItem(storageKey, JSON.stringify(payload));
  }, [answer, currentIndex, feedback, lastScore, questions, session, setupContext, storageKey, timerSeconds]);

  const currentQuestion = useMemo(() => questions[currentIndex] ?? null, [questions, currentIndex]);

  useEffect(() => {
    if (!currentQuestion) {
      return;
    }

    const interval = setInterval(() => {
      setTimerSeconds((prev) => prev + 1);
    }, 1000);

    return () => clearInterval(interval);
  }, [currentQuestion]);

  async function initializeInterview({
    jobDescription,
    interviewMode = "text",
    interviewLanguage = "en",
    interviewDifficulty = "medium",
    targetRole,
    targetCompany,
  }: SetupPayload): Promise<boolean> {
    setLoading(true);
    setError(null);
    setFeedback(null);
    setTimerSeconds(0);

    try {
      const generated = await api.generateQuestions(
        "",
        jobDescription,
        interviewLanguage,
        interviewMode,
        interviewDifficulty,
      );

      if (!generated.questions?.length || !generated.resume_id || !generated.job_parse_id) {
        throw new Error("No generated questions were returned by the API.");
      }

      const createdSession = await api.startInterviewSession(
        generated.resume_id,
        generated.job_parse_id,
        generated.questions.map((q) => q.id),
        {
          interview_mode: interviewMode,
          interview_language: interviewLanguage,
          interview_difficulty: interviewDifficulty,
          target_role: targetRole,
          target_company: targetCompany,
        },
      );

      setQuestions(generated.questions);
      setSession(createdSession);
      setCurrentIndex(0);
      setAnswer("");
      setFeedback(null);
      setLastScore(0);
      setSetupContext({
        jobDescription,
        interviewMode,
        interviewLanguage,
        interviewDifficulty,
        targetRole: targetRole ?? "",
        targetCompany: targetCompany ?? "",
      });
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to initialize interview.");
      return false;
    } finally {
      setLoading(false);
    }
  }

  async function submitCurrentAnswer(): Promise<void> {
    if (!session || !currentQuestion || !answer.trim()) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      await api.submitInterviewAnswer(session.id, currentQuestion.id, answer);
      const response = await api.generateFeedback(
        session.id,
        currentQuestion.id,
        currentQuestion.question,
        answer,
      );

      setFeedback(response);
      setLastScore(response.score);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit answer.");
    } finally {
      setLoading(false);
    }
  }

  async function submitAnswerText(answerText: string): Promise<boolean> {
    if (!session || !currentQuestion || !answerText.trim()) {
      return false;
    }

    setLoading(true);
    setError(null);

    try {
      await api.submitInterviewAnswer(session.id, currentQuestion.id, answerText);
      const response = await api.generateFeedback(
        session.id,
        currentQuestion.id,
        currentQuestion.question,
        answerText,
      );

      setFeedback(response);
      setLastScore(response.score);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit answer.");
      return false;
    } finally {
      setLoading(false);
    }
  }

  async function submitVoiceAnswer(answerText: string): Promise<boolean> {
    if (!session || !currentQuestion || !answerText.trim()) {
      return false;
    }

    setLoading(true);
    setError(null);

    try {
      await api.submitInterviewAnswer(session.id, currentQuestion.id, answerText);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to submit answer.");
      return false;
    } finally {
      setLoading(false);
    }
  }

  async function completeSession(): Promise<boolean> {
    if (!session) {
      return false;
    }

    setLoading(true);
    setError(null);

    try {
      const completed = await api.completeInterviewSession(session.id);
      setSession(completed);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to complete session.");
      return false;
    } finally {
      setLoading(false);
    }
  }

  function resetInterviewFlow(): void {
    setQuestions([]);
    setSession(null);
    setCurrentIndex(0);
    setAnswer("");
    setFeedback(null);
    setLastScore(0);
    setTimerSeconds(0);
    setSetupContext(null);
    setError(null);

    if (typeof window !== "undefined") {
      window.sessionStorage.removeItem(storageKey);
    }
  }

  function goToNextQuestion(): void {
    if (currentIndex >= questions.length - 1) {
      return;
    }

    setCurrentIndex((prev) => prev + 1);
    setAnswer("");
    setFeedback(null);
    setTimerSeconds(0);
  }

  return {
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
    timerSeconds,
    setupContext,
    initializeInterview,
    submitCurrentAnswer,
    submitAnswerText,
    submitVoiceAnswer,
    completeSession,
    resetInterviewFlow,
    goToNextQuestion,
    sessionCompleted,
  };
}

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

type UseInterviewFlowOptions = {
  storageKey?: string;
};

type PersistedInterviewFlowState = {
  session: PracticeSession | null;
  questions: StoredQuestion[];
  currentIndex: number;
  answer: string;
  feedback: FeedbackRecord | null;
  lastScore: number;
  sessionCompleted: boolean;
  timerSeconds: number;
};

export function useInterviewFlow(options: UseInterviewFlowOptions = {}) {
  const storageKey = options.storageKey ?? "interview-flow";

  const [questions, setQuestions] = useState<StoredQuestion[]>([]);
  const [session, setSession] = useState<PracticeSession | null>(null);
  const [currentIndex, setCurrentIndex] = useState(0);
  const [answer, setAnswer] = useState("");
  const [feedback, setFeedback] = useState<FeedbackRecord | null>(null);
  const [lastScore, setLastScore] = useState(0);
  const [sessionCompleted, setSessionCompleted] = useState(false);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [timerSeconds, setTimerSeconds] = useState(0);
  const [hydrated, setHydrated] = useState(false);

  const currentQuestion = useMemo(() => questions[currentIndex] ?? null, [questions, currentIndex]);

  useEffect(() => {
    if (typeof window === "undefined") {
      return;
    }

    try {
      const raw = window.sessionStorage.getItem(storageKey);
      if (raw) {
        const parsed = JSON.parse(raw) as Partial<PersistedInterviewFlowState>;

        if (Array.isArray(parsed.questions)) {
          setQuestions(parsed.questions);
        }

        if (parsed.session && typeof parsed.session === "object") {
          setSession(parsed.session as PracticeSession);
        }

        if (typeof parsed.currentIndex === "number" && Number.isFinite(parsed.currentIndex)) {
          setCurrentIndex(parsed.currentIndex);
        }

        if (typeof parsed.answer === "string") {
          setAnswer(parsed.answer);
        }

        if (parsed.feedback && typeof parsed.feedback === "object") {
          setFeedback(parsed.feedback as FeedbackRecord);
        }

        if (typeof parsed.lastScore === "number" && Number.isFinite(parsed.lastScore)) {
          setLastScore(parsed.lastScore);
        }

        if (typeof parsed.sessionCompleted === "boolean") {
          setSessionCompleted(parsed.sessionCompleted);
        }

        if (typeof parsed.timerSeconds === "number" && Number.isFinite(parsed.timerSeconds)) {
          setTimerSeconds(parsed.timerSeconds);
        }
      }
    } catch {
      window.sessionStorage.removeItem(storageKey);
    } finally {
      setHydrated(true);
    }
  }, [storageKey]);

  useEffect(() => {
    if (!hydrated || typeof window === "undefined") {
      return;
    }

    const payload: PersistedInterviewFlowState = {
      session,
      questions,
      currentIndex,
      answer,
      feedback,
      lastScore,
      sessionCompleted,
      timerSeconds,
    };

    window.sessionStorage.setItem(storageKey, JSON.stringify(payload));
  }, [answer, currentIndex, feedback, hydrated, lastScore, questions, session, sessionCompleted, storageKey, timerSeconds]);

  useEffect(() => {
    if (questions.length === 0 && currentIndex !== 0) {
      setCurrentIndex(0);
      return;
    }

    if (questions.length > 0 && currentIndex > questions.length - 1) {
      setCurrentIndex(questions.length - 1);
    }
  }, [currentIndex, questions]);

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
      const normalizedLanguage = normalizeInterviewLanguage(interviewLanguage);
      const normalizedMode = normalizeInterviewMode(interviewMode);
      const normalizedDifficulty = normalizeInterviewDifficulty(interviewDifficulty);
      const generated = await api.generateQuestions(
        "",
        jobDescription,
        normalizedLanguage,
        normalizedMode,
        normalizedDifficulty,
      );

      if (!generated.questions?.length || !generated.resume_id || !generated.job_parse_id) {
        throw new Error("No generated questions were returned by the API.");
      }

      const createdSession = await api.startInterviewSession(
        generated.resume_id,
        generated.job_parse_id,
        generated.questions.map((q) => q.id),
        {
          interview_mode: normalizedMode,
          interview_language: normalizedLanguage,
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
      setSessionCompleted(false);
      return true;
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to initialize interview.");
      return false;
    } finally {
      setLoading(false);
    }
  }

  async function submitCurrentAnswer(): Promise<boolean> {
    return submitAnswerText(answer);
  }

  async function submitAnswerText(answerText: string): Promise<boolean> {
    const normalizedAnswer = answerText.trim();

    if (!session || !currentQuestion || !normalizedAnswer) {
      return false;
    }

    setLoading(true);
    setError(null);
    setAnswer(normalizedAnswer);

    try {
      await api.submitInterviewAnswer(session.id, currentQuestion.id, normalizedAnswer);
      const interviewLanguage = normalizeInterviewLanguage(session.interview_language);
      const response = await api.generateFeedback(
        session.id,
        currentQuestion.id,
        currentQuestion.question,
        normalizedAnswer,
        interviewLanguage,
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

  async function completeSession(): Promise<void> {
    if (!session || sessionCompleted) {
      return;
    }

    setLoading(true);
    setError(null);

    try {
      const completed = await api.completeInterviewSession(session.id);
      setSession(completed);
      setLastScore(completed.score);
      setSessionCompleted(true);
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to complete interview session.");
    } finally {
      setLoading(false);
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
    sessionCompleted,
    loading,
    error,
    timerSeconds,
    initializeInterview,
    submitCurrentAnswer,
    submitAnswerText,
    completeSession,
    goToNextQuestion,
  };
}

function normalizeInterviewLanguage(language?: InterviewLanguage): InterviewLanguage {
  return language === "id" ? "id" : "en";
}

function normalizeInterviewMode(mode?: InterviewMode): InterviewMode {
  return mode === "voice" ? "voice" : "text";
}

function normalizeInterviewDifficulty(difficulty?: InterviewDifficulty): InterviewDifficulty {
  switch (difficulty) {
    case "easy":
      return "easy";
    case "hard":
      return "hard";
    default:
      return "medium";
  }
}

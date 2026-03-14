"use client";

import { useEffect, useMemo, useState } from "react";

import { api } from "@/lib/api/endpoints";
import type { FeedbackRecord, PracticeSession, StoredQuestion } from "@/lib/api/types";

type SetupPayload = {
  jobDescription: string;
  interviewMode?: "text" | "voice";
  targetRole?: string;
  targetCompany?: string;
};

export function useInterviewFlow() {
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

  async function initializeInterview({ jobDescription, interviewMode = "text", targetRole, targetCompany }: SetupPayload): Promise<void> {
    setLoading(true);
    setError(null);
    setFeedback(null);
    setTimerSeconds(0);

    try {
      const generated = await api.generateQuestions("", jobDescription);

      if (!generated.questions?.length || !generated.resume_id || !generated.job_parse_id) {
        throw new Error("No generated questions were returned by the API.");
      }

      const createdSession = await api.startInterviewSession(
        generated.resume_id,
        generated.job_parse_id,
        generated.questions.map((q) => q.id),
        {
          interview_mode: interviewMode,
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
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to initialize interview.");
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
    completeSession,
    goToNextQuestion,
  };
}

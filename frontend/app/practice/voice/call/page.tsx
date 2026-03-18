"use client";

import { Conversation, type Mode, type Status } from "@elevenlabs/client";
import { ArrowLeft, Mic, MicOff, PhoneOff, Sparkles } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useLanguage } from "@/components/providers/LanguageProvider";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";
import type { FeedbackRecord } from "@/lib/api/types";
import { cn } from "@/lib/utils";

type ConnectionMode = "initializing" | "agent" | "fallback";

type TranscriptItem = {
  id: string;
  role: "user" | "agent";
  message: string;
};

type PendingAnswerTurn = {
  question: string;
  answer: string;
};

const AGENT_QUESTION_PATTERN = /\?|^(what|why|how|when|where|which|who|tell me|walk me|describe|explain|can you|could you|would you|jelaskan|ceritakan|bagaimana|mengapa|kapan|siapa|apa|bisakah|dapatkah)/i;
const PUBLIC_APP_ENV = (process.env.NEXT_PUBLIC_APP_ENV ?? "").trim().toLowerCase();
const HIDE_TRANSCRIPT_IN_PRODUCTION = PUBLIC_APP_ENV === "production"
  || (PUBLIC_APP_ENV === "" && process.env.NODE_ENV === "production");
const DEFAULT_VOICE_CALL_MAX_SECONDS = 15 * 60;
const RAW_VOICE_CALL_MAX_SECONDS = Number.parseInt(process.env.NEXT_PUBLIC_VOICE_CALL_MAX_SECONDS ?? "", 10);
const VOICE_CALL_MAX_SECONDS = Number.isFinite(RAW_VOICE_CALL_MAX_SECONDS) && RAW_VOICE_CALL_MAX_SECONDS > 0
  ? RAW_VOICE_CALL_MAX_SECONDS
  : DEFAULT_VOICE_CALL_MAX_SECONDS;
const DASHBOARD_REDIRECT_DELAY_MS = 2500;
const DEFAULT_COMPLETION_POPUP_MESSAGE = "Latihan interview telah selesai dan Anda akan kembali ke halaman dashboard.";

function resolveSessionLanguage(interviewLanguage?: string): "en" | "id" {
  return interviewLanguage === "id" ? "id" : "en";
}

function normalizeTranscript(value: string): string {
  return value
    .replace(/\[[^\]]*\]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function isLikelyQuestion(value: string): boolean {
  const normalized = normalizeTranscript(value);
  if (!normalized) {
    return false;
  }

  return AGENT_QUESTION_PATTERN.test(normalized);
}

function truncateContext(value: string, maxLength: number): string {
  const normalized = normalizeTranscript(value);
  if (normalized.length <= maxLength) {
    return normalized;
  }

  return `${normalized.slice(0, maxLength)}...`;
}
export default function VoiceCallPage() {
  const router = useRouter();
  const { locale } = useLanguage();
  const {
    session,
    questions,
    currentQuestion,
    currentIndex,
    setupContext,
    feedback,
    error,
    timerSeconds,
    completeSession,
    resetInterviewFlow,
    sessionCompleted,
  } = useInterviewFlow();

  const interviewLanguageLabel = session?.interview_language === "id" ? "Bahasa Indonesia" : "English";
  const interviewDifficultyLabel = session?.interview_difficulty === "easy"
    ? "Easy"
    : session?.interview_difficulty === "hard"
      ? "Hard"
      : "Medium";
  const transcriptContainerRef = useRef<HTMLDivElement | null>(null);
  const conversationRef = useRef<Conversation | null>(null);
  const pendingAnswerQueueRef = useRef<PendingAnswerTurn[]>([]);
  const queueRunningRef = useRef(false);
  const startingConversationRef = useRef(false);
  const processedAgentEventIDsRef = useRef<Set<number>>(new Set());
  const processedUserEventIDsRef = useRef<Set<number>>(new Set());
  const recentMessageDedupRef = useRef<Map<string, number>>(new Map());
  const activeAgentQuestionRef = useRef<string>("");
  const activeUserAnswerPartsRef = useRef<string[]>([]);
  const userTurnFlushTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const sessionFinalizingRef = useRef(false);
  const timeLimitHandledRef = useRef(false);
  const warningThresholdShownRef = useRef(false);
  const usageCommittedRef = useRef(false);
  const isCallActiveRef = useRef(true);
  const dashboardRedirectTimeoutRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const questionsRef = useRef(questions);
  const evaluatedTurnsRef = useRef(Math.max(currentIndex, 0));
  const sessionCompletedRef = useRef(sessionCompleted);

  const [isCallActive, setIsCallActive] = useState(true);
  const [voiceError, setVoiceError] = useState<string | null>(null);
  const [voiceInfo, setVoiceInfo] = useState<string | null>(pickLocaleText(locale, "Menghubungkan ke interviewer AI...", "Connecting to AI interviewer..."));
  const [connectionMode, setConnectionMode] = useState<ConnectionMode>("initializing");
  const [agentStatus, setAgentStatus] = useState<Status>("disconnected");
  const [agentMode, setAgentMode] = useState<Mode>("listening");
  const [conversationID, setConversationID] = useState<string | null>(null);
  const [transcriptItems, setTranscriptItems] = useState<TranscriptItem[]>([]);
  const [isMuted, setIsMuted] = useState(false);
  const [activeQuestionLabel, setActiveQuestionLabel] = useState<string>(pickLocaleText(locale, "Interviewer AI akan segera mengajukan pertanyaan pertama.", "AI interviewer will ask the first question shortly."));
  const [latestFeedback, setLatestFeedback] = useState<FeedbackRecord | null>(feedback ?? null);
  const [evaluatedTurns, setEvaluatedTurns] = useState<number>(currentIndex);
  const [showCompletionPopup, setShowCompletionPopup] = useState(false);
  const [completionPopupMessage, setCompletionPopupMessage] = useState<string>(DEFAULT_COMPLETION_POPUP_MESSAGE);
  const [voiceLimitSeconds, setVoiceLimitSeconds] = useState<number>(VOICE_CALL_MAX_SECONDS);
  const [showBackExitModal, setShowBackExitModal] = useState(false);
  const [backExitAction, setBackExitAction] = useState<"idle" | "ending" | "pausing">("idle");
  const allowBackNavigationRef = useRef(false);
  const showTranscriptPanel = !HIDE_TRANSCRIPT_IN_PRODUCTION;
  const selectedInterviewLanguage = resolveSessionLanguage(session?.interview_language);
  const remainingCallSeconds = Math.max(voiceLimitSeconds - timerSeconds, 0);
  const callProgressPercent = Math.min(100, Math.round((timerSeconds / Math.max(voiceLimitSeconds, 1)) * 100));
  const hasActiveVoiceSession = Boolean(session?.id && session?.status === "active" && !sessionCompleted);

  useEffect(() => {
    if (!session?.id) {
      return;
    }

    let cancelled = false;

    void (async () => {
      try {
        const history = await api.getSessionHistory();
        const hasServerSession = history.sessions.some((item) => item.id === session.id);
        if (cancelled) {
          return;
        }

        if (!hasServerSession) {
          setVoiceError(pickLocaleText(locale, "Sesi interview tidak lagi tersedia di server. Silakan mulai ulang dari halaman Practice.", "Interview session is no longer available on the server. Please restart from Practice."));
          resetInterviewFlow();
          router.push("/practice");
          return;
        }
      } catch {
        return;
      }
    })();

    return () => {
      cancelled = true;
    };
  }, [locale, resetInterviewFlow, router, session?.id]);

  useEffect(() => {
    questionsRef.current = questions;
    sessionCompletedRef.current = sessionCompleted;
    evaluatedTurnsRef.current = Math.max(evaluatedTurnsRef.current, currentIndex);
    setEvaluatedTurns((previous) => Math.max(previous, evaluatedTurnsRef.current));
  }, [currentIndex, questions, sessionCompleted]);

  useEffect(() => {
    isCallActiveRef.current = isCallActive;
  }, [isCallActive]);

  useEffect(() => {
    if (!feedback) {
      return;
    }

    setLatestFeedback(feedback);
  }, [feedback]);

  useEffect(() => {
    if (!transcriptContainerRef.current) {
      return;
    }

    transcriptContainerRef.current.scrollTop = transcriptContainerRef.current.scrollHeight;
  }, [transcriptItems]);

  useEffect(() => {
    if (typeof window === "undefined" || !hasActiveVoiceSession) {
      return;
    }

    const guardState = { interviewBackGuard: "practice-voice" };
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
  }, [hasActiveVoiceSession]);

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

  const appendTranscript = useCallback((role: "user" | "agent", message: string) => {
    setTranscriptItems((previous) => {
      const entry: TranscriptItem = {
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        role,
        message,
      };

      return [...previous, entry].slice(-24);
    });
  }, []);

  const buildAgentContextInstruction = useCallback(() => {
    const languageInstruction = session?.interview_language === "id"
      ? "Use Bahasa Indonesia only for all spoken responses."
      : "Use English only for all spoken responses.";

    const targetRole = normalizeTranscript(setupContext?.targetRole ?? session?.target_role ?? "");
    const targetCompany = normalizeTranscript(setupContext?.targetCompany ?? session?.target_company ?? "");
    const roleInstruction = targetRole
      ? `Target role for this mock interview: ${targetRole}.`
      : "Target role is not explicitly provided, infer a realistic role context from the job description.";
    const companyInstruction = targetCompany
      ? `Target company context: ${targetCompany}.`
      : "Target company context is optional and may be ignored if unavailable.";

    const jobDescriptionContext = truncateContext(setupContext?.jobDescription ?? "", 1100);
    const jobInstruction = jobDescriptionContext
      ? `Use this job description as the primary source of interview topics: ${jobDescriptionContext}`
      : "Use available role and interview metadata when job description text is unavailable.";

    return [
      "You are the interviewer for a voice mock interview session.",
      languageInstruction,
      roleInstruction,
      companyInstruction,
      `Interview mode: ${session?.interview_mode ?? "voice"}.`,
      `Difficulty level: ${session?.interview_difficulty ?? "medium"}.`,
      jobInstruction,
      `Interview time limit: ${VOICE_CALL_MAX_SECONDS} seconds. Keep the conversation paced to fit within this duration.`,
      "Do not rely on a pre-scripted fixed list of questions.",
      "Ask one clear question at a time and wait for candidate response before moving on.",
      "Keep responses natural and conversational.",
      "Do not speak JSON, code blocks, or machine-readable payloads.",
    ].join(" ");
  }, [session?.interview_difficulty, session?.interview_language, session?.interview_mode, session?.target_company, session?.target_role, setupContext?.jobDescription, setupContext?.targetCompany, setupContext?.targetRole]);

  const redirectToDashboardNow = useCallback(() => {
    if (dashboardRedirectTimeoutRef.current) {
      clearTimeout(dashboardRedirectTimeoutRef.current);
      dashboardRedirectTimeoutRef.current = null;
    }

    resetInterviewFlow();
    router.push("/dashboard");
  }, [resetInterviewFlow, router]);

  const openCompletionPopupAndRedirect = useCallback((message?: string) => {
    setCompletionPopupMessage(message ?? DEFAULT_COMPLETION_POPUP_MESSAGE);
    setShowCompletionPopup(true);

    if (dashboardRedirectTimeoutRef.current) {
      clearTimeout(dashboardRedirectTimeoutRef.current);
    }

    dashboardRedirectTimeoutRef.current = setTimeout(() => {
      dashboardRedirectTimeoutRef.current = null;
      resetInterviewFlow();
      router.push("/dashboard");
    }, DASHBOARD_REDIRECT_DELAY_MS);
  }, [resetInterviewFlow, router]);

  const endAgentConversation = useCallback(async () => {
    const activeConversation = conversationRef.current;
    if (!activeConversation) {
      return;
    }

    conversationRef.current = null;
    try {
      await activeConversation.endSession();
    } catch {
      return;
    }
  }, []);

  useEffect(() => {
    return () => {
      if (userTurnFlushTimerRef.current) {
        clearTimeout(userTurnFlushTimerRef.current);
      }
      if (dashboardRedirectTimeoutRef.current) {
        clearTimeout(dashboardRedirectTimeoutRef.current);
      }
      void endAgentConversation();
    };
  }, [endAgentConversation]);

  const processPendingAnswers = useCallback(async () => {
    if (queueRunningRef.current) {
      return;
    }

    queueRunningRef.current = true;

    try {
      while (pendingAnswerQueueRef.current.length > 0) {
        const nextTurn = pendingAnswerQueueRef.current.shift();

        if (!nextTurn || sessionCompletedRef.current || !session?.id) {
          continue;
        }

        if (questionsRef.current.length === 0) {
          setVoiceError("No scoring question slots available for this session.");
          break;
        }

        const slotIndex = evaluatedTurnsRef.current % questionsRef.current.length;
        const slotQuestion = questionsRef.current[slotIndex];

        setVoiceInfo(pickLocaleText(locale, "Menganalisis jawaban Anda...", "Analyzing your answer..."));

        try {
          await api.submitInterviewAnswer(session.id, slotQuestion.id, nextTurn.answer);
          const scoredFeedback = await api.generateFeedback(
            session.id,
            slotQuestion.id,
            nextTurn.question,
            nextTurn.answer,
          );
          setLatestFeedback(scoredFeedback);
          evaluatedTurnsRef.current += 1;
          setEvaluatedTurns(evaluatedTurnsRef.current);
        } catch (submitError) {
          const message = submitError instanceof Error ? submitError.message : "Failed to evaluate answer.";
          setVoiceError(message);

          if (message.toLowerCase().includes("session not found")) {
            pendingAnswerQueueRef.current = [];
            setVoiceInfo(pickLocaleText(locale, "Sesi interview tidak ditemukan di server. Silakan mulai sesi voice baru dari halaman Practice.", "Interview session was not found on the server. Please start a new voice session from Practice."));
            resetInterviewFlow();
            router.push("/practice");
            return;
          }

          continue;
        }

        setVoiceInfo(pickLocaleText(locale, "Jawaban sudah dinilai. Lanjutkan ke pertanyaan berikutnya secara natural.", "Answer scored. Continue with the next question naturally."));
      }
    } finally {
      queueRunningRef.current = false;
    }
  }, [locale, resetInterviewFlow, router, session?.id]);

  const flushActiveTurnForScoring = useCallback(() => {
    if (userTurnFlushTimerRef.current) {
      clearTimeout(userTurnFlushTimerRef.current);
      userTurnFlushTimerRef.current = null;
    }

    const activeQuestion = normalizeTranscript(activeAgentQuestionRef.current);
    const activeAnswer = normalizeTranscript(activeUserAnswerPartsRef.current.join(" "));

    if (!activeQuestion || !activeAnswer) {
      return;
    }

    pendingAnswerQueueRef.current.push({ question: activeQuestion, answer: activeAnswer });
    activeUserAnswerPartsRef.current = [];
    void processPendingAnswers();
  }, [processPendingAnswers]);

  const scheduleTurnFlush = useCallback(() => {
    if (userTurnFlushTimerRef.current) {
      clearTimeout(userTurnFlushTimerRef.current);
    }

    userTurnFlushTimerRef.current = setTimeout(() => {
      flushActiveTurnForScoring();
    }, 3500);
  }, [flushActiveTurnForScoring]);

  const waitForPendingScoring = useCallback(async () => {
    for (let attempt = 0; attempt < 40; attempt += 1) {
      if (!queueRunningRef.current && pendingAnswerQueueRef.current.length === 0) {
        return;
      }

      await new Promise((resolve) => setTimeout(resolve, 120));
    }
  }, []);

  const commitVoiceUsageIfNeeded = useCallback(async () => {
    if (usageCommittedRef.current || !session?.id || timerSeconds <= 0 || connectionMode !== "agent") {
      return;
    }

    usageCommittedRef.current = true;

    try {
      await api.commitVoiceUsage(session.id, timerSeconds);
    } catch {
      usageCommittedRef.current = false;
    }
  }, [connectionMode, session?.id, timerSeconds]);

  const finalizeInterviewSession = useCallback(async (options?: {
    redirectToPractice?: boolean;
    redirectToDashboardWithPopup?: boolean;
    closingContextMessage?: string;
    finalInfoMessage?: string;
    completionPopupMessage?: string;
  }) => {
    if (sessionFinalizingRef.current) {
      return;
    }

    sessionFinalizingRef.current = true;

    try {
      setVoiceInfo(options?.finalInfoMessage ?? pickLocaleText(locale, "Menyelesaikan interview dan menyimpan hasil...", "Finalizing interview and saving results..."));
      flushActiveTurnForScoring();
      await waitForPendingScoring();
      await commitVoiceUsageIfNeeded();

      if (!sessionCompletedRef.current) {
        const completed = await completeSession();
        sessionCompletedRef.current = completed;
      }

      if (options?.closingContextMessage && conversationRef.current) {
        try {
          conversationRef.current.sendContextualUpdate(options.closingContextMessage);
          await new Promise((resolve) => setTimeout(resolve, 1800));
        } catch {
          setVoiceError("Unable to send final context update to the agent.");
        }
      }

      isCallActiveRef.current = false;
      await endAgentConversation();
      setIsCallActive(false);

      if (options?.redirectToDashboardWithPopup) {
        openCompletionPopupAndRedirect(options.completionPopupMessage);
        return;
      }

      if (options?.redirectToPractice) {
        resetInterviewFlow();
        router.push("/practice");
      }
    } finally {
      sessionFinalizingRef.current = false;
    }
  }, [commitVoiceUsageIfNeeded, completeSession, endAgentConversation, flushActiveTurnForScoring, locale, openCompletionPopupAndRedirect, resetInterviewFlow, router, waitForPendingScoring]);

  const startAgentConversation = useCallback(async () => {
    if (conversationRef.current || startingConversationRef.current || !isCallActive) {
      return;
    }

    startingConversationRef.current = true;

    setVoiceError(null);
    setVoiceInfo(pickLocaleText(locale, "Menghubungkan ke agen ElevenLabs...", "Connecting to ElevenLabs agent..."));
    setAgentStatus("connecting");
    timeLimitHandledRef.current = false;
    warningThresholdShownRef.current = false;
    usageCommittedRef.current = false;
    setVoiceLimitSeconds(VOICE_CALL_MAX_SECONDS);

    try {
      const agentSession = await api.createVoiceAgentSession();

      const conversation = await Conversation.startSession({
        signedUrl: agentSession.signed_url,
        connectionType: "websocket",
        userId: session?.id,
        overrides: {
          agent: {
            language: selectedInterviewLanguage,
          },
        },
        dynamicVariables: {
          interview_language: selectedInterviewLanguage,
        },
        onConnect: ({ conversationId }) => {
          setConversationID(conversationId ?? null);
          setVoiceInfo(pickLocaleText(locale, "Terhubung. Bicaralah secara natural dengan interviewer.", "Connected. Speak naturally with the interviewer."));
          try {
            conversationRef.current?.sendContextualUpdate(buildAgentContextInstruction());
          } catch {
            setVoiceError("Failed to send interview preferences to AI interviewer.");
          }
        },
        onStatusChange: ({ status }) => {
          setAgentStatus(status);
        },
        onModeChange: ({ mode }) => {
          setAgentMode(mode);
        },
        onError: (message) => {
          setVoiceError(message || "Agent connection error.");
        },
        onDisconnect: () => {
          flushActiveTurnForScoring();
          setAgentStatus("disconnected");

          if (!isCallActiveRef.current || sessionFinalizingRef.current) {
            return;
          }

          void finalizeInterviewSession({
            redirectToDashboardWithPopup: true,
            finalInfoMessage: "Agent ended the call. Finalizing interview and saving results...",
            completionPopupMessage: DEFAULT_COMPLETION_POPUP_MESSAGE,
          });
        },
        onMessage: ({ role, message, event_id }) => {
          const trimmedMessage = normalizeTranscript(message);
          if (!trimmedMessage) {
            return;
          }

          const now = Date.now();
          const dedupKey = `${role}:${trimmedMessage.toLowerCase()}`;
          const previousTimestamp = recentMessageDedupRef.current.get(dedupKey);
          if (typeof previousTimestamp === "number" && now-previousTimestamp < 3000) {
            return;
          }
          recentMessageDedupRef.current.set(dedupKey, now);
          if (recentMessageDedupRef.current.size > 300) {
            recentMessageDedupRef.current.clear();
          }

          if (role === "agent") {
            if (typeof event_id === "number") {
              if (processedAgentEventIDsRef.current.has(event_id)) {
                return;
              }
              processedAgentEventIDsRef.current.add(event_id);
              if (processedAgentEventIDsRef.current.size > 500) {
                processedAgentEventIDsRef.current.clear();
              }
            }

            if (showTranscriptPanel) {
              appendTranscript("agent", trimmedMessage);
            }

            if (isLikelyQuestion(trimmedMessage)) {
              if (activeAgentQuestionRef.current && activeUserAnswerPartsRef.current.length > 0) {
                flushActiveTurnForScoring();
              }

              activeAgentQuestionRef.current = trimmedMessage;
              setActiveQuestionLabel(trimmedMessage);
              setVoiceInfo(pickLocaleText(locale, "Interviewer AI sedang mengajukan pertanyaan baru.", "AI interviewer is asking a new question."));
            }

            return;
          }

          if (role !== "user") {
            return;
          }

          if (typeof event_id === "number") {
            if (processedUserEventIDsRef.current.has(event_id)) {
              return;
            }
            processedUserEventIDsRef.current.add(event_id);
            if (processedUserEventIDsRef.current.size > 500) {
              processedUserEventIDsRef.current.clear();
            }
          }

          if (showTranscriptPanel) {
            appendTranscript("user", trimmedMessage);
          }

          if (!activeAgentQuestionRef.current) {
            return;
          }

          activeUserAnswerPartsRef.current.push(trimmedMessage);
          scheduleTurnFlush();
        },
      });

      conversationRef.current = conversation;
      conversationRef.current?.setVolume?.({ volume: isMuted ? 0 : 1 });
      setConnectionMode("agent");
      setAgentStatus("connected");

      if (agentSession.conversation_id) {
        setConversationID(agentSession.conversation_id);
      }

      if (typeof agentSession.allowed_call_seconds === "number" && Number.isFinite(agentSession.allowed_call_seconds)) {
        const boundedSeconds = Math.max(1, Math.min(VOICE_CALL_MAX_SECONDS, Math.floor(agentSession.allowed_call_seconds)));
        setVoiceLimitSeconds(boundedSeconds);
      }

      if (agentSession.warning_threshold_reached) {
        setVoiceInfo(agentSession.voice_quota_message || pickLocaleText(locale, "Sisa menit voice Anda hampir habis.", "Your remaining voice minutes are almost exhausted."));
      }
    } catch (agentError) {
      setConnectionMode("fallback");
      setAgentStatus("disconnected");
      const message = agentError instanceof Error ? agentError.message : "Unable to connect to ElevenLabs agent.";
      setVoiceError(message);
      setVoiceInfo(pickLocaleText(locale, "Interviewer AI tidak tersedia. Akhiri panggilan dan coba lagi dari pengaturan voice.", "AI interviewer is unavailable. End call and try again from voice setup."));

      if (message.toLowerCase().includes("voice quota exceeded")) {
        router.push("/practice");
      }
    } finally {
      startingConversationRef.current = false;
    }
  }, [appendTranscript, buildAgentContextInstruction, finalizeInterviewSession, flushActiveTurnForScoring, isCallActive, isMuted, locale, router, scheduleTurnFlush, selectedInterviewLanguage, session?.id, showTranscriptPanel]);

  useEffect(() => {
    if (!isCallActive || !currentQuestion || connectionMode !== "initializing") {
      return;
    }

    void startAgentConversation();
  }, [connectionMode, currentQuestion, isCallActive, startAgentConversation]);

  useEffect(() => {
    if (!sessionCompleted) {
      return;
    }

    flushActiveTurnForScoring();
  }, [flushActiveTurnForScoring, sessionCompleted]);

  useEffect(() => {
    if (!isCallActive || sessionCompleted || connectionMode !== "agent") {
      return;
    }

    if (timerSeconds < voiceLimitSeconds || timeLimitHandledRef.current) {
      return;
    }

    timeLimitHandledRef.current = true;
    void finalizeInterviewSession({
      closingContextMessage: "The interview time limit has been reached. End the session now with one brief closing sentence and thank the candidate.",
      finalInfoMessage: "Call duration limit reached. Finalizing interview and score...",
      redirectToDashboardWithPopup: true,
      completionPopupMessage: DEFAULT_COMPLETION_POPUP_MESSAGE,
    });
  }, [connectionMode, finalizeInterviewSession, isCallActive, sessionCompleted, timerSeconds, voiceLimitSeconds]);

  useEffect(() => {
    if (!isCallActive || connectionMode !== "agent" || warningThresholdShownRef.current || voiceLimitSeconds <= 0) {
      return;
    }

    const warningSeconds = Math.max(1, Math.floor(voiceLimitSeconds * 0.1));
    if (remainingCallSeconds > warningSeconds) {
      return;
    }

    warningThresholdShownRef.current = true;
    setVoiceInfo(pickLocaleText(locale, "Peringatan: sisa menit voice kurang dari 10%.", "Warning: remaining voice minutes are below 10%."));
  }, [connectionMode, isCallActive, locale, remainingCallSeconds, voiceLimitSeconds]);

  const handleBackContinueLater = useCallback(async () => {
    if (backExitAction !== "idle") {
      return;
    }

    setBackExitAction("pausing");
    setVoiceError(null);
    setVoiceInfo(pickLocaleText(locale, "Menyimpan progres voice interview sebelum keluar...", "Saving voice interview progress before leaving..."));

    flushActiveTurnForScoring();
    await waitForPendingScoring();
    await commitVoiceUsageIfNeeded();

    isCallActiveRef.current = false;
    setIsCallActive(false);
    await endAgentConversation();

    setBackExitAction("idle");
    navigateBackWithoutPrompt();
  }, [backExitAction, commitVoiceUsageIfNeeded, endAgentConversation, flushActiveTurnForScoring, locale, navigateBackWithoutPrompt, waitForPendingScoring]);

  const handleBackEndSession = useCallback(async () => {
    if (backExitAction !== "idle") {
      return;
    }

    setBackExitAction("ending");
    setVoiceError(null);
    setVoiceInfo(pickLocaleText(locale, "Mengakhiri sesi interview dan menyimpan hasil...", "Ending interview session and saving results..."));

    flushActiveTurnForScoring();
    await waitForPendingScoring();
    await commitVoiceUsageIfNeeded();

    if (!sessionCompletedRef.current) {
      const completed = await completeSession();
      sessionCompletedRef.current = completed;

      if (!completed) {
        setBackExitAction("idle");
        return;
      }
    }

    isCallActiveRef.current = false;
    setIsCallActive(false);
    await endAgentConversation();

    resetInterviewFlow();
    setBackExitAction("idle");
    navigateBackWithoutPrompt();
  }, [backExitAction, commitVoiceUsageIfNeeded, completeSession, endAgentConversation, flushActiveTurnForScoring, locale, navigateBackWithoutPrompt, resetInterviewFlow, waitForPendingScoring]);

  function endCall() {
    void finalizeInterviewSession({
      redirectToPractice: true,
      finalInfoMessage: "Ending call and saving interview results...",
    });
  }

  function toggleMute() {
    const nextMuted = !isMuted;
    setIsMuted(nextMuted);
    conversationRef.current?.setVolume?.({ volume: nextMuted ? 0 : 1 });
  }

  function openVoiceSetup() {
    router.push("/practice");
  }

  function formatTimer(secondsTotal: number): string {
    const minutes = Math.floor(secondsTotal / 60);
    const seconds = secondsTotal % 60;
    return `${minutes.toString().padStart(2, "0")}:${seconds.toString().padStart(2, "0")}`;
  }

  const speakingState = connectionMode === "agent" && agentMode === "speaking";
  const listeningState = connectionMode === "agent" && agentMode === "listening";

  const statusLabel = connectionMode === "agent"
    ? agentStatus === "connecting"
      ? pickLocaleText(locale, "Menghubungkan ke interviewer", "Connecting to interviewer")
      : agentStatus === "disconnected"
        ? pickLocaleText(locale, "Terputus", "Disconnected")
        : agentMode === "speaking"
          ? pickLocaleText(locale, "Interviewer AI sedang berbicara", "AI Interviewer speaking")
          : pickLocaleText(locale, "Mendengarkan jawaban Anda", "Listening for your answer")
    : connectionMode === "initializing"
      ? pickLocaleText(locale, "Menghubungkan Interviewer AI", "Connecting AI Interviewer")
      : pickLocaleText(locale, "Terputus", "Disconnected");

  if (!currentQuestion) {
    return (
      <div className="relative min-h-screen overflow-hidden bg-[var(--color-bg)] text-white">
        <div className="absolute inset-0 grid-overlay opacity-20" />
        <div className="ambient-orb orb-primary left-[-80px] top-24 h-80 w-80" />
        <div className="ambient-orb orb-cyan right-[-100px] bottom-[-20px] h-80 w-80" />

        <div className="relative z-10 mx-auto flex min-h-screen w-full max-w-5xl flex-col items-center justify-center px-6 text-center">
          <div className="mb-5 inline-flex h-16 w-16 items-center justify-center rounded-3xl border border-white/15 bg-gradient-to-br from-purple-500/30 to-cyan-500/20">
            <Sparkles className="h-8 w-8 text-cyan-100" />
          </div>
          <h1 className="text-2xl font-semibold">{pickLocaleText(locale, "Tidak ada sesi interview aktif", "No active interview session")}</h1>
          <p className="mt-2 max-w-xl text-sm text-[var(--color-text-muted)]">
            {pickLocaleText(locale, "Mulai dari pengaturan Practice dan pilih mode interview suara terlebih dahulu.", "Start from Practice setup and choose Voice interview mode first.")}
          </p>
          <Button onClick={openVoiceSetup} className="mt-6">
            <ArrowLeft className="mr-2 h-4 w-4" />
            {pickLocaleText(locale, "Kembali ke pengaturan suara", "Back to voice setup")}
          </Button>
        </div>
      </div>
    );
  }

  return (
    <div className="relative min-h-screen overflow-hidden bg-[var(--color-bg)] text-white">
      <div className="absolute inset-0 grid-overlay opacity-20" />
      <div className="ambient-orb orb-primary left-[-90px] top-16 h-80 w-80" />
      <div className="ambient-orb orb-cyan right-[-90px] top-[28%] h-80 w-80" />
      <div className="absolute inset-x-0 bottom-[-140px] mx-auto h-80 w-[70vw] rounded-full bg-gradient-to-r from-purple-500/20 via-cyan-400/20 to-blue-500/20 blur-3xl" />

      <div className="relative z-10 mx-auto flex min-h-screen w-full max-w-7xl flex-col px-4 py-4 md:px-8 md:py-6">
        <header className="space-y-3 rounded-[20px] border border-white/10 bg-[rgba(17,24,36,0.72)] px-4 py-3 backdrop-blur-xl md:px-5">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h2 className="text-lg text-white tracking-tight">{pickLocaleText(locale, "Sesi Interview", "Interview Session")}</h2>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">
              {connectionMode === "agent" ? pickLocaleText(locale, "Mode suara · Interviewer AI", "Voice mode · AI Interviewer") : pickLocaleText(locale, "Menyiapkan Interviewer AI", "Preparing AI Interviewer")}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2 text-xs">
              <span className="rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-cyan-200">
                {pickLocaleText(locale, "Giliran dinilai", "Evaluated turns")}: {evaluatedTurns}
              </span>
              {session?.target_role && (
                <span className="rounded-full border border-fuchsia-300/30 bg-fuchsia-400/10 px-3 py-1 text-fuchsia-100">
                  {pickLocaleText(locale, "Role", "Role")}: {session.target_role}
                </span>
              )}
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
                {interviewLanguageLabel}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
                {interviewDifficultyLabel}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
                {formatTimer(timerSeconds)}
              </span>
              <span className="rounded-full border border-amber-300/35 bg-amber-400/10 px-3 py-1 text-amber-100">
                {pickLocaleText(locale, "Sisa", "Remaining")} {formatTimer(remainingCallSeconds)} / {formatTimer(voiceLimitSeconds)}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">{statusLabel}</span>
              {connectionMode === "agent" && conversationID && (
                <span className="rounded-full border border-emerald-300/30 bg-emerald-400/10 px-3 py-1 text-emerald-200">
                  #{conversationID.slice(0, 8)}
                </span>
              )}
            </div>
          </div>

          <div className="h-1.5 overflow-hidden rounded-full bg-white/10">
            <div
              className="h-full rounded-full bg-gradient-to-r from-cyan-400 to-blue-500 transition-all duration-500"
              style={{ width: `${callProgressPercent}%` }}
            />
          </div>
        </header>

        <main className="flex flex-1 flex-col items-center justify-center gap-5 py-6">
          <VoiceOrb isSpeaking={speakingState} isListening={listeningState} isCallActive={isCallActive} />

          <div className="flex flex-wrap items-center justify-center gap-2">
            {HIDE_TRANSCRIPT_IN_PRODUCTION && (
              <Button
                variant="secondary"
                onClick={toggleMute}
                disabled={connectionMode !== "agent"}
              >
                {isMuted ? <MicOff className="mr-2 h-4 w-4" /> : <Mic className="mr-2 h-4 w-4" />}
                    {isMuted ? pickLocaleText(locale, "Bunyikan", "Unmute") : pickLocaleText(locale, "Bisukan", "Mute")}
              </Button>
            )}
            <Button variant="secondary" onClick={endCall} className="border-red-400/50 text-red-200 hover:border-red-300/70">
              <PhoneOff className="mr-2 h-4 w-4" />
                  {pickLocaleText(locale, "Akhiri Sesi", "End Session")}
            </Button>
          </div>

          <div className="w-full max-w-5xl rounded-[20px] bg-gradient-to-r from-purple-500/35 via-cyan-500/30 to-purple-500/35 p-[1px]">
            <Card className="space-y-5 p-5 md:p-6">
              <div>
                <p className="text-xs uppercase tracking-wide text-[var(--color-text-muted)]">AI Interviewer</p>
                <p className="mt-2 text-base leading-relaxed text-white/95 md:text-lg">{activeQuestionLabel}</p>
              </div>

            {(error || voiceError || voiceInfo) && (
              <div className="flex flex-wrap gap-2">
                {error && <p className="inline-flex rounded-full border border-red-400/25 bg-red-500/10 px-3 py-1 text-xs text-red-300">{error}</p>}
                {voiceError && <p className="inline-flex rounded-full border border-red-400/25 bg-red-500/10 px-3 py-1 text-xs text-red-300">{voiceError}</p>}
                {voiceInfo && <p className="inline-flex rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">{voiceInfo}</p>}
              </div>
            )}

              {connectionMode === "agent" && showTranscriptPanel ? (
                <div>
                  <p className="mb-2 text-xs uppercase tracking-wide text-[var(--color-text-muted)]">Interview Transcript</p>
                  <div
                    ref={transcriptContainerRef}
                    className="max-h-56 space-y-2 overflow-y-auto rounded-2xl border border-white/10 bg-white/[0.03] p-3"
                  >
                    {transcriptItems.length === 0 && (
                      <p className="text-sm text-white/55">
                        Conversation transcript will appear here.
                      </p>
                    )}
                    {transcriptItems.map((item) => (
                      <div
                        key={item.id}
                        className={cn(
                          "rounded-xl px-3 py-2 text-sm leading-relaxed",
                          item.role === "agent"
                            ? "border border-cyan-300/20 bg-cyan-400/10 text-cyan-100"
                            : "border border-white/15 bg-white/[0.04] text-white/90",
                        )}
                      >
                        <p className="mb-1 text-[11px] uppercase tracking-wide opacity-70">
                          {item.role === "agent" ? "Interviewer" : "You"}
                        </p>
                        <p>{item.message}</p>
                      </div>
                    ))}
                  </div>
                </div>
              ) : connectionMode !== "agent" ? (
                <div className="rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3">
                  <p className="text-sm text-white/70">
                    Voice interview runs automatically. If connection drops, end the session and restart from setup.
                  </p>
                </div>
              ) : null}

              <p className="text-xs text-white/60">
                Keep speaking naturally. Each answer is evaluated automatically.
              </p>

              {!HIDE_TRANSCRIPT_IN_PRODUCTION && latestFeedback && (
                <div className="rounded-2xl border border-cyan-300/25 bg-cyan-400/10 px-4 py-3">
                  <p className="text-xs uppercase tracking-wide text-cyan-200/80">Latest Feedback</p>
                  <p className="mt-1 text-sm text-cyan-100">Score {latestFeedback.score}/100 · {latestFeedback.star_feedback}</p>
                </div>
              )}
            </Card>
          </div>
        </main>
      </div>

      {showBackExitModal && (
        <div className="fixed inset-0 z-60 flex items-center justify-center bg-slate-950/75 px-4 backdrop-blur-sm">
          <Card className="w-full max-w-md border border-white/15 bg-[rgba(10,15,26,0.96)] p-6">
            <h3 className="text-base font-semibold text-white">
              {pickLocaleText(locale, "Keluar dari sesi voice interview?", "Leave voice interview session?")}
            </h3>
            <p className="mt-2 text-sm text-white/70">
              {pickLocaleText(
                locale,
                "Pilih `Akhiri sesi` untuk menyelesaikan interview sekarang, atau `Lanjutkan nanti` untuk kembali tanpa menyelesaikan sesi.",
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
                onClick={() => {
                  void handleBackContinueLater();
                }}
                disabled={backExitAction !== "idle"}
              >
                {backExitAction === "pausing"
                  ? pickLocaleText(locale, "Menyimpan progres...", "Saving progress...")
                  : pickLocaleText(locale, "Lanjutkan nanti", "Continue later")}
              </Button>
              <Button
                type="button"
                onClick={() => {
                  void handleBackEndSession();
                }}
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

      {showCompletionPopup && (
        <div className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/70 px-4">
          <Card className="w-full max-w-md rounded-3xl border border-cyan-300/25 bg-[rgba(10,15,26,0.96)] p-6 text-center">
            <p className="text-xs uppercase tracking-wide text-cyan-200/80">Interview Completed</p>
            <h3 className="mt-2 text-xl font-semibold text-white">Sesi Latihan Selesai</h3>
            <p className="mt-3 text-sm leading-relaxed text-white/80">{completionPopupMessage}</p>
            <p className="mt-2 text-xs text-cyan-100/75">Mengalihkan ke dashboard...</p>

            <div className="mt-5 flex justify-center">
              <Button onClick={redirectToDashboardNow}>Ke Dashboard Sekarang</Button>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}

function VoiceOrb({
  isSpeaking,
  isListening,
  isCallActive,
}: {
  isSpeaking: boolean;
  isListening: boolean;
  isCallActive: boolean;
}) {
  const { locale } = useLanguage();
  const stateClass = isSpeaking
    ? "voice-liquid-orb--speaking"
    : isListening
      ? "voice-liquid-orb--listening"
      : "voice-liquid-orb--idle";

  return (
    <div className={cn("voice-liquid-orb", stateClass, !isCallActive && "voice-liquid-orb--inactive")}>
      <div className="voice-aurora voice-aurora--outer" />
      <div className="voice-aurora voice-aurora--inner" />

      <div className="voice-ripple-stack">
        {[0, 1, 2].map((index) => (
          <span
            key={index}
            className="voice-ripple-ring"
            style={{ animationDelay: `${index * 0.34}s` }}
          />
        ))}
      </div>

      <div className="voice-liquid-core">
        <div className="voice-liquid-surface" />
        <div className="voice-liquid-sheen" />
        <div className="voice-liquid-inner-ring" />
        <div className="voice-liquid-inner-ring voice-liquid-inner-ring--small" />

        <div className="relative z-10 flex flex-col items-center">
          <Sparkles className={cn("h-8 w-8 text-white/90", isSpeaking && "animate-pulse")} />
          <p className="mt-2 text-xs font-medium text-white/90">
            {isSpeaking
              ? pickLocaleText(locale, "Berbicara", "Speaking")
              : isListening
                ? pickLocaleText(locale, "Mendengarkan", "Listening")
                : isCallActive
                  ? pickLocaleText(locale, "Siap", "Ready")
                  : pickLocaleText(locale, "Diam", "Idle")}
          </p>
        </div>
      </div>
    </div>
  );
}

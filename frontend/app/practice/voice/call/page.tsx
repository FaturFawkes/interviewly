"use client";

import { Conversation, type Mode, type Status } from "@elevenlabs/client";
import { ArrowLeft, PhoneOff, Sparkles } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";
import { api } from "@/lib/api/endpoints";
import { cn } from "@/lib/utils";

type ConnectionMode = "initializing" | "agent" | "fallback";

type TranscriptItem = {
  id: string;
  role: "user" | "agent";
  message: string;
};

export default function VoiceCallPage() {
  const router = useRouter();
  const {
    session,
    questions,
    currentQuestion,
    currentIndex,
    feedback,
    error,
    timerSeconds,
    submitAnswerText,
    completeSession,
    goToNextQuestion,
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
  const announcedQuestionIDRef = useRef<string>("");
  const pendingAnswerQueueRef = useRef<string[]>([]);
  const queueRunningRef = useRef(false);
  const startingConversationRef = useRef(false);
  const processedAgentEventIDsRef = useRef<Set<number>>(new Set());
  const processedUserEventIDsRef = useRef<Set<number>>(new Set());
  const recentMessageDedupRef = useRef<Map<string, number>>(new Map());
  const questionsRef = useRef(questions);
  const currentIndexRef = useRef(currentIndex);
  const questionCountRef = useRef(questions.length);
  const sessionCompletedRef = useRef(sessionCompleted);

  const [isCallActive, setIsCallActive] = useState(true);
  const [voiceError, setVoiceError] = useState<string | null>(null);
  const [voiceInfo, setVoiceInfo] = useState<string | null>("Connecting to AI interviewer...");
  const [connectionMode, setConnectionMode] = useState<ConnectionMode>("initializing");
  const [agentStatus, setAgentStatus] = useState<Status>("disconnected");
  const [agentMode, setAgentMode] = useState<Mode>("listening");
  const [conversationID, setConversationID] = useState<string | null>(null);
  const [transcriptItems, setTranscriptItems] = useState<TranscriptItem[]>([]);

  useEffect(() => {
    questionsRef.current = questions;
    currentIndexRef.current = currentIndex;
    questionCountRef.current = questions.length;
    sessionCompletedRef.current = sessionCompleted;
  }, [currentIndex, questions, sessionCompleted]);

  useEffect(() => {
    if (!transcriptContainerRef.current) {
      return;
    }

    transcriptContainerRef.current.scrollTop = transcriptContainerRef.current.scrollHeight;
  }, [transcriptItems]);

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

    return [
      "You are the interviewer for a voice mock interview session.",
      languageInstruction,
      `Interview mode: ${session?.interview_mode ?? "voice"}.`,
      `Difficulty level: ${session?.interview_difficulty ?? "medium"}.`,
      "Keep responses natural and conversational.",
      "Do not speak JSON, code blocks, or machine-readable payloads.",
    ].join(" ");
  }, [session?.interview_difficulty, session?.interview_language, session?.interview_mode]);

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
        const nextAnswer = pendingAnswerQueueRef.current.shift();

        if (!nextAnswer || !isCallActive || sessionCompletedRef.current) {
          continue;
        }

        const lastQuestion = currentIndexRef.current >= questionCountRef.current - 1;
        setVoiceInfo("Analyzing your answer...");

        const submitted = await submitAnswerText(nextAnswer);
        if (!submitted) {
          setVoiceError("Failed to evaluate answer.");
          continue;
        }

        if (lastQuestion) {
          await completeSession();
          setVoiceInfo("Interview completed.");

          if (conversationRef.current) {
            try {
              conversationRef.current.sendContextualUpdate(
                "The interview has been completed. Thank the candidate and close the session politely.",
              );
            } catch {
              setVoiceError("Unable to send final context update to the agent.");
            }
          }

          break;
        }

        goToNextQuestion();
      }
    } finally {
      queueRunningRef.current = false;
    }
  }, [completeSession, goToNextQuestion, isCallActive, submitAnswerText]);

  const startAgentConversation = useCallback(async () => {
    if (conversationRef.current || startingConversationRef.current || !isCallActive) {
      return;
    }

    startingConversationRef.current = true;

    setVoiceError(null);
    setVoiceInfo("Connecting to ElevenLabs agent...");
    setAgentStatus("connecting");

    try {
      const agentSession = await api.createVoiceAgentSession();

      const conversation = await Conversation.startSession({
        signedUrl: agentSession.signed_url,
        connectionType: "websocket",
        userId: session?.id,
        onConnect: ({ conversationId }) => {
          setConversationID(conversationId ?? null);
          setVoiceInfo("Connected. Speak naturally with the interviewer.");
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
          setAgentStatus("disconnected");
          if (isCallActive) {
            setVoiceInfo("Agent disconnected.");
          }
        },
        onMessage: ({ role, message, event_id }) => {
          const trimmedMessage = message.trim();
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

            appendTranscript("agent", trimmedMessage);
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

          appendTranscript("user", trimmedMessage);
          pendingAnswerQueueRef.current.push(trimmedMessage);
          void processPendingAnswers();
        },
      });

      conversationRef.current = conversation;
      setConnectionMode("agent");
      setAgentStatus("connected");

      if (agentSession.conversation_id) {
        setConversationID(agentSession.conversation_id);
      }
    } catch (agentError) {
      setConnectionMode("fallback");
      setAgentStatus("disconnected");
      setVoiceError(agentError instanceof Error ? agentError.message : "Unable to connect to ElevenLabs agent.");
      setVoiceInfo("AI interviewer is unavailable. End call and try again from voice setup.");
    } finally {
      startingConversationRef.current = false;
    }
  }, [appendTranscript, buildAgentContextInstruction, isCallActive, processPendingAnswers, session?.id]);

  useEffect(() => {
    if (!isCallActive || !currentQuestion || connectionMode !== "initializing") {
      return;
    }

    void startAgentConversation();
  }, [connectionMode, currentQuestion, isCallActive, startAgentConversation]);

  useEffect(() => {
    if (
      connectionMode !== "agent" ||
      agentStatus !== "connected" ||
      !isCallActive ||
      !currentQuestion?.id ||
      !currentQuestion.question ||
      !conversationRef.current ||
      sessionCompleted
    ) {
      return;
    }

    if (announcedQuestionIDRef.current === currentQuestion.id) {
      return;
    }

    announcedQuestionIDRef.current = currentQuestion.id;

    try {
      conversationRef.current.sendContextualUpdate(
        `${buildAgentContextInstruction()} Interview context update: Ask this exact question: "${currentQuestion.question}". Wait for candidate answer before continuing.`,
      );
      setVoiceInfo("AI interviewer is asking the current question.");
    } catch {
      setVoiceError("Failed to send question context to AI interviewer.");
    }
  }, [agentStatus, buildAgentContextInstruction, connectionMode, currentQuestion?.id, currentQuestion?.question, isCallActive, sessionCompleted]);

  function endCall() {
    setIsCallActive(false);
    setVoiceInfo("Call ended.");
    void endAgentConversation();
    router.push("/practice");
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
    ? agentMode === "speaking"
      ? "AI Interviewer speaking"
      : "Listening for your answer"
    : connectionMode === "initializing"
      ? "Connecting AI Interviewer"
      : "Disconnected";

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
          <h1 className="text-2xl font-semibold">No active interview session</h1>
          <p className="mt-2 max-w-xl text-sm text-[var(--color-text-muted)]">
            Start from Practice setup and choose Voice interview mode first.
          </p>
          <Button onClick={openVoiceSetup} className="mt-6">
            <ArrowLeft className="mr-2 h-4 w-4" />
            Back to voice setup
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
              <h2 className="text-lg text-white tracking-tight">Interview Session</h2>
              <p className="mt-1 text-sm text-[var(--color-text-muted)]">
              {connectionMode === "agent" ? "Voice mode · AI Interviewer" : "Preparing AI Interviewer"}
              </p>
            </div>

            <div className="flex flex-wrap items-center gap-2 text-xs">
              <span className="rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-cyan-200">
                Question {currentIndex + 1}/{questions.length}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
                {interviewLanguageLabel}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
                {interviewDifficultyLabel}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
                {formatTimer(timerSeconds)}
              </span>
              <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">{statusLabel}</span>
              {connectionMode === "agent" && conversationID && (
                <span className="rounded-full border border-emerald-300/30 bg-emerald-400/10 px-3 py-1 text-emerald-200">
                  #{conversationID.slice(0, 8)}
                </span>
              )}
            </div>
          </div>

          <div className="flex items-center gap-1.5">
            {questions.map((item, index) => (
              <div
                key={item.id}
                className={cn(
                  "h-1.5 rounded-full transition-all",
                  index === currentIndex
                    ? "w-8 bg-gradient-to-r from-purple-500 to-cyan-400"
                    : index < currentIndex
                      ? "w-6 bg-purple-500/30"
                      : "w-6 bg-white/[0.08]",
                )}
              />
            ))}
          </div>
        </header>

        <main className="flex flex-1 flex-col items-center justify-center gap-5 py-6">
          <VoiceOrb isSpeaking={speakingState} isListening={listeningState} isCallActive={isCallActive} />

          <div className="w-full max-w-5xl rounded-[20px] bg-gradient-to-r from-purple-500/35 via-cyan-500/30 to-purple-500/35 p-[1px]">
            <Card className="space-y-5 p-5 md:p-6">
              <div>
                <p className="text-xs uppercase tracking-wide text-[var(--color-text-muted)]">AI Interviewer</p>
                <p className="mt-2 text-base leading-relaxed text-white/95 md:text-lg">{currentQuestion.question}</p>
              </div>

            {(error || voiceError || voiceInfo) && (
              <div className="flex flex-wrap gap-2">
                {error && <p className="inline-flex rounded-full border border-red-400/25 bg-red-500/10 px-3 py-1 text-xs text-red-300">{error}</p>}
                {voiceError && <p className="inline-flex rounded-full border border-red-400/25 bg-red-500/10 px-3 py-1 text-xs text-red-300">{voiceError}</p>}
                {voiceInfo && <p className="inline-flex rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-xs text-cyan-200">{voiceInfo}</p>}
              </div>
            )}

              {connectionMode === "agent" ? (
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
              ) : (
                <div className="rounded-2xl border border-white/10 bg-white/[0.03] px-4 py-3">
                  <p className="text-sm text-white/70">
                    Voice interview runs automatically. If connection drops, end the session and restart from setup.
                  </p>
                </div>
              )}

              <p className="text-xs text-white/60">
                Keep speaking naturally. Each answer is evaluated automatically.
              </p>

              <div className="flex flex-wrap items-center justify-end gap-2">
                <Button variant="secondary" onClick={endCall} className="border-red-400/50 text-red-200 hover:border-red-300/70">
                  <PhoneOff className="mr-2 h-4 w-4" />
                  End Session
                </Button>
              </div>

              {feedback && (
                <div className="rounded-2xl border border-cyan-300/25 bg-cyan-400/10 px-4 py-3">
                  <p className="text-xs uppercase tracking-wide text-cyan-200/80">Latest Feedback</p>
                  <p className="mt-1 text-sm text-cyan-100">Score {feedback.score}/100 · {feedback.star_feedback}</p>
                </div>
              )}
            </Card>
          </div>
        </main>
      </div>
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
            {isSpeaking ? "Speaking" : isListening ? "Listening" : isCallActive ? "Ready" : "Idle"}
          </p>
        </div>
      </div>
    </div>
  );
}

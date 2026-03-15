"use client";

import { Conversation, type Mode, type Status } from "@elevenlabs/client";
import { ArrowLeft, PhoneOff, Sparkles } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
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
  } = useInterviewFlow({ storageKey: "interview-flow-voice" });

  const interviewLanguageLabel = session?.interview_language === "id" ? "Bahasa Indonesia" : "English";
  const transcriptContainerRef = useRef<HTMLDivElement | null>(null);
  const conversationRef = useRef<Conversation | null>(null);
  const announcedQuestionIDRef = useRef<string>("");
  const pendingAnswerQueueRef = useRef<string[]>([]);
  const queueRunningRef = useRef(false);
  const processedUserEventIDsRef = useRef<Set<number>>(new Set());
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
    currentIndexRef.current = currentIndex;
    questionCountRef.current = questions.length;
    sessionCompletedRef.current = sessionCompleted;
  }, [currentIndex, questions.length, sessionCompleted]);

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
    if (conversationRef.current || !isCallActive) {
      return;
    }

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
          setConversationID(conversationId);
          setVoiceInfo("Connected. Speak naturally with the interviewer.");
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

          if (role === "agent") {
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
    }
  }, [appendTranscript, isCallActive, processPendingAnswers, session?.id]);

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
        `Interview context update: Ask this exact question in ${interviewLanguageLabel}: "${currentQuestion.question}". Wait for candidate answer before continuing.`,
      );
      setVoiceInfo("AI interviewer is asking the current question.");
    } catch {
      setVoiceError("Failed to send question context to AI interviewer.");
    }
  }, [agentStatus, connectionMode, currentQuestion?.id, currentQuestion?.question, interviewLanguageLabel, isCallActive, sessionCompleted]);

  function endCall() {
    setIsCallActive(false);
    setVoiceInfo("Call ended.");
    void endAgentConversation();
    router.push("/practice/voice");
  }

  function openVoiceSetup() {
    router.push("/practice/voice");
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
      ? "Agent speaking"
      : "Listening"
    : connectionMode === "initializing"
      ? "Connecting"
      : "Disconnected";

  if (!currentQuestion) {
    return (
      <div className="relative min-h-screen overflow-hidden bg-[var(--color-bg)] text-white">
        <div className="absolute inset-0 grid-overlay opacity-20" />
        <div className="ambient-orb orb-primary left-[-80px] top-24 h-80 w-80" />
        <div className="ambient-orb orb-cyan right-[-100px] bottom-[-20px] h-80 w-80" />

        <div className="relative z-10 mx-auto flex min-h-screen w-full max-w-5xl flex-col items-center justify-center px-6 text-center">
          <div className="mb-5 inline-flex h-16 w-16 items-center justify-center rounded-3xl border border-white/15 bg-white/5">
            <Sparkles className="h-8 w-8 text-cyan-200" />
          </div>
          <h1 className="text-2xl font-semibold">No active voice interview</h1>
          <p className="mt-2 max-w-xl text-sm text-[var(--color-text-muted)]">
            Prepare interview questions first from the voice setup page.
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

      <div className="relative z-10 flex min-h-screen flex-col px-4 py-4 md:px-8 md:py-6">
        <header className="flex flex-wrap items-center justify-between gap-3 rounded-2xl border border-white/10 bg-white/5 px-4 py-3 backdrop-blur-md">
          <div className="flex items-center gap-2">
            <p className="text-sm text-[var(--color-text-muted)]">
              {connectionMode === "agent" ? "ElevenLabs realtime interviewer" : "Waiting for AI interviewer"}
            </p>
          </div>

          <div className="flex items-center gap-2 text-xs">
            <span className="rounded-full border border-cyan-300/30 bg-cyan-400/10 px-3 py-1 text-cyan-200">
              Question {currentIndex + 1}/{questions.length}
            </span>
            <span className="rounded-full border border-white/20 bg-white/5 px-3 py-1 text-white/85">
              {interviewLanguageLabel}
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
        </header>

        <main className="flex flex-1 flex-col items-center justify-center gap-6 py-6">
          <VoiceOrb isSpeaking={speakingState} isListening={listeningState} isCallActive={isCallActive} />

          <div className="w-full max-w-4xl space-y-4 rounded-[20px] border border-white/10 bg-[rgba(17,24,36,0.72)] p-4 backdrop-blur-md md:p-6">
            <div>
              <p className="text-xs uppercase tracking-wide text-[var(--color-text-muted)]">AI interviewer question</p>
              <p className="mt-2 text-base leading-relaxed text-white/95 md:text-lg">{currentQuestion.question}</p>
            </div>

            {(error || voiceError || voiceInfo) && (
              <div className="space-y-1">
                {error && <p className="text-sm text-red-300">{error}</p>}
                {voiceError && <p className="text-sm text-red-300">{voiceError}</p>}
                {voiceInfo && <p className="text-sm text-cyan-200">{voiceInfo}</p>}
              </div>
            )}

            {connectionMode === "agent" ? (
              <div>
                <p className="mb-2 text-xs uppercase tracking-wide text-[var(--color-text-muted)]">Live transcript</p>
                <div
                  ref={transcriptContainerRef}
                  className="max-h-56 space-y-2 overflow-y-auto rounded-2xl border border-white/10 bg-white/[0.03] p-3"
                >
                  {transcriptItems.length === 0 && (
                    <p className="text-sm text-white/55">
                      Transcript will appear here once conversation starts.
                    </p>
                  )}
                  {transcriptItems.map((item) => (
                    <div
                      key={item.id}
                      className={cn(
                        "rounded-xl px-3 py-2 text-sm",
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
                  Voice call is fully automatic. If connection drops, end the call and restart from voice setup.
                </p>
              </div>
            )}

            <p className="text-xs text-white/60">
              Answer processing is automatic. Keep speaking naturally and the app evaluates each response.
            </p>

            <div className="flex flex-wrap gap-2">
              <Button variant="secondary" onClick={endCall} className="border-red-400/50 text-red-200 hover:border-red-300/70">
                <PhoneOff className="mr-2 h-4 w-4" />
                End call
              </Button>
            </div>

            {feedback && (
              <div className="rounded-2xl border border-cyan-300/25 bg-cyan-400/10 px-4 py-3">
                <p className="text-xs uppercase tracking-wide text-cyan-200/80">Latest feedback</p>
                <p className="mt-1 text-sm text-cyan-100">Score {feedback.score}/100 · {feedback.star_feedback}</p>
              </div>
            )}
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

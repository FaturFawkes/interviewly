"use client";

import { Conversation, type Mode, type Status } from "@elevenlabs/client";
import { ArrowLeft, CheckCircle, Mic, MicOff, PhoneOff, Send, SkipForward, Sparkles, Volume2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useRouter } from "next/navigation";

import { Button } from "@/components/ui/Button";
import { TextArea } from "@/components/ui/Input";
import { useInterviewFlow } from "@/hooks/useInterviewFlow";
import { getAuthToken } from "@/lib/auth/token-provider";
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
    answer,
    setAnswer,
    feedback,
    loading,
    error,
    timerSeconds,
    submitCurrentAnswer,
    submitAnswerText,
    completeSession,
    goToNextQuestion,
    sessionCompleted,
  } = useInterviewFlow({ storageKey: "interview-flow-voice" });

  const isLastQuestion = currentIndex >= questions.length - 1;
  const interviewLanguageLabel = session?.interview_language === "id" ? "Bahasa Indonesia" : "English";
  const apiBaseUrl = process.env.NEXT_PUBLIC_API_BASE_URL ?? "/api-proxy";

  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const mediaStreamRef = useRef<MediaStream | null>(null);
  const voiceChunksRef = useRef<BlobPart[]>([]);
  const audioPlayerRef = useRef<HTMLAudioElement | null>(null);
  const audioUrlRef = useRef<string | null>(null);
  const lastSpokenQuestionRef = useRef<string>("");
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
  const [isListening, setIsListening] = useState(false);
  const [isSpeaking, setIsSpeaking] = useState(false);
  const [voiceError, setVoiceError] = useState<string | null>(null);
  const [voiceInfo, setVoiceInfo] = useState<string | null>("Connecting to AI interviewer...");
  const [connectionMode, setConnectionMode] = useState<ConnectionMode>("initializing");
  const [agentStatus, setAgentStatus] = useState<Status>("disconnected");
  const [agentMode, setAgentMode] = useState<Mode>("listening");
  const [conversationID, setConversationID] = useState<string | null>(null);
  const [transcriptItems, setTranscriptItems] = useState<TranscriptItem[]>([]);

  const recordingSupported = useMemo(() => {
    if (typeof window === "undefined") {
      return false;
    }

    return Boolean(window.MediaRecorder && navigator.mediaDevices?.getUserMedia);
  }, []);

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

  const stopVoiceInput = useCallback(() => {
    if (mediaRecorderRef.current && mediaRecorderRef.current.state !== "inactive") {
      mediaRecorderRef.current.stop();
    }
  }, []);

  const stopQuestionAudio = useCallback(() => {
    if (audioPlayerRef.current) {
      audioPlayerRef.current.pause();
      audioPlayerRef.current = null;
    }

    if (audioUrlRef.current) {
      URL.revokeObjectURL(audioUrlRef.current);
      audioUrlRef.current = null;
    }

    setIsSpeaking(false);
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
      stopVoiceInput();

      if (mediaStreamRef.current) {
        mediaStreamRef.current.getTracks().forEach((track) => track.stop());
      }

      stopQuestionAudio();
      void endAgentConversation();
    };
  }, [endAgentConversation, stopQuestionAudio, stopVoiceInput]);

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
          setAnswer(trimmedMessage);
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
      setVoiceInfo("Using classic voice mode as fallback.");
    }
  }, [appendTranscript, isCallActive, processPendingAnswers, session?.id, setAnswer]);

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

  async function transcribeRecordedAudio(blob: Blob): Promise<string> {
    const token = await getAuthToken();
    if (!token) {
      throw new Error("Authentication token is missing.");
    }

    const formData = new FormData();
    formData.append("audio", blob, "answer.webm");

    const response = await fetch(`${apiBaseUrl}/api/voice/stt`, {
      method: "POST",
      headers: {
        Authorization: `Bearer ${token}`,
      },
      body: formData,
    });

    const payload = (await response.json().catch(() => ({}))) as { text?: string; error?: string };
    if (!response.ok) {
      throw new Error(payload.error ?? "Failed to transcribe voice.");
    }

    return (payload.text ?? "").trim();
  }

  async function startVoiceInput() {
    setVoiceError(null);
    setVoiceInfo(null);

    if (connectionMode !== "fallback") {
      setVoiceError("Manual recording is only available in fallback mode.");
      return;
    }

    if (!isCallActive) {
      setVoiceError("Call is not active.");
      return;
    }

    if (!recordingSupported || typeof window === "undefined") {
      setVoiceError("Voice input is not supported in this browser.");
      return;
    }

    try {
      const stream = await navigator.mediaDevices.getUserMedia({ audio: true });
      mediaStreamRef.current = stream;
      voiceChunksRef.current = [];

      const recorder = new MediaRecorder(stream);
      mediaRecorderRef.current = recorder;

      recorder.ondataavailable = (event) => {
        if (event.data && event.data.size > 0) {
          voiceChunksRef.current.push(event.data);
        }
      };

      recorder.onstop = () => {
        void (async () => {
          try {
            const audioBlob = new Blob(voiceChunksRef.current, { type: "audio/webm" });
            const transcript = await transcribeRecordedAudio(audioBlob);

            if (!transcript) {
              setVoiceError("No speech detected. Please try again.");
              return;
            }

            appendTranscript("user", transcript);

            setAnswer((previousAnswer) => {
              const trimmed = previousAnswer.trim();
              return trimmed ? `${trimmed} ${transcript}` : transcript;
            });

            setVoiceInfo("Voice converted to text successfully.");
          } catch (voiceProcessError) {
            setVoiceError(
              voiceProcessError instanceof Error
                ? voiceProcessError.message
                : "Failed to process voice input.",
            );
          } finally {
            if (mediaStreamRef.current) {
              mediaStreamRef.current.getTracks().forEach((track) => track.stop());
              mediaStreamRef.current = null;
            }

            setIsListening(false);
          }
        })();
      };

      recorder.start();
      setIsListening(true);
    } catch {
      setVoiceError("Microphone permission denied or unavailable.");
    }
  }

  const speakCurrentQuestion = useCallback(async () => {
    if (connectionMode !== "fallback") {
      return;
    }

    setVoiceError(null);
    setVoiceInfo(null);

    if (!currentQuestion?.question) {
      return;
    }

    try {
      const token = await getAuthToken();
      if (!token) {
        throw new Error("Authentication token is missing.");
      }

      setIsSpeaking(true);

      const response = await fetch(`${apiBaseUrl}/api/voice/tts`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${token}`,
        },
        body: JSON.stringify({ text: currentQuestion.question }),
      });

      if (!response.ok) {
        const payload = (await response.json().catch(() => ({}))) as { error?: string };
        throw new Error(payload.error ?? "Failed to generate voice output.");
      }

      const audioBlob = await response.blob();
      const audioUrl = URL.createObjectURL(audioBlob);

      stopQuestionAudio();

      const player = new Audio(audioUrl);
      audioPlayerRef.current = player;
      audioUrlRef.current = audioUrl;

      player.onended = () => {
        setIsSpeaking(false);

        if (audioUrlRef.current) {
          URL.revokeObjectURL(audioUrlRef.current);
          audioUrlRef.current = null;
        }
      };

      player.onerror = () => {
        setIsSpeaking(false);
        setVoiceError("Unable to play generated voice.");

        if (audioUrlRef.current) {
          URL.revokeObjectURL(audioUrlRef.current);
          audioUrlRef.current = null;
        }
      };

      appendTranscript("agent", currentQuestion.question);
      await player.play();
    } catch (ttsError) {
      setIsSpeaking(false);
      setVoiceError(ttsError instanceof Error ? ttsError.message : "Unable to read the question aloud.");
    }
  }, [apiBaseUrl, appendTranscript, connectionMode, currentQuestion?.question, stopQuestionAudio]);

  useEffect(() => {
    if (connectionMode !== "fallback" || !isCallActive || !currentQuestion?.question) {
      return;
    }

    if (lastSpokenQuestionRef.current === currentQuestion.question) {
      return;
    }

    lastSpokenQuestionRef.current = currentQuestion.question;
    void speakCurrentQuestion();
  }, [connectionMode, currentQuestion?.question, isCallActive, speakCurrentQuestion]);

  function retryAgentConnection() {
    if (connectionMode !== "fallback") {
      return;
    }

    setConnectionMode("initializing");
    setVoiceError(null);
    setVoiceInfo("Retrying AI interviewer connection...");
  }

  function endCall() {
    stopVoiceInput();
    stopQuestionAudio();
    setIsCallActive(false);
    setIsListening(false);
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

  const speakingState = connectionMode === "agent" ? agentMode === "speaking" : isSpeaking;
  const listeningState = connectionMode === "agent" ? agentMode === "listening" : isListening;

  const statusLabel = connectionMode === "agent"
    ? agentMode === "speaking"
      ? "Agent speaking"
      : "Listening"
    : connectionMode === "initializing"
      ? "Connecting"
      : isSpeaking
        ? "Agent speaking (fallback)"
        : isListening
          ? "Listening (fallback)"
          : "Fallback ready";

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
            Prepare interview questions first from the voice setup page, then press Start call.
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
            <Button variant="secondary" onClick={openVoiceSetup}>
              <ArrowLeft className="mr-2 h-4 w-4" />
              Back
            </Button>
            <p className="text-sm text-[var(--color-text-muted)]">
              {connectionMode === "agent" ? "ElevenLabs realtime interviewer" : "Classic voice fallback"}
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
              <>
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

                <p className="text-xs text-white/60">
                  Answer processing is automatic. Keep speaking naturally and the app evaluates each response.
                </p>

                <div className="flex flex-wrap gap-2">
                  <Button variant="secondary" onClick={endCall} className="border-red-400/50 text-red-200 hover:border-red-300/70">
                    <PhoneOff className="mr-2 h-4 w-4" />
                    End call
                  </Button>
                </div>
              </>
            ) : (
              <>
                <TextArea
                  value={answer}
                  onChange={(event) => setAnswer(event.target.value)}
                  placeholder="Your answer will appear here..."
                  className="min-h-28"
                />

                <div className="flex flex-wrap gap-2">
                  <Button
                    variant="secondary"
                    onClick={isListening ? stopVoiceInput : startVoiceInput}
                    disabled={!isCallActive || isSpeaking || loading}
                  >
                    {isListening ? <MicOff className="mr-2 h-4 w-4" /> : <Mic className="mr-2 h-4 w-4" />}
                    {isListening ? "Stop talking" : "Talk"}
                  </Button>

                  <Button
                    variant="secondary"
                    onClick={() => void speakCurrentQuestion()}
                    disabled={!isCallActive || isSpeaking || isListening || loading}
                  >
                    <Volume2 className="mr-2 h-4 w-4" />
                    {isSpeaking ? "Reading..." : "Replay"}
                  </Button>

                  <Button onClick={() => void submitCurrentAnswer()} disabled={loading || !answer.trim()}>
                    <Send className="mr-2 h-4 w-4" />
                    {loading ? "Submitting..." : "Submit answer"}
                  </Button>

                  <Button variant="secondary" onClick={goToNextQuestion} disabled={loading || isLastQuestion}>
                    <SkipForward className="mr-2 h-4 w-4" />
                    Next question
                  </Button>

                  {isLastQuestion && !sessionCompleted && (
                    <Button variant="secondary" onClick={() => void completeSession()} disabled={loading}>
                      <CheckCircle className="mr-2 h-4 w-4" />
                      Finish session
                    </Button>
                  )}

                  <Button variant="secondary" onClick={retryAgentConnection} disabled={loading}>
                    Retry AI agent
                  </Button>

                  <Button variant="secondary" onClick={endCall} className="border-red-400/50 text-red-200 hover:border-red-300/70">
                    <PhoneOff className="mr-2 h-4 w-4" />
                    End call
                  </Button>
                </div>
              </>
            )}

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

"use client";

import { Conversation, type Mode, type Status } from "@elevenlabs/client";
import { ArrowLeft, Mic, MicOff, PhoneOff, Sparkles } from "lucide-react";
import { useRouter, useSearchParams } from "next/navigation";
import { Suspense, useCallback, useEffect, useMemo, useRef, useState } from "react";

import { useLanguage } from "@/components/providers/LanguageProvider";
import { Button } from "@/components/ui/Button";
import { Card } from "@/components/ui/Card";
import { api } from "@/lib/api/endpoints";
import { pickLocaleText } from "@/lib/i18n";
import type { InterviewLanguage } from "@/lib/api/types";

type TranscriptItem = {
  id: string;
  role: "user" | "agent";
  message: string;
};

function getRoleLabel(locale: "id" | "en", role: "user" | "agent"): string {
  if (role === "agent") {
    return pickLocaleText(locale, "Coach", "Coach");
  }

  return pickLocaleText(locale, "Anda", "You");
}

function getSessionTypeLabel(locale: "id" | "en", sessionType: "review" | "recovery"): string {
  return sessionType === "recovery"
    ? pickLocaleText(locale, "Post-interview recovery", "Post-interview recovery")
    : pickLocaleText(locale, "Review biasa", "Standard review");
}

function getAgentStatusLabel(locale: "id" | "en", status: Status): string {
  if (status === "connected") {
    return pickLocaleText(locale, "Terhubung", "Connected");
  }
  if (status === "connecting") {
    return pickLocaleText(locale, "Menghubungkan", "Connecting");
  }
  if (status === "disconnected") {
    return pickLocaleText(locale, "Terputus", "Disconnected");
  }

  return status;
}

function getAgentModeLabel(locale: "id" | "en", mode: Mode): string {
  if (mode === "listening") {
    return pickLocaleText(locale, "Mendengar", "Listening");
  }
  if (mode === "speaking") {
    return pickLocaleText(locale, "Berbicara", "Speaking");
  }

  return mode;
}

function normalizeTranscript(value: string): string {
  return value
    .replace(/\[[^\]]*\]/g, " ")
    .replace(/\s+/g, " ")
    .trim();
}

function serializeTranscript(items: TranscriptItem[]): string {
  return items
    .map((item) => `${item.role === "agent" ? "Coach" : "You"}: ${item.message}`)
    .join("\n");
}

function resolveInterviewLanguage(value: string | null): InterviewLanguage {
  return value === "en" ? "en" : "id";
}

function ReviewVoiceCallPageContent() {
  const router = useRouter();
  const searchParams = useSearchParams();
  const { locale } = useLanguage();

  const sessionType = searchParams.get("session_type") === "recovery" ? "recovery" : "review";
  const interviewLanguage = resolveInterviewLanguage(searchParams.get("interview_language"));
  const targetRole = (searchParams.get("target_role") ?? "").trim();
  const targetCompany = (searchParams.get("target_company") ?? "").trim();
  const interviewPrompt = (searchParams.get("interview_prompt") ?? "").trim();

  const conversationRef = useRef<Conversation | null>(null);
  const isCallActiveRef = useRef(true);
  const callStartedAtRef = useRef<number>(Date.now());
  const finalizingRef = useRef(false);

  const [isCallActive, setIsCallActive] = useState(true);
  const [isMuted, setIsMuted] = useState(false);
  const [agentStatus, setAgentStatus] = useState<Status>("disconnected");
  const [agentMode, setAgentMode] = useState<Mode>("listening");
  const [voiceInfo, setVoiceInfo] = useState<string | null>(pickLocaleText(locale, "Menghubungkan ke coach AI...", "Connecting to AI coach..."));
  const [voiceError, setVoiceError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);
  const [transcriptItems, setTranscriptItems] = useState<TranscriptItem[]>([]);

  const buildAgentContextInstruction = useCallback(() => {
    const languageInstruction = interviewLanguage === "id"
      ? "Gunakan Bahasa Indonesia saja untuk seluruh respons percakapan."
      : "Use English only for all conversational responses.";

    return [
      "You are an AI career coach for a live voice review call.",
      languageInstruction,
      `Session type: ${sessionType}.`,
      `Target role: ${targetRole || "not provided"}.`,
      `Target company: ${targetCompany || "not provided"}.`,
      `Interview prompt: ${interviewPrompt || "not provided"}.`,
      "Ask one focused follow-up at a time and wait for user response.",
      "Keep responses natural, concise, and actionable.",
      "Do not produce JSON or machine-readable output while speaking.",
    ].join(" ");
  }, [interviewLanguage, interviewPrompt, sessionType, targetCompany, targetRole]);

  useEffect(() => {
    isCallActiveRef.current = isCallActive;
  }, [isCallActive]);

  const appendTranscript = useCallback((role: "user" | "agent", message: string) => {
    const normalized = normalizeTranscript(message);
    if (!normalized) {
      return;
    }

    setTranscriptItems((previous) => ([
      ...previous,
      {
        id: `${Date.now()}-${Math.random().toString(36).slice(2, 8)}`,
        role,
        message: normalized,
      },
    ]).slice(-80));
  }, []);

  const finalizeVoiceReview = useCallback(async () => {
    if (finalizingRef.current) {
      return;
    }

    finalizingRef.current = true;
    setSubmitting(true);
    setVoiceError(null);

    try {
      const transcriptText = serializeTranscript(transcriptItems);
      if (!transcriptText.trim()) {
        throw new Error(pickLocaleText(locale, "Belum ada transkrip voice untuk direview.", "No voice transcript captured for review yet."));
      }

      const review = await api.startReview({
        session_type: sessionType,
        input_mode: "voice",
        interview_language: interviewLanguage,
        transcript_text: transcriptText,
        interview_prompt: interviewPrompt,
        target_role: targetRole,
        target_company: targetCompany,
      });

      const elapsedSeconds = Math.max(1, Math.round((Date.now() - callStartedAtRef.current) / 1000));
      try {
        await api.commitVoiceUsage(review.session.id, elapsedSeconds);
      } catch {
        // non-blocking
      }

      await api.endReview(review.session.id);
      router.push("/review");
    } catch (err) {
      setVoiceError(err instanceof Error ? err.message : pickLocaleText(locale, "Gagal menyelesaikan voice review.", "Failed to finish voice review."));
    } finally {
      setSubmitting(false);
      finalizingRef.current = false;
    }
  }, [interviewLanguage, interviewPrompt, locale, router, sessionType, targetCompany, targetRole, transcriptItems]);

  const endConversation = useCallback(async () => {
    setIsCallActive(false);
    const conversation = conversationRef.current;
    conversationRef.current = null;
    if (!conversation) {
      return;
    }

    try {
      await conversation.endSession();
    } catch {
      // ignored
    }
  }, []);

  const handleEndCall = useCallback(async () => {
    await endConversation();
    await finalizeVoiceReview();
  }, [endConversation, finalizeVoiceReview]);

  useEffect(() => {
    let cancelled = false;

    void (async () => {
      try {
        const agentSession = await api.createReviewVoiceAgentSession();
        if (cancelled) {
          return;
        }

        const conversation = await Conversation.startSession({
          signedUrl: agentSession.signed_url,
          connectionType: "websocket",
          overrides: {
            agent: {
              language: interviewLanguage,
            },
          },
          dynamicVariables: {
            interview_language: interviewLanguage,
          },
          onConnect: () => {
            setVoiceInfo(pickLocaleText(locale, "Terhubung. Ceritakan pengalaman interview Anda.", "Connected. Share your interview experience."));
            try {
              conversationRef.current?.sendContextualUpdate(buildAgentContextInstruction());
            } catch {
              setVoiceError(pickLocaleText(locale, "Gagal mengirim preferensi bahasa ke coach.", "Failed to send language preferences to coach."));
            }
          },
          onStatusChange: ({ status }) => {
            setAgentStatus(status);
          },
          onModeChange: ({ mode }) => {
            setAgentMode(mode);
          },
          onError: (message) => {
            setVoiceError(message || pickLocaleText(locale, "Koneksi ke agen bermasalah.", "Agent connection error."));
          },
          onMessage: ({ role, message }) => {
            if (role === "agent" || role === "user") {
              appendTranscript(role, message);
            }
          },
          onDisconnect: () => {
            if (!isCallActiveRef.current || finalizingRef.current) {
              return;
            }

            void finalizeVoiceReview();
          },
        });

        conversationRef.current = conversation;
        callStartedAtRef.current = Date.now();
      } catch (err) {
        if (cancelled) {
          return;
        }

        setVoiceError(err instanceof Error ? err.message : pickLocaleText(locale, "Gagal terhubung ke coach voice.", "Failed to connect to voice coach."));
      }
    })();

    return () => {
      cancelled = true;
      void endConversation();
    };
  }, [appendTranscript, buildAgentContextInstruction, endConversation, finalizeVoiceReview, interviewLanguage, locale]);

  const transcriptPreview = useMemo(() => transcriptItems.slice(-14), [transcriptItems]);

  return (
    <div className="min-h-screen bg-[#070A14] px-4 py-6 text-white md:px-8">
      <div className="mx-auto w-full max-w-5xl space-y-5">
        <div className="flex items-center justify-between gap-3">
          <Button variant="secondary" type="button" onClick={() => router.push("/review")}> 
            <ArrowLeft className="mr-2 h-4 w-4" />
            {pickLocaleText(locale, "Kembali", "Back")}
          </Button>
          <p className="text-xs text-white/50">{pickLocaleText(locale, "Mode: Voice Review Coach", "Mode: Voice Review Coach")}</p>
        </div>

        <Card className="space-y-5 p-6">
          <div className="flex flex-wrap items-center justify-between gap-3">
            <div>
              <h1 className="text-lg font-semibold text-white">{pickLocaleText(locale, "Voice Review Call", "Voice Review Call")}</h1>
              <p className="mt-1 text-sm text-white/65">
                {pickLocaleText(locale, "Sampaikan pengalaman interview Anda secara natural. Coach akan memberi arahan lalu hasil review disimpan otomatis.", "Share your interview experience naturally. The coach guides you, then review results are saved automatically.")}
              </p>
            </div>
            <div className="rounded-full border border-white/10 bg-white/5 px-3 py-1 text-xs text-white/70">
              {getAgentStatusLabel(locale, agentStatus)} · {getAgentModeLabel(locale, agentMode)}
            </div>
          </div>

          {voiceInfo && <p className="text-sm text-cyan-200">{voiceInfo}</p>}
          {voiceError && <p className="text-sm text-red-300">{voiceError}</p>}

          <div className="grid gap-4 md:grid-cols-4">
            <div className="rounded-xl border border-white/10 bg-white/5 p-4">
              <p className="text-xs text-white/55">{pickLocaleText(locale, "Session type", "Session type")}</p>
              <p className="mt-1 text-sm font-medium text-white">{getSessionTypeLabel(locale, sessionType)}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/5 p-4">
              <p className="text-xs text-white/55">{pickLocaleText(locale, "Interview language", "Interview language")}</p>
              <p className="mt-1 text-sm font-medium text-white">{interviewLanguage === "id" ? "Bahasa Indonesia" : "English"}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/5 p-4">
              <p className="text-xs text-white/55">{pickLocaleText(locale, "Target role", "Target role")}</p>
              <p className="mt-1 text-sm font-medium text-white">{targetRole || "-"}</p>
            </div>
            <div className="rounded-xl border border-white/10 bg-white/5 p-4">
              <p className="text-xs text-white/55">{pickLocaleText(locale, "Target company", "Target company")}</p>
              <p className="mt-1 text-sm font-medium text-white">{targetCompany || "-"}</p>
            </div>
          </div>

          <div className="flex flex-wrap gap-2">
            <Button
              type="button"
              variant={isMuted ? "secondary" : "primary"}
              onClick={async () => {
                const next = !isMuted;
                setIsMuted(next);
                try {
                  conversationRef.current?.setVolume?.({ volume: next ? 0 : 1 });
                } catch {
                  setVoiceError(pickLocaleText(locale, "Gagal memperbarui status mikrofon.", "Failed to update microphone state."));
                }
              }}
              disabled={!isCallActive || submitting}
            >
              {isMuted ? <MicOff className="mr-2 h-4 w-4" /> : <Mic className="mr-2 h-4 w-4" />}
              {isMuted ? pickLocaleText(locale, "Unmute", "Unmute") : pickLocaleText(locale, "Mute", "Mute")}
            </Button>

            <Button type="button" onClick={() => void handleEndCall()} disabled={!isCallActive || submitting}>
              <PhoneOff className="mr-2 h-4 w-4" />
              {submitting ? pickLocaleText(locale, "Menyimpan review...", "Saving review...") : pickLocaleText(locale, "Akhiri & Simpan Review", "End & Save Review")}
            </Button>
          </div>
        </Card>

        <Card className="p-5">
          <div className="mb-3 flex items-center gap-2 text-sm font-medium text-white">
            <Sparkles className="h-4 w-4 text-purple-300" />
            {pickLocaleText(locale, "Live transcript", "Live transcript")}
          </div>
          <div className="max-h-95 space-y-2 overflow-y-auto pr-1">
            {transcriptPreview.length === 0 && (
              <p className="text-sm text-white/55">{pickLocaleText(locale, "Menunggu percakapan dimulai...", "Waiting for conversation to begin...")}</p>
            )}
            {transcriptPreview.map((item) => (
              <div key={item.id} className="rounded-xl border border-white/10 bg-white/5 px-3 py-2 text-sm">
                <p className="mb-1 text-xs text-white/45">{getRoleLabel(locale, item.role)}</p>
                <p className="text-white/90">{item.message}</p>
              </div>
            ))}
          </div>
        </Card>
      </div>
    </div>
  );
}

export default function ReviewVoiceCallPage() {
  return (
    <Suspense fallback={<ReviewVoiceCallPageFallback />}>
      <ReviewVoiceCallPageContent />
    </Suspense>
  );
}

function ReviewVoiceCallPageFallback() {
  const { locale } = useLanguage();

  return (
    <div className="min-h-screen bg-[#070A14] px-4 py-6 text-white md:px-8">
      <div className="mx-auto w-full max-w-5xl space-y-5">
        <Card className="p-6">
          <p className="text-sm text-white/70">{pickLocaleText(locale, "Memuat voice review...", "Loading voice review...")}</p>
        </Card>
      </div>
    </div>
  );
}

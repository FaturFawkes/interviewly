import { apiRequest } from "@/lib/api/client";
import { getAuthToken } from "@/lib/auth/token-provider";
import type {
  AnalyticsOverview,
  FeedbackRecord,
  InterviewDifficulty,
  GenerateQuestionsResponse,
  InterviewLanguage,
  InterviewMode,
  ParsedJobDescription,
  PracticeSession,
  ProgressMetrics,
  ResumeAnalysisResponse,
  ResumeRecord,
  SessionStartMetadata,
  SessionAnswer,
  SessionHistoryResponse,
  VoiceAgentSession,
} from "@/lib/api/types";

const API_BASE_URL = process.env.NEXT_PUBLIC_API_BASE_URL ?? "http://localhost:8080";

export const api = {
  parseJobDescription: (jobDescription: string): Promise<ParsedJobDescription> =>
    apiRequest("/api/job/parse", {
      method: "POST",
      body: { job_description: jobDescription },
    }),

  saveResume: (resumeText: string, file?: File): Promise<ResumeRecord> => {
    if (file) {
      const payload = new FormData();
      payload.append("content", resumeText);
      payload.append("file", file);

      return apiRequest("/api/resume", {
        method: "POST",
        body: payload,
      });
    }

    return apiRequest("/api/resume", {
      method: "POST",
      body: { content: resumeText },
    });
  },

  getLatestResume: (): Promise<ResumeRecord> => apiRequest("/api/resume"),

  analyzeResume: (resumeText: string, file?: File): Promise<ResumeAnalysisResponse> => {
    if (file) {
      const payload = new FormData();
      payload.append("content", resumeText);
      payload.append("file", file);

      return apiRequest("/api/resume/analyze", {
        method: "POST",
        body: payload,
      });
    }

    return apiRequest("/api/resume/analyze", {
      method: "POST",
      body: { content: resumeText },
    });
  },

  downloadLatestResume: async (): Promise<void> => {
    if (typeof window === "undefined") {
      throw new Error("Download can only run in browser.");
    }

    const token = await getAuthToken();
    const headers = new Headers();
    if (token) {
      headers.set("Authorization", `Bearer ${token}`);
    }

    const response = await fetch(`${API_BASE_URL}/api/resume/download`, {
      method: "GET",
      headers,
      cache: "no-store",
    });

    if (!response.ok) {
      const payload = (await response.json().catch(() => ({}))) as { error?: string };
      throw new Error(payload.error ?? "Failed to download CV.");
    }

    const blob = await response.blob();
    const downloadUrl = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = downloadUrl;
    link.download = extractDownloadFileName(response.headers.get("Content-Disposition"), "resume-download");
    document.body.append(link);
    link.click();
    link.remove();
    URL.revokeObjectURL(downloadUrl);
  },

  generateQuestions: (
    resumeText: string,
    jobDescription: string,
    interviewLanguage: InterviewLanguage,
    interviewMode: InterviewMode,
    interviewDifficulty: InterviewDifficulty,
  ): Promise<GenerateQuestionsResponse> =>
    apiRequest("/api/questions/generate", {
      method: "POST",
      body: {
        resume_text: resumeText,
        job_description: jobDescription,
        interview_language: interviewLanguage,
        interview_mode: interviewMode,
        interview_difficulty: interviewDifficulty,
      },
    }),

  startInterviewSession: (
    resumeID: string,
    jobParseID: string,
    questionIDs: string[],
    metadata?: SessionStartMetadata,
  ): Promise<PracticeSession> =>
    apiRequest("/api/session/start", {
      method: "POST",
      body: {
        resume_id: resumeID,
        job_parse_id: jobParseID,
        question_ids: questionIDs,
        ...(metadata ?? {}),
      },
    }),

  submitInterviewAnswer: (sessionID: string, questionID: string, answer: string): Promise<SessionAnswer> =>
    apiRequest("/api/session/answer", {
      method: "POST",
      body: { session_id: sessionID, question_id: questionID, answer },
    }),

  completeInterviewSession: (sessionID: string): Promise<PracticeSession> =>
    apiRequest("/api/session/complete", {
      method: "POST",
      body: { session_id: sessionID },
    }),

  generateFeedback: (
    sessionID: string,
    questionID: string,
    question: string,
    answer: string,
    interviewLanguage: InterviewLanguage,
  ): Promise<FeedbackRecord> =>
    apiRequest("/api/feedback/generate", {
      method: "POST",
      body: {
        session_id: sessionID,
        question_id: questionID,
        question,
        answer,
        interview_language: interviewLanguage,
      },
    }),

  getProgress: (): Promise<ProgressMetrics> => apiRequest("/api/progress"),

  getAnalyticsOverview: (): Promise<AnalyticsOverview> => apiRequest("/api/analytics/overview"),

  getSessionHistory: (): Promise<SessionHistoryResponse> => apiRequest("/api/session/history"),

  createVoiceAgentSession: (includeConversationID = false): Promise<VoiceAgentSession> =>
    apiRequest("/api/voice/agent/session", {
      method: "POST",
      body: { include_conversation_id: includeConversationID },
    }),
};

function extractDownloadFileName(contentDisposition: string | null, fallback: string): string {
  if (!contentDisposition) {
    return fallback;
  }

  const utf8NameMatch = contentDisposition.match(/filename\*=UTF-8''([^;]+)/i);
  if (utf8NameMatch?.[1]) {
    const decoded = decodeURIComponent(utf8NameMatch[1]).trim();
    if (decoded) {
      return decoded;
    }
  }

  const nameMatch = contentDisposition.match(/filename="?([^";]+)"?/i);
  if (nameMatch?.[1]) {
    const cleaned = nameMatch[1].trim();
    if (cleaned) {
      return cleaned;
    }
  }

  return fallback;
}

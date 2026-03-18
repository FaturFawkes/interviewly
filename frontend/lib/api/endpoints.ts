import { apiRequest } from "@/lib/api/client";
import type {
  AgentFeedbackPayload,
  FeedbackRecord,
  GenerateQuestionsResponse,
  InterviewDifficulty,
  InterviewLanguage,
  InterviewMode,
  ParsedJobDescription,
  ResumeAIAnalysis,
  PracticeSession,
  ProgressMetrics,
  ResumeRecord,
  SessionStartMetadata,
  SessionAnswer,
  SessionHistoryResponse,
  AnalyticsOverview,
  PaymentCheckoutSession,
  PaymentPlanID,
  SubscriptionStatus,
  VoiceAgentSession,
  VoiceUsageCommitResponse,
} from "@/lib/api/types";

export const api = {
  parseJobDescription: (jobDescription: string): Promise<ParsedJobDescription> =>
    apiRequest("/api/job/parse", {
      method: "POST",
      body: { job_description: jobDescription },
    }),

  saveResume: (resumeText: string): Promise<ResumeRecord> =>
    apiRequest("/api/resume", {
      method: "POST",
      body: { content: resumeText },
    }),

  getLatestResume: (): Promise<ResumeRecord> =>
    apiRequest("/api/resume", {
      method: "GET",
    }),

  analyzeResume: (resumeText?: string): Promise<ResumeAIAnalysis> =>
    apiRequest("/api/resume/analyze", {
      method: "POST",
      body: { content: resumeText ?? "" },
    }),

  generateQuestions: (
    resumeText: string,
    jobDescription: string,
    interviewLanguage: InterviewLanguage = "en",
    interviewMode: InterviewMode = "text",
    interviewDifficulty: InterviewDifficulty = "medium",
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

  completeInterviewSession: (sessionID: string): Promise<PracticeSession> =>
    apiRequest("/api/session/complete", {
      method: "POST",
      body: { session_id: sessionID },
    }),

  submitInterviewAnswer: (sessionID: string, questionID: string, answer: string): Promise<SessionAnswer> =>
    apiRequest("/api/session/answer", {
      method: "POST",
      body: { session_id: sessionID, question_id: questionID, answer },
    }),

  generateFeedback: (sessionID: string, questionID: string, question: string, answer: string): Promise<FeedbackRecord> =>
    apiRequest("/api/feedback/generate", {
      method: "POST",
      body: { session_id: sessionID, question_id: questionID, question, answer },
    }),

  createVoiceAgentSession: (includeConversationID = false): Promise<VoiceAgentSession> =>
    apiRequest("/api/voice/agent/session", {
      method: "POST",
      body: { include_conversation_id: includeConversationID },
    }),

  commitVoiceUsage: (sessionID: string, elapsedSeconds: number): Promise<VoiceUsageCommitResponse> =>
    apiRequest("/api/voice/usage/commit", {
      method: "POST",
      body: {
        session_id: sessionID,
        elapsed_seconds: elapsedSeconds,
      },
    }),

  submitAgentFeedback: (payload: AgentFeedbackPayload): Promise<FeedbackRecord> =>
    apiRequest("/api/feedback/agent", {
      method: "POST",
      body: payload,
    }),

  getAnalyticsOverview: (): Promise<AnalyticsOverview> => apiRequest("/api/analytics/overview"),

  getProgress: (): Promise<ProgressMetrics> => apiRequest("/api/progress"),

  getSessionHistory: (): Promise<SessionHistoryResponse> => apiRequest("/api/session/history"),

  getSubscriptionStatus: (): Promise<SubscriptionStatus> => apiRequest("/api/subscription/status"),

  createCheckoutSession: (planID: PaymentPlanID): Promise<PaymentCheckoutSession> =>
    apiRequest("/payments/checkout", {
      method: "POST",
      body: { plan_id: planID },
    }),
};

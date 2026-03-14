import { apiRequest } from "@/lib/api/client";
import type {
  AnalyticsOverview,
  FeedbackRecord,
  GenerateQuestionsResponse,
  ParsedJobDescription,
  PracticeSession,
  ProgressMetrics,
  ResumeRecord,
  SessionStartMetadata,
  SessionAnswer,
  SessionHistoryResponse,
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

  generateQuestions: (resumeText: string, jobDescription: string): Promise<GenerateQuestionsResponse> =>
    apiRequest("/api/questions/generate", {
      method: "POST",
      body: { resume_text: resumeText, job_description: jobDescription },
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

  generateFeedback: (sessionID: string, questionID: string, question: string, answer: string): Promise<FeedbackRecord> =>
    apiRequest("/api/feedback/generate", {
      method: "POST",
      body: { session_id: sessionID, question_id: questionID, question, answer },
    }),

  getProgress: (): Promise<ProgressMetrics> => apiRequest("/api/progress"),

  getAnalyticsOverview: (): Promise<AnalyticsOverview> => apiRequest("/api/analytics/overview"),

  getSessionHistory: (): Promise<SessionHistoryResponse> => apiRequest("/api/session/history"),
};

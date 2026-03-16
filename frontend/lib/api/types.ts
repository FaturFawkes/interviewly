export type JobInsights = {
  skills: string[];
  keywords: string[];
  themes: string[];
  seniority: string;
};

export type ParsedJobDescription = {
  id: string;
  user_id: string;
  raw_description: string;
  insights: JobInsights;
  created_at: string;
};

export type ResumeRecord = {
  id: string;
  user_id: string;
  content: string;
  created_at: string;
};

export type StoredQuestion = {
  id: string;
  user_id: string;
  resume_id: string;
  job_parse_id: string;
  type: string;
  question: string;
  created_at: string;
};

export type GenerateQuestionsResponse = {
  questions: StoredQuestion[];
  resume_id?: string;
  job_parse_id?: string;
};

export type InterviewMode = "text" | "voice";
export type InterviewLanguage = "en" | "id";
export type InterviewDifficulty = "easy" | "medium" | "hard";

export type SessionStartMetadata = {
  interview_mode?: InterviewMode;
  interview_language?: InterviewLanguage;
  interview_difficulty?: InterviewDifficulty;
  target_role?: string;
  target_company?: string;
};

export type PracticeSession = {
  id: string;
  user_id: string;
  resume_id: string;
  job_parse_id: string;
  question_ids: string[];
  interview_mode: InterviewMode;
  interview_language: InterviewLanguage;
  interview_difficulty: InterviewDifficulty;
  target_role?: string;
  target_company?: string;
  status: "active" | "completed" | "abandoned";
  score: number;
  created_at: string;
  completed_at?: string;
};

export type SessionHistoryResponse = {
  sessions: PracticeSession[];
};

export type VoiceAgentSession = {
  signed_url: string;
  conversation_id?: string;
};

export type AgentFeedbackPayload = {
  session_id: string;
  question_id: string;
  question: string;
  answer: string;
  score: number;
  strengths: string[];
  weaknesses: string[];
  improvements: string[];
  star_feedback: string;
};

export type SessionAnswer = {
  id: string;
  session_id: string;
  question_id: string;
  user_id: string;
  answer: string;
  created_at: string;
};

export type FeedbackRecord = {
  id: string;
  user_id: string;
  session_id: string;
  question_id: string;
  question: string;
  answer: string;
  score: number;
  strengths: string[];
  weaknesses: string[];
  improvements: string[];
  star_feedback: string;
  created_at: string;
};

export type ProgressMetrics = {
  user_id: string;
  average_score: number;
  weak_areas: string[];
  sessions_completed: number;
  updated_at: string;
};

export type AnalyticsOverview = {
  average_score: number;
  sessions_completed: number;
  weak_areas: string[];
  recent_sessions: PracticeSession[];
};

export type APIError = {
  error: string;
};

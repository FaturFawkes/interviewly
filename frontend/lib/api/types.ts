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

export type ResumeAIAnalysis = {
  summary: string;
  response: string;
  highlights: string[];
  recommendations: string[];
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
  total_voice_minutes?: number;
  used_voice_minutes?: number;
  remaining_voice_minutes?: number;
  allowed_call_seconds?: number;
  warning_threshold_reached?: boolean;
  voice_quota_message?: string;
};

export type VoiceUsageCommitResponse = {
  total_voice_minutes: number;
  used_voice_minutes: number;
  remaining_voice_minutes: number;
  period_start: string;
  period_end: string;
};

export type SubscriptionStatus = {
  plan_id: string;
  is_free_tier: boolean;
  total_voice_minutes: number;
  used_voice_minutes: number;
  remaining_voice_minutes: number;
  total_sessions: number;
  used_sessions: number;
  remaining_sessions: number;
  total_text_requests: number;
  used_text_requests: number;
  remaining_text_requests: number;
  text_fup_exceeded: boolean;
  should_slowdown_response: boolean;
  suggested_downgrade_model?: string;
  total_jd_limit: number;
  used_jd_parses: number;
  remaining_jd_parses: number;
  total_voice_topup_minutes: number;
  used_voice_topup_minutes: number;
  remaining_voice_topup_minutes: number;
  trial_available: boolean;
  trial_active: boolean;
  trial_duration_hours: number;
  trial_bonus_voice_minutes: number;
  trial_ends_at?: string;
  trigger_required_sessions: number;
  trigger_progress_sessions: number;
  upsell_messages: string[];
  anti_abuse_rules: string[];
};

export type PaymentPlanID = "starter" | "pro" | "elite";
export type VoiceTopupPackageCode = "voice_topup_10" | "voice_topup_30";
export type PaymentCheckoutType = "subscription" | "voice_topup";

export type PaymentCheckoutSession = {
  checkout_url: string;
  checkout_session_id?: string;
  checkout_type: PaymentCheckoutType;
  plan_id?: PaymentPlanID;
  package_code?: VoiceTopupPackageCode;
  voice_minutes?: number;
  currency: string;
  amount_cents: number;
};

export type SessionHeartbeatResponse = {
  session_id: string;
  status: "ok";
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

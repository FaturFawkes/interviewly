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

export type PaymentCheckoutSession = {
  checkout_url: string;
  plan_id: PaymentPlanID;
  currency: string;
  amount_cents: number;
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

export type ReviewFeedback = {
  score: number;
  communication: number;
  structure_star: number;
  confidence: number;
  strengths: string[];
  weaknesses: string[];
  suggestions: string[];
  better_answer: string;
  insight: string;
  follow_up_question: string;
  recovery_simulation?: string;
};

export type ReviewSession = {
  id: string;
  user_id: string;
  session_type: "review" | "recovery";
  input_mode: InterviewMode;
  input_text?: string;
  voice_url?: string;
  transcript_text?: string;
  role_target?: string;
  company_target?: string;
  status: "active" | "completed" | "abandoned";
  feedback: ReviewFeedback;
  created_at: string;
  updated_at: string;
  completed_at?: string;
};

export type ImprovementPlan = {
  focus_areas: string[];
  practice_plan: string[];
  weekly_target: string;
  next_session_focus: string;
};

export type ReviewStartPayload = {
  session_type?: "review" | "recovery";
  input_mode?: InterviewMode;
  input_text?: string;
  voice_url?: string;
  transcript_text?: string;
  interview_prompt?: string;
  target_role?: string;
  target_company?: string;
};

export type ReviewRespondPayload = {
  session_id: string;
  input_text?: string;
  voice_url?: string;
  transcript_text?: string;
  interview_prompt?: string;
};

export type ReviewResponse = {
  session: ReviewSession;
  feedback: ReviewFeedback;
  score: number;
  improvement_tips: string[];
};

export type ReviewEndResponse = {
  session_id: string;
  feedback: ReviewFeedback;
  score: number;
  improvement_tips: string[];
  improvement_plan: ImprovementPlan;
  coaching_summary: string;
};

export type ReviewProgressPoint = {
  review_session_id?: string;
  communication: number;
  structure_star: number;
  confidence: number;
  overall_score: number;
  notes?: string;
  created_at: string;
};

export type ReviewProgress = {
  user_id: string;
  communication_trend: ReviewProgressPoint[];
  structure_trend: ReviewProgressPoint[];
  confidence_trend: ReviewProgressPoint[];
  latest_overall_score: number;
  average_overall_score: number;
};

export type ProgressResponse = {
  interview_progress: ProgressMetrics;
  review_progress: ReviewProgress;
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

# AI Career Coach (Review Mode)

Dokumen ini merangkum implementasi backend + kontrak API untuk fitur **AI Career Coach (Review Mode)**.

## Endpoint

Semua endpoint berada di namespace auth `/api`:

- `POST /api/review/start`
- `POST /api/review/respond`
- `POST /api/review/end`
- `GET /api/progress` (sekarang return `interview_progress` + `review_progress`)
- `GET /api/coaching-summary`

## Contoh Request

### `POST /api/review/start`

```json
{
  "session_type": "review",
  "input_mode": "text",
  "input_text": "Saya gagal di interview behavioral karena jawaban saya muter-muter...",
  "interview_prompt": "Tell me about a conflict you handled",
  "target_role": "Backend Engineer",
  "target_company": "Tech Corp"
}
```

### `POST /api/review/respond`

```json
{
  "session_id": "e0d53f56-9d07-4c4d-b4e6-2f26a2856f6d",
  "input_text": "Pertanyaannya tentang conflict, saya jawab soal bug fixing tapi tidak pakai struktur jelas"
}
```

### `POST /api/review/end`

```json
{
  "session_id": "e0d53f56-9d07-4c4d-b4e6-2f26a2856f6d"
}
```

## Contoh Response

### `POST /api/review/start` / `POST /api/review/respond`

```json
{
  "session": {
    "id": "e0d53f56-9d07-4c4d-b4e6-2f26a2856f6d",
    "user_id": "user-123",
    "session_type": "review",
    "input_mode": "text",
    "status": "active",
    "feedback": {
      "score": 72,
      "communication": 74,
      "structure_star": 66,
      "confidence": 70,
      "strengths": ["clear ownership"],
      "weaknesses": ["result not measurable"],
      "suggestions": ["add quantified impact"],
      "better_answer": "Use STAR...",
      "insight": "Main gap is structure + measurable impact",
      "follow_up_question": "What exact result did you achieve?"
    },
    "created_at": "2026-03-20T08:00:00Z",
    "updated_at": "2026-03-20T08:01:00Z"
  },
  "feedback": {
    "score": 72,
    "communication": 74,
    "structure_star": 66,
    "confidence": 70,
    "strengths": ["clear ownership"],
    "weaknesses": ["result not measurable"],
    "suggestions": ["add quantified impact"],
    "better_answer": "Use STAR...",
    "insight": "Main gap is structure + measurable impact",
    "follow_up_question": "What exact result did you achieve?"
  },
  "score": 72,
  "improvement_tips": ["add quantified impact"]
}
```

### `POST /api/review/end`

```json
{
  "session_id": "e0d53f56-9d07-4c4d-b4e6-2f26a2856f6d",
  "feedback": {
    "score": 76,
    "communication": 78,
    "structure_star": 70,
    "confidence": 72,
    "strengths": ["relevance improved"],
    "weaknesses": ["still needs crisper result statement"],
    "suggestions": ["close answer with metric + lesson learned"],
    "better_answer": "Use STAR with quantified result...",
    "insight": "You are improving on structure and clarity",
    "follow_up_question": "Ready to simulate the same question again?"
  },
  "score": 76,
  "improvement_tips": [
    "Record 1 voice reflection using STAR for a failed interview question",
    "Rewrite 2 past answers with quantified results"
  ],
  "improvement_plan": {
    "focus_areas": [
      "STAR structure consistency",
      "clearer confidence language",
      "stronger measurable outcomes"
    ],
    "practice_plan": [
      "Record 1 voice reflection using STAR for a failed interview question",
      "Rewrite 2 past answers with quantified results",
      "Do 1 mock response in under 90 seconds"
    ],
    "weekly_target": "Complete at least 2 review sessions and 1 recovery simulation this week",
    "next_session_focus": "Answer relevance and stronger action/result details"
  },
  "coaching_summary": "You have clear potential. Focus on structure and measurable impact in your next answers."
}
```

## Prompt System (Current)

Prompt sistem Review Mode di backend menargetkan perilaku berikut:

- role: senior career coach (review mode)
- output: strict JSON (machine-parseable)
- constraints:
  - feedback harus spesifik, tidak generic
  - selalu actionable
  - ajukan follow-up question jika data kurang
  - nilai dimensi: communication, structure_star, confidence

Implementasi berada di:

- `backend/internal/service/ai/service.go`
  - `remoteAnalyzeReview(...)`
  - `remoteGenerateImprovementPlan(...)`

## Monetization Rules (Implemented)

Di subscription service:

- **Free**: maksimal 2 review session per periode weekly, text-only
- **Pro/Elite**: review tidak dibatasi hard-limit (mengikuti FUP product policy), voice coaching diizinkan
- Tracking usage type baru:
  - `review_count`
  - `review_voice_minutes` (siap dipakai perluasan commit usage)

Implementasi berada di:

- `backend/internal/service/subscription/service.go`
  - `CanStartReviewSession(...)`
  - `ConsumeReviewSession(...)`
  - `CheckReviewVoiceQuota(...)`

## Schema

Migration baru:

- `backend/migrations/000024_add_review_mode_tables.up.sql`
- `backend/migrations/000024_add_review_mode_tables.down.sql`

Tabel baru:

- `app_review_sessions`
- `app_coaching_memory`
- `app_progress_tracking`

Dan constraint usage type pada `app_usage_tracking` diperluas untuk review mode.

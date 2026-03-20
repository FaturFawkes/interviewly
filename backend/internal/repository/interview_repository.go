package repository

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type interviewRepository struct {
	pool             *pgxpool.Pool
	mu               sync.RWMutex
	parsedJobs       map[string]*domain.ParsedJobDescription
	resumes          map[string]*domain.ResumeRecord
	resumeAnalyses   map[string]domain.ResumeAnalysisRecord
	questions        map[string]domain.StoredQuestion
	sessions         map[string]domain.PracticeSession
	answers          map[string]domain.SessionAnswer
	feedbacks        map[string]domain.FeedbackRecord
	progress         map[string]domain.ProgressMetrics
	reviewSessions   map[string]domain.ReviewSession
	coachingMemory   map[string]domain.CoachingMemory
	progressTracking map[string][]domain.ProgressTrackingPoint
}

// NewInterviewRepository creates interview repository with postgres support and in-memory fallback.
func NewInterviewRepository(pool *pgxpool.Pool) domain.InterviewRepository {
	return &interviewRepository{
		pool:             pool,
		parsedJobs:       make(map[string]*domain.ParsedJobDescription),
		resumes:          make(map[string]*domain.ResumeRecord),
		resumeAnalyses:   make(map[string]domain.ResumeAnalysisRecord),
		questions:        make(map[string]domain.StoredQuestion),
		sessions:         make(map[string]domain.PracticeSession),
		answers:          make(map[string]domain.SessionAnswer),
		feedbacks:        make(map[string]domain.FeedbackRecord),
		progress:         make(map[string]domain.ProgressMetrics),
		reviewSessions:   make(map[string]domain.ReviewSession),
		coachingMemory:   make(map[string]domain.CoachingMemory),
		progressTracking: make(map[string][]domain.ProgressTrackingPoint),
	}
}

func (r *interviewRepository) SaveParsedJob(userID, rawDescription string, insights *domain.JobInsights) (*domain.ParsedJobDescription, error) {
	now := time.Now().UTC()
	parsed := &domain.ParsedJobDescription{
		ID:             uuid.NewString(),
		UserID:         userID,
		RawDescription: rawDescription,
		Insights:       *insights,
		CreatedAt:      now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_job_parses
				(id, user_id, raw_description, skills, keywords, themes, seniority, created_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8)`,
			parsed.ID,
			parsed.UserID,
			parsed.RawDescription,
			parsed.Insights.Skills,
			parsed.Insights.Keywords,
			parsed.Insights.Themes,
			parsed.Insights.Seniority,
			parsed.CreatedAt,
		)
		if err == nil {
			return parsed, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.parsedJobs[parsed.ID] = parsed
	return parsed, nil
}

func (r *interviewRepository) SaveResume(userID, content string) (*domain.ResumeRecord, error) {
	return r.SaveResumeWithFilePath(userID, content, "")
}

func (r *interviewRepository) SaveResumeWithFilePath(userID, content, filePath string) (*domain.ResumeRecord, error) {
	now := time.Now().UTC()
	resume := &domain.ResumeRecord{
		ID:        uuid.NewString(),
		UserID:    userID,
		Content:   content,
		MinIOPath: strings.TrimSpace(filePath),
		CreatedAt: now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_resumes
				(id, user_id, content, minio_path, created_at)
			 VALUES
				($1, $2, $3, $4, $5)`,
			resume.ID,
			resume.UserID,
			resume.Content,
			resume.MinIOPath,
			resume.CreatedAt,
		)
		if err == nil {
			return resume, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.resumes[resume.ID] = resume
	return resume, nil
}

func (r *interviewRepository) GetLatestResume(userID string) (*domain.ResumeRecord, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var resume domain.ResumeRecord
		err := r.pool.QueryRow(
			ctx,
			`SELECT id, user_id, content, minio_path, created_at
			   FROM app_resumes
			  WHERE user_id = $1
			    AND NULLIF(BTRIM(content), '') IS NOT NULL
			  ORDER BY created_at DESC, id DESC
			  LIMIT 1`,
			userID,
		).Scan(
			&resume.ID,
			&resume.UserID,
			&resume.Content,
			&resume.MinIOPath,
			&resume.CreatedAt,
		)
		if err == nil {
			return &resume, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *domain.ResumeRecord
	for _, resume := range r.resumes {
		if resume.UserID != userID {
			continue
		}
		if strings.TrimSpace(resume.Content) == "" {
			continue
		}
		if latest == nil || resume.CreatedAt.After(latest.CreatedAt) {
			copy := *resume
			latest = &copy
		}
	}

	return latest, nil
}

func (r *interviewRepository) SaveResumeAnalysis(userID, resumeID, contentHash, model string, analysis *domain.ResumeAIAnalysis) (*domain.ResumeAnalysisRecord, error) {
	now := time.Now().UTC()
	record := &domain.ResumeAnalysisRecord{
		ID:              uuid.NewString(),
		UserID:          strings.TrimSpace(userID),
		ResumeID:        strings.TrimSpace(resumeID),
		ContentHash:     strings.TrimSpace(contentHash),
		Model:           strings.TrimSpace(model),
		Summary:         strings.TrimSpace(analysis.Summary),
		Response:        strings.TrimSpace(analysis.Response),
		Highlights:      append([]string{}, analysis.Highlights...),
		Recommendations: append([]string{}, analysis.Recommendations...),
		CreatedAt:       now,
	}

	rawResponse, marshalErr := json.Marshal(analysis)
	if marshalErr != nil {
		rawResponse = []byte("{}")
	}
	record.RawResponse = string(rawResponse)

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var scanned domain.ResumeAnalysisRecord
		var rawJSON []byte
		err := r.pool.QueryRow(
			ctx,
			`INSERT INTO app_resume_analyses
				(id, user_id, resume_id, content_hash, model, summary, response, highlights, recommendations, raw_response, created_at)
			 VALUES
				($1, $2, NULLIF($3, '')::uuid, $4, $5, $6, $7, $8, $9, $10::jsonb, $11)
			 ON CONFLICT (user_id, content_hash, model)
			 DO UPDATE SET
				resume_id = NULLIF(EXCLUDED.resume_id::text, '')::uuid,
				summary = EXCLUDED.summary,
				response = EXCLUDED.response,
				highlights = EXCLUDED.highlights,
				recommendations = EXCLUDED.recommendations,
				raw_response = EXCLUDED.raw_response,
				created_at = NOW()
			 RETURNING id, user_id, COALESCE(resume_id::text, ''), content_hash, model, summary, response, highlights, recommendations, raw_response, created_at`,
			record.ID,
			record.UserID,
			record.ResumeID,
			record.ContentHash,
			record.Model,
			record.Summary,
			record.Response,
			record.Highlights,
			record.Recommendations,
			record.RawResponse,
			record.CreatedAt,
		).Scan(
			&scanned.ID,
			&scanned.UserID,
			&scanned.ResumeID,
			&scanned.ContentHash,
			&scanned.Model,
			&scanned.Summary,
			&scanned.Response,
			&scanned.Highlights,
			&scanned.Recommendations,
			&rawJSON,
			&scanned.CreatedAt,
		)
		if err == nil {
			scanned.RawResponse = string(rawJSON)
			return &scanned, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id, existing := range r.resumeAnalyses {
		if existing.UserID == record.UserID && existing.ContentHash == record.ContentHash && existing.Model == record.Model {
			record.ID = id
			break
		}
	}

	r.resumeAnalyses[record.ID] = *record
	copy := *record
	return &copy, nil
}

func (r *interviewRepository) FindResumeAnalysisByHash(userID, contentHash, model string) (*domain.ResumeAnalysisRecord, error) {
	userID = strings.TrimSpace(userID)
	contentHash = strings.TrimSpace(contentHash)
	model = strings.TrimSpace(model)

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var record domain.ResumeAnalysisRecord
		var rawJSON []byte
		err := r.pool.QueryRow(
			ctx,
			`SELECT id, user_id, COALESCE(resume_id::text, ''), content_hash, model, summary, response, highlights, recommendations, raw_response, created_at
			   FROM app_resume_analyses
			  WHERE user_id = $1 AND content_hash = $2 AND model = $3
			  LIMIT 1`,
			userID,
			contentHash,
			model,
		).Scan(
			&record.ID,
			&record.UserID,
			&record.ResumeID,
			&record.ContentHash,
			&record.Model,
			&record.Summary,
			&record.Response,
			&record.Highlights,
			&record.Recommendations,
			&rawJSON,
			&record.CreatedAt,
		)
		if err == nil {
			record.RawResponse = string(rawJSON)
			return &record, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	for _, record := range r.resumeAnalyses {
		if record.UserID == userID && record.ContentHash == contentHash && record.Model == model {
			copy := record
			return &copy, nil
		}
	}

	return nil, nil
}

func (r *interviewRepository) GetLatestResumeAnalysis(userID string) (*domain.ResumeAnalysisRecord, error) {
	userID = strings.TrimSpace(userID)

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var record domain.ResumeAnalysisRecord
		var rawJSON []byte
		err := r.pool.QueryRow(
			ctx,
			`SELECT id, user_id, COALESCE(resume_id::text, ''), content_hash, model, summary, response, highlights, recommendations, raw_response, created_at
			   FROM app_resume_analyses
			  WHERE user_id = $1
			  ORDER BY created_at DESC
			  LIMIT 1`,
			userID,
		).Scan(
			&record.ID,
			&record.UserID,
			&record.ResumeID,
			&record.ContentHash,
			&record.Model,
			&record.Summary,
			&record.Response,
			&record.Highlights,
			&record.Recommendations,
			&rawJSON,
			&record.CreatedAt,
		)
		if err == nil {
			record.RawResponse = string(rawJSON)
			return &record, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *domain.ResumeAnalysisRecord
	for _, record := range r.resumeAnalyses {
		if record.UserID != userID {
			continue
		}
		if latest == nil || record.CreatedAt.After(latest.CreatedAt) {
			copy := record
			latest = &copy
		}
	}

	return latest, nil
}

func (r *interviewRepository) GetLatestResumeAnalysisByLanguage(userID, language string) (*domain.ResumeAnalysisRecord, error) {
	userID = strings.TrimSpace(userID)
	language = strings.TrimSpace(strings.ToLower(language))
	if language == "" {
		language = "en"
	}
	modelSuffix := "|lang:" + language

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var record domain.ResumeAnalysisRecord
		var rawJSON []byte
		err := r.pool.QueryRow(
			ctx,
			`SELECT id, user_id, COALESCE(resume_id::text, ''), content_hash, model, summary, response, highlights, recommendations, raw_response, created_at
			   FROM app_resume_analyses
			  WHERE user_id = $1
			    AND model LIKE $2
			  ORDER BY created_at DESC
			  LIMIT 1`,
			userID,
			"%"+modelSuffix,
		).Scan(
			&record.ID,
			&record.UserID,
			&record.ResumeID,
			&record.ContentHash,
			&record.Model,
			&record.Summary,
			&record.Response,
			&record.Highlights,
			&record.Recommendations,
			&rawJSON,
			&record.CreatedAt,
		)
		if err == nil {
			record.RawResponse = string(rawJSON)
			return &record, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	var latest *domain.ResumeAnalysisRecord
	for _, record := range r.resumeAnalyses {
		if record.UserID != userID {
			continue
		}
		if !strings.Contains(strings.ToLower(record.Model), modelSuffix) {
			continue
		}
		if latest == nil || record.CreatedAt.After(latest.CreatedAt) {
			copy := record
			latest = &copy
		}
	}

	return latest, nil
}

func (r *interviewRepository) SaveGeneratedQuestions(userID, resumeID, jobParseID string, questions []domain.GeneratedQuestion) ([]domain.StoredQuestion, error) {
	now := time.Now().UTC()
	stored := make([]domain.StoredQuestion, 0, len(questions))

	for _, item := range questions {
		record := domain.StoredQuestion{
			ID:         uuid.NewString(),
			UserID:     userID,
			ResumeID:   resumeID,
			JobParseID: jobParseID,
			Type:       item.Type,
			Question:   item.Question,
			CreatedAt:  now,
		}

		if r.pool != nil {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			_, err := r.pool.Exec(
				ctx,
				`INSERT INTO app_questions
					(id, user_id, resume_id, job_parse_id, question_type, question_text, created_at)
				 VALUES
					($1, $2, $3, $4, $5, $6, $7)`,
				record.ID,
				record.UserID,
				record.ResumeID,
				record.JobParseID,
				record.Type,
				record.Question,
				record.CreatedAt,
			)
			cancel()
			if err != nil {
				r.mu.Lock()
				r.questions[record.ID] = record
				r.mu.Unlock()
			}
		} else {
			r.mu.Lock()
			r.questions[record.ID] = record
			r.mu.Unlock()
		}

		stored = append(stored, record)
	}

	return stored, nil
}

func (r *interviewRepository) CreatePracticeSession(
	userID,
	resumeID,
	jobParseID string,
	questionIDs []string,
	metadata domain.SessionMetadata,
) (*domain.PracticeSession, error) {
	if err := r.validateSessionCreationReferences(userID, resumeID, jobParseID, questionIDs); err != nil {
		return nil, err
	}

	normalizedMode := string(domain.NormalizeInterviewMode(metadata.InterviewMode))
	normalizedLanguage := string(domain.NormalizeInterviewLanguage(string(metadata.InterviewLanguage)))
	normalizedDifficulty := string(domain.NormalizeInterviewDifficulty(string(metadata.InterviewDifficulty)))

	now := time.Now().UTC()
	session := &domain.PracticeSession{
		ID:                  uuid.NewString(),
		UserID:              userID,
		ResumeID:            resumeID,
		JobParseID:          jobParseID,
		QuestionIDs:         append([]string{}, questionIDs...),
		InterviewMode:       normalizedMode,
		InterviewLanguage:   normalizedLanguage,
		InterviewDifficulty: normalizedDifficulty,
		TargetRole:          metadata.TargetRole,
		TargetCompany:       metadata.TargetCompany,
		Status:              domain.SessionStatusActive,
		Score:               0,
		CreatedAt:           now,
		LastActivityAt:      now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_practice_sessions
				(id, user_id, resume_id, job_parse_id, question_ids, interview_mode, interview_language, interview_difficulty, target_role, target_company, status, score, created_at, last_activity_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)`,
			session.ID,
			session.UserID,
			session.ResumeID,
			session.JobParseID,
			session.QuestionIDs,
			session.InterviewMode,
			session.InterviewLanguage,
			session.InterviewDifficulty,
			session.TargetRole,
			session.TargetCompany,
			session.Status,
			session.Score,
			session.CreatedAt,
			session.LastActivityAt,
		)
		if err != nil {
			return nil, err
		}

		return session, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.sessions[session.ID] = *session
	return session, nil
}

func (r *interviewRepository) ListPracticeSessions(userID string) ([]domain.PracticeSession, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := r.pool.Query(
			ctx,
			`SELECT id, user_id, resume_id, job_parse_id, question_ids, interview_mode, interview_language, interview_difficulty, target_role, target_company, status, score, created_at, last_activity_at, completed_at
			   FROM app_practice_sessions
			  WHERE user_id = $1
			  ORDER BY created_at DESC`,
			userID,
		)
		if err == nil {
			defer rows.Close()

			result := make([]domain.PracticeSession, 0)
			for rows.Next() {
				var session domain.PracticeSession
				if scanErr := rows.Scan(
					&session.ID,
					&session.UserID,
					&session.ResumeID,
					&session.JobParseID,
					&session.QuestionIDs,
					&session.InterviewMode,
					&session.InterviewLanguage,
					&session.InterviewDifficulty,
					&session.TargetRole,
					&session.TargetCompany,
					&session.Status,
					&session.Score,
					&session.CreatedAt,
					&session.LastActivityAt,
					&session.CompletedAt,
				); scanErr == nil {
					result = append(result, session)
				}
			}
			return result, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.PracticeSession, 0)
	for _, session := range r.sessions {
		if session.UserID == userID {
			result = append(result, session)
		}
	}

	return result, nil
}

func (r *interviewRepository) CompletePracticeSession(userID, sessionID string) (*domain.PracticeSession, error) {
	completedAt := time.Now().UTC()

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var session domain.PracticeSession
		err := r.pool.QueryRow(
			ctx,
			`UPDATE app_practice_sessions
			    SET status = $1,
			        completed_at = $2,
			        last_activity_at = $2
			  WHERE id = $3
			    AND user_id = $4
			    AND status = $5
			RETURNING id, user_id, resume_id, job_parse_id, question_ids, interview_mode, interview_language, interview_difficulty, target_role, target_company, status, score, created_at, last_activity_at, completed_at`,
			domain.SessionStatusCompleted,
			completedAt,
			sessionID,
			userID,
			domain.SessionStatusActive,
		).Scan(
			&session.ID,
			&session.UserID,
			&session.ResumeID,
			&session.JobParseID,
			&session.QuestionIDs,
			&session.InterviewMode,
			&session.InterviewLanguage,
			&session.InterviewDifficulty,
			&session.TargetRole,
			&session.TargetCompany,
			&session.Status,
			&session.Score,
			&session.CreatedAt,
			&session.LastActivityAt,
			&session.CompletedAt,
		)
		if err == nil {
			return &session, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists || session.UserID != userID {
		return nil, errors.New("session not found")
	}
	if session.Status != domain.SessionStatusActive {
		return nil, errors.New("session is not active")
	}

	session.Status = domain.SessionStatusCompleted
	session.LastActivityAt = completedAt
	session.CompletedAt = &completedAt
	r.sessions[sessionID] = session

	return &session, nil
}

func (r *interviewRepository) SaveSessionAnswer(userID, sessionID, questionID, answer string) (*domain.SessionAnswer, error) {
	if err := r.validateSessionQuestionReference(userID, sessionID, questionID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	record := &domain.SessionAnswer{
		ID:         uuid.NewString(),
		SessionID:  sessionID,
		QuestionID: questionID,
		UserID:     userID,
		Answer:     answer,
		CreatedAt:  now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		touchTag, err := tx.Exec(
			ctx,
			`UPDATE app_practice_sessions
			    SET last_activity_at = $3
			  WHERE id = $1
			    AND user_id = $2
			    AND status = $4`,
			sessionID,
			userID,
			now,
			domain.SessionStatusActive,
		)
		if err != nil {
			return nil, err
		}
		if touchTag.RowsAffected() == 0 {
			return nil, errors.New("session is not active")
		}

		_, err = tx.Exec(
			ctx,
			`INSERT INTO app_session_answers
				(id, session_id, question_id, user_id, answer_text, created_at)
			 VALUES
				($1, $2, $3, $4, $5, $6)`,
			record.ID,
			record.SessionID,
			record.QuestionID,
			record.UserID,
			record.Answer,
			record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return record, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists || session.UserID != userID || session.Status != domain.SessionStatusActive {
		return nil, errors.New("session is not active")
	}
	session.LastActivityAt = now
	r.sessions[sessionID] = session

	r.answers[record.ID] = *record
	return record, nil
}

func (r *interviewRepository) SaveFeedback(
	userID, sessionID, questionID, question, answer string,
	analysis *domain.AnswerAnalysis,
) (*domain.FeedbackRecord, error) {
	if err := r.validateSessionQuestionReference(userID, sessionID, questionID); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	record := &domain.FeedbackRecord{
		ID:           uuid.NewString(),
		UserID:       userID,
		SessionID:    sessionID,
		QuestionID:   questionID,
		Question:     question,
		Answer:       answer,
		Score:        analysis.Score,
		Strengths:    append([]string{}, analysis.Strengths...),
		Weaknesses:   append([]string{}, analysis.Weaknesses...),
		Improvements: append([]string{}, analysis.Improvements...),
		STARFeedback: analysis.STARFeedback,
		CreatedAt:    now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
		if err != nil {
			return nil, err
		}
		defer tx.Rollback(ctx)

		touchTag, err := tx.Exec(
			ctx,
			`UPDATE app_practice_sessions
			    SET last_activity_at = $3
			  WHERE id = $1
			    AND user_id = $2
			    AND status = $4`,
			sessionID,
			userID,
			now,
			domain.SessionStatusActive,
		)
		if err != nil {
			return nil, err
		}
		if touchTag.RowsAffected() == 0 {
			return nil, errors.New("session is not active")
		}

		_, err = tx.Exec(
			ctx,
			`INSERT INTO app_feedback
				(id, user_id, session_id, question_id, question_text, answer_text, score, strengths, weaknesses, improvements, star_feedback, created_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			record.ID,
			record.UserID,
			record.SessionID,
			record.QuestionID,
			record.Question,
			record.Answer,
			record.Score,
			record.Strengths,
			record.Weaknesses,
			record.Improvements,
			record.STARFeedback,
			record.CreatedAt,
		)
		if err != nil {
			return nil, err
		}

		if err := tx.Commit(ctx); err != nil {
			return nil, err
		}

		return record, nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists || session.UserID != userID || session.Status != domain.SessionStatusActive {
		return nil, errors.New("session is not active")
	}
	session.LastActivityAt = now
	r.sessions[sessionID] = session

	r.feedbacks[record.ID] = *record
	return record, nil
}

func (r *interviewRepository) ListFeedbackByUser(userID string) ([]domain.FeedbackRecord, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := r.pool.Query(
			ctx,
			`SELECT id, user_id, session_id, question_id, question_text, answer_text, score, strengths, weaknesses, improvements, star_feedback, created_at
			   FROM app_feedback
			  WHERE user_id = $1
			  ORDER BY created_at DESC`,
			userID,
		)
		if err == nil {
			defer rows.Close()
			result := make([]domain.FeedbackRecord, 0)
			for rows.Next() {
				var item domain.FeedbackRecord
				if scanErr := rows.Scan(
					&item.ID,
					&item.UserID,
					&item.SessionID,
					&item.QuestionID,
					&item.Question,
					&item.Answer,
					&item.Score,
					&item.Strengths,
					&item.Weaknesses,
					&item.Improvements,
					&item.STARFeedback,
					&item.CreatedAt,
				); scanErr == nil {
					result = append(result, item)
				}
			}
			return result, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.FeedbackRecord, 0)
	for _, item := range r.feedbacks {
		if item.UserID == userID {
			result = append(result, item)
		}
	}
	return result, nil
}

func (r *interviewRepository) SaveProgressMetrics(userID string, averageScore float64, weakAreas []string, sessionsCompleted int) (*domain.ProgressMetrics, error) {
	updatedAt := time.Now().UTC()
	metrics := &domain.ProgressMetrics{
		UserID:            userID,
		AverageScore:      averageScore,
		WeakAreas:         append([]string{}, weakAreas...),
		SessionsCompleted: sessionsCompleted,
		UpdatedAt:         updatedAt,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_progress_metrics
				(user_id, average_score, weak_areas, sessions_completed, updated_at)
			 VALUES
				($1, $2, $3, $4, $5)
			 ON CONFLICT (user_id)
			 DO UPDATE
			    SET average_score = EXCLUDED.average_score,
			        weak_areas = EXCLUDED.weak_areas,
			        sessions_completed = EXCLUDED.sessions_completed,
			        updated_at = EXCLUDED.updated_at`,
			metrics.UserID,
			metrics.AverageScore,
			metrics.WeakAreas,
			metrics.SessionsCompleted,
			metrics.UpdatedAt,
		)
		if err == nil {
			return metrics, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.progress[userID] = *metrics
	return metrics, nil
}

func (r *interviewRepository) GetProgressMetrics(userID string) (*domain.ProgressMetrics, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var metrics domain.ProgressMetrics
		err := r.pool.QueryRow(
			ctx,
			`SELECT user_id, average_score, weak_areas, sessions_completed, updated_at
			   FROM app_progress_metrics
			  WHERE user_id = $1`,
			userID,
		).Scan(
			&metrics.UserID,
			&metrics.AverageScore,
			&metrics.WeakAreas,
			&metrics.SessionsCompleted,
			&metrics.UpdatedAt,
		)
		if err == nil {
			return &metrics, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	metrics, exists := r.progress[userID]
	if !exists {
		return &domain.ProgressMetrics{
			UserID:            userID,
			AverageScore:      0,
			WeakAreas:         []string{},
			SessionsCompleted: 0,
			UpdatedAt:         time.Now().UTC(),
		}, nil
	}

	copy := metrics
	return &copy, nil
}

func (r *interviewRepository) CreateReviewSession(userID string, input domain.ReviewStartInput) (*domain.ReviewSession, error) {
	now := time.Now().UTC()
	sessionType := strings.TrimSpace(strings.ToLower(input.SessionType))
	if sessionType == "" {
		sessionType = domain.ReviewSessionTypeStandard
	}
	inputMode := strings.TrimSpace(strings.ToLower(input.InputMode))
	if inputMode == "" {
		inputMode = string(domain.InterviewModeText)
	}

	session := &domain.ReviewSession{
		ID:                uuid.NewString(),
		UserID:            userID,
		SessionType:       sessionType,
		InputMode:         inputMode,
		InterviewLanguage: string(domain.NormalizeInterviewLanguage(input.InterviewLanguage)),
		InputText:         strings.TrimSpace(input.InputText),
		VoiceURL:          strings.TrimSpace(input.VoiceURL),
		TranscriptText:    strings.TrimSpace(input.TranscriptText),
		RoleTarget:        strings.TrimSpace(input.TargetRole),
		CompanyTarget:     strings.TrimSpace(input.TargetCompany),
		Status:            domain.SessionStatusActive,
		Feedback: domain.ReviewAIFeedback{
			Strengths:   []string{},
			Weaknesses:  []string{},
			Suggestions: []string{},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	if r.pool != nil {
		payload, marshalErr := json.Marshal(session.Feedback)
		if marshalErr != nil {
			return nil, marshalErr
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_review_sessions
				(id, user_id, session_type, input_mode, interview_language, input_text, voice_url, transcript_text, role_target, company_target, status, ai_feedback, score, communication_score, structure_score, confidence_score, created_at, updated_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12::jsonb, $13, $14, $15, $16, $17, $18)`,
			session.ID,
			session.UserID,
			session.SessionType,
			session.InputMode,
			session.InterviewLanguage,
			session.InputText,
			session.VoiceURL,
			session.TranscriptText,
			session.RoleTarget,
			session.CompanyTarget,
			session.Status,
			string(payload),
			session.Feedback.Score,
			session.Feedback.Communication,
			session.Feedback.StructureSTAR,
			session.Feedback.Confidence,
			session.CreatedAt,
			session.UpdatedAt,
		)
		if err == nil {
			return session, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviewSessions[session.ID] = *session
	return session, nil
}

func (r *interviewRepository) GetReviewSession(userID, sessionID string) (*domain.ReviewSession, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var session domain.ReviewSession
		var feedbackJSON []byte
		err := r.pool.QueryRow(
			ctx,
			`SELECT id, user_id, session_type, input_mode, interview_language, input_text, voice_url, transcript_text, role_target, company_target, status,
					ai_feedback, score, communication_score, structure_score, confidence_score, created_at, updated_at, completed_at
			   FROM app_review_sessions
			  WHERE id = $1 AND user_id = $2`,
			sessionID,
			userID,
		).Scan(
			&session.ID,
			&session.UserID,
			&session.SessionType,
			&session.InputMode,
			&session.InterviewLanguage,
			&session.InputText,
			&session.VoiceURL,
			&session.TranscriptText,
			&session.RoleTarget,
			&session.CompanyTarget,
			&session.Status,
			&feedbackJSON,
			&session.Feedback.Score,
			&session.Feedback.Communication,
			&session.Feedback.StructureSTAR,
			&session.Feedback.Confidence,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.CompletedAt,
		)
		if err == nil {
			if len(feedbackJSON) > 0 {
				_ = json.Unmarshal(feedbackJSON, &session.Feedback)
			}
			return &session, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, exists := r.reviewSessions[sessionID]
	if !exists || session.UserID != userID {
		return nil, errors.New("review session not found")
	}
	copy := session
	return &copy, nil
}

func (r *interviewRepository) UpdateReviewSessionFeedback(userID, sessionID string, feedback domain.ReviewAIFeedback, appendInput string) (*domain.ReviewSession, error) {
	now := time.Now().UTC()
	feedback.Score = clampReviewScore(feedback.Score)
	feedback.Communication = clampReviewScore(feedback.Communication)
	feedback.StructureSTAR = clampReviewScore(feedback.StructureSTAR)
	feedback.Confidence = clampReviewScore(feedback.Confidence)

	if r.pool != nil {
		payload, marshalErr := json.Marshal(feedback)
		if marshalErr != nil {
			return nil, marshalErr
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var session domain.ReviewSession
		var feedbackJSON []byte
		err := r.pool.QueryRow(
			ctx,
			`UPDATE app_review_sessions
				SET ai_feedback = $3::jsonb,
					score = $4,
					communication_score = $5,
					structure_score = $6,
					confidence_score = $7,
					transcript_text = CASE WHEN $8 = '' THEN transcript_text ELSE CONCAT_WS(E'\n', transcript_text, $8) END,
					updated_at = $9
			  WHERE id = $1 AND user_id = $2
			  RETURNING id, user_id, session_type, input_mode, interview_language, input_text, voice_url, transcript_text, role_target, company_target, status,
						ai_feedback, score, communication_score, structure_score, confidence_score, created_at, updated_at, completed_at`,
			sessionID,
			userID,
			string(payload),
			feedback.Score,
			feedback.Communication,
			feedback.StructureSTAR,
			feedback.Confidence,
			strings.TrimSpace(appendInput),
			now,
		).Scan(
			&session.ID,
			&session.UserID,
			&session.SessionType,
			&session.InputMode,
			&session.InterviewLanguage,
			&session.InputText,
			&session.VoiceURL,
			&session.TranscriptText,
			&session.RoleTarget,
			&session.CompanyTarget,
			&session.Status,
			&feedbackJSON,
			&session.Feedback.Score,
			&session.Feedback.Communication,
			&session.Feedback.StructureSTAR,
			&session.Feedback.Confidence,
			&session.CreatedAt,
			&session.UpdatedAt,
			&session.CompletedAt,
		)
		if err == nil {
			if len(feedbackJSON) > 0 {
				_ = json.Unmarshal(feedbackJSON, &session.Feedback)
			}
			return &session, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.reviewSessions[sessionID]
	if !exists || session.UserID != userID {
		return nil, errors.New("review session not found")
	}
	session.Feedback = feedback
	if trimmedInput := strings.TrimSpace(appendInput); trimmedInput != "" {
		session.TranscriptText = strings.TrimSpace(strings.Join([]string{session.TranscriptText, trimmedInput}, "\n"))
	}
	session.UpdatedAt = now
	r.reviewSessions[sessionID] = session
	copy := session
	return &copy, nil
}

func (r *interviewRepository) CompleteReviewSession(userID, sessionID string, plan *domain.ImprovementPlan, summary string) (*domain.ReviewSession, error) {
	now := time.Now().UTC()

	session, err := r.GetReviewSession(userID, sessionID)
	if err != nil {
		return nil, err
	}

	if plan != nil {
		session.ImprovementPlan = plan
	}
	if strings.TrimSpace(summary) != "" {
		session.Feedback.Insight = strings.TrimSpace(summary)
	}
	session.Status = domain.SessionStatusCompleted
	session.CompletedAt = &now
	session.UpdatedAt = now

	if r.pool != nil {
		payload, marshalErr := json.Marshal(session.Feedback)
		if marshalErr != nil {
			return nil, marshalErr
		}
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`UPDATE app_review_sessions
				SET status = $3,
					ai_feedback = $4::jsonb,
					score = $5,
					communication_score = $6,
					structure_score = $7,
					confidence_score = $8,
					completed_at = $9,
					updated_at = $10
			  WHERE id = $1 AND user_id = $2`,
			sessionID,
			userID,
			session.Status,
			string(payload),
			session.Feedback.Score,
			session.Feedback.Communication,
			session.Feedback.StructureSTAR,
			session.Feedback.Confidence,
			now,
			now,
		)
		if err == nil {
			return session, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.reviewSessions[sessionID] = *session
	return session, nil
}

func (r *interviewRepository) ListRecentReviewSessions(userID string, limit int) ([]domain.ReviewSession, error) {
	if limit <= 0 {
		limit = 10
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := r.pool.Query(
			ctx,
			`SELECT id, user_id, session_type, input_mode, interview_language, input_text, voice_url, transcript_text, role_target, company_target, status,
					ai_feedback, score, communication_score, structure_score, confidence_score, created_at, updated_at, completed_at
			   FROM app_review_sessions
			  WHERE user_id = $1
			  ORDER BY created_at DESC
			  LIMIT $2`,
			userID,
			limit,
		)
		if err == nil {
			defer rows.Close()
			result := make([]domain.ReviewSession, 0)
			for rows.Next() {
				var session domain.ReviewSession
				var feedbackJSON []byte
				if scanErr := rows.Scan(
					&session.ID,
					&session.UserID,
					&session.SessionType,
					&session.InputMode,
					&session.InterviewLanguage,
					&session.InputText,
					&session.VoiceURL,
					&session.TranscriptText,
					&session.RoleTarget,
					&session.CompanyTarget,
					&session.Status,
					&feedbackJSON,
					&session.Feedback.Score,
					&session.Feedback.Communication,
					&session.Feedback.StructureSTAR,
					&session.Feedback.Confidence,
					&session.CreatedAt,
					&session.UpdatedAt,
					&session.CompletedAt,
				); scanErr == nil {
					if len(feedbackJSON) > 0 {
						_ = json.Unmarshal(feedbackJSON, &session.Feedback)
					}
					result = append(result, session)
				}
			}
			return result, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]domain.ReviewSession, 0)
	for _, session := range r.reviewSessions {
		if session.UserID == userID {
			result = append(result, session)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})
	if len(result) > limit {
		result = result[:limit]
	}
	return result, nil
}

func (r *interviewRepository) GetCoachingMemory(userID string) (*domain.CoachingMemory, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		memory := &domain.CoachingMemory{}
		err := r.pool.QueryRow(
			ctx,
			`SELECT user_id, target_role, strengths, weaknesses, preferred_language, last_summary, focus_areas, next_actions, updated_at
			   FROM app_coaching_memory
			  WHERE user_id = $1`,
			userID,
		).Scan(
			&memory.UserID,
			&memory.TargetRole,
			&memory.Strengths,
			&memory.Weaknesses,
			&memory.PreferredLanguage,
			&memory.LastSummary,
			&memory.FocusAreas,
			&memory.NextActions,
			&memory.UpdatedAt,
		)
		if err == nil {
			return memory, nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	memory, exists := r.coachingMemory[userID]
	if exists {
		copy := memory
		return &copy, nil
	}

	return &domain.CoachingMemory{
		UserID:            userID,
		Strengths:         []string{},
		Weaknesses:        []string{},
		PreferredLanguage: "en",
		FocusAreas:        []string{},
		NextActions:       []string{},
		UpdatedAt:         time.Now().UTC(),
	}, nil
}

func (r *interviewRepository) UpsertCoachingMemory(memory domain.CoachingMemory) (*domain.CoachingMemory, error) {
	memory.UpdatedAt = time.Now().UTC()

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_coaching_memory
				(user_id, target_role, strengths, weaknesses, preferred_language, last_summary, focus_areas, next_actions, updated_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (user_id)
			 DO UPDATE SET
				target_role = EXCLUDED.target_role,
				strengths = EXCLUDED.strengths,
				weaknesses = EXCLUDED.weaknesses,
				preferred_language = EXCLUDED.preferred_language,
				last_summary = EXCLUDED.last_summary,
				focus_areas = EXCLUDED.focus_areas,
				next_actions = EXCLUDED.next_actions,
				updated_at = EXCLUDED.updated_at`,
			memory.UserID,
			memory.TargetRole,
			memory.Strengths,
			memory.Weaknesses,
			memory.PreferredLanguage,
			memory.LastSummary,
			memory.FocusAreas,
			memory.NextActions,
			memory.UpdatedAt,
		)
		if err == nil {
			return &memory, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.coachingMemory[memory.UserID] = memory
	copy := memory
	return &copy, nil
}

func (r *interviewRepository) SaveProgressTrackingPoint(userID, reviewSessionID string, point domain.ProgressTrackingPoint) (*domain.ProgressTrackingPoint, error) {
	point.ReviewSessionID = reviewSessionID
	if point.CreatedAt.IsZero() {
		point.CreatedAt = time.Now().UTC()
	}
	point.Communication = clampReviewScore(point.Communication)
	point.StructureSTAR = clampReviewScore(point.StructureSTAR)
	point.Confidence = clampReviewScore(point.Confidence)
	point.OverallScore = clampReviewScore(point.OverallScore)

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_progress_tracking
				(id, user_id, review_session_id, communication_score, structure_score, confidence_score, overall_score, notes, created_at)
			 VALUES
				($1, $2, $3::uuid, $4, $5, $6, $7, $8, $9)`,
			uuid.NewString(),
			userID,
			reviewSessionID,
			point.Communication,
			point.StructureSTAR,
			point.Confidence,
			point.OverallScore,
			point.Notes,
			point.CreatedAt,
		)
		if err == nil {
			return &point, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.progressTracking[userID] = append(r.progressTracking[userID], point)
	copy := point
	return &copy, nil
}

func (r *interviewRepository) ListProgressTracking(userID string, limit int) ([]domain.ProgressTrackingPoint, error) {
	if limit <= 0 {
		limit = 20
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		rows, err := r.pool.Query(
			ctx,
			`SELECT review_session_id, communication_score, structure_score, confidence_score, overall_score, notes, created_at
			   FROM app_progress_tracking
			  WHERE user_id = $1
			  ORDER BY created_at DESC
			  LIMIT $2`,
			userID,
			limit,
		)
		if err == nil {
			defer rows.Close()
			result := make([]domain.ProgressTrackingPoint, 0)
			for rows.Next() {
				var point domain.ProgressTrackingPoint
				if scanErr := rows.Scan(
					&point.ReviewSessionID,
					&point.Communication,
					&point.StructureSTAR,
					&point.Confidence,
					&point.OverallScore,
					&point.Notes,
					&point.CreatedAt,
				); scanErr == nil {
					result = append(result, point)
				}
			}
			return result, nil
		}
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	points := append([]domain.ProgressTrackingPoint{}, r.progressTracking[userID]...)
	sort.Slice(points, func(i, j int) bool {
		return points[i].CreatedAt.After(points[j].CreatedAt)
	})
	if len(points) > limit {
		points = points[:limit]
	}
	return points, nil
}

func clampReviewScore(value int) int {
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func (r *interviewRepository) validateSessionCreationReferences(userID, resumeID, jobParseID string, questionIDs []string) error {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var exists bool

		if err := r.pool.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM app_resumes WHERE id = $1 AND user_id = $2)`,
			resumeID,
			userID,
		).Scan(&exists); err == nil {
			if !exists {
				return errors.New("resume not found")
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		if err := r.pool.QueryRow(
			ctx,
			`SELECT EXISTS(SELECT 1 FROM app_job_parses WHERE id = $1 AND user_id = $2)`,
			jobParseID,
			userID,
		).Scan(&exists); err == nil {
			if !exists {
				return errors.New("job parse not found")
			}
		} else if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}

		for _, questionID := range questionIDs {
			if err := r.pool.QueryRow(
				ctx,
				`SELECT EXISTS(
					SELECT 1
					  FROM app_questions
					 WHERE id = $1 AND user_id = $2 AND resume_id = $3 AND job_parse_id = $4
				)`,
				questionID,
				userID,
				resumeID,
				jobParseID,
			).Scan(&exists); err == nil {
				if !exists {
					return errors.New("question not found for session")
				}
			} else if !errors.Is(err, pgx.ErrNoRows) {
				return err
			}
		}

		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	resume, resumeExists := r.resumes[resumeID]
	if !resumeExists || resume.UserID != userID {
		return errors.New("resume not found")
	}

	job, jobExists := r.parsedJobs[jobParseID]
	if !jobExists || job.UserID != userID {
		return errors.New("job parse not found")
	}

	for _, questionID := range questionIDs {
		question, questionExists := r.questions[questionID]
		if !questionExists || question.UserID != userID || question.ResumeID != resumeID || question.JobParseID != jobParseID {
			return errors.New("question not found for session")
		}
	}

	return nil
}

func (r *interviewRepository) validateSessionQuestionReference(userID, sessionID, questionID string) error {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var sessionUserID string
		var sessionStatus string
		var questionIDs []string
		err := r.pool.QueryRow(
			ctx,
			`SELECT user_id, status, question_ids FROM app_practice_sessions WHERE id = $1`,
			sessionID,
		).Scan(&sessionUserID, &sessionStatus, &questionIDs)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("session not found")
			}
			return err
		}

		if sessionUserID != userID {
			return errors.New("session not found")
		}

		if sessionStatus != domain.SessionStatusActive {
			return errors.New("session is not active")
		}

		if !containsString(questionIDs, questionID) {
			return errors.New("question not found for session")
		}

		var questionUserID string
		err = r.pool.QueryRow(
			ctx,
			`SELECT user_id FROM app_questions WHERE id = $1`,
			questionID,
		).Scan(&questionUserID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("question not found")
			}
			return err
		}

		if questionUserID != userID {
			return errors.New("question not found")
		}

		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	session, sessionExists := r.sessions[sessionID]
	if !sessionExists || session.UserID != userID {
		return errors.New("session not found")
	}

	if session.Status != domain.SessionStatusActive {
		return errors.New("session is not active")
	}

	if !containsString(session.QuestionIDs, questionID) {
		return errors.New("question not found for session")
	}

	question, questionExists := r.questions[questionID]
	if !questionExists || question.UserID != userID {
		return errors.New("question not found")
	}

	if question.ResumeID != session.ResumeID || question.JobParseID != session.JobParseID {
		return errors.New("question not found for session")
	}

	return nil
}

func (r *interviewRepository) TouchSessionActivity(userID, sessionID string) error {
	now := time.Now().UTC()

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tag, err := r.pool.Exec(
			ctx,
			`UPDATE app_practice_sessions
			    SET last_activity_at = $3
			  WHERE id = $1
			    AND user_id = $2
			    AND status = $4`,
			sessionID,
			userID,
			now,
			domain.SessionStatusActive,
		)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return errors.New("session is not active")
		}

		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists || session.UserID != userID {
		return errors.New("session not found")
	}
	if session.Status != domain.SessionStatusActive {
		return errors.New("session is not active")
	}

	session.LastActivityAt = now
	r.sessions[sessionID] = session

	return nil
}

func (r *interviewRepository) AbandonIdleSessions(idleFor time.Duration) (int64, error) {
	if idleFor <= 0 {
		return 0, errors.New("idle duration must be greater than zero")
	}

	cutoff := time.Now().UTC().Add(-idleFor)
	now := time.Now().UTC()

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		tag, err := r.pool.Exec(
			ctx,
			`UPDATE app_practice_sessions
			    SET status = $1,
			        completed_at = COALESCE(completed_at, $2),
			        last_activity_at = $2
			  WHERE status = $3
			    AND last_activity_at < $4`,
			domain.SessionStatusAbandoned,
			now,
			domain.SessionStatusActive,
			cutoff,
		)
		if err != nil {
			return 0, err
		}

		return tag.RowsAffected(), nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var count int64
	for sessionID, session := range r.sessions {
		if session.Status != domain.SessionStatusActive {
			continue
		}

		lastActivity := session.LastActivityAt
		if lastActivity.IsZero() {
			lastActivity = session.CreatedAt
		}

		if !lastActivity.Before(cutoff) {
			continue
		}

		session.Status = domain.SessionStatusAbandoned
		session.CompletedAt = &now
		session.LastActivityAt = now
		r.sessions[sessionID] = session
		count++
	}

	return count, nil
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

package repository

import (
	"context"
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type interviewRepository struct {
	pool       *pgxpool.Pool
	mu         sync.RWMutex
	parsedJobs map[string]*domain.ParsedJobDescription
	resumes    map[string]*domain.ResumeRecord
	questions  map[string]domain.StoredQuestion
	sessions   map[string]domain.PracticeSession
	answers    map[string]domain.SessionAnswer
	feedbacks  map[string]domain.FeedbackRecord
	progress   map[string]domain.ProgressMetrics
}

// NewInterviewRepository creates interview repository with postgres support and in-memory fallback.
func NewInterviewRepository(pool *pgxpool.Pool) domain.InterviewRepository {
	return &interviewRepository{
		pool:       pool,
		parsedJobs: make(map[string]*domain.ParsedJobDescription),
		resumes:    make(map[string]*domain.ResumeRecord),
		questions:  make(map[string]domain.StoredQuestion),
		sessions:   make(map[string]domain.PracticeSession),
		answers:    make(map[string]domain.SessionAnswer),
		feedbacks:  make(map[string]domain.FeedbackRecord),
		progress:   make(map[string]domain.ProgressMetrics),
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

func (r *interviewRepository) SaveResume(userID, content, minIOPath string) (*domain.ResumeRecord, error) {
	now := time.Now().UTC()
	resume := &domain.ResumeRecord{
		UserID:    userID,
		Content:   content,
		MinIOPath: minIOPath,
		CreatedAt: now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
		if err == nil {
			defer tx.Rollback(ctx)

			rows, queryErr := tx.Query(
				ctx,
				`SELECT id
				   FROM app_resumes
				  WHERE user_id = $1
				  ORDER BY created_at DESC`,
				userID,
			)
			if queryErr == nil {
				resumeIDs := make([]string, 0)
				for rows.Next() {
					var id string
					if scanErr := rows.Scan(&id); scanErr == nil {
						resumeIDs = append(resumeIDs, id)
					}
				}
				rows.Close()

				if len(resumeIDs) == 0 {
					resume.ID = uuid.NewString()
					_, err = tx.Exec(
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
				} else {
					resume.ID = resumeIDs[0]
					_, err = tx.Exec(
						ctx,
						`UPDATE app_resumes
						    SET content = $2,
						        minio_path = $3,
						        created_at = $4
						  WHERE id = $1`,
						resume.ID,
						resume.Content,
						resume.MinIOPath,
						resume.CreatedAt,
					)
					if err == nil {
						_, err = tx.Exec(
							ctx,
							`DELETE FROM app_resumes
							  WHERE user_id = $1 AND id <> $2`,
							userID,
							resume.ID,
						)
					}
				}

				if err == nil {
					err = tx.Commit(ctx)
				}
				if err == nil {
					return resume, nil
				}
			}
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	var latestID string
	var latestTime time.Time
	userResumeIDs := make([]string, 0)
	for id, record := range r.resumes {
		if record.UserID != userID {
			continue
		}
		userResumeIDs = append(userResumeIDs, id)
		if latestID == "" || record.CreatedAt.After(latestTime) {
			latestID = id
			latestTime = record.CreatedAt
		}
	}

	if latestID == "" {
		resume.ID = uuid.NewString()
		r.resumes[resume.ID] = resume
		return resume, nil
	}

	resume.ID = latestID
	r.resumes[resume.ID] = resume
	for _, id := range userResumeIDs {
		if id == resume.ID {
			continue
		}
		delete(r.resumes, id)
	}

	return resume, nil
}

func (r *interviewRepository) GetLatestResume(userID string) (*domain.ResumeRecord, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var resume domain.ResumeRecord
		err := r.pool.QueryRow(
			ctx,
			`SELECT id, user_id, content, COALESCE(minio_path, ''), created_at
			   FROM app_resumes
			  WHERE user_id = $1
			  ORDER BY created_at DESC
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
		if latest == nil || resume.CreatedAt.After(latest.CreatedAt) {
			copy := *resume
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

func (r *interviewRepository) CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string, metadata domain.SessionMetadata) (*domain.PracticeSession, error) {
	if err := r.validateSessionCreationReferences(userID, resumeID, jobParseID, questionIDs); err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	session := &domain.PracticeSession{
		ID:                uuid.NewString(),
		UserID:            userID,
		ResumeID:          resumeID,
		JobParseID:        jobParseID,
		InterviewMode:     metadata.InterviewMode,
		InterviewLanguage: metadata.InterviewLanguage,
		TargetRole:        metadata.TargetRole,
		TargetCompany:     metadata.TargetCompany,
		QuestionIDs:       append([]string{}, questionIDs...),
		Status:            domain.SessionStatusActive,
		Score:             0,
		CreatedAt:         now,
	}

	if session.InterviewMode == "" {
		session.InterviewMode = "text"
	}
	if session.InterviewLanguage == "" {
		session.InterviewLanguage = domain.InterviewLanguageEnglish
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_practice_sessions
				(id, user_id, resume_id, job_parse_id, interview_mode, interview_language, target_role, target_company, question_ids, status, score, created_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`,
			session.ID,
			session.UserID,
			session.ResumeID,
			session.JobParseID,
			session.InterviewMode,
			session.InterviewLanguage,
			session.TargetRole,
			session.TargetCompany,
			session.QuestionIDs,
			session.Status,
			session.Score,
			session.CreatedAt,
		)
		if err == nil {
			return session, nil
		}
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
			`SELECT id, user_id, resume_id, job_parse_id, interview_mode, interview_language, target_role, target_company, question_ids, status, score, created_at, completed_at
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
					&session.InterviewMode,
					&session.InterviewLanguage,
					&session.TargetRole,
					&session.TargetCompany,
					&session.QuestionIDs,
					&session.Status,
					&session.Score,
					&session.CreatedAt,
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

	sort.Slice(result, func(i, j int) bool {
		return result[i].CreatedAt.After(result[j].CreatedAt)
	})

	return result, nil
}

func (r *interviewRepository) CompletePracticeSession(userID, sessionID string) (*domain.PracticeSession, error) {
	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		var score float64
		var feedbackCount int
		if err := r.pool.QueryRow(
			ctx,
			`SELECT COALESCE(AVG(score), 0), COUNT(*)
			   FROM app_feedback
			  WHERE user_id = $1 AND session_id = $2`,
			userID,
			sessionID,
		).Scan(&score, &feedbackCount); err != nil {
			return nil, err
		}

		if feedbackCount == 0 {
			return nil, errors.New("cannot complete session without feedback")
		}

		finalScore := int(score + 0.5)

		var session domain.PracticeSession
		err := r.pool.QueryRow(
			ctx,
			`UPDATE app_practice_sessions
			    SET status = $3,
			        score = $4,
			        completed_at = COALESCE(completed_at, NOW())
			  WHERE id = $1 AND user_id = $2
			  RETURNING id, user_id, resume_id, job_parse_id, interview_mode, interview_language, target_role, target_company, question_ids, status, score, created_at, completed_at`,
			sessionID,
			userID,
			domain.SessionStatusCompleted,
			finalScore,
		).Scan(
			&session.ID,
			&session.UserID,
			&session.ResumeID,
			&session.JobParseID,
			&session.InterviewMode,
			&session.InterviewLanguage,
			&session.TargetRole,
			&session.TargetCompany,
			&session.QuestionIDs,
			&session.Status,
			&session.Score,
			&session.CreatedAt,
			&session.CompletedAt,
		)
		if err == nil {
			return &session, nil
		}
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("session not found")
		}
		return nil, err
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	session, exists := r.sessions[sessionID]
	if !exists || session.UserID != userID {
		return nil, errors.New("session not found")
	}

	total := 0
	count := 0
	for _, item := range r.feedbacks {
		if item.UserID == userID && item.SessionID == sessionID {
			total += item.Score
			count++
		}
	}
	if count == 0 {
		return nil, errors.New("cannot complete session without feedback")
	}

	session.Status = domain.SessionStatusCompleted
	session.Score = int(float64(total)/float64(count) + 0.5)
	now := time.Now().UTC()
	session.CompletedAt = &now
	r.sessions[sessionID] = session

	copy := session
	return &copy, nil
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

		_, err := r.pool.Exec(
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
		if err == nil {
			return record, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
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

		_, err := r.pool.Exec(
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
		if err == nil {
			return record, nil
		}
	}

	r.mu.Lock()
	defer r.mu.Unlock()
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
		var questionIDs []string
		err := r.pool.QueryRow(
			ctx,
			`SELECT user_id, question_ids FROM app_practice_sessions WHERE id = $1`,
			sessionID,
		).Scan(&sessionUserID, &questionIDs)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return errors.New("session not found")
			}
			return err
		}

		if sessionUserID != userID {
			return errors.New("session not found")
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

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

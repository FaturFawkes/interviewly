package repository

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/interview_app/backend/internal/domain"
	"github.com/jackc/pgx/v5/pgxpool"
)

type interviewRepository struct {
	pool       *pgxpool.Pool
	mu         sync.RWMutex
	parsedJobs map[string]*domain.ParsedJobDescription
	resumes    map[string]*domain.ResumeRecord
	questions  map[string]domain.StoredQuestion
	sessions   map[string]domain.PracticeSession
}

// NewInterviewRepository creates interview repository with postgres support and in-memory fallback.
func NewInterviewRepository(pool *pgxpool.Pool) domain.InterviewRepository {
	return &interviewRepository{
		pool:       pool,
		parsedJobs: make(map[string]*domain.ParsedJobDescription),
		resumes:    make(map[string]*domain.ResumeRecord),
		questions:  make(map[string]domain.StoredQuestion),
		sessions:   make(map[string]domain.PracticeSession),
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
	now := time.Now().UTC()
	resume := &domain.ResumeRecord{
		ID:        uuid.NewString(),
		UserID:    userID,
		Content:   content,
		CreatedAt: now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_resumes
				(id, user_id, content, created_at)
			 VALUES
				($1, $2, $3, $4)`,
			resume.ID,
			resume.UserID,
			resume.Content,
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

func (r *interviewRepository) CreatePracticeSession(userID, resumeID, jobParseID string, questionIDs []string) (*domain.PracticeSession, error) {
	now := time.Now().UTC()
	session := &domain.PracticeSession{
		ID:          uuid.NewString(),
		UserID:      userID,
		ResumeID:    resumeID,
		JobParseID:  jobParseID,
		QuestionIDs: append([]string{}, questionIDs...),
		Status:      domain.SessionStatusActive,
		Score:       0,
		CreatedAt:   now,
	}

	if r.pool != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		_, err := r.pool.Exec(
			ctx,
			`INSERT INTO app_practice_sessions
				(id, user_id, resume_id, job_parse_id, question_ids, status, score, created_at)
			 VALUES
				($1, $2, $3, $4, $5, $6, $7, $8)`,
			session.ID,
			session.UserID,
			session.ResumeID,
			session.JobParseID,
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
			`SELECT id, user_id, resume_id, job_parse_id, question_ids, status, score, created_at, completed_at
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

	return result, nil
}

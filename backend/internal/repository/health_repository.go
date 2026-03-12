package repository

import "github.com/interview_app/backend/internal/domain"

type healthRepository struct{}

// NewHealthRepository returns a new instance of HealthRepository.
func NewHealthRepository() domain.HealthRepository {
	return &healthRepository{}
}

// Check provides the current health data.
func (r *healthRepository) Check() (*domain.HealthStatus, error) {
	return &domain.HealthStatus{
		Status:  "ok",
		Version: "1.0.0",
	}, nil
}

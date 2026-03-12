package usecase

import "github.com/interview_app/backend/internal/domain"

type healthUseCase struct {
	healthRepo domain.HealthRepository
}

// NewHealthUseCase creates a new HealthUseCase with the provided repository.
func NewHealthUseCase(repo domain.HealthRepository) domain.HealthUseCase {
	return &healthUseCase{healthRepo: repo}
}

// GetHealthStatus retrieves the application health status.
func (uc *healthUseCase) GetHealthStatus() (*domain.HealthStatus, error) {
	return uc.healthRepo.Check()
}

package domain

// HealthStatus represents the health state of the application.
type HealthStatus struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

// HealthRepository defines the contract for health data access.
type HealthRepository interface {
	Check() (*HealthStatus, error)
}

// HealthUseCase defines the contract for health business logic.
type HealthUseCase interface {
	GetHealthStatus() (*HealthStatus, error)
}

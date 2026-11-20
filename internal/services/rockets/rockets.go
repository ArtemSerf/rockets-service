package rockets

import (
	"context"
	"time"

	"rockets/internal/models"
	repository "rockets/internal/repositories/rockets"
)

type Service struct {
	repository *repository.Repository
	now        func() time.Time
}

func NewService(repository *repository.Repository) *Service {
	return &Service{repository: repository, now: time.Now}
}

func (s *Service) Ping(ctx context.Context) error {
	return s.repository.Ping(ctx)
}

func (s *Service) Rocket(ctx context.Context, channel string) (models.Rocket, error) {
	return s.repository.Rocket(ctx, channel)
}

func (s *Service) Rockets(ctx context.Context, opts models.ListOptions) ([]models.Rocket, error) {
	if err := validateListOptions(opts); err != nil {
		return nil, err
	}
	return s.repository.Rockets(ctx, opts)
}

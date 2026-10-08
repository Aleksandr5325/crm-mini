package sources

import (
	"context"
	"strings"
)

// Store — то, что сервису нужно от хранилища источников
type Store interface {
	List(ctx context.Context) ([]Source, error)
	Create(ctx context.Context, s Source) error
}

// Service — бизнес-логика источников
type Service struct {
	store Store
}

// NewService создаёт сервис источников.
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List возвращает все источники
func (s *Service) List(ctx context.Context) ([]Source, error) {
	return s.store.List(ctx)
}

// Create проверяет данные и сохраняет новый источник
func (s *Service) Create(ctx context.Context, src Source) (Source, error) {
	src.Code = strings.TrimSpace(src.Code)
	src.Name = strings.TrimSpace(src.Name)

	if err := src.Validate(); err != nil {
		return Source{}, err
	}

	if err := s.store.Create(ctx, src); err != nil {
		return Source{}, err
	}

	return src, nil
}

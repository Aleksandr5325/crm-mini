package employees

import (
	"context"
)

// Store - то, что сервису нужно от хранилища сотрудников
type Store interface {
	List(ctx context.Context) ([]Employee, error)
}

// Service - бизнес-логика сотрудников
type Service struct {
	store Store
}

// NewService - создает сервис сотрудников
func NewService(store Store) *Service {
	return &Service{store: store}
}

// List возвращает всех сотрудников
func (s *Service) List(ctx context.Context) ([]Employee, error) {
	return s.store.List(ctx)
}

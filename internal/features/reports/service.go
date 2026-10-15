package reports

import "context"

// Store — то, что сервису нужно от хранилища отчётов
type Store interface {
	Funnel(ctx context.Context) (Funnel, error)
}

// Service — отчёты
type Service struct {
	store Store
}

// NewService создаёт сервис отчётов
func NewService(store Store) *Service {
	return &Service{store: store}
}

// Funnel возвращает отчёт «Воронка продаж»
func (s *Service) Funnel(ctx context.Context) (Funnel, error) {
	return s.store.Funnel(ctx)
}

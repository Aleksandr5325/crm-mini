package leads

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
	"github.com/google/uuid"
)

// Store — то, что сервису нужно от хранилища лидов
type Store interface {
	Create(ctx context.Context, l Lead) (Lead, error)
	Get(ctx context.Context, id uuid.UUID) (Lead, error)
	List(ctx context.Context, f ListFilter) ([]Lead, error)
}

// Service — бизнес-логика лидов
type Service struct {
	store Store
}

// NewService создаёт сервис лидов
func NewService(store Store) *Service {
	return &Service{store: store}
}

// CreateInput — данные, из которых создаётся лид
type CreateInput struct {
	SourceCode string    `json:"source_code"`
	OwnerID    uuid.UUID `json:"owner_id"`
	FullName   string    `json:"full_name"`
	Phone      *string   `json:"phone"`
	Email      *string   `json:"email"`
	Comment    *string   `json:"comment"`
}

// Create проверяет данные и создаёт лида в стадии new
func (s *Service) Create(ctx context.Context, in CreateInput) (Lead, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return Lead{}, fmt.Errorf("generate lead id: %w", err)
	}

	now := time.Now().UTC().Truncate(time.Microsecond)

	lead := Lead{
		ID:           id,
		SourceCode:   strings.TrimSpace(in.SourceCode),
		OwnerID:      in.OwnerID,
		CurrentStage: StageNew,
		FullName:     strings.TrimSpace(in.FullName),
		Phone:        cleanOptional(in.Phone),
		Email:        cleanOptional(in.Email),
		Comment:      cleanOptional(in.Comment),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := lead.Validate(); err != nil {
		return Lead{}, err
	}

	return s.store.Create(ctx, lead)
}

// Get возвращает лида по id
func (s *Service) Get(ctx context.Context, id uuid.UUID) (Lead, error) {
	return s.store.Get(ctx, id)
}

// Размер страницы списка лидов
const (
	defaultLimit = 50
	maxLimit     = 100
)

// ListFilter — фильтры и страница для списка лидов. Пустые поля не фильтруют
type ListFilter struct {
	SourceCode string
	OwnerID    uuid.UUID
	Stage      Stage
	Limit      int
	Offset     int
}

// List возвращает страницу лидов, подходящих под фильтр, — сначала новые
func (s *Service) List(ctx context.Context, f ListFilter) ([]Lead, error) {
	if f.Stage != "" && !f.Stage.Valid() {
		return nil, fmt.Errorf("unknown stage %q: %w", f.Stage, apperr.ErrInvalid)
	}

	if f.Limit == 0 {
		f.Limit = defaultLimit
	}
	if f.Limit < 1 || f.Limit > maxLimit {
		return nil, fmt.Errorf("limit must be between 1 and %d: %w", maxLimit, apperr.ErrInvalid)
	}

	if f.Offset < 0 {
		return nil, fmt.Errorf("offset must not be negative: %w", apperr.ErrInvalid)
	}

	return s.store.List(ctx, f)
}

package leads

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
	"github.com/Aleksandr5325/crm-mini/internal/core/request"
	"github.com/google/uuid"
)

// Store — то, что сервису нужно от хранилища лидов
type Store interface {
	Create(ctx context.Context, l Lead) (Lead, error)
	Get(ctx context.Context, id uuid.UUID) (Lead, error)
	List(ctx context.Context, f ListFilter) ([]Lead, error)
	Update(ctx context.Context, l Lead) (Lead, error)
	SoftDelete(ctx context.Context, id uuid.UUID, at time.Time) error
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

// UpdateInput — изменения лида для PATCH. Поля, которых нет в JSON, не меняются
type UpdateInput struct {
	Version    int64                       `json:"version"`
	SourceCode request.Optional[string]    `json:"source_code"`
	OwnerID    request.Optional[uuid.UUID] `json:"owner_id"`
	FullName   request.Optional[string]    `json:"full_name"`
	Phone      request.Optional[string]    `json:"phone"`
	Email      request.Optional[string]    `json:"email"`
	Comment    request.Optional[string]    `json:"comment"`
}

// Update меняет только присланные поля лида. Версия должна совпадать с версией в базе —
// это оптимистическая блокировка: иначе apperr.ErrConflict.
func (s *Service) Update(ctx context.Context, id uuid.UUID, in UpdateInput) (Lead, error) {
	if in.Version < 1 {
		return Lead{}, fmt.Errorf("version is required: %w", apperr.ErrInvalid)
	}

	lead, err := s.store.Get(ctx, id)
	if err != nil {
		return Lead{}, err
	}

	if lead.Version != in.Version {
		return Lead{}, fmt.Errorf("lead %s was changed by someone else (current version %d, yours %d), reload it: %w",
			id, lead.Version, in.Version, apperr.ErrConflict)
	}

	if in.SourceCode.Set {
		if in.SourceCode.Value == nil {
			return Lead{}, fmt.Errorf("source_code cannot be null: %w", apperr.ErrInvalid)
		}
		lead.SourceCode = strings.TrimSpace(*in.SourceCode.Value)
	}
	if in.OwnerID.Set {
		if in.OwnerID.Value == nil {
			return Lead{}, fmt.Errorf("owner_id cannot be null: %w", apperr.ErrInvalid)
		}
		lead.OwnerID = *in.OwnerID.Value
	}
	if in.FullName.Set {
		if in.FullName.Value == nil {
			return Lead{}, fmt.Errorf("full_name cannot be null: %w", apperr.ErrInvalid)
		}
		lead.FullName = strings.TrimSpace(*in.FullName.Value)
	}
	if in.Phone.Set {
		lead.Phone = cleanOptional(in.Phone.Value)
	}
	if in.Email.Set {
		lead.Email = cleanOptional(in.Email.Value)
	}
	if in.Comment.Set {
		lead.Comment = cleanOptional(in.Comment.Value)
	}

	if err := lead.Validate(); err != nil {
		return Lead{}, err
	}

	lead.UpdatedAt = time.Now().UTC().Truncate(time.Microsecond)

	return s.store.Update(ctx, lead)
}

// Delete мягко удаляет лида: он пропадает из API, но остаётся в базе вместе с историей
func (s *Service) Delete(ctx context.Context, id uuid.UUID) error {
	return s.store.SoftDelete(ctx, id, time.Now().UTC().Truncate(time.Microsecond))
}

package leads

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Store — то, что сервису нужно от хранилища лидов
type Store interface {
	Create(ctx context.Context, l Lead) (Lead, error)
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

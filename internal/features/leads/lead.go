// Package leads - лиды: потенциальные клиенты и их движение по воронке продаж
package leads

import (
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

// Stage — стадия воронки продаж
type Stage string

// Стадии воронки по порядку. won и lost — конечные
const (
	StageNew       Stage = "new"
	StageQualified Stage = "qualified"
	StageProposal  Stage = "proposal"
	StageWon       Stage = "won"
	StageLost      Stage = "lost"
)

// Lead — лид
type Lead struct {
	ID           uuid.UUID `json:"id"`
	Number       int64     `json:"number"`
	ExternalID   *string   `json:"external_id"`
	Version      int64     `json:"version"`
	SourceCode   string    `json:"source_code"`
	OwnerID      uuid.UUID `json:"owner_id"`
	CurrentStage Stage     `json:"current_stage"`
	FullName     string    `json:"full_name"`
	Phone        *string   `json:"phone"`
	Email        *string   `json:"email"`
	Comment      *string   `json:"comment"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

var phonePattern = regexp.MustCompile(`^\+[0-9]{10,15}$`)

// Validate проверяет поля лида по тем же правилам, что и CHECK в базе.
func (l Lead) Validate() error {
	if l.SourceCode == "" {
		return fmt.Errorf("source_code is required: %w", apperr.ErrInvalid)
	}

	if l.OwnerID == uuid.Nil {
		return fmt.Errorf("owner_id is required: %w", apperr.ErrInvalid)
	}

	if n := utf8.RuneCountInString(l.FullName); n < 1 || n > 255 {
		return fmt.Errorf("full_name must be 1-255 characters long: %w", apperr.ErrInvalid)
	}

	if l.Phone != nil && !phonePattern.MatchString(*l.Phone) {
		return fmt.Errorf("phone must look like +79991234567: %w", apperr.ErrInvalid)
	}

	if l.Email != nil && (len(*l.Email) > 255 || strings.Count(*l.Email, "@") != 1) {
		return fmt.Errorf("email is invalid: %w", apperr.ErrInvalid)
	}

	if l.Phone == nil && l.Email == nil {
		return fmt.Errorf("phone or email is required: %w", apperr.ErrInvalid)
	}

	if l.Comment != nil && utf8.RuneCountInString(*l.Comment) > 2000 {
		return fmt.Errorf("comment must be at most 2000 characters long: %w", apperr.ErrInvalid)
	}

	return nil
}

// cleanOptional убирает пробелы по краям; пустая строка превращается в nil (NULL в базе).
func cleanOptional(s *string) *string {
	if s == nil {
		return nil
	}

	v := strings.TrimSpace(*s)
	if v == "" {
		return nil
	}

	return &v
}

// Valid сообщает, является ли s одной из стадий воронки
func (s Stage) Valid() bool {
	switch s {
	case StageNew, StageQualified, StageProposal, StageWon, StageLost:
		return true
	}
	return false
}

package leads

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
)

type importFile struct {
	Sources []importSource `json:"sources"`
	Leads   []importLead   `json:"leads"`
}

type importSource struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type importLead struct {
	ExternalID   string        `json:"lead_id"`
	CreatedAt    time.Time     `json:"created_at"`
	SourceCode   string        `json:"source"`
	CurrnetStage Stage         `json:"current_stage"`
	Owner        string        `json:"owner"`
	FullName     string        `json:"full_name"`
	Phone        string        `json:"phone"`
	Email        string        `json:"email"`
	Comment      string        `json:"comment"`
	Stages       []StagePeriod `json:"stages"`
}

// StagePeriod — период, который лид провёл в стадии. LeftAt == nil — стадия текущая
type StagePeriod struct {
	Stage     Stage      `json:"stage"`
	EnteredAt time.Time  `json:"entered_at"`
	LeftAt    *time.Time `json:"left_at"`
}

// importBatch — проверенные данные импорта, готовые к записи в базу
type importBatch struct {
	FileHash   string
	ImportedAt time.Time
	Sources    []importSource
	Leads      []importItem
}

// importItem — лид из файла вместе с именем ответственного и историей стадий
type importItem struct {
	Lead   Lead
	Owner  string
	Stages []StagePeriod
}

// ImportResult — итог импорта
type ImportResult struct {
	Imported int `json:"imported"`
	Skipped  int `json:"skipped"`
}

// Import загружает выгрузку лидов из 1С. Один и тот же файл повторно не загружается:
// по SHA-256 его содержимого — apperr.ErrConflict
func (s *Service) Import(ctx context.Context, data []byte) (ImportResult, error) {
	sum := sha256.Sum256(data)

	var f importFile
	if err := json.Unmarshal(data, &f); err != nil {
		return ImportResult{}, fmt.Errorf("invalid import file: %w", apperr.ErrInvalid)
	}
	if len(f.Leads) == 0 {
		return ImportResult{}, fmt.Errorf("import file has no leads: %w", apperr.ErrInvalid)
	}

	batch := importBatch{
		FileHash:   hex.EncodeToString(sum[:]),
		ImportedAt: time.Now().UTC().Truncate(time.Microsecond),
		Sources:    f.Sources,
	}

	for _, il := range f.Leads {
		item, err := il.toItem(batch.ImportedAt)
		if err != nil {
			return ImportResult{}, fmt.Errorf("lead %s: %w", il.ExternalID, err)
		}
		batch.Leads = append(batch.Leads, item)
	}

	return s.store.Import(ctx, batch)
}

// toItem превращает лида из файла в Lead и проверяет его так же, как при создании через API
func (il importLead) toItem(importedAt time.Time) (importItem, error) {
	externalID := strings.TrimSpace(il.ExternalID)
	if externalID == "" {
		return importItem{}, fmt.Errorf("lead_id is required: %w", apperr.ErrInvalid)
	}

	owner := strings.TrimSpace(il.Owner)
	if owner == "" {
		return importItem{}, fmt.Errorf("owner is required: %w", apperr.ErrInvalid)
	}

	lead := Lead{
		ExternalID:   &externalID,
		SourceCode:   strings.TrimSpace(il.SourceCode),
		CurrentStage: il.CurrnetStage,
		FullName:     strings.TrimSpace(il.FullName),
		Phone:        cleanOptional(&il.Phone),
		Email:        cleanOptional(&il.Email),
		Comment:      cleanOptional(&il.Comment),
		CreatedAt:    il.CreatedAt.UTC(),
		UpdatedAt:    importedAt,
	}

	if lead.SourceCode == "" {
		return importItem{}, fmt.Errorf("source is required: %w", apperr.ErrInvalid)
	}
	if err := lead.validateDetails(); err != nil {
		return importItem{}, err
	}
	if err := validateHistory(lead.CurrentStage, il.Stages); err != nil {
		return importItem{}, err
	}

	return importItem{Lead: lead, Owner: owner, Stages: il.Stages}, nil
}

// validateHistory проверяет историю стадий: стадии известны, закрыты все, кроме последней,
// выход не раньше входа, а последняя стадия совпадает с текущей стадией лида.
func validateHistory(current Stage, stages []StagePeriod) error {
	if len(stages) == 0 {
		return fmt.Errorf("stage history is empty: %w", apperr.ErrInvalid)
	}

	for i, st := range stages {
		if !st.Stage.Valid() {
			return fmt.Errorf("unknown stage %q in history: %w", st.Stage, apperr.ErrInvalid)
		}

		last := i == len(stages)-1
		if last && st.LeftAt != nil {
			return fmt.Errorf("last stage %q must be open (left_at = null): %w", st.Stage, apperr.ErrInvalid)
		}
		if !last && st.LeftAt == nil {
			return fmt.Errorf("only the last stage can be open, but %q is open too: %w", st.Stage, apperr.ErrInvalid)
		}
		if st.LeftAt != nil && st.LeftAt.Before(st.EnteredAt) {
			return fmt.Errorf("stage %q: left_at is before entered_at: %w", st.Stage, apperr.ErrInvalid)
		}
	}

	if last := stages[len(stages)-1].Stage; last != current {
		return fmt.Errorf("current_stage %q does not match last stage in history %q: %w", current, last, apperr.ErrInvalid)
	}

	return nil
}

package leads

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
	"github.com/Aleksandr5325/crm-mini/internal/core/postgres"
)

// Repository читает и пишет лидов в PostgreSQL
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создаёт репозиторий лидов
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// Create сохраняет нового лида и первую запись в истории стадий — в одной транзакции
// Возвращает лида с номером, который выдала база
func (r *Repository) Create(ctx context.Context, l Lead) (Lead, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Lead{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO crm.leads
			(id, source_code, owner_id, current_stage, full_name, phone, email, comment, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING number, version`,
		l.ID, l.SourceCode, l.OwnerID, l.CurrentStage, l.FullName,
		l.Phone, l.Email, l.Comment, l.CreatedAt, l.UpdatedAt,
	).Scan(&l.Number, &l.Version)
	if constraint, ok := postgres.ForeignKeyViolation(err); ok {
		switch constraint {
		case "leads_source_code_fkey":
			return Lead{}, fmt.Errorf("source %q not found: %w", l.SourceCode, apperr.ErrInvalid)
		case "leads_owner_id_fkey":
			return Lead{}, fmt.Errorf("owner %s not found: %w", l.OwnerID, apperr.ErrInvalid)
		}
	}
	if err != nil {
		return Lead{}, fmt.Errorf("insert lead: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO crm.lead_stages (lead_id, stage, entered_at) VALUES ($1, $2, $3)`,
		l.ID, l.CurrentStage, l.CreatedAt,
	)
	if err != nil {
		return Lead{}, fmt.Errorf("insert first stage: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Lead{}, fmt.Errorf("commit: %w", err)
	}

	return l, nil
}

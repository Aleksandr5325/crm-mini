package leads

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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
	if fkErr := leadFKError(err, l); fkErr != nil {
		return Lead{}, fkErr
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

// leadColumns — колонки лида в том порядке, в котором их читает scanLead
const leadColumns = `id, number, external_id, version, source_code, owner_id, current_stage, full_name, phone, email, comment, created_at, updated_at`

// scanner — то, из чего можно прочитать строку: pgx.Row или pgx.Rows
type scanner interface {
	Scan(dest ...any) error
}

// scanLead раскладывает строку из базы по полям лида
func scanLead(row scanner) (Lead, error) {
	var l Lead
	err := row.Scan(
		&l.ID, &l.Number, &l.ExternalID, &l.Version, &l.SourceCode, &l.OwnerID, &l.CurrentStage,
		&l.FullName, &l.Phone, &l.Email, &l.Comment, &l.CreatedAt, &l.UpdatedAt,
	)

	return l, err
}

// Get возвращает лида по id. Удалённых не возвращает
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (Lead, error) {
	row := r.pool.QueryRow(ctx,
		`SELECT `+leadColumns+` FROM crm.leads WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)

	l, err := scanLead(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lead{}, fmt.Errorf("lead %s: %w", id, apperr.ErrNotFound)
	}
	if err != nil {
		return Lead{}, fmt.Errorf("select lead: %w", err)
	}

	return l, nil
}

// List возвращает страницу лидов по фильтру, сначала новые. Удалённых не возвращает
func (r *Repository) List(ctx context.Context, f ListFilter) ([]Lead, error) {
	where := []string{"deleted_at IS NULL"}
	var args []any

	if f.SourceCode != "" {
		args = append(args, f.SourceCode)
		where = append(where, fmt.Sprintf("source_code = $%d", len(args)))
	}
	if f.OwnerID != uuid.Nil {
		args = append(args, f.OwnerID)
		where = append(where, fmt.Sprintf("owner_id = $%d", len(args)))
	}
	if f.Stage != "" {
		args = append(args, f.Stage)
		where = append(where, fmt.Sprintf("current_stage = $%d", len(args)))
	}

	args = append(args, f.Limit, f.Offset)
	query := fmt.Sprintf(
		`SELECT %s FROM crm.leads WHERE %s ORDER BY number DESC LIMIT $%d OFFSET $%d`,
		leadColumns, strings.Join(where, " AND "), len(args)-1, len(args),
	)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query leads: %w", err)
	}
	defer rows.Close()

	result := []Lead{}
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, fmt.Errorf("scan lead: %w", err)
		}
		result = append(result, l)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read leads: %w", err)
	}

	return result, nil
}

// leadFKError переводит нарушение внешних ключей лида в понятную ошибку
// Если внешний ключ не нарушен — возвращает nil
func leadFKError(err error, l Lead) error {
	constraint, ok := postgres.ForeignKeyViolation(err)
	if !ok {
		return nil
	}

	switch constraint {
	case "leads_source_code_fkey":
		return fmt.Errorf("source %q not found: %w", l.SourceCode, apperr.ErrInvalid)
	case "leads_owner_id_fkey":
		return fmt.Errorf("owner %s not found: %w", l.OwnerID, apperr.ErrInvalid)
	}

	return nil
}

// Update сохраняет изменения лида, только если его версия в базе всё ещё равна l.Version
// Если кто-то успел изменить лида раньше — возвращает apperr.ErrConflict
func (r *Repository) Update(ctx context.Context, l Lead) (Lead, error) {
	err := r.pool.QueryRow(ctx, `
		UPDATE crm.leads
		SET source_code = $3, owner_id = $4, full_name = $5, phone = $6, email = $7, comment = $8, 
			updated_at = $9, version = version + 1
		WHERE id = $1 AND version = $2 AND deleted_at IS NULL
		RETURNING version`,
		l.ID, l.Version, l.SourceCode, l.OwnerID, l.FullName, l.Phone, l.Email, l.Comment, l.UpdatedAt,
	).Scan(&l.Version)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lead{}, fmt.Errorf("lead %s was changed by someone else, reload it: %w", l.ID, apperr.ErrConflict)
	}
	if fkErr := leadFKError(err, l); fkErr != nil {
		return Lead{}, fkErr
	}
	if err != nil {
		return Lead{}, fmt.Errorf("update lead: %w", err)
	}

	return l, nil
}

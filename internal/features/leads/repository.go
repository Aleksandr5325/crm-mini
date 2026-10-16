package leads

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

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

// SoftDelete помечает лида удалённым. Строка и её история остаются в базе
// Если лида нет или он уже удалён — возвращает apperr.ErrNotFound
func (r *Repository) SoftDelete(ctx context.Context, id uuid.UUID, at time.Time) error {
	tag, err := r.pool.Exec(ctx, `
		UPDATE crm.leads
		SET deleted_at = $2, updated_at = $2, version = version + 1
		WHERE id = $1 AND deleted_at IS NULL`,
		id, at,
	)
	if err != nil {
		return fmt.Errorf("soft delete lead: %w", err)
	}

	if tag.RowsAffected() == 0 {
		return fmt.Errorf("lead %s: %w", id, apperr.ErrNotFound)
	}

	return nil
}

// MoveStage переводит лида в стадию to и пишет историю — всё в одной транзакции:
// меняет current_stage (с проверкой версии), закрывает открытую стадию и открывает новую
func (r *Repository) MoveStage(ctx context.Context, l Lead, to Stage, at time.Time) (Lead, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Lead{}, fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	row := tx.QueryRow(ctx, `
		UPDATE crm.leads
		SET current_stage = $3, updated_at = $4, version = version + 1
		WHERE id = $1 AND version = $2 AND deleted_at IS NULL
		RETURNING `+leadColumns,
		l.ID, l.Version, to, at,
	)
	moved, err := scanLead(row)
	if errors.Is(err, pgx.ErrNoRows) {
		return Lead{}, fmt.Errorf("lead %s was changed by someone else, reload it: %w", l.ID, apperr.ErrConflict)
	}
	if err != nil {
		return Lead{}, fmt.Errorf("update lead stage: %w", err)
	}

	_, err = tx.Exec(ctx,
		`UPDATE crm.lead_stages SET left_at = $2 WHERE lead_id = $1 AND left_at IS NULL`,
		l.ID, at,
	)
	if err != nil {
		return Lead{}, fmt.Errorf("close current stage: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO crm.lead_stages (lead_id, stage, entered_at) VALUES ($1, $2, $3)`,
		l.ID, to, at,
	)
	if err != nil {
		return Lead{}, fmt.Errorf("open new stage: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Lead{}, fmt.Errorf("commit: %w", err)
	}

	return moved, nil
}

// Import записывает весь файл импорта в одной транзакции. Первым делом сохраняет хеш файла:
// если такой уже есть — apperr.ErrConflict. Лиды, чей external_id уже есть в базе, пропускает
func (r *Repository) Import(ctx context.Context, b importBatch) (ImportResult, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return ImportResult{}, fmt.Errorf("begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var logID int64
	err = tx.QueryRow(ctx,
		`INSERT INTO crm.import_logs (file_hash, rows_imported, imported_at) VALUES ($1, 0, $2) RETURNING id`,
		b.FileHash, b.ImportedAt,
	).Scan(&logID)
	if postgres.IsUniqueViolation(err) {
		return ImportResult{}, fmt.Errorf("this file has already been imported: %w", apperr.ErrConflict)
	}
	if err != nil {
		return ImportResult{}, fmt.Errorf("insert import log: %w", err)
	}

	for _, s := range b.Sources {
		_, err := tx.Exec(ctx,
			`INSERT INTO crm.sources (code, name) VALUES ($1, $2) ON CONFLICT (code) DO NOTHING`,
			s.Code, s.Name,
		)
		if err != nil {
			return ImportResult{}, fmt.Errorf("insert sources %q: %w", s.Code, err)
		}
	}

	owners := map[string]uuid.UUID{}
	var res ImportResult

	for _, item := range b.Leads {
		l := item.Lead

		ownerID, ok := owners[item.Owner]
		if !ok {
			ownerID, err = findOrCreateEmployee(ctx, tx, item.Owner, b.ImportedAt)
			if err != nil {
				return ImportResult{}, err
			}
			owners[item.Owner] = ownerID
		}
		l.OwnerID = ownerID

		l.ID, err = uuid.NewV7()
		if err != nil {
			return ImportResult{}, fmt.Errorf("generate lead id: %w", err)
		}

		err = tx.QueryRow(ctx, `
			INSERT INTO crm.leads
				(id, external_id, source_code, owner_id, current_stage, full_name, phone, email, comment, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (external_id) DO NOTHING
			RETURNING id`,
			l.ID, l.ExternalID, l.SourceCode, l.OwnerID, l.CurrentStage, l.FullName,
			l.Phone, l.Email, l.Comment, l.CreatedAt, l.UpdatedAt,
		).Scan(&l.ID)
		if errors.Is(err, pgx.ErrNoRows) {
			res.Skipped++
			continue
		}
		if fkErr := leadFKError(err, l); fkErr != nil {
			return ImportResult{}, fmt.Errorf("lead %s: %w", *l.ExternalID, fkErr)
		}
		if err != nil {
			return ImportResult{}, fmt.Errorf("insert lead %s: %w", *l.ExternalID, err)
		}

		for _, st := range item.Stages {
			_, err := tx.Exec(ctx,
				`INSERT INTO crm.lead_stages (lead_id, stage, entered_at, left_at) VALUES ($1, $2, $3, $4)`,
				l.ID, st.Stage, st.EnteredAt, st.LeftAt,
			)
			if err != nil {
				return ImportResult{}, fmt.Errorf("insert stage history of lead %s: %w", *l.ExternalID, err)
			}
		}

		res.Imported++
	}

	_, err = tx.Exec(ctx, `UPDATE crm.import_logs SET rows_imported = $2 WHERE id = $1`, logID, res.Imported)
	if err != nil {
		return ImportResult{}, fmt.Errorf("update import log: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return ImportResult{}, fmt.Errorf("commit: %w", err)
	}

	return res, nil
}

// findOrCreateEmployee возвращает id сотрудника с таким ФИО, а если его нет — создаёт
func findOrCreateEmployee(ctx context.Context, tx pgx.Tx, fullName string, at time.Time) (uuid.UUID, error) {
	var id uuid.UUID
	err := tx.QueryRow(ctx,
		`SELECT id FROM crm.employees WHERE full_name = $1 ORDER BY created_at LIMIT 1`,
		fullName,
	).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, fmt.Errorf("find employee %q: %w", fullName, err)
	}

	id, err = uuid.NewV7()
	if err != nil {
		return uuid.Nil, fmt.Errorf("generate employee id: %w", err)
	}

	_, err = tx.Exec(ctx,
		`INSERT INTO crm.employees (id, full_name, created_at) VALUES ($1, $2, $3)`,
		id, fullName, at,
	)
	if err != nil {
		return uuid.Nil, fmt.Errorf("create employee %q: %w", fullName, err)
	}

	return id, nil
}

// Stages возвращает историю стадий лида по порядку.
// У каждого лида есть хотя бы одна стадия, поэтому пустая история значит, что лида нет — apperr.ErrNotFound
func (r *Repository) Stages(ctx context.Context, id uuid.UUID) ([]StagePeriod, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT ls.stage, ls.entered_at, ls.left_at
		FROM crm.lead_stages ls
		JOIN crm.leads l ON l.id = ls.lead_id
		WHERE ls.lead_id = $1 AND l.deleted_at IS NULL
		ORDER BY ls.entered_at`,
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("query lead stages: %w", err)
	}
	defer rows.Close()

	result := []StagePeriod{}
	for rows.Next() {
		var p StagePeriod
		if err := rows.Scan(&p.Stage, &p.EnteredAt, &p.LeftAt); err != nil {
			return nil, fmt.Errorf("scan lead stages: %w", err)
		}
		result = append(result, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read lead stages: %w", err)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("lead %s: %w", id, apperr.ErrNotFound)
	}

	return result, nil
}

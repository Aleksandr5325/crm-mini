package sources

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Aleksandr5325/crm-mini/internal/core/apperr"
	"github.com/Aleksandr5325/crm-mini/internal/core/postgres"
)

// Repository читает и пишет источники в PostgreSQL.
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создаёт репозиторий источников.
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List возвращает все источники, отсортированные по коду
func (r *Repository) List(ctx context.Context) ([]Source, error) {
	rows, err := r.pool.Query(ctx, `SELECT code, name FROM crm.sources ORDER BY code`)
	if err != nil {
		return nil, fmt.Errorf("query sources: %w", err)
	}
	defer rows.Close()

	sources := []Source{}
	for rows.Next() {
		var s Source
		if err := rows.Scan(&s.Code, &s.Name); err != nil {
			return nil, fmt.Errorf("scan sources: %w", err)
		}
		sources = append(sources, s)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read sources: %w", err)
	}

	return sources, nil
}

// Create добавляет новый источник. Если код уже занят — возвращает apperr.ErrConflict.
func (r *Repository) Create(ctx context.Context, s Source) error {
	_, err := r.pool.Exec(ctx, `INSERT INTO crm.sources (code, name) VALUES ($1, $2)`, s.Code, s.Name)
	if postgres.IsUniqueViolation(err) {
		return fmt.Errorf("source %q already exists: %w", s.Code, apperr.ErrConflict)
	}
	if err != nil {
		return fmt.Errorf("insert source: %w", err)
	}

	return nil
}

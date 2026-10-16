package employees

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository читает сотрудников из PostgreSQL
type Repository struct {
	pool *pgxpool.Pool
}

// NewRepository создает репозиторий сотрудников
func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool}
}

// List возвращает всех сотрудников, отсортированные по имени
func (r *Repository) List(ctx context.Context) ([]Employee, error) {
	rows, err := r.pool.Query(ctx, `SELECT id, full_name, email, role FROM crm.employees ORDER BY full_name`)
	if err != nil {
		return nil, fmt.Errorf("query employees: %w", err)
	}
	defer rows.Close()

	employees := []Employee{}
	for rows.Next() {
		var em Employee
		if err := rows.Scan(&em.ID, &em.FullName, &em.Email, &em.Role); err != nil {
			return nil, fmt.Errorf("scan employees: %w", err)
		}
		employees = append(employees, em)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read employees: %w", err)
	}

	return employees, nil
}

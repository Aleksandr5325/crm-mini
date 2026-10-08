package postgres

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// IsUniqueViolation сообщает, нарушено ли ограничение уникальности (код Postgres 23505).
func IsUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

// ForeignKeyViolation сообщает, нарушен ли внешний ключ (код Postgres 23503),
// и возвращает имя нарушенного ограничения.
func ForeignKeyViolation(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return pgErr.ConstraintName, true
	}

	return "", false
}

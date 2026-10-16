// Package employees — сотрудники: ответственные за лидов
package employees

import "github.com/google/uuid"

// Employee — сотрудник
type Employee struct {
	ID       uuid.UUID `json:"id"`
	FullName string    `json:"full_name"`
	Email    *string   `json:"email"`
	Role     string    `json:"role"`
}

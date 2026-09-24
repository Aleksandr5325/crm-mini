package models

import "time"

type Employee struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	FullName  string    `gorm:"size:255" json:"full_name"`
	Email     string    `gorm:"uniqueIndex;size:255" json:"email"`
	Role      string    `gorm:"size:50" json:"role"`
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}
package models

import (
	"time"
)

type Lead struct {
	LeadID       string    `gorm:"primaryKey;type:text" json:"lead_id"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
	Source       string    `gorm:"not null;size:50" json:"source"`
	CurrentStage string    `gorm:"not null;size:20" json:"current_stage"`
	Owner        string    `gorm:"not null;size:100" json:"owner"`

	FullName string `gorm:"size:255" json:"full_name"`
	Phone    string `gorm:"size:50" json:"phone"`
	Email    string `gorm:"size:255;index" json:"email"`
	Comment  string `gorm:"type:text" json:"comment"`
}

type LeadStage struct {
	ID        uint       `gorm:"primaryKey" json:"id"`
	LeadID    string     `gorm:"not null;index" json:"lead_id"`
	Stage     string     `gorm:"not null;size:20" json:"stage"`
	EnteredAt time.Time  `gorm:"not null" json:"entered_at"`
	LeftAt    *time.Time `json:"left_at"`
}

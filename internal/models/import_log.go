package models

import (
	"time"
)

type ImportLog struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	FileHash     string    `gorm:"uniqueIndex;size:64;not null" json:"file_hash"`
	FileName     string    `gorm:"size:255" json:"file_name"`
	ImportedAt   time.Time `gorm:"not null;autoCreateTime" json:"imported_at"`
	RowsImported int       `gorm:"not null" json:"rows_imported"`
}

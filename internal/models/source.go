package models

type Source struct {
	Code string `gorm:"primaryKey;size:50" json:"code"`
	Name string `gorm:"not null;size:100" json:"name"`
}

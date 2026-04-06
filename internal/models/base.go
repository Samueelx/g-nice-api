package models

import (
	"time"

	"gorm.io/gorm"
)

// Base embeds into every model, providing ID, timestamps and soft-delete.
// Using uint primary key instead of gorm.Model to keep JSON tags explicit.
type Base struct {
	ID        uint           `gorm:"primarykey"                   json:"id"`
	CreatedAt time.Time      `                                    json:"created_at"`
	UpdatedAt time.Time      `                                    json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index"                        json:"-"`
}

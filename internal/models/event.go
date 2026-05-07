package models

import "time"

// Event represents a scheduled event created by an admin.
type Event struct {
	Base

	Title       string    `gorm:"not null;size:255"  json:"title"`
	Description string    `gorm:"type:text;not null" json:"description"`
	Location    string    `gorm:"size:255"           json:"location"`
	Date        time.Time `gorm:"not null;index"     json:"date"`
	Price       float64   `gorm:"default:0"          json:"price"`
	Category    string    `gorm:"size:100;index"     json:"category"`
	ImageURL    string    `gorm:"size:500"           json:"image_url"`
	IsFeatured  bool      `gorm:"default:false;index" json:"is_featured"`

	// FK to the admin user who created the event.
	UserID uint `gorm:"not null;index" json:"user_id"`
	Author User `gorm:"foreignKey:UserID" json:"author,omitempty"`
}

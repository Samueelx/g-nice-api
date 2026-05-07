package repository

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// EventFilter holds optional query parameters for listing events.
type EventFilter struct {
	Category  string
	Search    string
	SortBy    string // "date", "price", "title" — defaults to "date"
	SortOrder string // "asc" or "desc" — defaults to "asc"
	Featured  *bool
}

// EventRepository defines the data-access contract for Event entities.
type EventRepository interface {
	Create(event *models.Event) error
	FindByID(id uint) (*models.Event, error)
	FindByIDWithAuthor(id uint) (*models.Event, error)
	// List returns paginated events with optional filtering and sorting.
	// Author is preloaded on each result.
	List(filter EventFilter, limit, offset int) ([]models.Event, int64, error)
	UpdateFields(id uint, fields map[string]interface{}) error
	Delete(id uint) error
}

type eventRepository struct {
	db *gorm.DB
}

// NewEventRepository constructs the GORM-backed EventRepository.
func NewEventRepository(db *gorm.DB) EventRepository {
	return &eventRepository{db: db}
}

func (r *eventRepository) Create(event *models.Event) error {
	return r.db.Create(event).Error
}

func (r *eventRepository) FindByID(id uint) (*models.Event, error) {
	var event models.Event
	err := r.db.First(&event, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &event, err
}

func (r *eventRepository) FindByIDWithAuthor(id uint) (*models.Event, error) {
	var event models.Event
	err := r.db.Preload("Author").First(&event, id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	return &event, err
}

func (r *eventRepository) List(filter EventFilter, limit, offset int) ([]models.Event, int64, error) {
	var events []models.Event
	var total int64

	base := r.db.Model(&models.Event{})

	// ── Filters ───────────────────────────────────────────────────────────────
	if filter.Category != "" {
		base = base.Where("category = ?", filter.Category)
	}
	if filter.Search != "" {
		pattern := "%" + filter.Search + "%"
		base = base.Where("title ILIKE ?", pattern)
	}
	if filter.Featured != nil {
		base = base.Where("is_featured = ?", *filter.Featured)
	}

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// ── Sorting ───────────────────────────────────────────────────────────────
	sortCol := "date"
	switch filter.SortBy {
	case "price":
		sortCol = "price"
	case "title":
		sortCol = "title"
	}

	sortDir := "asc"
	if filter.SortOrder == "desc" {
		sortDir = "desc"
	}

	err := base.
		Preload("Author").
		Order(sortCol + " " + sortDir).
		Limit(limit).
		Offset(offset).
		Find(&events).Error

	return events, total, err
}

func (r *eventRepository) UpdateFields(id uint, fields map[string]interface{}) error {
	result := r.db.Model(&models.Event{}).Where("id = ?", id).Updates(fields)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *eventRepository) Delete(id uint) error {
	result := r.db.Delete(&models.Event{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

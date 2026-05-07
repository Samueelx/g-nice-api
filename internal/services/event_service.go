package services

import (
	"errors"
	"time"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── DTOs ─────────────────────────────────────────────────────────────────────

// CreateEventRequest is the payload for POST /api/v1/events.
type CreateEventRequest struct {
	Title       string    `json:"title"       binding:"required,max=255"`
	Description string    `json:"description" binding:"required"`
	Location    string    `json:"location"    binding:"omitempty,max=255"`
	Date        time.Time `json:"date"        binding:"required"`
	Price       float64   `json:"price"       binding:"omitempty,min=0"`
	Category    string    `json:"category"    binding:"omitempty,max=100"`
	ImageURL    string    `json:"image_url"   binding:"omitempty,max=500"`
	IsFeatured  bool      `json:"is_featured"`
}

// UpdateEventRequest is the payload for PATCH /api/v1/events/:id.
// Pointer fields: nil means "leave unchanged".
type UpdateEventRequest struct {
	Title       *string    `json:"title"       binding:"omitempty,max=255"`
	Description *string    `json:"description"`
	Location    *string    `json:"location"    binding:"omitempty,max=255"`
	Date        *time.Time `json:"date"`
	Price       *float64   `json:"price"       binding:"omitempty,min=0"`
	Category    *string    `json:"category"    binding:"omitempty,max=100"`
	ImageURL    *string    `json:"image_url"   binding:"omitempty,max=500"`
	IsFeatured  *bool      `json:"is_featured"`
}

// EventListFilter mirrors repository.EventFilter but lives in the service layer
// so handlers do not import the repository package directly.
type EventListFilter struct {
	Category  string
	Search    string
	SortBy    string
	SortOrder string
	Featured  *bool
}

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrEventNotFound = errors.New("event not found")

// ── Interface ─────────────────────────────────────────────────────────────────

// EventService handles all event-related business logic.
type EventService interface {
	CreateEvent(adminID uint, req *CreateEventRequest) (*models.Event, error)
	GetEvent(id uint) (*models.Event, error)
	UpdateEvent(id uint, req *UpdateEventRequest) (*models.Event, error)
	DeleteEvent(id uint) error
	ListEvents(filter EventListFilter, page, pageSize int) (*pagination.Page[models.Event], error)
}

// ── Implementation ────────────────────────────────────────────────────────────

type eventService struct {
	eventRepo repository.EventRepository
}

// NewEventService constructs an EventService.
func NewEventService(eventRepo repository.EventRepository) EventService {
	return &eventService{eventRepo: eventRepo}
}

// CreateEvent persists a new event created by the given admin user.
func (s *eventService) CreateEvent(adminID uint, req *CreateEventRequest) (*models.Event, error) {
	event := &models.Event{
		UserID:      adminID,
		Title:       req.Title,
		Description: req.Description,
		Location:    req.Location,
		Date:        req.Date,
		Price:       req.Price,
		Category:    req.Category,
		ImageURL:    req.ImageURL,
		IsFeatured:  req.IsFeatured,
	}

	if err := s.eventRepo.Create(event); err != nil {
		return nil, err
	}

	return s.eventRepo.FindByIDWithAuthor(event.ID)
}

// GetEvent retrieves a single event with its author preloaded.
func (s *eventService) GetEvent(id uint) (*models.Event, error) {
	event, err := s.eventRepo.FindByIDWithAuthor(id)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrEventNotFound
	}
	return event, err
}

// UpdateEvent applies a partial update to an existing event.
func (s *eventService) UpdateEvent(id uint, req *UpdateEventRequest) (*models.Event, error) {
	// Verify the event exists before patching.
	if _, err := s.eventRepo.FindByID(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, ErrEventNotFound
		}
		return nil, err
	}

	fields := make(map[string]interface{})
	if req.Title != nil {
		fields["title"] = *req.Title
	}
	if req.Description != nil {
		fields["description"] = *req.Description
	}
	if req.Location != nil {
		fields["location"] = *req.Location
	}
	if req.Date != nil {
		fields["date"] = *req.Date
	}
	if req.Price != nil {
		fields["price"] = *req.Price
	}
	if req.Category != nil {
		fields["category"] = *req.Category
	}
	if req.ImageURL != nil {
		fields["image_url"] = *req.ImageURL
	}
	if req.IsFeatured != nil {
		fields["is_featured"] = *req.IsFeatured
	}

	if len(fields) == 0 {
		// Nothing to update — return current state.
		return s.eventRepo.FindByIDWithAuthor(id)
	}

	if err := s.eventRepo.UpdateFields(id, fields); err != nil {
		return nil, err
	}

	return s.eventRepo.FindByIDWithAuthor(id)
}

// DeleteEvent soft-deletes an event by ID.
func (s *eventService) DeleteEvent(id uint) error {
	if err := s.eventRepo.Delete(id); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return ErrEventNotFound
		}
		return err
	}
	return nil
}

// ListEvents returns a paginated, filtered list of events.
func (s *eventService) ListEvents(filter EventListFilter, page, pageSize int) (*pagination.Page[models.Event], error) {
	repoFilter := repository.EventFilter{
		Category:  filter.Category,
		Search:    filter.Search,
		SortBy:    filter.SortBy,
		SortOrder: filter.SortOrder,
		Featured:  filter.Featured,
	}

	offset := pagination.Offset(page, pageSize)
	events, total, err := s.eventRepo.List(repoFilter, pageSize, offset)
	if err != nil {
		return nil, err
	}

	return pagination.New(events, total, page, pageSize), nil
}

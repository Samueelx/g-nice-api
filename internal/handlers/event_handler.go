package handlers

import (
	"errors"
	"log"
	"strconv"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// EventHandler handles HTTP requests for event operations.
type EventHandler struct {
	eventSvc services.EventService
}

// NewEventHandler constructs an EventHandler.
func NewEventHandler(eventSvc services.EventService) *EventHandler {
	return &EventHandler{eventSvc: eventSvc}
}

// List godoc
// GET /api/v1/events
// Optional query params: page, page_size, category, search, sort_by, sort_order, featured
func (h *EventHandler) List(c *gin.Context) {
	page, pageSize := parsePagination(c)

	filter := services.EventListFilter{
		Category:  c.Query("category"),
		Search:    c.Query("search"),
		SortBy:    c.Query("sort_by"),
		SortOrder: c.Query("sort_order"),
	}

	if featuredStr := c.Query("featured"); featuredStr != "" {
		v, err := strconv.ParseBool(featuredStr)
		if err != nil {
			BadRequest(c, "featured must be a boolean (true or false)")
			return
		}
		filter.Featured = &v
	}

	result, err := h.eventSvc.ListEvents(filter, page, pageSize)
	if err != nil {
		log.Printf("ListEvents error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// GetEvent godoc
// GET /api/v1/events/:id
func (h *EventHandler) GetEvent(c *gin.Context) {
	eventID, ok := parseEventID(c)
	if !ok {
		return
	}

	event, err := h.eventSvc.GetEvent(eventID)
	if err != nil {
		if errors.Is(err, services.ErrEventNotFound) {
			NotFound(c, "event not found")
			return
		}
		log.Printf("GetEvent error: %v", err)
		InternalError(c)
		return
	}

	OK(c, event)
}

// CreateEvent godoc
// POST /api/v1/events  [Admin]
// Body: { title, description, location, date, price, category, image_url, is_featured? }
func (h *EventHandler) CreateEvent(c *gin.Context) {
	adminID, ok := extractUserID(c)
	if !ok {
		return
	}

	var req services.CreateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	event, err := h.eventSvc.CreateEvent(adminID, &req)
	if err != nil {
		log.Printf("CreateEvent error: %v", err)
		InternalError(c)
		return
	}

	Created(c, event)
}

// UpdateEvent godoc
// PATCH /api/v1/events/:id  [Admin]
// Body: (any subset of event fields)
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	eventID, ok := parseEventID(c)
	if !ok {
		return
	}

	var req services.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		BadRequest(c, err.Error())
		return
	}

	event, err := h.eventSvc.UpdateEvent(eventID, &req)
	if err != nil {
		if errors.Is(err, services.ErrEventNotFound) {
			NotFound(c, "event not found")
			return
		}
		log.Printf("UpdateEvent error: %v", err)
		InternalError(c)
		return
	}

	OK(c, event)
}

// DeleteEvent godoc
// DELETE /api/v1/events/:id  [Admin]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	eventID, ok := parseEventID(c)
	if !ok {
		return
	}

	if err := h.eventSvc.DeleteEvent(eventID); err != nil {
		if errors.Is(err, services.ErrEventNotFound) {
			NotFound(c, "event not found")
			return
		}
		log.Printf("DeleteEvent error: %v", err)
		InternalError(c)
		return
	}

	ok204(c)
}

// ── Private helpers ───────────────────────────────────────────────────────────

func parseEventID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "invalid event id")
		return 0, false
	}
	return uint(id), true
}

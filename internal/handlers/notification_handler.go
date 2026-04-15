package handlers

import (
	"errors"
	"log"
	"strconv"

	"github.com/Samueelx/g-nice-api/internal/services"
	"github.com/gin-gonic/gin"
)

// NotificationHandler handles HTTP requests for notification operations.
type NotificationHandler struct {
	notifSvc services.NotificationService
}

// NewNotificationHandler constructs a NotificationHandler.
func NewNotificationHandler(notifSvc services.NotificationService) *NotificationHandler {
	return &NotificationHandler{notifSvc: notifSvc}
}

// List godoc
//
//	GET /api/v1/notifications?page=1&page_size=20
//
// Returns the authenticated user's paginated notifications, newest first.
func (h *NotificationHandler) List(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	page, pageSize := parsePagination(c)

	result, err := h.notifSvc.List(userID, page, pageSize)
	if err != nil {
		log.Printf("NotificationHandler.List error: %v", err)
		InternalError(c)
		return
	}

	OK(c, result)
}

// UnreadCount godoc
//
//	GET /api/v1/notifications/unread-count
//
// Returns the number of unread notifications for the authenticated user.
func (h *NotificationHandler) UnreadCount(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	count, err := h.notifSvc.UnreadCount(userID)
	if err != nil {
		log.Printf("NotificationHandler.UnreadCount error: %v", err)
		InternalError(c)
		return
	}

	OK(c, gin.H{"unread_count": count})
}

// MarkRead godoc
//
//	PATCH /api/v1/notifications/:nid/read
//
// Marks a single notification as read. Returns 404 if the notification does not
// exist or does not belong to the authenticated user.
func (h *NotificationHandler) MarkRead(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	notifID, ok := parseNotifID(c)
	if !ok {
		return
	}

	if err := h.notifSvc.MarkRead(notifID, userID); err != nil {
		if errors.Is(err, services.ErrNotificationNotFound) {
			NotFound(c, "notification not found")
			return
		}
		log.Printf("NotificationHandler.MarkRead error: %v", err)
		InternalError(c)
		return
	}

	ok204(c)
}

// MarkAllRead godoc
//
//	PATCH /api/v1/notifications/read-all
//
// Marks all unread notifications as read for the authenticated user.
func (h *NotificationHandler) MarkAllRead(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	if err := h.notifSvc.MarkAllRead(userID); err != nil {
		log.Printf("NotificationHandler.MarkAllRead error: %v", err)
		InternalError(c)
		return
	}

	ok204(c)
}

// Delete godoc
//
//	DELETE /api/v1/notifications/:nid
//
// Permanently deletes a notification. Returns 404 if the notification does not
// exist or does not belong to the authenticated user.
func (h *NotificationHandler) Delete(c *gin.Context) {
	userID, ok := extractUserID(c)
	if !ok {
		return
	}

	notifID, ok := parseNotifID(c)
	if !ok {
		return
	}

	if err := h.notifSvc.Delete(notifID, userID); err != nil {
		if errors.Is(err, services.ErrNotificationNotFound) {
			NotFound(c, "notification not found")
			return
		}
		log.Printf("NotificationHandler.Delete error: %v", err)
		InternalError(c)
		return
	}

	ok204(c)
}

// ── Private helpers ───────────────────────────────────────────────────────────

func parseNotifID(c *gin.Context) (uint, bool) {
	id, err := strconv.ParseUint(c.Param("nid"), 10, 64)
	if err != nil || id == 0 {
		BadRequest(c, "invalid notification id")
		return 0, false
	}
	return uint(id), true
}

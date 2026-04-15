package services

import (
	"errors"

	"github.com/Samueelx/g-nice-api/internal/models"
	"github.com/Samueelx/g-nice-api/internal/pagination"
	"github.com/Samueelx/g-nice-api/internal/repository"
)

// ── Sentinel errors ───────────────────────────────────────────────────────────

var ErrNotificationNotFound = errors.New("notification not found")

// ── Interface ─────────────────────────────────────────────────────────────────

// NotificationService manages notification persistence and retrieval.
type NotificationService interface {
	// Notify creates a notification for userID (the recipient) triggered by actorID.
	// It is a no-op when actorID == userID (actor is the recipient).
	// Errors are intentionally swallowed — callers treat this as best-effort.
	Notify(userID, actorID uint, notifType models.NotificationType, targetID *uint, targetType string) error

	// List returns a paginated list of notifications for the authenticated user.
	List(userID uint, page, pageSize int) (*pagination.Page[models.Notification], error)

	// UnreadCount returns the number of unread notifications for the user.
	UnreadCount(userID uint) (int64, error)

	// MarkRead marks a single notification as read (ownership enforced).
	MarkRead(notifID, userID uint) error

	// MarkAllRead marks every unread notification as read for the user.
	MarkAllRead(userID uint) error

	// Delete permanently removes a notification (ownership enforced).
	Delete(notifID, userID uint) error
}

// ── Implementation ────────────────────────────────────────────────────────────

type notificationService struct {
	notifRepo repository.NotificationRepository
}

// NewNotificationService constructs a NotificationService.
func NewNotificationService(notifRepo repository.NotificationRepository) NotificationService {
	return &notificationService{notifRepo: notifRepo}
}

// Notify persists a notification, skipping self-notifications silently.
func (s *notificationService) Notify(
	userID, actorID uint,
	notifType models.NotificationType,
	targetID *uint,
	targetType string,
) error {
	// Never notify a user about their own actions.
	if userID == actorID {
		return nil
	}

	return s.notifRepo.Create(&models.Notification{
		UserID:     userID,
		ActorID:    actorID,
		Type:       notifType,
		TargetID:   targetID,
		TargetType: targetType,
	})
}

func (s *notificationService) List(userID uint, page, pageSize int) (*pagination.Page[models.Notification], error) {
	offset := pagination.Offset(page, pageSize)
	notifications, total, err := s.notifRepo.ListByUserID(userID, pageSize, offset)
	if err != nil {
		return nil, err
	}
	return pagination.New(notifications, total, page, pageSize), nil
}

func (s *notificationService) UnreadCount(userID uint) (int64, error) {
	return s.notifRepo.CountUnread(userID)
}

func (s *notificationService) MarkRead(notifID, userID uint) error {
	err := s.notifRepo.MarkRead(notifID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotificationNotFound
	}
	return err
}

func (s *notificationService) MarkAllRead(userID uint) error {
	return s.notifRepo.MarkAllRead(userID)
}

func (s *notificationService) Delete(notifID, userID uint) error {
	err := s.notifRepo.Delete(notifID, userID)
	if errors.Is(err, repository.ErrNotFound) {
		return ErrNotificationNotFound
	}
	return err
}

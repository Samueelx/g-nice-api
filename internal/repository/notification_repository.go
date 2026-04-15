package repository

import (
	"github.com/Samueelx/g-nice-api/internal/models"
	"gorm.io/gorm"
)

// NotificationRepository defines the data-access contract for Notification entities.
type NotificationRepository interface {
	// Create persists a new notification.
	Create(n *models.Notification) error
	// ListByUserID returns paginated notifications for the given recipient,
	// ordered newest first, with Actor preloaded.
	ListByUserID(userID uint, limit, offset int) ([]models.Notification, int64, error)
	// CountUnread returns the number of unread notifications for a user.
	CountUnread(userID uint) (int64, error)
	// MarkRead marks a single notification as read.
	// Scoped by userID so a user cannot touch another user's notifications.
	MarkRead(notifID, userID uint) error
	// MarkAllRead marks all unread notifications as read for the given user.
	MarkAllRead(userID uint) error
	// Delete permanently removes a notification owned by the given user.
	Delete(notifID, userID uint) error
}

type notificationRepository struct {
	db *gorm.DB
}

// NewNotificationRepository constructs the GORM-backed implementation.
func NewNotificationRepository(db *gorm.DB) NotificationRepository {
	return &notificationRepository{db: db}
}

func (r *notificationRepository) Create(n *models.Notification) error {
	return r.db.Create(n).Error
}

func (r *notificationRepository) ListByUserID(userID uint, limit, offset int) ([]models.Notification, int64, error) {
	var notifications []models.Notification
	var total int64

	base := r.db.Model(&models.Notification{}).Where("user_id = ?", userID)

	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	err := base.
		Preload("Actor").
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&notifications).Error

	return notifications, total, err
}

func (r *notificationRepository) CountUnread(userID uint) (int64, error) {
	var count int64
	err := r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Count(&count).Error
	return count, err
}

func (r *notificationRepository) MarkRead(notifID, userID uint) error {
	result := r.db.Model(&models.Notification{}).
		Where("id = ? AND user_id = ?", notifID, userID).
		Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *notificationRepository) MarkAllRead(userID uint) error {
	return r.db.Model(&models.Notification{}).
		Where("user_id = ? AND is_read = ?", userID, false).
		Update("is_read", true).Error
}

func (r *notificationRepository) Delete(notifID, userID uint) error {
	result := r.db.
		Where("id = ? AND user_id = ?", notifID, userID).
		Delete(&models.Notification{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

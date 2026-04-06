package models

// NotificationType categorises what triggered a notification.
type NotificationType string

const (
	NotifTypeLikePost      NotificationType = "like_post"
	NotifTypeLikeComment   NotificationType = "like_comment"
	NotifTypeComment       NotificationType = "comment"
	NotifTypeReply         NotificationType = "reply"
	NotifTypeFollow        NotificationType = "follow"
	NotifTypeMention       NotificationType = "mention"
)

// Notification is delivered to a User when something happens related to them.
// ActorID is the user who triggered the event.
// TargetID + TargetType point to the entity involved (e.g. a Post or Comment).
type Notification struct {
	Base

	UserID  uint             `gorm:"not null;index"          json:"user_id"`  // recipient
	ActorID uint             `gorm:"not null;index"          json:"actor_id"` // who did the action
	Type    NotificationType `gorm:"type:varchar(30);not null" json:"type"`

	// Polymorphic target (optional)
	TargetID   *uint  `gorm:"index"          json:"target_id,omitempty"`
	TargetType string `gorm:"size:20"        json:"target_type,omitempty"`

	IsRead bool `gorm:"default:false;index" json:"is_read"`

	// Associations
	User  User `gorm:"foreignKey:UserID"  json:"-"`
	Actor User `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
}

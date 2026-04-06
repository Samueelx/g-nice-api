package models

// LikeTargetType identifies what kind of entity was liked.
type LikeTargetType string

const (
	LikeTargetPost    LikeTargetType = "post"
	LikeTargetComment LikeTargetType = "comment"
)

// Like records that a User liked a Post or Comment.
// The composite unique index prevents double-liking.
type Like struct {
	Base

	UserID     uint           `gorm:"not null;index"                              json:"user_id"`
	TargetID   uint           `gorm:"not null;index"                              json:"target_id"`
	TargetType LikeTargetType `gorm:"type:varchar(20);not null"                   json:"target_type"`

	// Associations
	User User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName overrides the default table name.
func (Like) TableName() string { return "likes" }

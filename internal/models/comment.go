package models

// Comment is a reply to a Post.
// ParentID enables one level of threading (reply-to-comment).
// Set ParentID to nil for top-level comments.
type Comment struct {
	Base

	PostID   uint   `gorm:"not null;index"     json:"post_id"`
	UserID   uint   `gorm:"not null;index"     json:"user_id"`
	Content  string `gorm:"type:text;not null" json:"content"`

	// Optional: self-referential for threaded replies
	ParentID *uint `gorm:"index" json:"parent_id,omitempty"`

	// Denormalised counter
	LikesCount int `gorm:"default:0" json:"likes_count"`

	// Associations
	Author  User      `gorm:"foreignKey:UserID"   json:"author,omitempty"`
	Post    Post      `gorm:"foreignKey:PostID"   json:"-"`
	Parent  *Comment  `gorm:"foreignKey:ParentID" json:"-"`
	Replies []Comment `gorm:"foreignKey:ParentID" json:"replies,omitempty"`
}

package models

// MediaType defines the kind of media attached to a post.
type MediaType string

const (
	MediaTypeNone  MediaType = ""
	MediaTypeImage MediaType = "image"
	MediaTypeVideo MediaType = "video"
	MediaTypeGIF   MediaType = "gif"
)

// Post is a piece of content published by a User.
type Post struct {
	Base

	UserID  uint   `gorm:"not null;index"  json:"user_id"`
	Content string `gorm:"type:text;not null" json:"content"`

	// Optional media attachment
	MediaURL  string    `gorm:"size:500" json:"media_url,omitempty"`
	MediaType MediaType `gorm:"size:20"  json:"media_type,omitempty"`

	// Visibility
	IsPublic bool `gorm:"default:true;index" json:"is_public"`

	// Denormalised counters
	LikesCount    int `gorm:"default:0" json:"likes_count"`
	CommentsCount int `gorm:"default:0" json:"comments_count"`

	// Associations
	Author   User      `gorm:"foreignKey:UserID"  json:"author,omitempty"`
	Comments []Comment `gorm:"foreignKey:PostID"  json:"comments,omitempty"`
	Likes    []Like    `gorm:"foreignKey:TargetID;references:ID" json:"-"`
}

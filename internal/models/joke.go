package models

import "time"

// Joke represents a Jumbotron joke of the day
type Joke struct {
	Base

	Content           string    `gorm:"type:text;not null" json:"content"`
	SponsorName       *string   `gorm:"size:150"           json:"sponsor_name"`
	SponsorLogoURL    *string   `gorm:"type:text"          json:"sponsor_logo_url"`
	SponsorWebsiteURL *string   `gorm:"type:text"          json:"sponsor_website_url"`
	ActiveDate        time.Time `gorm:"type:date;unique;not null" json:"active_date"`
	
	CreatedByID       uint      `gorm:"not null"           json:"created_by_id"`
	CreatedBy         User      `gorm:"foreignKey:CreatedByID" json:"created_by"`

	LikesCount        int       `gorm:"default:0"          json:"likes_count"`
	CommentsCount     int       `gorm:"default:0"          json:"comments_count"`
	IsLiked           bool      `gorm:"-"                  json:"is_liked"`
	
	// Associations
	Likes             []JokeLike    `gorm:"foreignKey:JokeID" json:"-"`
	Comments          []JokeComment `gorm:"foreignKey:JokeID" json:"-"`
}

// JokeLike represents a user liking a joke
type JokeLike struct {
	Base

	JokeID uint `gorm:"uniqueIndex:idx_joke_like;not null" json:"joke_id"`
	UserID uint `gorm:"uniqueIndex:idx_joke_like;not null" json:"user_id"`
	User   User `gorm:"foreignKey:UserID" json:"-"`
}

// JokeComment represents a comment on a joke
type JokeComment struct {
	Base

	JokeID       uint         `gorm:"not null" json:"joke_id"`
	UserID       uint         `gorm:"not null" json:"user_id"`
	User         User         `gorm:"foreignKey:UserID" json:"author"`
	ParentID     *uint        `json:"parent_id"`
	Parent       *JokeComment `gorm:"foreignKey:ParentID" json:"-"`
	Content      string       `gorm:"type:text;not null" json:"content"`
	
	LikesCount   int          `gorm:"default:0" json:"likes_count"`
	RepliesCount int          `gorm:"default:0" json:"replies_count"`
	IsLiked      bool         `gorm:"-"         json:"is_liked"`

	// Associations
	Replies      []JokeComment     `gorm:"foreignKey:ParentID" json:"-"`
	Likes        []JokeCommentLike `gorm:"foreignKey:CommentID" json:"-"`
}

// JokeCommentLike represents a user liking a joke comment
type JokeCommentLike struct {
	Base

	CommentID uint `gorm:"uniqueIndex:idx_joke_comment_like;not null" json:"comment_id"`
	UserID    uint `gorm:"uniqueIndex:idx_joke_comment_like;not null" json:"user_id"`
	User      User `gorm:"foreignKey:UserID" json:"-"`
}

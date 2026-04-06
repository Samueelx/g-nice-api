package models

// User represents an account in the platform.
// PasswordHash is always excluded from JSON responses.
type User struct {
	Base

	Username     string  `gorm:"uniqueIndex;not null;size:50"  json:"username"`
	Email        string  `gorm:"uniqueIndex;not null;size:255" json:"email"`
	PasswordHash string  `gorm:"not null"                      json:"-"`
	DisplayName  string  `gorm:"size:100"                      json:"display_name"`
	Bio          string  `gorm:"type:text"                     json:"bio"`
	AvatarURL    string  `gorm:"size:500"                      json:"avatar_url"`
	IsVerified   bool    `gorm:"default:false"                 json:"is_verified"`
	IsPrivate    bool    `gorm:"default:false"                 json:"is_private"`

	// Denormalised counters — updated via service layer; avoids expensive COUNT(*) on hot paths.
	PostsCount     int `gorm:"default:0" json:"posts_count"`
	FollowersCount int `gorm:"default:0" json:"followers_count"`
	FollowingCount int `gorm:"default:0" json:"following_count"`

	// Associations (not loaded by default — use Preload explicitly)
	Posts         []Post         `gorm:"foreignKey:UserID"   json:"-"`
	Comments      []Comment      `gorm:"foreignKey:UserID"   json:"-"`
	Likes         []Like         `gorm:"foreignKey:UserID"   json:"-"`
	Followers     []Follow       `gorm:"foreignKey:FolloweeID" json:"-"`
	Following     []Follow       `gorm:"foreignKey:FollowerID" json:"-"`
	Notifications []Notification `gorm:"foreignKey:UserID"   json:"-"`
}

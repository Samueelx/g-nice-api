package models

// Follow represents a directed "follower → followee" relationship.
// A unique index on (follower_id, followee_id) prevents duplicate follows.
type Follow struct {
	Base

	FollowerID uint `gorm:"not null;index" json:"follower_id"`
	FolloweeID uint `gorm:"not null;index" json:"followee_id"`

	// Associations
	Follower User `gorm:"foreignKey:FollowerID" json:"follower,omitempty"`
	Followee User `gorm:"foreignKey:FolloweeID" json:"followee,omitempty"`
}

// TableName overrides the default table name.
func (Follow) TableName() string { return "follows" }

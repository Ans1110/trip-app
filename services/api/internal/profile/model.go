package profile

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Profile struct {
	UserID          uuid.UUID      `gorm:"type:uuid;primaryKey;column:user_id"`
	Username        string         `gorm:"column:username;type:citext;not null;uniqueIndex"`
	Bio             string         `gorm:"column:bio;not null;default:''"`
	AvatarURL       string         `gorm:"column:avatar_url;not null;default:''"`
	CoverImage      string         `gorm:"column:cover_image;not null;default:''"`
	TravelTags      pq.StringArray `gorm:"column:travel_tags;type:text[]"`
	SocialInstagram string         `gorm:"column:social_instagram;not null;default:''"`
	SocialX         string         `gorm:"column:social_x;not null;default:''"`
	SocialYoutube   string         `gorm:"column:social_youtube;not null;default:''"`
	SocialTiktok    string         `gorm:"column:social_tiktok;not null;default:''"`
	CreatedAt       time.Time      `gorm:"column:created_at"`
	UpdatedAt       time.Time      `gorm:"column:updated_at"`
}

func (Profile) TableName() string { return "profile.profiles" }

type Follow struct {
	FollowerID uuid.UUID `gorm:"type:uuid;primaryKey;column:follower_id"`
	FolloweeID uuid.UUID `gorm:"type:uuid;primaryKey;column:followee_id"`
	CreatedAt  time.Time `gorm:"column:created_at"`
}

func (Follow) TableName() string { return "profile.follows" }

type FollowCounts struct {
	UserID         uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	FollowersCount int       `gorm:"column:followers_count;not null;default:0"`
	FollowingCount int       `gorm:"column:following_count;not null;default:0"`
	UpdatedAt      time.Time `gorm:"column:updated_at"`
}

func (FollowCounts) TableName() string { return "profile.follow_counts" }

type FeedEntry struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID      uuid.UUID `gorm:"type:uuid;column:user_id;not null;index"`
	ActorID     uuid.UUID `gorm:"type:uuid;column:actor_id;not null"`
	EventType   string    `gorm:"column:event_type;not null"`
	TripID      *uuid.UUID `gorm:"type:uuid;column:trip_id"`
	PublishedAt time.Time `gorm:"column:published_at;not null"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func (FeedEntry) TableName() string { return "profile.feed" }

const (
	EventTripPublished = "TRIP_PUBLISHED"
)

package profile

import "time"

type UpdateProfilePayload struct {
	Username        *string   `json:"username,omitempty" binding:"omitempty,min=3,max=32"`
	Bio             *string   `json:"bio,omitempty" binding:"omitempty,max=500"`
	AvatarURL       *string   `json:"avatar_url,omitempty" binding:"omitempty,max=1024,url"`
	CoverImage      *string   `json:"cover_image,omitempty" binding:"omitempty,max=1024,url"`
	TravelTags      *[]string `json:"travel_tags,omitempty" binding:"omitempty,max=20,dive,max=32"`
	SocialInstagram *string   `json:"social_instagram,omitempty" binding:"omitempty,max=256,url"`
	SocialX         *string   `json:"social_x,omitempty" binding:"omitempty,max=256,url"`
	SocialYoutube   *string   `json:"social_youtube,omitempty" binding:"omitempty,max=256,url"`
	SocialTiktok    *string   `json:"social_tiktok,omitempty" binding:"omitempty,max=256,url"`
}

type ProfileResponse struct {
	UserID          string    `json:"user_id"`
	Username        string    `json:"username"`
	Name            string    `json:"name"`
	Bio             string    `json:"bio"`
	AvatarURL       string    `json:"avatar_url"`
	CoverImage      string    `json:"cover_image"`
	TravelTags      []string  `json:"travel_tags"`
	SocialInstagram string    `json:"social_instagram,omitempty"`
	SocialX         string    `json:"social_x,omitempty"`
	SocialYoutube   string    `json:"social_youtube,omitempty"`
	SocialTiktok    string    `json:"social_tiktok,omitempty"`
	FollowersCount  int       `json:"followers_count"`
	FollowingCount  int       `json:"following_count"`
	IsFollowing     bool      `json:"is_following"`
	IsSelf          bool      `json:"is_self"`
	CreatedAt       time.Time `json:"created_at"`
}

type UserSummary struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type ListUsersResponse struct {
	Users      []UserSummary `json:"users"`
	NextCursor string        `json:"next_cursor,omitempty"`
}

type FeedItemResponse struct {
	ID          string      `json:"id"`
	EventType   string      `json:"event_type"`
	Actor       UserSummary `json:"actor"`
	TripID      string      `json:"trip_id,omitempty"`
	PublishedAt time.Time   `json:"published_at"`
}

type FeedResponse struct {
	Items      []FeedItemResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

type RecommendationResponse struct {
	Users []UserSummary `json:"users"`
}

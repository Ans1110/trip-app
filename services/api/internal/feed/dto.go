package feed

import (
	"time"

	"github.com/Ans1110/trip-app/internal/post"
)

type FeedItemResponse struct {
	ID          string             `json:"id"`
	EventType   string             `json:"event_type"`
	SubjectType string             `json:"subject_type"`
	SubjectID   string             `json:"subject_id"`
	PublishedAt time.Time          `json:"published_at"`
	Post        *post.PostResponse `json:"post,omitempty"`
}

type ListFeedResponse struct {
	Items      []FeedItemResponse `json:"items"`
	NextCursor string             `json:"next_cursor,omitempty"`
}

package feed

import "github.com/Ans1110/trip-app/internal/post"

type ListFeedResponse struct {
	Posts      []post.PostResponse `json:"posts"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

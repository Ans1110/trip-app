package search

import "github.com/Ans1110/trip-app/internal/post"

type SearchPostsResponse struct {
	Query      string              `json:"query"`
	Posts      []post.PostResponse `json:"posts"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

package post

import "time"

type CreatePostPayload struct {
	Title      string   `json:"title" binding:"required,min=1,max=200"`
	Content    string   `json:"content" binding:"required,max=20000"`
	CoverImage string   `json:"cover_image" binding:"omitempty,max=1024,url"`
	Tags       []string `json:"tags" binding:"omitempty,max=20,dive,max=32"`
}

type UpdatePostPayload struct {
	Title      *string   `json:"title,omitempty" binding:"omitempty,min=1,max=200"`
	Content    *string   `json:"content,omitempty" binding:"omitempty,max=20000"`
	CoverImage *string   `json:"cover_image,omitempty" binding:"omitempty,max=1024,url"`
	Tags       *[]string `json:"tags,omitempty" binding:"omitempty,max=20,dive,max=32"`
}

type PostResponse struct {
	ID           string      `json:"id"`
	Author       UserSummary `json:"author"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	CoverImage   string      `json:"cover_image,omitempty"`
	Tags         []string    `json:"tags"`
	LikeCount    int         `json:"like_count"`
	CommentCount int         `json:"comment_count"`
	IsLiked      bool        `json:"is_liked"`
	IsAuthor     bool        `json:"is_author"`
	PublishedAt  time.Time   `json:"published_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type ListPostsResponse struct {
	Posts      []PostResponse `json:"posts"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

type CreateCommentPayload struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

type UpdateCommentPayload struct {
	Content string `json:"content" binding:"required,min=1,max=2000"`
}

type CommentResponse struct {
	ID        string      `json:"id"`
	PostID    string      `json:"post_id"`
	Author    UserSummary `json:"author"`
	Content   string      `json:"content"`
	IsAuthor  bool        `json:"is_author"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
}

type ListCommentsResponse struct {
	Comments   []CommentResponse `json:"comments"`
	NextCursor string            `json:"next_cursor,omitempty"`
}

type UserSummary struct {
	UserID    string `json:"user_id"`
	Username  string `json:"username,omitempty"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

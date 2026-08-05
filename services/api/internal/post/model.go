package post

import (
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

type Post struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AuthorID     uuid.UUID      `gorm:"type:uuid;column:author_id;not null;index"`
	Title        string         `gorm:"column:title;not null"`
	Content      string         `gorm:"column:content;not null;default:''"`
	CoverImage   string         `gorm:"column:cover_image;not null;default:''"`
	Tags         pq.StringArray `gorm:"column:tags;type:text[]"`
	LikeCount    int            `gorm:"column:like_count;not null;default:0"`
	CommentCount int            `gorm:"column:comment_count;not null;default:0"`
	PublishedAt  time.Time      `gorm:"column:published_at;not null"`
	CreatedAt    time.Time      `gorm:"column:created_at"`
	UpdatedAt    time.Time      `gorm:"column:updated_at"`
	DeletedAt    *time.Time     `gorm:"column:deleted_at"`
}

func (Post) TableName() string { return "post.posts" }

type Like struct {
	PostID    uuid.UUID `gorm:"type:uuid;primaryKey;column:post_id"`
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey;column:user_id"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func (Like) TableName() string { return "post.likes" }

type Comment struct {
	ID          uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PostID      uuid.UUID  `gorm:"type:uuid;column:post_id;not null;index"`
	AuthorID    uuid.UUID  `gorm:"type:uuid;column:author_id;not null"`
	ParentID    *uuid.UUID `gorm:"type:uuid;column:parent_id"`
	InReplyToID *uuid.UUID `gorm:"type:uuid;column:in_reply_to_id"`
	Content     string     `gorm:"column:content;not null"`
	ReplyCount  int        `gorm:"column:reply_count;not null;default:0"`
	CreatedAt   time.Time  `gorm:"column:created_at"`
	UpdatedAt   time.Time  `gorm:"column:updated_at"`
	DeletedAt   *time.Time `gorm:"column:deleted_at"`
}

func (Comment) TableName() string { return "post.comments" }

const (
	OpPostPublished = "POST_PUBLISHED"
	OpPostUpdated   = "POST_UPDATED"
	OpPostDeleted   = "POST_DELETED"

	SubjectPost = "post"
)

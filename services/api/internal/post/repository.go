package post

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/outbox"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

var (
	ErrPostNotFound    = errors.New("post not found")
	ErrCommentNotFound = errors.New("comment not found")
	ErrForbidden       = errors.New("forbidden")
	ErrAlreadyLiked    = errors.New("already liked")
	ErrNotLiked        = errors.New("not liked")
	ErrParentNotFound  = errors.New("parent comment not found")
	ErrInvalidParent   = errors.New("invalid parent comment")
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	CreatePost(ctx context.Context, p *Post) error
	UpdatePost(ctx context.Context, p *Post) error
	SoftDeletePost(ctx context.Context, id uuid.UUID) error
	FindPost(ctx context.Context, id uuid.UUID) (*Post, error)
	FindPostsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Post, error)
	ListPosts(ctx context.Context, filter ListPostsFilter, cursor *PostCursor, limit int) ([]Post, error)

	InsertLike(ctx context.Context, postID, userID uuid.UUID) error
	DeleteLike(ctx context.Context, postID, userID uuid.UUID) error
	AreLiked(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	CountLikes(ctx context.Context, postID uuid.UUID) (int, error)
	BumpLikeCount(ctx context.Context, postID uuid.UUID, delta int) error

	InsertBookmark(ctx context.Context, userID, postID uuid.UUID) error
	DeleteBookmark(ctx context.Context, userID, postID uuid.UUID) error
	AreBookmarked(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	ListBookmarks(ctx context.Context, userID uuid.UUID, cursor *BookmarkCursor, limit int) ([]Bookmark, error)

	CreateComment(ctx context.Context, c *Comment) error
	UpdateCommentContent(ctx context.Context, commentID, authorID uuid.UUID, content string) (*Comment, error)
	SoftDeleteComment(ctx context.Context, commentID uuid.UUID) error
	FindComment(ctx context.Context, id uuid.UUID) (*Comment, error)
	FindCommentAny(ctx context.Context, id uuid.UUID) (*Comment, error)
	ListComments(ctx context.Context, postID uuid.UUID, cursor *CommentCursor, limit int) ([]Comment, error)
	ListReplies(ctx context.Context, parentID uuid.UUID, cursor *ReplyCursor, limit int) ([]Comment, error)
	BumpCommentCount(ctx context.Context, postID uuid.UUID, delta int) error
	BumpReplyCount(ctx context.Context, commentID uuid.UUID, delta int) error

	FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error)

	InsertOutbox(ctx context.Context, row *outbox.Outbox) error
}

type PostCursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type CommentCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type ReplyCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type BookmarkCursor struct {
	CreatedAt time.Time
	PostID    uuid.UUID
}

type ListPostsFilter struct {
	AuthorID uuid.UUID
	Statuses []string
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

func (r *repository) CreatePost(ctx context.Context, p *Post) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	now := time.Now()
	if p.PublishedAt.IsZero() {
		p.PublishedAt = now
	}
	if p.Tags == nil {
		p.Tags = pq.StringArray{}
	}
	if p.Status == "" {
		p.Status = StatusPublished
	}
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) UpdatePost(ctx context.Context, p *Post) error {
	if p.Tags == nil {
		p.Tags = pq.StringArray{}
	}
	if p.Status == "" {
		p.Status = StatusPublished
	}
	p.UpdatedAt = time.Now()
	res := r.db.WithContext(ctx).Model(&Post{}).
		Where("id = ? AND deleted_at IS NULL", p.ID).
		Updates(map[string]any{
			"title":       p.Title,
			"content":     p.Content,
			"cover_image": p.CoverImage,
			"tags":        p.Tags,
			"status":      p.Status,
			"updated_at":  p.UpdatedAt,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *repository) SoftDeletePost(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&Post{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *repository) FindPost(ctx context.Context, id uuid.UUID) (*Post, error) {
	var p Post
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) FindPostsByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]Post, error) {
	out := make(map[uuid.UUID]Post, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var rows []Post
	err := r.db.WithContext(ctx).
		Where("id IN ? AND deleted_at IS NULL", ids).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, p := range rows {
		out[p.ID] = p
	}
	return out, nil
}

func (r *repository) ListPosts(ctx context.Context, filter ListPostsFilter, cursor *PostCursor, limit int) ([]Post, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&Post{}).Where("deleted_at IS NULL")
	if filter.AuthorID != uuid.Nil {
		q = q.Where("author_id = ?", filter.AuthorID)
	}
	if len(filter.Statuses) > 0 {
		q = q.Where("status IN ?", filter.Statuses)
	}
	if cursor != nil {
		q = q.Where("(published_at, id) < (?, ?)", cursor.PublishedAt, cursor.ID)
	}
	var rows []Post
	err := q.Order("published_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) InsertLike(ctx context.Context, postID, userID uuid.UUID) error {
	res := r.db.WithContext(ctx).Exec(`
		INSERT INTO post.likes (post_id, user_id, created_at)
		VALUES (?, ?, now())
		ON CONFLICT DO NOTHING
	`, postID, userID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrAlreadyLiked
	}
	return nil
}

func (r *repository) DeleteLike(ctx context.Context, postID, userID uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("post_id = ? AND user_id = ?", postID, userID).
		Delete(&Like{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrNotLiked
	}
	return nil
}

func (r *repository) AreLiked(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(postIDs))
	if userID == uuid.Nil || len(postIDs) == 0 {
		return out, nil
	}
	var rows []Like
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, l := range rows {
		out[l.PostID] = true
	}
	return out, nil
}

func (r *repository) CountLikes(ctx context.Context, postID uuid.UUID) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).Model(&Like{}).
		Where("post_id = ?", postID).
		Count(&n).Error
	return int(n), err
}

func (r *repository) BumpLikeCount(ctx context.Context, postID uuid.UUID, delta int) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE post.posts
		SET like_count = GREATEST(like_count + ?, 0)
		WHERE id = ? AND deleted_at IS NULL
	`, delta, postID).Error
}

func (r *repository) InsertBookmark(ctx context.Context, userID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO post.bookmarks (user_id, post_id, created_at)
		VALUES (?, ?, now())
		ON CONFLICT DO NOTHING
	`, userID, postID).Error
}

func (r *repository) DeleteBookmark(ctx context.Context, userID, postID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("user_id = ? AND post_id = ?", userID, postID).
		Delete(&Bookmark{}).Error
}

func (r *repository) AreBookmarked(ctx context.Context, userID uuid.UUID, postIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	out := make(map[uuid.UUID]bool, len(postIDs))
	if userID == uuid.Nil || len(postIDs) == 0 {
		return out, nil
	}
	var rows []Bookmark
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND post_id IN ?", userID, postIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, b := range rows {
		out[b.PostID] = true
	}
	return out, nil
}

func (r *repository) ListBookmarks(ctx context.Context, userID uuid.UUID, cursor *BookmarkCursor, limit int) ([]Bookmark, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&Bookmark{}).Where("user_id = ?", userID)
	if cursor != nil {
		q = q.Where("(created_at, post_id) < (?, ?)", cursor.CreatedAt, cursor.PostID)
	}
	var rows []Bookmark
	err := q.Order("created_at DESC, post_id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) CreateComment(ctx context.Context, c *Comment) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(c).Error
}

func (r *repository) UpdateCommentContent(ctx context.Context, commentID, authorID uuid.UUID, content string) (*Comment, error) {
	res := r.db.WithContext(ctx).Model(&Comment{}).
		Where("id = ? AND author_id = ? AND deleted_at IS NULL", commentID, authorID).
		Updates(map[string]any{
			"content":    content,
			"updated_at": time.Now(),
		})
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		existing, err := r.FindComment(ctx, commentID)
		if err != nil {
			return nil, err
		}
		if existing == nil {
			return nil, ErrCommentNotFound
		}
		return nil, ErrForbidden
	}
	return r.FindComment(ctx, commentID)
}

func (r *repository) SoftDeleteComment(ctx context.Context, commentID uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&Comment{}).
		Where("id = ? AND deleted_at IS NULL", commentID).
		Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

func (r *repository) FindComment(ctx context.Context, id uuid.UUID) (*Comment, error) {
	var c Comment
	err := r.db.WithContext(ctx).
		Where("id = ? AND deleted_at IS NULL", id).
		First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) FindCommentAny(ctx context.Context, id uuid.UUID) (*Comment, error) {
	var c Comment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) ListComments(ctx context.Context, postID uuid.UUID, cursor *CommentCursor, limit int) ([]Comment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&Comment{}).
		Where("post_id = ? AND parent_id IS NULL AND (deleted_at IS NULL OR reply_count > 0)", postID)
	if cursor != nil {
		q = q.Where("(created_at, id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	var rows []Comment
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) ListReplies(ctx context.Context, parentID uuid.UUID, cursor *ReplyCursor, limit int) ([]Comment, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&Comment{}).
		Where("parent_id = ? AND deleted_at IS NULL", parentID)
	if cursor != nil {
		q = q.Where("(created_at, id) > (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	var rows []Comment
	err := q.Order("created_at ASC, id ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) BumpCommentCount(ctx context.Context, postID uuid.UUID, delta int) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE post.posts
		SET comment_count = GREATEST(comment_count + ?, 0)
		WHERE id = ? AND deleted_at IS NULL
	`, delta, postID).Error
}

func (r *repository) BumpReplyCount(ctx context.Context, commentID uuid.UUID, delta int) error {
	return r.db.WithContext(ctx).Exec(`
		UPDATE post.comments
		SET reply_count = GREATEST(reply_count + ?, 0)
		WHERE id = ?
	`, delta, commentID).Error
}

func (r *repository) FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	out := make(map[uuid.UUID]auth.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []auth.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

func (r *repository) InsertOutbox(ctx context.Context, row *outbox.Outbox) error {
	if row == nil {
		return nil
	}
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(row).Error
}

type PostEventPayload struct {
	PostID      uuid.UUID `json:"post_id"`
	ActorID     uuid.UUID `json:"actor_id"`
	PublishedAt time.Time `json:"published_at"`
}

func BuildPostEvent(opType string, postID, actorID uuid.UUID, publishedAt time.Time) (*outbox.Outbox, error) {
	payload, err := json.Marshal(PostEventPayload{
		PostID:      postID,
		ActorID:     actorID,
		PublishedAt: publishedAt,
	})
	if err != nil {
		return nil, err
	}
	return &outbox.Outbox{
		AggregateType: SubjectPost,
		AggregateID:   postID,
		OpType:        opType,
		Stream:        stream.StreamPostEvents,
		ActorID:       outbox.ActorPtr(actorID),
		Payload:       payload,
	}, nil
}

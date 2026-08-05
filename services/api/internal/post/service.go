package post

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type IService interface {
	CreatePost(ctx context.Context, authorID uuid.UUID, payload CreatePostPayload) (*PostResponse, error)
	UpdatePost(ctx context.Context, authorID, postID uuid.UUID, payload UpdatePostPayload) (*PostResponse, error)
	DeletePost(ctx context.Context, authorID, postID uuid.UUID) error
	GetPost(ctx context.Context, viewerID, postID uuid.UUID) (*PostResponse, error)
	GetPostsByIDs(ctx context.Context, viewerID uuid.UUID, ids []uuid.UUID) ([]PostResponse, error)
	ListPosts(ctx context.Context, viewerID, authorID uuid.UUID, cursor *PostCursor, limit int) (ListPostsResponse, error)

	LikePost(ctx context.Context, userID, postID uuid.UUID) error
	UnlikePost(ctx context.Context, userID, postID uuid.UUID) error

	CreateComment(ctx context.Context, authorID, postID uuid.UUID, payload CreateCommentPayload) (*CommentResponse, error)
	UpdateComment(ctx context.Context, authorID, commentID uuid.UUID, payload UpdateCommentPayload) (*CommentResponse, error)
	DeleteComment(ctx context.Context, authorID, commentID uuid.UUID) error
	ListComments(ctx context.Context, viewerID, postID uuid.UUID, cursor *CommentCursor, limit int) (ListCommentsResponse, error)
	ListReplies(ctx context.Context, viewerID, commentID uuid.UUID, cursor *ReplyCursor, limit int) (ListRepliesResponse, error)
}

type ProfileLookup interface {
	FindProfileByUserID(ctx context.Context, userID uuid.UUID) (username, avatarURL string, err error)
}

type service struct {
	repo      IRepository
	profiles  ProfileLookup
	likeCache *likeCache
	logger    *zap.Logger
}

type ServiceConfig struct {
	Repo     IRepository
	Redis    *redis.Client
	Profiles ProfileLookup
	Logger   *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:      cfg.Repo,
		profiles:  cfg.Profiles,
		likeCache: newLikeCache(cfg.Redis),
		logger:    logger.With(zap.String("layer", "post.service")),
	}
}

func (s *service) CreatePost(ctx context.Context, authorID uuid.UUID, payload CreatePostPayload) (*PostResponse, error) {
	now := time.Now()
	p := &Post{
		ID:          uuid.New(),
		AuthorID:    authorID,
		Title:       strings.TrimSpace(payload.Title),
		Content:     payload.Content,
		CoverImage:  payload.CoverImage,
		Tags:        normalizeTags(payload.Tags),
		PublishedAt: now,
	}
	err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreatePost(ctx, p); err != nil {
			return err
		}
		row, err := BuildPostEvent(OpPostPublished, p.ID, authorID, p.PublishedAt)
		if err != nil {
			return err
		}
		return tx.InsertOutbox(ctx, row)
	})
	if err != nil {
		return nil, err
	}
	return s.hydratePost(ctx, authorID, p)
}

func (s *service) UpdatePost(ctx context.Context, authorID, postID uuid.UUID, payload UpdatePostPayload) (*PostResponse, error) {
	existing, err := s.repo.FindPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, ErrPostNotFound
	}
	if existing.AuthorID != authorID {
		return nil, ErrForbidden
	}
	if payload.Title != nil {
		existing.Title = strings.TrimSpace(*payload.Title)
	}
	if payload.Content != nil {
		existing.Content = *payload.Content
	}
	if payload.CoverImage != nil {
		existing.CoverImage = *payload.CoverImage
	}
	if payload.Tags != nil {
		existing.Tags = normalizeTags(*payload.Tags)
	}
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.UpdatePost(ctx, existing); err != nil {
			return err
		}
		row, err := BuildPostEvent(OpPostUpdated, existing.ID, authorID, existing.PublishedAt)
		if err != nil {
			return err
		}
		return tx.InsertOutbox(ctx, row)
	})
	if err != nil {
		return nil, err
	}
	updated, err := s.repo.FindPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrPostNotFound
	}
	return s.hydratePost(ctx, authorID, updated)
}

func (s *service) DeletePost(ctx context.Context, authorID, postID uuid.UUID) error {
	existing, err := s.repo.FindPost(ctx, postID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrPostNotFound
	}
	if existing.AuthorID != authorID {
		return ErrForbidden
	}
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.SoftDeletePost(ctx, postID); err != nil {
			return err
		}
		row, err := BuildPostEvent(OpPostDeleted, postID, authorID, existing.PublishedAt)
		if err != nil {
			return err
		}
		return tx.InsertOutbox(ctx, row)
	})
	if err != nil {
		return err
	}
	s.likeCache.del(ctx, postID)
	return nil
}

func (s *service) GetPost(ctx context.Context, viewerID, postID uuid.UUID) (*PostResponse, error) {
	p, err := s.repo.FindPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPostNotFound
	}
	return s.hydratePost(ctx, viewerID, p)
}

func (s *service) GetPostsByIDs(ctx context.Context, viewerID uuid.UUID, ids []uuid.UUID) ([]PostResponse, error) {
	if len(ids) == 0 {
		return []PostResponse{}, nil
	}
	postsByID, err := s.repo.FindPostsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if len(postsByID) == 0 {
		return []PostResponse{}, nil
	}
	rows := make([]Post, 0, len(postsByID))
	authorIDs := make([]uuid.UUID, 0, len(postsByID))
	postIDs := make([]uuid.UUID, 0, len(postsByID))
	seen := make(map[uuid.UUID]struct{}, len(postsByID))
	for _, id := range ids {
		p, ok := postsByID[id]
		if !ok {
			continue
		}
		rows = append(rows, p)
		postIDs = append(postIDs, p.ID)
		if _, dup := seen[p.AuthorID]; !dup {
			seen[p.AuthorID] = struct{}{}
			authorIDs = append(authorIDs, p.AuthorID)
		}
	}
	users, err := s.repo.FindUsersByIDs(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	profiles := s.batchProfiles(ctx, authorIDs)
	liked, err := s.repo.AreLiked(ctx, viewerID, postIDs)
	if err != nil {
		return nil, err
	}
	counts := s.likeCache.getMany(ctx, rows)
	out := make([]PostResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, buildPostResponse(p, users[p.AuthorID], profiles[p.AuthorID], liked[p.ID], counts[p.ID], p.AuthorID == viewerID))
	}
	return out, nil
}

func (s *service) ListPosts(ctx context.Context, viewerID, authorID uuid.UUID, cursor *PostCursor, limit int) (ListPostsResponse, error) {
	rows, err := s.repo.ListPosts(ctx, authorID, cursor, limit)
	if err != nil {
		return ListPostsResponse{}, err
	}
	if len(rows) == 0 {
		return ListPostsResponse{Posts: []PostResponse{}}, nil
	}
	authorIDs := make([]uuid.UUID, 0, len(rows))
	postIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{}, len(rows))
	for _, p := range rows {
		postIDs = append(postIDs, p.ID)
		if _, ok := seen[p.AuthorID]; !ok {
			seen[p.AuthorID] = struct{}{}
			authorIDs = append(authorIDs, p.AuthorID)
		}
	}
	users, err := s.repo.FindUsersByIDs(ctx, authorIDs)
	if err != nil {
		return ListPostsResponse{}, err
	}
	profiles := s.batchProfiles(ctx, authorIDs)
	liked, err := s.repo.AreLiked(ctx, viewerID, postIDs)
	if err != nil {
		return ListPostsResponse{}, err
	}
	counts := s.likeCache.getMany(ctx, rows)
	out := make([]PostResponse, 0, len(rows))
	for _, p := range rows {
		out = append(out, buildPostResponse(p, users[p.AuthorID], profiles[p.AuthorID], liked[p.ID], counts[p.ID], p.AuthorID == viewerID))
	}
	last := rows[len(rows)-1]
	return ListPostsResponse{
		Posts:      out,
		NextCursor: encodeCursor(last.PublishedAt, last.ID),
	}, nil
}

func (s *service) LikePost(ctx context.Context, userID, postID uuid.UUID) error {
	p, err := s.repo.FindPost(ctx, postID)
	if err != nil {
		return err
	}
	if p == nil {
		return ErrPostNotFound
	}
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.InsertLike(ctx, postID, userID); err != nil {
			return err
		}
		return tx.BumpLikeCount(ctx, postID, +1)
	})
	if err != nil {
		return err
	}
	s.likeCache.incr(ctx, postID)
	return nil
}

func (s *service) UnlikePost(ctx context.Context, userID, postID uuid.UUID) error {
	err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.DeleteLike(ctx, postID, userID); err != nil {
			return err
		}
		return tx.BumpLikeCount(ctx, postID, -1)
	})
	if err != nil {
		return err
	}
	s.likeCache.decr(ctx, postID)
	return nil
}

func (s *service) CreateComment(ctx context.Context, authorID, postID uuid.UUID, payload CreateCommentPayload) (*CommentResponse, error) {
	p, err := s.repo.FindPost(ctx, postID)
	if err != nil {
		return nil, err
	}
	if p == nil {
		return nil, ErrPostNotFound
	}
	var parentID *uuid.UUID
	var inReplyToID *uuid.UUID
	if payload.ParentID != nil && *payload.ParentID != "" {
		pid, err := uuid.Parse(*payload.ParentID)
		if err != nil {
			return nil, ErrInvalidParent
		}
		parent, err := s.repo.FindComment(ctx, pid)
		if err != nil {
			return nil, err
		}
		if parent == nil || parent.PostID != postID {
			return nil, ErrParentNotFound
		}
		root := pid
		if parent.ParentID != nil {
			rootParent, err := s.repo.FindComment(ctx, *parent.ParentID)
			if err != nil {
				return nil, err
			}
			if rootParent == nil || rootParent.PostID != postID {
				return nil, ErrParentNotFound
			}
			root = rootParent.ID
		}
		parentID = &root
		inReplyToID = &pid
	}
	c := &Comment{
		ID:          uuid.New(),
		PostID:      postID,
		AuthorID:    authorID,
		ParentID:    parentID,
		InReplyToID: inReplyToID,
		Content:     strings.TrimSpace(payload.Content),
	}
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreateComment(ctx, c); err != nil {
			return err
		}
		if err := tx.BumpCommentCount(ctx, postID, +1); err != nil {
			return err
		}
		if parentID != nil {
			return tx.BumpReplyCount(ctx, *parentID, +1)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return s.hydrateComment(ctx, authorID, c)
}

func (s *service) UpdateComment(ctx context.Context, authorID, commentID uuid.UUID, payload UpdateCommentPayload) (*CommentResponse, error) {
	updated, err := s.repo.UpdateCommentContent(ctx, commentID, authorID, strings.TrimSpace(payload.Content))
	if err != nil {
		return nil, err
	}
	if updated == nil {
		return nil, ErrCommentNotFound
	}
	return s.hydrateComment(ctx, authorID, updated)
}

func (s *service) DeleteComment(ctx context.Context, authorID, commentID uuid.UUID) error {
	existing, err := s.repo.FindComment(ctx, commentID)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrCommentNotFound
	}
	return s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.SoftDeleteComment(ctx, commentID, authorID); err != nil {
			return err
		}
		if err := tx.BumpCommentCount(ctx, existing.PostID, -1); err != nil {
			return err
		}
		if existing.ParentID != nil {
			return tx.BumpReplyCount(ctx, *existing.ParentID, -1)
		}
		return nil
	})
}

func (s *service) ListComments(ctx context.Context, viewerID, postID uuid.UUID, cursor *CommentCursor, limit int) (ListCommentsResponse, error) {
	rows, err := s.repo.ListComments(ctx, postID, cursor, limit)
	if err != nil {
		return ListCommentsResponse{}, err
	}
	if len(rows) == 0 {
		return ListCommentsResponse{Comments: []CommentResponse{}}, nil
	}
	out, err := s.hydrateComments(ctx, viewerID, rows)
	if err != nil {
		return ListCommentsResponse{}, err
	}
	last := rows[len(rows)-1]
	return ListCommentsResponse{
		Comments:   out,
		NextCursor: encodeCursor(last.CreatedAt, last.ID),
	}, nil
}

func (s *service) ListReplies(ctx context.Context, viewerID, commentID uuid.UUID, cursor *ReplyCursor, limit int) (ListRepliesResponse, error) {
	parent, err := s.repo.FindCommentAny(ctx, commentID)
	if err != nil {
		return ListRepliesResponse{}, err
	}
	if parent == nil || parent.ParentID != nil {
		return ListRepliesResponse{}, ErrCommentNotFound
	}
	rows, err := s.repo.ListReplies(ctx, commentID, cursor, limit)
	if err != nil {
		return ListRepliesResponse{}, err
	}
	if len(rows) == 0 {
		return ListRepliesResponse{Replies: []CommentResponse{}}, nil
	}
	out, err := s.hydrateComments(ctx, viewerID, rows)
	if err != nil {
		return ListRepliesResponse{}, err
	}
	last := rows[len(rows)-1]
	return ListRepliesResponse{
		Replies:    out,
		NextCursor: encodeCursor(last.CreatedAt, last.ID),
	}, nil
}

func (s *service) hydrateComments(ctx context.Context, viewerID uuid.UUID, rows []Comment) ([]CommentResponse, error) {
	authorIDs := make([]uuid.UUID, 0, len(rows))
	seen := make(map[uuid.UUID]struct{}, len(rows))
	for _, c := range rows {
		if _, ok := seen[c.AuthorID]; !ok {
			seen[c.AuthorID] = struct{}{}
			authorIDs = append(authorIDs, c.AuthorID)
		}
	}
	users, err := s.repo.FindUsersByIDs(ctx, authorIDs)
	if err != nil {
		return nil, err
	}
	profiles := s.batchProfiles(ctx, authorIDs)
	out := make([]CommentResponse, 0, len(rows))
	for _, c := range rows {
		out = append(out, buildCommentResponse(c, users[c.AuthorID], profiles[c.AuthorID], c.AuthorID == viewerID))
	}
	return out, nil
}

type profileHit struct {
	username  string
	avatarURL string
}

func (s *service) batchProfiles(ctx context.Context, ids []uuid.UUID) map[uuid.UUID]profileHit {
	out := make(map[uuid.UUID]profileHit, len(ids))
	if s.profiles == nil {
		return out
	}
	for _, id := range ids {
		username, avatarURL, err := s.profiles.FindProfileByUserID(ctx, id)
		if err != nil {
			s.logger.Warn("profile lookup failed", zap.String("user_id", id.String()), zap.Error(err))
			continue
		}
		out[id] = profileHit{username: username, avatarURL: avatarURL}
	}
	return out
}

func (s *service) hydratePost(ctx context.Context, viewerID uuid.UUID, p *Post) (*PostResponse, error) {
	users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{p.AuthorID})
	if err != nil {
		return nil, err
	}
	profiles := s.batchProfiles(ctx, []uuid.UUID{p.AuthorID})
	liked, err := s.repo.AreLiked(ctx, viewerID, []uuid.UUID{p.ID})
	if err != nil {
		return nil, err
	}
	counts := s.likeCache.getMany(ctx, []Post{*p})
	resp := buildPostResponse(*p, users[p.AuthorID], profiles[p.AuthorID], liked[p.ID], counts[p.ID], p.AuthorID == viewerID)
	return &resp, nil
}

func (s *service) hydrateComment(ctx context.Context, viewerID uuid.UUID, c *Comment) (*CommentResponse, error) {
	users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{c.AuthorID})
	if err != nil {
		return nil, err
	}
	profiles := s.batchProfiles(ctx, []uuid.UUID{c.AuthorID})
	resp := buildCommentResponse(*c, users[c.AuthorID], profiles[c.AuthorID], c.AuthorID == viewerID)
	return &resp, nil
}

func buildPostResponse(p Post, u auth.User, prof profileHit, liked bool, likeCount int, isAuthor bool) PostResponse {
	tags := []string(p.Tags)
	if tags == nil {
		tags = []string{}
	}
	return PostResponse{
		ID:           p.ID.String(),
		Author:       toUserSummary(p.AuthorID, u, prof),
		Title:        p.Title,
		Content:      p.Content,
		CoverImage:   p.CoverImage,
		Tags:         tags,
		LikeCount:    likeCount,
		CommentCount: p.CommentCount,
		IsLiked:      liked,
		IsAuthor:     isAuthor,
		PublishedAt:  p.PublishedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func buildCommentResponse(c Comment, u auth.User, prof profileHit, isAuthor bool) CommentResponse {
	deleted := c.DeletedAt != nil
	resp := CommentResponse{
		ID:         c.ID.String(),
		PostID:     c.PostID.String(),
		Author:     toUserSummary(c.AuthorID, u, prof),
		Content:    c.Content,
		ReplyCount: c.ReplyCount,
		IsAuthor:   isAuthor && !deleted,
		IsDeleted:  deleted,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
	if c.ParentID != nil {
		resp.ParentID = c.ParentID.String()
	}
	if c.InReplyToID != nil {
		resp.InReplyToID = c.InReplyToID.String()
	}
	if deleted {
		resp.Content = ""
	}
	return resp
}

func toUserSummary(id uuid.UUID, u auth.User, prof profileHit) UserSummary {
	s := UserSummary{
		UserID:    id.String(),
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
	if prof.username != "" {
		s.Username = prof.username
	}
	if prof.avatarURL != "" {
		s.AvatarURL = prof.avatarURL
	}
	return s
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		key := strings.ToLower(t)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, t)
	}
	return out
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano) + ":" + id.String()
}

func decodeCursor(token string) (*time.Time, uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, uuid.Nil, nil
	}
	sep := strings.LastIndex(token, ":")
	if sep <= 0 || sep == len(token)-1 {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, token[:sep])
	if err != nil {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(token[sep+1:])
	if err != nil {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	return &t, id, nil
}

func DecodePostCursor(token string) (*PostCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &PostCursor{PublishedAt: *t, ID: id}, nil
}

func DecodeCommentCursor(token string) (*CommentCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &CommentCursor{CreatedAt: *t, ID: id}, nil
}

func DecodeReplyCursor(token string) (*ReplyCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &ReplyCursor{CreatedAt: *t, ID: id}, nil
}

const (
	likeCacheKeyPrefix = "post:"
	likeCacheKeySuffix = ":likes"
	likeCacheTTL       = 30 * time.Minute
)

type likeCache struct {
	rdb *redis.Client
}

func newLikeCache(rdb *redis.Client) *likeCache { return &likeCache{rdb: rdb} }

func likeCacheKey(postID uuid.UUID) string {
	return likeCacheKeyPrefix + postID.String() + likeCacheKeySuffix
}

func (c *likeCache) getMany(ctx context.Context, posts []Post) map[uuid.UUID]int {
	out := make(map[uuid.UUID]int, len(posts))
	if len(posts) == 0 {
		return out
	}
	if c.rdb == nil {
		for _, p := range posts {
			out[p.ID] = p.LikeCount
		}
		return out
	}
	keys := make([]string, len(posts))
	for i, p := range posts {
		keys[i] = likeCacheKey(p.ID)
	}
	values, err := c.rdb.MGet(ctx, keys...).Result()
	if err != nil {
		for _, p := range posts {
			out[p.ID] = p.LikeCount
		}
		return out
	}
	pipe := c.rdb.Pipeline()
	needFill := false
	for i, p := range posts {
		if i < len(values) && values[i] != nil {
			if s, ok := values[i].(string); ok {
				if n, err := strconv.Atoi(s); err == nil {
					out[p.ID] = n
					continue
				}
			}
		}
		out[p.ID] = p.LikeCount
		pipe.Set(ctx, keys[i], p.LikeCount, likeCacheTTL)
		needFill = true
	}
	if needFill {
		_, _ = pipe.Exec(ctx)
	}
	return out
}

func (c *likeCache) incr(ctx context.Context, postID uuid.UUID) {
	if c.rdb == nil {
		return
	}
	// EXISTS guard: skip when unset so backfill stays authoritative.
	c.rdb.Eval(ctx, `if redis.call('EXISTS', KEYS[1]) == 1 then return redis.call('INCR', KEYS[1]) else return nil end`, []string{likeCacheKey(postID)})
}

func (c *likeCache) decr(ctx context.Context, postID uuid.UUID) {
	if c.rdb == nil {
		return
	}
	c.rdb.Eval(ctx, `if redis.call('EXISTS', KEYS[1]) == 1 then return redis.call('DECR', KEYS[1]) else return nil end`, []string{likeCacheKey(postID)})
}

func (c *likeCache) del(ctx context.Context, postID uuid.UUID) {
	if c.rdb == nil {
		return
	}
	c.rdb.Del(ctx, likeCacheKey(postID))
}

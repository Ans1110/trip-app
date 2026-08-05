package search

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/post"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var ErrQueryRequired = errors.New("query is required")

type IService interface {
	SearchPosts(ctx context.Context, viewerID uuid.UUID, query string, cursor *Cursor, limit int) (SearchPostsResponse, error)
}

type PostLookup interface {
	GetPostsByIDs(ctx context.Context, viewerID uuid.UUID, ids []uuid.UUID) ([]post.PostResponse, error)
}

type service struct {
	repo   IRepository
	posts  PostLookup
	logger *zap.Logger
}

type ServiceConfig struct {
	Repo   IRepository
	Posts  PostLookup
	Logger *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		posts:  cfg.Posts,
		logger: logger.With(zap.String("layer", "search.service")),
	}
}

func (s *service) SearchPosts(ctx context.Context, viewerID uuid.UUID, query string, cursor *Cursor, limit int) (SearchPostsResponse, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return SearchPostsResponse{}, ErrQueryRequired
	}
	hits, err := s.repo.SearchPosts(ctx, q, cursor, limit)
	if err != nil {
		return SearchPostsResponse{}, err
	}
	if len(hits) == 0 {
		return SearchPostsResponse{Query: q, Posts: []post.PostResponse{}}, nil
	}
	ids := make([]uuid.UUID, 0, len(hits))
	for _, h := range hits {
		ids = append(ids, h.ID)
	}
	posts := []post.PostResponse{}
	if s.posts != nil {
		posts, err = s.posts.GetPostsByIDs(ctx, viewerID, ids)
		if err != nil {
			return SearchPostsResponse{}, err
		}
	}
	last := hits[len(hits)-1]
	return SearchPostsResponse{
		Query:      q,
		Posts:      posts,
		NextCursor: encodeCursor(last.PublishedAt, last.ID),
	}, nil
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano) + ":" + id.String()
}

func DecodeCursor(token string) (*Cursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	sep := strings.LastIndex(token, ":")
	if sep <= 0 || sep == len(token)-1 {
		return nil, errors.New("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, token[:sep])
	if err != nil {
		return nil, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(token[sep+1:])
	if err != nil {
		return nil, errors.New("invalid cursor")
	}
	return &Cursor{PublishedAt: t, ID: id}, nil
}

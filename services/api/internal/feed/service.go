package feed

import (
	"context"

	"github.com/Ans1110/trip-app/internal/post"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IService interface {
	ListFeed(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) (ListFeedResponse, error)
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
		logger: logger.With(zap.String("layer", "feed.service")),
	}
}

func (s *service) ListFeed(ctx context.Context, userID uuid.UUID, cursor *Cursor, limit int) (ListFeedResponse, error) {
	rows, refTime, err := s.repo.ListRanked(ctx, userID, cursor, limit)
	if err != nil {
		return ListFeedResponse{}, err
	}
	if len(rows) == 0 {
		return ListFeedResponse{Posts: []post.PostResponse{}}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for _, r := range rows {
		ids = append(ids, r.ID)
	}
	hydrated := []post.PostResponse{}
	if s.posts != nil {
		hydrated, err = s.posts.GetPostsByIDs(ctx, userID, ids)
		if err != nil {
			return ListFeedResponse{}, err
		}
	}
	last := rows[len(rows)-1]
	return ListFeedResponse{
		Posts: hydrated,
		NextCursor: encodeCursor(Cursor{
			RefTime:     refTime,
			Score:       last.Score,
			PublishedAt: last.PublishedAt,
			PostID:      last.ID,
		}),
	}, nil
}

package feed

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/post"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IService interface {
	ListFeed(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) (ListFeedResponse, error)
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

func (s *service) ListFeed(ctx context.Context, userID uuid.UUID, cursor *FeedCursor, limit int) (ListFeedResponse, error) {
	rows, err := s.repo.ListForUser(ctx, userID, cursor, limit)
	if err != nil {
		return ListFeedResponse{}, err
	}
	if len(rows) == 0 {
		return ListFeedResponse{Items: []FeedItemResponse{}}, nil
	}

	postsByID := map[uuid.UUID]post.PostResponse{}
	if s.posts != nil {
		ids := make([]uuid.UUID, 0, len(rows))
		seen := make(map[uuid.UUID]struct{}, len(rows))
		for _, r := range rows {
			if r.SubjectType != post.SubjectPost {
				continue
			}
			if _, ok := seen[r.SubjectID]; ok {
				continue
			}
			seen[r.SubjectID] = struct{}{}
			ids = append(ids, r.SubjectID)
		}
		if len(ids) > 0 {
			hydrated, err := s.posts.GetPostsByIDs(ctx, userID, ids)
			if err != nil {
				return ListFeedResponse{}, err
			}
			for _, p := range hydrated {
				pid, err := uuid.Parse(p.ID)
				if err != nil {
					continue
				}
				postsByID[pid] = p
			}
		}
	}

	items := make([]FeedItemResponse, 0, len(rows))
	for _, r := range rows {
		item := FeedItemResponse{
			ID:          r.ID.String(),
			EventType:   r.EventType,
			SubjectType: r.SubjectType,
			SubjectID:   r.SubjectID.String(),
			PublishedAt: r.PublishedAt,
		}
		if r.SubjectType == post.SubjectPost {
			if p, ok := postsByID[r.SubjectID]; ok {
				pp := p
				item.Post = &pp
			}
		}
		items = append(items, item)
	}

	last := rows[len(rows)-1]
	return ListFeedResponse{
		Items:      items,
		NextCursor: encodeCursor(last.PublishedAt, last.ID),
	}, nil
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano) + ":" + id.String()
}

func DecodeCursor(token string) (*FeedCursor, error) {
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
	return &FeedCursor{PublishedAt: t, ID: id}, nil
}

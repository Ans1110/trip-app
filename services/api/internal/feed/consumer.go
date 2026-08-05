package feed

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Ans1110/trip-app/internal/post"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type FollowerLookup func(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error)

type FanoutConsumer struct {
	repo        IRepository
	consumer    *stream.Consumer
	logger      *zap.Logger
	batchInsert int
	followerFn  FollowerLookup
}

type FanoutConsumerConfig struct {
	Repo           IRepository
	Redis          *redis.Client
	Logger         *zap.Logger
	ConsumerID     string
	FollowerLookup FollowerLookup
}

func NewFanoutConsumer(cfg FanoutConsumerConfig) *FanoutConsumer {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.With(zap.String("layer", "feed.fanout"))

	c := &FanoutConsumer{
		repo:        cfg.Repo,
		logger:      logger,
		batchInsert: 500,
		followerFn:  cfg.FollowerLookup,
	}
	c.consumer = stream.NewConsumer(
		cfg.Redis,
		stream.StreamPostEvents,
		stream.GroupFeedFanout,
		cfg.ConsumerID,
		c.handle,
		logger,
	)
	return c
}

func (c *FanoutConsumer) Start(ctx context.Context) { c.consumer.Start(ctx) }

func (c *FanoutConsumer) handle(ctx context.Context, msgID string, values map[string]any) error {
	opType, _ := values["op_type"].(string)
	payloadStr, _ := values["payload"].(string)
	if payloadStr == "" {
		return errors.New("empty payload")
	}
	var p post.PostEventPayload
	if err := json.Unmarshal([]byte(payloadStr), &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}

	switch opType {
	case post.OpPostPublished:
		return c.fanOutPublish(ctx, msgID, p)
	case post.OpPostDeleted:
		if err := c.repo.DeleteBySubject(ctx, post.SubjectPost, p.PostID); err != nil {
			return fmt.Errorf("delete feed items: %w", err)
		}
		return nil
	default:
		return nil
	}
}

func (c *FanoutConsumer) fanOutPublish(ctx context.Context, msgID string, p post.PostEventPayload) error {
	if c.followerFn == nil {
		return errors.New("follower lookup not configured")
	}
	publishedAt := p.PublishedAt
	if publishedAt.IsZero() {
		publishedAt = time.Now()
	}

	followers, err := c.followerFn(ctx, p.ActorID)
	if err != nil {
		return fmt.Errorf("lookup followers: %w", err)
	}

	items := make([]FeedItem, 0, len(followers)+1)
	// Author also gets a row so their own post appears on their home feed.
	items = append(items, FeedItem{
		UserID:      p.ActorID,
		ActorID:     p.ActorID,
		EventType:   post.OpPostPublished,
		SubjectType: post.SubjectPost,
		SubjectID:   p.PostID,
		PublishedAt: publishedAt,
	})
	for _, followerID := range followers {
		if followerID == p.ActorID {
			continue
		}
		items = append(items, FeedItem{
			UserID:      followerID,
			ActorID:     p.ActorID,
			EventType:   post.OpPostPublished,
			SubjectType: post.SubjectPost,
			SubjectID:   p.PostID,
			PublishedAt: publishedAt,
		})
	}

	for i := 0; i < len(items); i += c.batchInsert {
		end := i + c.batchInsert
		if end > len(items) {
			end = len(items)
		}
		if err := c.repo.InsertMany(ctx, items[i:end]); err != nil {
			return fmt.Errorf("insert feed items: %w", err)
		}
	}
	c.logger.Debug("fanout complete",
		zap.String("msg_id", msgID),
		zap.String("post_id", p.PostID.String()),
		zap.String("actor_id", p.ActorID.String()),
		zap.Int("recipients", len(items)),
	)
	return nil
}

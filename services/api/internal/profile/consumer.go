package profile

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

// FanoutConsumer consumes TRIP_PUBLISHED events from events:profile and writes
// one profile.feed row per follower of the actor.
type FanoutConsumer struct {
	repo        IRepository
	consumer    *stream.Consumer
	logger      *zap.Logger
	batchInsert int
	followerFn  func(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error)
}

type FanoutConsumerConfig struct {
	Repo       IRepository
	Redis      *redis.Client
	Logger     *zap.Logger
	ConsumerID string

	// FollowerLookup lists every follower of the actor. Injected so tests
	// don't need to hit the real repository. Defaults to Repo.ListFollowers
	// paginated to completion.
	FollowerLookup func(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error)
}

func NewFanoutConsumer(cfg FanoutConsumerConfig) *FanoutConsumer {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	logger = logger.With(zap.String("layer", "profile.fanout"))

	lookup := cfg.FollowerLookup
	if lookup == nil {
		lookup = func(ctx context.Context, actorID uuid.UUID) ([]uuid.UUID, error) {
			return listAllFollowers(ctx, cfg.Repo, actorID)
		}
	}

	c := &FanoutConsumer{
		repo:        cfg.Repo,
		logger:      logger,
		batchInsert: 500,
		followerFn:  lookup,
	}
	c.consumer = stream.NewConsumer(
		cfg.Redis,
		stream.StreamProfileEvents,
		stream.GroupProfileFanout,
		cfg.ConsumerID,
		c.handle,
		logger,
	)
	return c
}

func (c *FanoutConsumer) Start(ctx context.Context) { c.consumer.Start(ctx) }

// handle processes a single message. Non-TRIP_PUBLISHED events are ignored so
// this consumer can safely share events:profile with future event types.
func (c *FanoutConsumer) handle(ctx context.Context, msgID string, values map[string]any) error {
	opType, _ := values["op_type"].(string)
	if opType != EventTripPublished {
		return nil
	}

	payloadStr, _ := values["payload"].(string)
	if payloadStr == "" {
		return errors.New("empty payload")
	}
	var p struct {
		TripID      string `json:"trip_id"`
		ActorID     string `json:"actor_id"`
		PublishedAt string `json:"published_at"`
	}
	if err := json.Unmarshal([]byte(payloadStr), &p); err != nil {
		return fmt.Errorf("decode payload: %w", err)
	}
	tripID, err := uuid.Parse(p.TripID)
	if err != nil {
		return fmt.Errorf("invalid trip_id: %w", err)
	}
	actorID, err := uuid.Parse(p.ActorID)
	if err != nil {
		return fmt.Errorf("invalid actor_id: %w", err)
	}
	publishedAt, err := time.Parse(time.RFC3339Nano, p.PublishedAt)
	if err != nil {
		return fmt.Errorf("invalid published_at: %w", err)
	}

	followers, err := c.followerFn(ctx, actorID)
	if err != nil {
		return fmt.Errorf("lookup followers: %w", err)
	}
	if len(followers) == 0 {
		return nil
	}

	entries := make([]FeedEntry, 0, len(followers))
	for _, followerID := range followers {
		if followerID == actorID {
			continue
		}
		tid := tripID
		entries = append(entries, FeedEntry{
			UserID:      followerID,
			ActorID:     actorID,
			EventType:   EventTripPublished,
			TripID:      &tid,
			PublishedAt: publishedAt,
		})
	}

	for i := 0; i < len(entries); i += c.batchInsert {
		end := i + c.batchInsert
		if end > len(entries) {
			end = len(entries)
		}
		if err := c.repo.InsertFeedEntries(ctx, entries[i:end]); err != nil {
			return fmt.Errorf("insert feed entries: %w", err)
		}
	}
	c.logger.Debug("fanout complete",
		zap.String("msg_id", msgID),
		zap.String("trip_id", tripID.String()),
		zap.String("actor_id", actorID.String()),
		zap.Int("followers", len(entries)),
	)
	return nil
}

func listAllFollowers(ctx context.Context, repo IRepository, actorID uuid.UUID) ([]uuid.UUID, error) {
	const pageSize = 200
	var (
		out    []uuid.UUID
		cursor *FollowCursor
	)
	for {
		rows, err := repo.ListFollowers(ctx, actorID, cursor, pageSize)
		if err != nil {
			return nil, err
		}
		if len(rows) == 0 {
			return out, nil
		}
		for _, r := range rows {
			out = append(out, r.FollowerID)
		}
		if len(rows) < pageSize {
			return out, nil
		}
		last := rows[len(rows)-1]
		cursor = &FollowCursor{CreatedAt: last.CreatedAt, OtherID: last.FollowerID}
	}
}

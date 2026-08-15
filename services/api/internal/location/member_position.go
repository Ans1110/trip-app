package location

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	memberPositionTTL    = 30 * time.Minute
	memberPositionPrefix = "location:member:"
	userSharingPrefix    = "location:user:"
)

type MemberPosition struct {
	UserID    uuid.UUID `json:"user_id"`
	Lat       float64   `json:"lat"`
	Lng       float64   `json:"lng"`
	AccuracyM float64   `json:"accuracy_m,omitempty"`
	SharedAt  time.Time `json:"shared_at"`
}

type PositionStore struct {
	rdb *redis.Client
}

func NewPositionStore(rdb *redis.Client) *PositionStore {
	return &PositionStore{rdb: rdb}
}

func tripKey(tripID uuid.UUID) string {
	return memberPositionPrefix + tripID.String()
}

func userSharingKey(userID uuid.UUID) string {
	return userSharingPrefix + userID.String() + ":trips"
}

func (s *PositionStore) Set(ctx context.Context, tripID uuid.UUID, pos MemberPosition) error {
	if s == nil || s.rdb == nil {
		return errors.New("position store not configured")
	}
	buf, err := json.Marshal(pos)
	if err != nil {
		return err
	}
	key := tripKey(tripID)
	ukey := userSharingKey(pos.UserID)
	pipe := s.rdb.TxPipeline()
	pipe.HSet(ctx, key, pos.UserID.String(), buf)
	// Refresh whole-trip TTL on every write; individual field expiry isn't
	// available in Redis <7, so we prune stale fields on read instead.
	pipe.Expire(ctx, key, memberPositionTTL)
	pipe.SAdd(ctx, ukey, tripID.String())
	pipe.Expire(ctx, ukey, memberPositionTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *PositionStore) Delete(ctx context.Context, tripID, userID uuid.UUID) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	pipe := s.rdb.TxPipeline()
	pipe.HDel(ctx, tripKey(tripID), userID.String())
	pipe.SRem(ctx, userSharingKey(userID), tripID.String())
	_, err := pipe.Exec(ctx)
	return err
}

func (s *PositionStore) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	if s == nil || s.rdb == nil {
		return nil
	}
	ukey := userSharingKey(userID)
	trips, err := s.rdb.SMembers(ctx, ukey).Result()
	if err != nil {
		return err
	}
	if len(trips) == 0 {
		return nil
	}
	pipe := s.rdb.TxPipeline()
	for _, t := range trips {
		pipe.HDel(ctx, memberPositionPrefix+t, userID.String())
	}
	pipe.Del(ctx, ukey)
	_, err = pipe.Exec(ctx)
	return err
}

// ListForTrip returns all live positions for a trip.
func (s *PositionStore) ListForTrip(ctx context.Context, tripID uuid.UUID) ([]MemberPosition, error) {
	if s == nil || s.rdb == nil {
		return nil, nil
	}
	raw, err := s.rdb.HGetAll(ctx, tripKey(tripID)).Result()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		return nil, nil
	}
	cutoff := time.Now().Add(-memberPositionTTL)
	out := make([]MemberPosition, 0, len(raw))
	var stale []string
	for field, v := range raw {
		var pos MemberPosition
		if err := json.Unmarshal([]byte(v), &pos); err != nil {
			stale = append(stale, field)
			continue
		}
		if pos.SharedAt.Before(cutoff) {
			stale = append(stale, field)
			continue
		}
		out = append(out, pos)
	}
	if len(stale) > 0 {
		if err := s.rdb.HDel(ctx, tripKey(tripID), stale...).Err(); err != nil {
			return out, fmt.Errorf("prune stale positions: %w", err)
		}
	}
	return out, nil
}

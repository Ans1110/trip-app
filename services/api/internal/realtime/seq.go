package realtime

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// roomSeqKey returns the Redis key holding the monotonic sequence for a
// trip's room. Keys live as long as the room is "active" plus a sliding TTL
// — long enough that reconnecting clients within the window can still rely
// on monotonic ordering.
func roomSeqKey(tripID uuid.UUID) string {
	return fmt.Sprintf("rt:seq:trip:{%s}", tripID.String())
}

// nextSeq atomically reserves the next sequence number for a room. INCR on
// a non-existent key starts at 1; TTL is refreshed on each call so an
// abandoned room eventually rolls off.
func nextSeq(ctx context.Context, rdb *redis.Client, tripID uuid.UUID) (int64, error) {
	key := roomSeqKey(tripID)
	pipe := rdb.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 7*24*time.Hour)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, err
	}
	return incr.Val(), nil
}

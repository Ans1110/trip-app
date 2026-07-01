package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type OutboxDispatcher struct {
	repo      trip.IRepository
	rdb       *redis.Client
	logger    *zap.Logger
	batchSize int
	interval  time.Duration
}

type OutboxDispatcherConfig struct {
	Repo      trip.IRepository
	Redis     *redis.Client
	Logger    *zap.Logger
	BatchSize int
	Interval  time.Duration
}

func NewOutboxDispatcher(cfg OutboxDispatcherConfig) *OutboxDispatcher {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	batch := cfg.BatchSize
	if batch <= 0 {
		batch = 64
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 200 * time.Millisecond
	}
	return &OutboxDispatcher{
		repo:      cfg.Repo,
		rdb:       cfg.Redis,
		logger:    logger.With(zap.String("layer", "realtime.outbox")),
		batchSize: batch,
		interval:  interval,
	}
}

// Start blocks until ctx is cancelled. Errors on individual ticks are logged
// but don't kill the loop.
func (d *OutboxDispatcher) Start(ctx context.Context) {
	t := time.NewTicker(d.interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			// Drain aggressively — if a tick emptied a full batch there's
			// probably more work behind it, so loop until we get less than
			// batchSize.
			for {
				n, err := d.tick(ctx)
				if err != nil {
					d.logger.Warn("outbox tick failed", zap.Error(err))
					break
				}
				if n < d.batchSize {
					break
				}
			}
		}
	}
}

// tick claims a batch, pipelines XADDs, marks dispatched. Returns the
// number of rows processed.
func (d *OutboxDispatcher) tick(ctx context.Context) (int, error) {
	var processed int
	err := d.repo.WithTx(ctx, func(txRepo trip.IRepository) error {
		rows, err := txRepo.ClaimOutbox(ctx, d.batchSize)
		if err != nil {
			return err
		}
		if len(rows) == 0 {
			return nil
		}

		// Pipeline the XADDs so N events cost 1 Redis RTT.
		pipe := d.rdb.Pipeline()
		cmds := make([]*redis.StringCmd, len(rows))
		for i, row := range rows {
			values, verr := outboxRowToValues(row)
			if verr != nil {
				// Bad payload — record failure for this row but keep going.
				if ferr := d.repo.RecordOutboxFailure(ctx, row.ID, verr.Error()); ferr != nil {
					d.logger.Warn("record outbox failure", zap.Error(ferr))
				}
				continue
			}
			target := row.Stream
			if target == "" {
				target = stream.StreamTripEvents
			}
			cmds[i] = pipe.XAdd(ctx, &redis.XAddArgs{
				Stream: target,
				MaxLen: 10000,
				Approx: true,
				Values: values,
			})
		}
		// pipe.Exec runs all queued commands even if some fail; it just
		// surfaces the first non-nil per-cmd error.
		if _, execErr := pipe.Exec(ctx); execErr != nil && !errors.Is(execErr, redis.Nil) {
			d.logger.Debug("outbox pipeline had per-cmd errors", zap.Error(execErr))
		}

		var dispatched []uuid.UUID
		for i, cmd := range cmds {
			if cmd == nil {
				continue // skipped bad row above
			}
			if err := cmd.Err(); err != nil {
				if ferr := d.repo.RecordOutboxFailure(ctx, rows[i].ID, err.Error()); ferr != nil {
					d.logger.Warn("record outbox failure", zap.Error(ferr))
				}
				continue
			}
			dispatched = append(dispatched, rows[i].ID)
		}
		if len(dispatched) > 0 {
			if err := txRepo.MarkOutboxDispatched(ctx, dispatched); err != nil {
				return err
			}
		}
		processed = len(rows)
		return nil
	})
	return processed, err
}

func outboxRowToValues(row trip.Outbox) (map[string]any, error) {
	var ev trip.OutboxEvent
	if err := json.Unmarshal(row.Payload, &ev); err != nil {
		return nil, err
	}
	// The outbox row is the source of truth for trace_id — if the caller
	// stored one on the row (not in the JSON payload) prefer that.
	if ev.TraceID == "" {
		ev.TraceID = row.TraceID
	}
	return outboxEventValues(&ev), nil
}

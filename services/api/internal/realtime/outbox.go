package realtime

import (
	"context"
	"errors"
	"time"

	"github.com/Ans1110/trip-app/internal/outbox"
	"github.com/Ans1110/trip-app/pkg/stream"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type OutboxDispatcher struct {
	repo         outbox.IRepository
	rdb          *redis.Client
	logger       *zap.Logger
	batchSize    int
	interval     time.Duration
	gcInterval   time.Duration
	gcRetention  time.Duration
	gcBatchLimit int
}

type OutboxDispatcherConfig struct {
	Repo      outbox.IRepository
	Redis     *redis.Client
	Logger    *zap.Logger
	BatchSize int
	Interval  time.Duration

	GCInterval   time.Duration
	GCRetention  time.Duration
	GCBatchLimit int
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
		interval = time.Second
	}
	gcInterval := cfg.GCInterval
	if gcInterval <= 0 {
		gcInterval = 5 * time.Minute
	}
	gcRetention := cfg.GCRetention
	if gcRetention <= 0 {
		gcRetention = 24 * time.Hour
	}
	gcBatch := cfg.GCBatchLimit
	if gcBatch <= 0 {
		gcBatch = 1000
	}
	return &OutboxDispatcher{
		repo:         cfg.Repo,
		rdb:          cfg.Redis,
		logger:       logger.With(zap.String("layer", "realtime.outbox")),
		batchSize:    batch,
		interval:     interval,
		gcInterval:   gcInterval,
		gcRetention:  gcRetention,
		gcBatchLimit: gcBatch,
	}
}

// Start blocks until ctx is cancelled. Errors on individual ticks are logged
// but don't kill the loop.
func (d *OutboxDispatcher) Start(ctx context.Context) {
	go d.gcLoop(ctx)

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

// gcLoop periodically deletes dispatched rows older than the retention window.
// Without this, the base table (and by extension the partial index for pending
// rows) grows unbounded and the pending-poll query slows down as dead tuples
// accumulate between autovacuum runs.
func (d *OutboxDispatcher) gcLoop(ctx context.Context) {
	t := time.NewTicker(d.gcInterval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			cutoff := time.Now().UTC().Add(-d.gcRetention)
			n, err := d.repo.PruneDispatched(ctx, cutoff, d.gcBatchLimit)
			if err != nil {
				d.logger.Warn("outbox gc failed", zap.Error(err))
				continue
			}
			if n > 0 {
				d.logger.Info("outbox gc pruned rows", zap.Int64("count", n))
			}
		}
	}
}

// tick claims a batch, pipelines XADDs, marks dispatched. Returns the
// number of rows processed.
func (d *OutboxDispatcher) tick(ctx context.Context) (int, error) {
	var processed int
	err := d.repo.WithTx(ctx, func(txRepo outbox.IRepository) error {
		rows, err := txRepo.Claim(ctx, d.batchSize)
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
			target := row.Stream
			if target == "" {
				target = stream.StreamTripEvents
			}
			cmds[i] = pipe.XAdd(ctx, &redis.XAddArgs{
				Stream: target,
				MaxLen: 10000,
				Approx: true,
				Values: outboxRowToValues(row),
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
				if ferr := d.repo.RecordFailure(ctx, rows[i].ID, err.Error()); ferr != nil {
					d.logger.Warn("record outbox failure", zap.Error(ferr))
				}
				continue
			}
			dispatched = append(dispatched, rows[i].ID)
		}
		if len(dispatched) > 0 {
			if err := txRepo.MarkDispatched(ctx, dispatched); err != nil {
				return err
			}
		}
		processed = len(rows)
		return nil
	})
	return processed, err
}

// outboxRowToValues renders an outbox row for XADD. The wire format is generic
// across event types: standard row fields as top-level keys plus the raw
// payload JSON string for consumers to decode.
func outboxRowToValues(row outbox.Outbox) map[string]any {
	values := map[string]any{
		"op_type":        row.OpType,
		"aggregate_type": row.AggregateType,
		"aggregate_id":   row.AggregateID.String(),
		"payload":        string(row.Payload),
		"ts":             time.Now().UnixMilli(),
	}
	if row.ActorID != nil && *row.ActorID != uuid.Nil {
		values["actor_id"] = row.ActorID.String()
	}
	if row.TraceID != "" {
		values["trace_id"] = row.TraceID
	}
	return values
}

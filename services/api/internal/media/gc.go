package media

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// GC is a background sweeper that cleans up abandoned uploads. A session that
// expires without a matching Complete leaves behind either (a) nothing (client
// never PUT the object) or (b) an orphaned S3 object (client PUT but never
// called Complete, or Complete failed). Either way we try to delete the object
// then the session row. Safe to run concurrently on multiple instances — each
// session id gets deleted at most once thanks to the row lock.
type GC struct {
	repo     IRepository
	storage  Storage
	interval time.Duration
	batch    int
	logger   *zap.Logger
}

type GCConfig struct {
	Repo     IRepository
	Storage  Storage
	Interval time.Duration
	Batch    int
	Logger   *zap.Logger
}

func NewGC(cfg GCConfig) *GC {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 5 * time.Minute
	}
	batch := cfg.Batch
	if batch <= 0 {
		batch = 100
	}
	return &GC{
		repo:     cfg.Repo,
		storage:  cfg.Storage,
		interval: interval,
		batch:    batch,
		logger:   logger.With(zap.String("layer", "media.gc")),
	}
}

// Start runs the sweep loop until ctx is cancelled. Returns immediately.
func (g *GC) Start(ctx context.Context) {
	go g.loop(ctx)
}

func (g *GC) loop(ctx context.Context) {
	t := time.NewTicker(g.interval)
	defer t.Stop()

	// First tick fires after `interval`; run one immediately on boot so a
	// server that just started doesn't wait a full window before sweeping.
	g.sweepOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			g.sweepOnce(ctx)
		}
	}
}

func (g *GC) sweepOnce(ctx context.Context) {
	rows, err := g.repo.ListExpiredPendingSessions(ctx, time.Now().UTC(), g.batch)
	if err != nil {
		g.logger.Error("media: list expired sessions", zap.Error(err))
		return
	}
	if len(rows) == 0 {
		return
	}
	g.logger.Info("media: sweeping expired upload sessions", zap.Int("count", len(rows)))
	for i := range rows {
		s := &rows[i]
		// Object removal is idempotent — RemoveObject swallows NoSuchKey. If
		// S3 is unreachable the session row stays so the next sweep retries.
		if err := g.storage.RemoveObject(ctx, s.ObjectKey); err != nil {
			g.logger.Warn("media: remove object",
				zap.String("session_id", s.ID.String()),
				zap.String("object_key", s.ObjectKey),
				zap.Error(err),
			)
			continue
		}
		if err := g.repo.DeleteSession(ctx, s.ID); err != nil {
			g.logger.Warn("media: delete session",
				zap.String("session_id", s.ID.String()),
				zap.Error(err),
			)
		}
	}
}

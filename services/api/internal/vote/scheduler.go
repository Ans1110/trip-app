package vote

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// Scheduler wakes up on a fixed cadence and closes polls whose deadline has
// passed. Errors on a single sweep are logged and the loop continues; a
// stalled sweep never blocks the next tick.
type Scheduler struct {
	svc      IService
	logger   *zap.Logger
	interval time.Duration
}

type SchedulerConfig struct {
	Service  IService
	Logger   *zap.Logger
	Interval time.Duration
}

func NewScheduler(cfg SchedulerConfig) *Scheduler {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	interval := cfg.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Scheduler{
		svc:      cfg.Service,
		logger:   logger.With(zap.String("layer", "vote.scheduler")),
		interval: interval,
	}
}

func (s *Scheduler) Start(ctx context.Context) {
	t := time.NewTicker(s.interval)
	defer t.Stop()
	// Fire once on startup so a restart doesn't leave already-expired polls
	// hanging until the first tick.
	s.tick(ctx)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s.tick(ctx)
		}
	}
}

func (s *Scheduler) tick(ctx context.Context) {
	n, err := s.svc.SweepExpiredPolls(ctx)
	if err != nil {
		s.logger.Warn("vote sweep failed", zap.Error(err))
		return
	}
	if n > 0 {
		s.logger.Info("vote: auto-closed polls", zap.Int("count", n))
	}
}

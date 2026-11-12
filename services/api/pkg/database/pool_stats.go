package database

import (
	"context"
	"database/sql"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// PoolStatsLogger periodically logs sql.DB.Stats() so we can correlate slow
// query spikes with pool contention. WaitCount / WaitDuration climbing at the
// same wall-clock as slow-SQL logs is the definitive signature of "sweeper is
// blocked getting a connection, not on Postgres." MaxIdleClosed /
// MaxLifetimeClosed rising alongside them points at ConnMaxLifetime / idle
// eviction forcing fresh TCP handshakes.
//
// Counters (WaitCount, WaitDuration, MaxIdleClosed, MaxLifetimeClosed) are
// monotonic — read them as deltas between ticks, not absolutes.
func StartPoolStatsLogger(ctx context.Context, db *gorm.DB, logger *zap.Logger, interval time.Duration) {
	if db == nil || logger == nil {
		return
	}
	sqlDB, err := db.DB()
	if err != nil {
		logger.Warn("pool stats: cannot access underlying sql.DB", zap.Error(err))
		return
	}
	if interval <= 0 {
		interval = 60 * time.Second
	}
	log := logger.With(zap.String("layer", "database.pool_stats"))
	go runPoolStats(ctx, sqlDB, log, interval)
}

func runPoolStats(ctx context.Context, sqlDB *sql.DB, log *zap.Logger, interval time.Duration) {
	t := time.NewTicker(interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			s := sqlDB.Stats()
			log.Info("pool stats",
				zap.Int("max_open_connections", s.MaxOpenConnections),
				zap.Int("open_connections", s.OpenConnections),
				zap.Int("in_use", s.InUse),
				zap.Int("idle", s.Idle),
				zap.Int64("wait_count", s.WaitCount),
				zap.Duration("wait_duration", s.WaitDuration),
				zap.Int64("max_idle_closed", s.MaxIdleClosed),
				zap.Int64("max_lifetime_closed", s.MaxLifetimeClosed),
			)
		}
	}
}

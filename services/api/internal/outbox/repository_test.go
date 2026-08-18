package outbox_test

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/outbox"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tcpostgres "github.com/testcontainers/testcontainers-go/modules/postgres"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func setupDB(t *testing.T) *gorm.DB {
	t.Helper()
	ctx := context.Background()

	ctr, err := tcpostgres.Run(ctx,
		"postgres:16-alpine",
		tcpostgres.WithDatabase("testdb"),
		tcpostgres.WithUsername("postgres"),
		tcpostgres.WithPassword("postgres"),
		tcpostgres.BasicWaitStrategies(),
	)
	require.NoError(t, err)
	t.Cleanup(func() { _ = ctr.Terminate(ctx) })

	dsn, err := ctr.ConnectionString(ctx, "sslmode=disable")
	require.NoError(t, err)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	require.NoError(t, err)

	sqls := []string{
		`CREATE SCHEMA IF NOT EXISTS platform`,
		`CREATE TABLE IF NOT EXISTS platform.outbox (
			id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			aggregate_type TEXT        NOT NULL,
			aggregate_id   UUID,
			op_type        TEXT        NOT NULL,
			stream         TEXT        NOT NULL,
			actor_id       UUID,
			trace_id       TEXT        NOT NULL DEFAULT '',
			payload        JSONB       NOT NULL,
			created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
			dispatched_at  TIMESTAMPTZ,
			attempts       INTEGER     NOT NULL DEFAULT 0,
			last_error     TEXT        NOT NULL DEFAULT ''
		)`,
	}
	for _, s := range sqls {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func newRow() *outbox.Outbox {
	actor := uuid.New()
	return &outbox.Outbox{
		AggregateType: "trip",
		AggregateID:   uuid.New(),
		OpType:        "TRIP_PUBLISHED",
		Stream:        "trip_events",
		ActorID:       &actor,
		TraceID:       "trace-1",
		Payload:       []byte(`{"foo":"bar"}`),
	}
}

func TestInsert_AssignsIDAndDefaults(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	row := newRow()
	require.NoError(t, repo.Insert(ctx, row))
	assert.NotEqual(t, uuid.Nil, row.ID, "Insert should assign an ID when uuid.Nil")

	var got outbox.Outbox
	require.NoError(t, db.Where("id = ?", row.ID).First(&got).Error)
	assert.False(t, got.CreatedAt.IsZero())
	assert.Nil(t, got.DispatchedAt)
	assert.Equal(t, 0, got.Attempts)
	assert.Empty(t, got.LastError)
}

func TestInsert_PreservesProvidedID(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	row := newRow()
	row.ID = uuid.New()
	want := row.ID
	require.NoError(t, repo.Insert(ctx, row))
	assert.Equal(t, want, row.ID)
}

func TestInsert_NilRowIsNoOp(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	require.NoError(t, repo.Insert(context.Background(), nil))

	var n int64
	require.NoError(t, db.Table("platform.outbox").Count(&n).Error)
	assert.EqualValues(t, 0, n)
}

func TestClaim_FIFOOrderAndLimit(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	// Insert 3 rows with staggered created_at so ordering is deterministic.
	ids := make([]uuid.UUID, 3)
	base := time.Now().UTC().Add(-time.Hour)
	for i := 0; i < 3; i++ {
		row := newRow()
		row.ID = uuid.New()
		ids[i] = row.ID
		require.NoError(t, repo.Insert(ctx, row))
		require.NoError(t, db.Exec(
			`UPDATE platform.outbox SET created_at = ? WHERE id = ?`,
			base.Add(time.Duration(i)*time.Minute), row.ID,
		).Error)
	}

	// Claim needs an outer transaction because it uses FOR UPDATE.
	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		txRepo := outbox.NewRepository(tx)
		claimed, err := txRepo.Claim(ctx, 2)
		require.NoError(t, err)
		require.Len(t, claimed, 2, "limit should cap the result set")
		assert.Equal(t, ids[0], claimed[0].ID, "oldest first")
		assert.Equal(t, ids[1], claimed[1].ID)
		return nil
	}))
}

// SKIP LOCKED means a second concurrent claimer sees only the rows the first
// one did not lock — no head-of-line blocking on the queue.
func TestClaim_SkipLockedAllowsParallelClaimers(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 4; i++ {
		require.NoError(t, repo.Insert(ctx, newRow()))
	}

	// Hold the first two rows in tx1 while tx2 tries to claim.
	tx1 := db.Begin()
	t.Cleanup(func() { tx1.Rollback() })
	claimed1, err := outbox.NewRepository(tx1).Claim(ctx, 2)
	require.NoError(t, err)
	require.Len(t, claimed1, 2)

	tx2 := db.Begin()
	t.Cleanup(func() { tx2.Rollback() })
	claimed2, err := outbox.NewRepository(tx2).Claim(ctx, 10)
	require.NoError(t, err)
	assert.Len(t, claimed2, 2, "SKIP LOCKED should hand tx2 the remaining rows")

	seen := map[uuid.UUID]bool{}
	for _, r := range claimed1 {
		seen[r.ID] = true
	}
	for _, r := range claimed2 {
		assert.False(t, seen[r.ID], "concurrent claimers must never see the same row")
	}
}

func TestClaim_DefaultLimitWhenNonPositive(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		require.NoError(t, repo.Insert(ctx, newRow()))
	}

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		claimed, err := outbox.NewRepository(tx).Claim(ctx, 0)
		require.NoError(t, err)
		assert.Len(t, claimed, 5, "limit 0 should fall back to the default (>=5)")
		return nil
	}))
}

func TestClaim_SkipsDispatched(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	live := newRow()
	dispatched := newRow()
	require.NoError(t, repo.Insert(ctx, live))
	require.NoError(t, repo.Insert(ctx, dispatched))

	now := time.Now().UTC()
	require.NoError(t, db.Exec(
		`UPDATE platform.outbox SET dispatched_at = ? WHERE id = ?`,
		now, dispatched.ID,
	).Error)

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		claimed, err := outbox.NewRepository(tx).Claim(ctx, 10)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		assert.Equal(t, live.ID, claimed[0].ID)
		return nil
	}))
}

func TestMarkDispatched_DeletesRows(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	a, b, keep := newRow(), newRow(), newRow()
	require.NoError(t, repo.Insert(ctx, a))
	require.NoError(t, repo.Insert(ctx, b))
	require.NoError(t, repo.Insert(ctx, keep))

	require.NoError(t, repo.MarkDispatched(ctx, []uuid.UUID{a.ID, b.ID}))

	var remaining []outbox.Outbox
	require.NoError(t, db.Find(&remaining).Error)
	require.Len(t, remaining, 1)
	assert.Equal(t, keep.ID, remaining[0].ID)
}

func TestMarkDispatched_EmptyIDsIsNoop(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	row := newRow()
	require.NoError(t, repo.Insert(ctx, row))

	require.NoError(t, repo.MarkDispatched(ctx, nil))
	require.NoError(t, repo.MarkDispatched(ctx, []uuid.UUID{}))

	var n int64
	require.NoError(t, db.Table("platform.outbox").Count(&n).Error)
	assert.EqualValues(t, 1, n)
}

func TestRecordFailure_IncrementsAttempts(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	row := newRow()
	require.NoError(t, repo.Insert(ctx, row))

	require.NoError(t, repo.RecordFailure(ctx, row.ID, "boom"))
	require.NoError(t, repo.RecordFailure(ctx, row.ID, "again"))

	var got outbox.Outbox
	require.NoError(t, db.Where("id = ?", row.ID).First(&got).Error)
	assert.Equal(t, 2, got.Attempts, "attempts should compound across calls")
	assert.Equal(t, "again", got.LastError, "last_error should reflect the most recent failure")
}

func TestRecordFailure_TruncatesLongMessage(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	row := newRow()
	require.NoError(t, repo.Insert(ctx, row))

	long := strings.Repeat("x", 1000)
	require.NoError(t, repo.RecordFailure(ctx, row.ID, long))

	var got outbox.Outbox
	require.NoError(t, db.Where("id = ?", row.ID).First(&got).Error)
	assert.Len(t, got.LastError, 512, "error should be truncated at maxErrLen (512)")
}

func TestRecordFailure_MissingIDNoRowsAffected(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	// Not an error — GORM returns nil for zero-row updates on this path.
	require.NoError(t, repo.RecordFailure(ctx, uuid.New(), "orphan"))
}

func TestPruneDispatched_HonorsCutoffAndLimit(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	old1 := newRow()
	old2 := newRow()
	recent := newRow()
	pending := newRow()
	for _, r := range []*outbox.Outbox{old1, old2, recent, pending} {
		require.NoError(t, repo.Insert(ctx, r))
	}
	require.NoError(t, db.Exec(
		`UPDATE platform.outbox SET dispatched_at = ? WHERE id IN (?, ?)`,
		now.Add(-2*time.Hour), old1.ID, old2.ID,
	).Error)
	require.NoError(t, db.Exec(
		`UPDATE platform.outbox SET dispatched_at = ? WHERE id = ?`,
		now.Add(-1*time.Minute), recent.ID,
	).Error)

	// Cutoff at 1h ago — only old1 and old2 qualify.
	deleted, err := repo.PruneDispatched(ctx, now.Add(-time.Hour), 100)
	require.NoError(t, err)
	assert.EqualValues(t, 2, deleted)

	var remainingIDs []uuid.UUID
	require.NoError(t, db.Table("platform.outbox").Order("id").Pluck("id", &remainingIDs).Error)
	assert.ElementsMatch(t, []uuid.UUID{recent.ID, pending.ID}, remainingIDs)
}

func TestPruneDispatched_LimitCap(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	for i := 0; i < 5; i++ {
		row := newRow()
		require.NoError(t, repo.Insert(ctx, row))
		require.NoError(t, db.Exec(
			`UPDATE platform.outbox SET dispatched_at = ? WHERE id = ?`,
			now.Add(-time.Duration(i+1)*time.Hour), row.ID,
		).Error)
	}

	deleted, err := repo.PruneDispatched(ctx, now, 2)
	require.NoError(t, err)
	assert.EqualValues(t, 2, deleted)

	var remaining int64
	require.NoError(t, db.Table("platform.outbox").Count(&remaining).Error)
	assert.EqualValues(t, 3, remaining)
}

func TestPruneDispatched_DefaultLimit(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	now := time.Now().UTC()
	// 3 rows all past the cutoff; limit 0 should default (>=1000) and delete all.
	for i := 0; i < 3; i++ {
		row := newRow()
		require.NoError(t, repo.Insert(ctx, row))
		require.NoError(t, db.Exec(
			`UPDATE platform.outbox SET dispatched_at = ? WHERE id = ?`,
			now.Add(-time.Hour), row.ID,
		).Error)
	}

	deleted, err := repo.PruneDispatched(ctx, now, 0)
	require.NoError(t, err)
	assert.EqualValues(t, 3, deleted)
}

func TestWithTx_CommitsAndRollsBack(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	// Commit path.
	committed := newRow()
	require.NoError(t, repo.WithTx(ctx, func(tx outbox.IRepository) error {
		return tx.Insert(ctx, committed)
	}))
	var n1 int64
	require.NoError(t, db.Table("platform.outbox").Where("id = ?", committed.ID).Count(&n1).Error)
	assert.EqualValues(t, 1, n1)

	// Rollback path.
	rolled := newRow()
	sentinel := errors.New("rollback")
	err := repo.WithTx(ctx, func(tx outbox.IRepository) error {
		if err := tx.Insert(ctx, rolled); err != nil {
			return err
		}
		return sentinel
	})
	require.ErrorIs(t, err, sentinel)

	var n2 int64
	require.NoError(t, db.Table("platform.outbox").Where("id = ?", rolled.ID).Count(&n2).Error)
	assert.EqualValues(t, 0, n2, "row must not be visible after rollback")
}

// End-to-end: producer inserts inside a tx, dispatcher claims + marks
// dispatched. The row should vanish from the queue.
func TestOutboxDispatchLoop_HappyPath(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	row := newRow()
	require.NoError(t, repo.WithTx(ctx, func(tx outbox.IRepository) error {
		return tx.Insert(ctx, row)
	}))

	require.NoError(t, db.Transaction(func(tx *gorm.DB) error {
		claimed, err := outbox.NewRepository(tx).Claim(ctx, 10)
		require.NoError(t, err)
		require.Len(t, claimed, 1)
		assert.Equal(t, row.ID, claimed[0].ID)

		ids := make([]uuid.UUID, len(claimed))
		for i, c := range claimed {
			ids[i] = c.ID
		}
		return outbox.NewRepository(tx).MarkDispatched(ctx, ids)
	}))

	var n int64
	require.NoError(t, db.Table("platform.outbox").Count(&n).Error)
	assert.EqualValues(t, 0, n)
}

// Regression guard: two goroutines calling Claim in parallel transactions
// must never observe the same row (SKIP LOCKED guarantee).
func TestClaim_ParallelClaimersDisjoint(t *testing.T) {
	db := setupDB(t)
	repo := outbox.NewRepository(db)
	ctx := context.Background()

	const total = 20
	for i := 0; i < total; i++ {
		require.NoError(t, repo.Insert(ctx, newRow()))
	}

	var (
		wg  sync.WaitGroup
		mu  sync.Mutex
		all = map[uuid.UUID]int{}
	)
	claim := func() {
		defer wg.Done()
		_ = db.Transaction(func(tx *gorm.DB) error {
			rows, err := outbox.NewRepository(tx).Claim(ctx, 8)
			if err != nil {
				return err
			}
			mu.Lock()
			for _, r := range rows {
				all[r.ID]++
			}
			mu.Unlock()
			ids := make([]uuid.UUID, len(rows))
			for i, r := range rows {
				ids[i] = r.ID
			}
			return outbox.NewRepository(tx).MarkDispatched(ctx, ids)
		})
	}
	wg.Add(3)
	go claim()
	go claim()
	go claim()
	wg.Wait()

	for id, hits := range all {
		assert.Equalf(t, 1, hits, "row %s claimed more than once", id)
	}
}

package audit_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
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

	// The audit table lives in the auth schema (historical name — the table
	// itself is generic). Only the columns audit.Log maps to are required.
	sqls := []string{
		`CREATE SCHEMA IF NOT EXISTS auth`,
		`CREATE TABLE IF NOT EXISTS auth.audit_logs (
			id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
			actor_user_id  UUID,
			target_user_id UUID,
			action         VARCHAR(64) NOT NULL,
			status         VARCHAR(32) NOT NULL,
			resource_type  VARCHAR(64),
			resource_id    TEXT,
			ip_address     INET,
			user_agent     TEXT,
			request_id     UUID,
			trace_id       TEXT,
			detail         JSONB,
			created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
		)`,
	}
	for _, s := range sqls {
		require.NoError(t, db.Exec(s).Error)
	}
	return db
}

func TestAuditRepository_Create_RoundTrip(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)
	ctx := context.Background()

	actor := uuid.New()
	target := uuid.New()
	req := uuid.New()
	ip := "10.0.0.1"

	log := &audit.Log{
		ID:           uuid.New(),
		ActorUserID:  &actor,
		TargetUserID: &target,
		Action:       audit.Action("USER_BLOCKED"),
		Status:       audit.Success,
		ResourceType: "user",
		ResourceID:   target.String(),
		IPAddress:    &ip,
		UserAgent:    "test-agent",
		RequestID:    &req,
		TraceID:      "trace-xyz",
		Detail:       map[string]any{"reason": "policy violation", "count": float64(3)},
	}
	require.NoError(t, repo.Create(ctx, log))

	var got audit.Log
	require.NoError(t, db.Where("id = ?", log.ID).First(&got).Error)
	assert.Equal(t, log.ID, got.ID)
	require.NotNil(t, got.ActorUserID)
	assert.Equal(t, actor, *got.ActorUserID)
	require.NotNil(t, got.TargetUserID)
	assert.Equal(t, target, *got.TargetUserID)
	assert.Equal(t, audit.Action("USER_BLOCKED"), got.Action)
	assert.Equal(t, audit.Success, got.Status)
	assert.Equal(t, "user", got.ResourceType)
	assert.Equal(t, target.String(), got.ResourceID)
	require.NotNil(t, got.IPAddress)
	assert.Equal(t, "10.0.0.1", *got.IPAddress)
	assert.Equal(t, "test-agent", got.UserAgent)
	require.NotNil(t, got.RequestID)
	assert.Equal(t, req, *got.RequestID)
	assert.Equal(t, "trace-xyz", got.TraceID)
	assert.Equal(t, "policy violation", got.Detail["reason"])
	assert.EqualValues(t, 3, got.Detail["count"])
	assert.False(t, got.CreatedAt.IsZero(), "created_at should be defaulted by the DB")
}

func TestAuditRepository_Create_NullableFields(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)
	ctx := context.Background()

	// System-emitted event: no actor, no target, no request context.
	log := &audit.Log{
		ID:     uuid.New(),
		Action: audit.Action("SYSTEM_CLEANUP"),
		Status: audit.Success,
	}
	require.NoError(t, repo.Create(ctx, log))

	var got audit.Log
	require.NoError(t, db.Where("id = ?", log.ID).First(&got).Error)
	assert.Nil(t, got.ActorUserID)
	assert.Nil(t, got.TargetUserID)
	assert.Nil(t, got.IPAddress)
	assert.Nil(t, got.RequestID)
	assert.Empty(t, got.Detail)
}

func TestAuditRepository_Create_FailureStatus(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)
	ctx := context.Background()

	actor := uuid.New()
	log := &audit.Log{
		ID:          uuid.New(),
		ActorUserID: &actor,
		Action:      audit.Action("LOGIN_ATTEMPT"),
		Status:      audit.Failure,
		Detail:      map[string]any{"error": "invalid_password"},
	}
	require.NoError(t, repo.Create(ctx, log))

	var count int64
	require.NoError(t, db.Table("auth.audit_logs").
		Where("actor_user_id = ? AND status = ?", actor, audit.Failure).
		Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

func TestAuditRepository_QueryByIndexedColumns(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)
	ctx := context.Background()

	actor := uuid.New()
	target := uuid.New()
	other := uuid.New()

	for _, l := range []*audit.Log{
		{ID: uuid.New(), ActorUserID: &actor, TargetUserID: &target, Action: "A", Status: audit.Success},
		{ID: uuid.New(), ActorUserID: &actor, TargetUserID: &target, Action: "B", Status: audit.Success},
		{ID: uuid.New(), ActorUserID: &other, TargetUserID: &target, Action: "A", Status: audit.Success},
	} {
		require.NoError(t, repo.Create(ctx, l))
	}

	var byActor int64
	require.NoError(t, db.Table("auth.audit_logs").
		Where("actor_user_id = ?", actor).Count(&byActor).Error)
	assert.EqualValues(t, 2, byActor)

	var byAction int64
	require.NoError(t, db.Table("auth.audit_logs").
		Where("action = ?", "A").Count(&byAction).Error)
	assert.EqualValues(t, 2, byAction)

	var byTarget int64
	require.NoError(t, db.Table("auth.audit_logs").
		Where("target_user_id = ?", target).Count(&byTarget).Error)
	assert.EqualValues(t, 3, byTarget)
}

// The Detail column is `jsonb`; the model uses the `serializer:json` tag so
// nested structures survive round-trip and are queryable via PG's JSON ops.
func TestAuditRepository_JSONBDetailIsQueryable(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)
	ctx := context.Background()

	require.NoError(t, repo.Create(ctx, &audit.Log{
		ID:     uuid.New(),
		Action: "X",
		Status: audit.Success,
		Detail: map[string]any{"kind": "wanted", "n": float64(1)},
	}))
	require.NoError(t, repo.Create(ctx, &audit.Log{
		ID:     uuid.New(),
		Action: "X",
		Status: audit.Success,
		Detail: map[string]any{"kind": "other", "n": float64(2)},
	}))

	var count int64
	require.NoError(t, db.Table("auth.audit_logs").
		Where("detail->>'kind' = ?", "wanted").Count(&count).Error)
	assert.EqualValues(t, 1, count)
}

// Ensures the writer respects context cancellation — a canceled ctx must not
// silently commit a row.
func TestAuditRepository_Create_CanceledContext(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := repo.Create(ctx, &audit.Log{
		ID:     uuid.New(),
		Action: "X",
		Status: audit.Success,
	})
	require.Error(t, err)

	var count int64
	require.NoError(t, db.WithContext(context.Background()).
		Table("auth.audit_logs").Count(&count).Error)
	assert.EqualValues(t, 0, count)
}

// A crashing caller (context deadline exceeded partway through a batch)
// should not partially commit — Create is a single statement and either
// lands as-is or not at all.
func TestAuditRepository_Create_DeadlineExceeded(t *testing.T) {
	db := setupDB(t)
	repo := audit.NewRepository(db)

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()
	time.Sleep(2 * time.Millisecond)

	err := repo.Create(ctx, &audit.Log{
		ID:     uuid.New(),
		Action: "X",
		Status: audit.Success,
	})
	require.Error(t, err)
}

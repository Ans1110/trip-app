package outbox

import (
	"context"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error
	Tx() *gorm.DB

	Insert(ctx context.Context, row *Outbox) error
	Claim(ctx context.Context, limit int) ([]Outbox, error)
	MarkDispatched(ctx context.Context, ids []uuid.UUID) error
	RecordFailure(ctx context.Context, id uuid.UUID, errMsg string) error
	PruneDispatched(ctx context.Context, olderThan time.Time, limit int) (int64, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

func (r *repository) Tx() *gorm.DB { return r.db }

func (r *repository) Insert(ctx context.Context, row *Outbox) error {
	if row == nil {
		return nil
	}
	if row.ID == uuid.Nil {
		row.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(row).Error
}

func (r *repository) Claim(ctx context.Context, limit int) ([]Outbox, error) {
	if limit <= 0 {
		limit = 32
	}
	var rows []Outbox
	err := r.db.WithContext(ctx).
		Raw(`SELECT * FROM platform.outbox
		     WHERE dispatched_at IS NULL
		     ORDER BY created_at ASC
		     LIMIT ?
		     FOR UPDATE SKIP LOCKED`, limit).
		Scan(&rows).Error
	return rows, err
}

// MarkDispatched deletes rows outright; retention is handled by the caller's
// choice not to keep dispatched history. PruneDispatched exists for callers
// that would rather flip dispatched_at first (currently unused).
func (r *repository) MarkDispatched(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Where("id IN ?", ids).
		Delete(&Outbox{}).Error
}

func (r *repository) RecordFailure(ctx context.Context, id uuid.UUID, errMsg string) error {
	const maxErrLen = 512
	if len(errMsg) > maxErrLen {
		errMsg = errMsg[:maxErrLen]
	}
	return r.db.WithContext(ctx).
		Model(&Outbox{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"attempts":   gorm.Expr("attempts + 1"),
			"last_error": errMsg,
		}).Error
}

func (r *repository) PruneDispatched(ctx context.Context, olderThan time.Time, limit int) (int64, error) {
	if limit <= 0 {
		limit = 1000
	}
	res := r.db.WithContext(ctx).Exec(
		`DELETE FROM platform.outbox
		 WHERE id IN (
		     SELECT id FROM platform.outbox
		     WHERE dispatched_at IS NOT NULL AND dispatched_at < ?
		     ORDER BY dispatched_at ASC
		     LIMIT ?
		 )`,
		olderThan, limit,
	)
	return res.RowsAffected, res.Error
}

// ActorPtr converts a uuid.UUID to a nullable *uuid.UUID suitable for the
// Outbox.ActorID column (nil for uuid.Nil, otherwise a pointer to a copy).
func ActorPtr(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	cp := id
	return &cp
}

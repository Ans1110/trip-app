package vote

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"gorm.io/gorm"
)

const pgUniqueViolation = "23505"

func isUniqueViolation(err error) bool {
	var pgErr *pq.Error
	if errors.As(err, &pgErr) {
		return pgErr.Code.Name() == "unique_violation" || string(pgErr.Code) == pgUniqueViolation
	}
	return false
}

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	CreatePoll(ctx context.Context, p *Poll) error
	FindPoll(ctx context.Context, id uuid.UUID) (*Poll, error)
	ListPollsByTrip(ctx context.Context, tripID uuid.UUID) ([]Poll, error)
	MarkClosed(ctx context.Context, id uuid.UUID, closedAt time.Time) error
	ClaimExpiredPolls(ctx context.Context, now time.Time, limit int) ([]Poll, error)
	DeletePoll(ctx context.Context, id uuid.UUID) error

	CreateOption(ctx context.Context, o *Option) error
	FindOption(ctx context.Context, id uuid.UUID) (*Option, error)
	ListOptions(ctx context.Context, pollID uuid.UUID) ([]Option, error)
	DeleteOption(ctx context.Context, id uuid.UUID) error

	ReplaceBallots(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID) error
	ListBallots(ctx context.Context, pollID uuid.UUID) ([]Ballot, error)
	ListMyChoices(ctx context.Context, pollID, userID uuid.UUID) ([]uuid.UUID, error)
	CountVotersByOption(ctx context.Context, pollID uuid.UUID) (map[uuid.UUID]int, error)
	CountDistinctVoters(ctx context.Context, pollID uuid.UUID) (int, error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository { return &repository{db: db} }

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

// ---- Poll ----

func (r *repository) CreatePoll(ctx context.Context, p *Poll) error {
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(p).Error
}

func (r *repository) FindPoll(ctx context.Context, id uuid.UUID) (*Poll, error) {
	var p Poll
	err := r.db.WithContext(ctx).First(&p, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *repository) ListPollsByTrip(ctx context.Context, tripID uuid.UUID) ([]Poll, error) {
	var rows []Poll
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("created_at DESC").Find(&rows).Error
	return rows, err
}

func (r *repository) MarkClosed(ctx context.Context, id uuid.UUID, closedAt time.Time) error {
	return r.db.WithContext(ctx).Model(&Poll{}).
		Where("id = ? AND status = ?", id, StatusOpen).
		Updates(map[string]any{
			"status":     StatusClosed,
			"closed_at":  closedAt,
			"updated_at": closedAt,
		}).Error
}

// ClaimExpiredPolls returns polls that have hit their deadline while still
// open. The sweeper uses this to bulk-close them.
func (r *repository) ClaimExpiredPolls(ctx context.Context, now time.Time, limit int) ([]Poll, error) {
	if limit <= 0 {
		limit = 64
	}
	var rows []Poll
	err := r.db.WithContext(ctx).
		Where("status = ? AND deadline_at IS NOT NULL AND deadline_at <= ?", StatusOpen, now).
		Order("deadline_at ASC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) DeletePoll(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Poll{}, "id = ?", id).Error
}

// ---- Option ----

func (r *repository) CreateOption(ctx context.Context, o *Option) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	err := r.db.WithContext(ctx).Create(o).Error
	if isUniqueViolation(err) {
		return ErrDuplicateOption
	}
	return err
}

func (r *repository) FindOption(ctx context.Context, id uuid.UUID) (*Option, error) {
	var o Option
	err := r.db.WithContext(ctx).First(&o, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &o, err
}

func (r *repository) ListOptions(ctx context.Context, pollID uuid.UUID) ([]Option, error) {
	var rows []Option
	err := r.db.WithContext(ctx).
		Where("poll_id = ?", pollID).
		Order("sort_order ASC, created_at ASC").Find(&rows).Error
	return rows, err
}

func (r *repository) DeleteOption(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&Option{}, "id = ?", id).Error
}

// ---- Ballots ----

// ReplaceBallots atomically swaps a user's ballot set for the given poll.
// Caller-provided option ids must already be validated against the poll.
func (r *repository) ReplaceBallots(ctx context.Context, pollID, userID uuid.UUID, optionIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("poll_id = ? AND user_id = ?", pollID, userID).Delete(&Ballot{}).Error; err != nil {
			return err
		}
		if len(optionIDs) == 0 {
			return nil
		}
		now := time.Now()
		rows := make([]Ballot, 0, len(optionIDs))
		for _, oid := range optionIDs {
			rows = append(rows, Ballot{
				PollID:    pollID,
				UserID:    userID,
				OptionID:  oid,
				CreatedAt: now,
			})
		}
		return tx.Create(&rows).Error
	})
}

func (r *repository) ListBallots(ctx context.Context, pollID uuid.UUID) ([]Ballot, error) {
	var rows []Ballot
	err := r.db.WithContext(ctx).Where("poll_id = ?", pollID).Find(&rows).Error
	return rows, err
}

func (r *repository) ListMyChoices(ctx context.Context, pollID, userID uuid.UUID) ([]uuid.UUID, error) {
	var rows []Ballot
	err := r.db.WithContext(ctx).
		Where("poll_id = ? AND user_id = ?", pollID, userID).Find(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, b := range rows {
		out = append(out, b.OptionID)
	}
	return out, nil
}

func (r *repository) CountVotersByOption(ctx context.Context, pollID uuid.UUID) (map[uuid.UUID]int, error) {
	type row struct {
		OptionID uuid.UUID
		N        int
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT option_id AS option_id, COUNT(*) AS n
		     FROM vote.ballot WHERE poll_id = ?
		     GROUP BY option_id`, pollID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]int, len(rows))
	for _, x := range rows {
		out[x.OptionID] = x.N
	}
	return out, nil
}

func (r *repository) CountDistinctVoters(ctx context.Context, pollID uuid.UUID) (int, error) {
	var n int64
	err := r.db.WithContext(ctx).
		Raw(`SELECT COUNT(DISTINCT user_id) FROM vote.ballot WHERE poll_id = ?`, pollID).
		Scan(&n).Error
	return int(n), err
}

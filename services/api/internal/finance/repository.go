package finance

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

var (
	ErrExpenseNotFound    = errors.New("finance: expense not found")
	ErrBudgetNotFound     = errors.New("finance: budget not found")
	ErrSettlementNotFound = errors.New("finance: settlement not found")
	ErrForbidden          = errors.New("finance: forbidden")
	ErrInvalidPayload     = errors.New("finance: invalid payload")
	ErrShareSumMismatch   = errors.New("finance: shares must sum to expense amount")
	ErrPctSumMismatch     = errors.New("finance: percentages must sum to 100")
	ErrParticipantMissing = errors.New("finance: participant not a trip member")
	ErrCurrencyMismatch   = errors.New("finance: currency mismatch")
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	// Expense
	CreateExpense(ctx context.Context, e *Expense) error
	UpdateExpense(ctx context.Context, e *Expense) error
	SoftDeleteExpense(ctx context.Context, id uuid.UUID) error
	FindExpense(ctx context.Context, id uuid.UUID) (*Expense, error)
	ListExpensesByTrip(ctx context.Context, tripID uuid.UUID) ([]Expense, error)
	ListExpensesForExport(ctx context.Context, tripID uuid.UUID, from, to *time.Time) ([]Expense, error)

	// Shares
	ReplaceShares(ctx context.Context, expenseID uuid.UUID, shares []ExpenseShare) error
	ListSharesByExpense(ctx context.Context, expenseID uuid.UUID) ([]ExpenseShare, error)
	ListSharesByExpenses(ctx context.Context, expenseIDs []uuid.UUID) (map[uuid.UUID][]ExpenseShare, error)
	SetSharePaid(ctx context.Context, expenseID, userID uuid.UUID, paidAt *time.Time) (*ExpenseShare, error)

	// Budget
	UpsertBudget(ctx context.Context, b *Budget) error
	FindBudget(ctx context.Context, tripID uuid.UUID, category string) (*Budget, error)
	ListBudgets(ctx context.Context, tripID uuid.UUID) ([]Budget, error)
	DeleteBudget(ctx context.Context, tripID, id uuid.UUID) error

	// FX
	FindFxRate(ctx context.Context, base, quote string, asOf time.Time) (*FxSnapshot, error)
	UpsertFxRate(ctx context.Context, s *FxSnapshot) error
	ListFxRates(ctx context.Context, base string) ([]FxSnapshot, error)

	// Settlement
	CreateSettlement(ctx context.Context, s *Settlement) error
	FindSettlement(ctx context.Context, id uuid.UUID) (*Settlement, error)
	UpdateSettlementStatus(ctx context.Context, id uuid.UUID, status SettlementStatus, at time.Time) error
	ListSettlements(ctx context.Context, tripID uuid.UUID) ([]Settlement, error)
	DeleteProposedSettlements(ctx context.Context, tripID uuid.UUID) error

	// Aggregates
	SumPaidByUser(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)
	SumOwedByUser(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)
	SumSpentByCategory(ctx context.Context, tripID uuid.UUID) (map[string]decimal.Decimal, error)
	SumUserPaidByCategory(ctx context.Context, tripID, userID uuid.UUID) (map[string]CategoryAgg, error)
}

// CategoryAgg is the per-category rollup used by personal-stats aggregation.
// Exported so external test packages can satisfy IRepository.
type CategoryAgg struct {
	AmountBase decimal.Decimal
	Count      int
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

// ---- Expense ----

func (r *repository) CreateExpense(ctx context.Context, e *Expense) error {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(e).Error
}

func (r *repository) UpdateExpense(ctx context.Context, e *Expense) error {
	return r.db.WithContext(ctx).Save(e).Error
}

func (r *repository) SoftDeleteExpense(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Delete(&Expense{}, "id = ?", id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrExpenseNotFound
	}
	return nil
}

func (r *repository) FindExpense(ctx context.Context, id uuid.UUID) (*Expense, error) {
	var e Expense
	err := r.db.WithContext(ctx).First(&e, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrExpenseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func (r *repository) ListExpensesByTrip(ctx context.Context, tripID uuid.UUID) ([]Expense, error) {
	var rows []Expense
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("occurred_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) ListExpensesForExport(ctx context.Context, tripID uuid.UUID, from, to *time.Time) ([]Expense, error) {
	q := r.db.WithContext(ctx).Where("trip_id = ?", tripID)
	if from != nil {
		q = q.Where("occurred_at >= ?", *from)
	}
	if to != nil {
		q = q.Where("occurred_at <= ?", *to)
	}
	var rows []Expense
	err := q.Order("occurred_at ASC").Find(&rows).Error
	return rows, err
}

// ---- Shares ----

func (r *repository) ReplaceShares(ctx context.Context, expenseID uuid.UUID, shares []ExpenseShare) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Preserve paid_at across edits — surviving participants keep their
		// prior settlement tick when the creator touches unrelated fields.
		var prior []ExpenseShare
		if err := tx.Where("expense_id = ?", expenseID).Find(&prior).Error; err != nil {
			return err
		}
		priorPaid := make(map[uuid.UUID]*time.Time, len(prior))
		for i := range prior {
			priorPaid[prior[i].UserID] = prior[i].PaidAt
		}
		if err := tx.Where("expense_id = ?", expenseID).Delete(&ExpenseShare{}).Error; err != nil {
			return err
		}
		if len(shares) == 0 {
			return nil
		}
		for i := range shares {
			if shares[i].PaidAt == nil {
				if p, ok := priorPaid[shares[i].UserID]; ok {
					shares[i].PaidAt = p
				}
			}
		}
		return tx.Create(&shares).Error
	})
}

func (r *repository) SetSharePaid(ctx context.Context, expenseID, userID uuid.UUID, paidAt *time.Time) (*ExpenseShare, error) {
	res := r.db.WithContext(ctx).
		Model(&ExpenseShare{}).
		Where("expense_id = ? AND user_id = ?", expenseID, userID).
		Update("paid_at", paidAt)
	if res.Error != nil {
		return nil, res.Error
	}
	if res.RowsAffected == 0 {
		return nil, ErrExpenseNotFound
	}
	var out ExpenseShare
	if err := r.db.WithContext(ctx).
		Where("expense_id = ? AND user_id = ?", expenseID, userID).
		First(&out).Error; err != nil {
		return nil, err
	}
	return &out, nil
}

func (r *repository) ListSharesByExpense(ctx context.Context, expenseID uuid.UUID) ([]ExpenseShare, error) {
	var rows []ExpenseShare
	err := r.db.WithContext(ctx).
		Where("expense_id = ?", expenseID).
		Find(&rows).Error
	return rows, err
}

func (r *repository) ListSharesByExpenses(ctx context.Context, expenseIDs []uuid.UUID) (map[uuid.UUID][]ExpenseShare, error) {
	out := make(map[uuid.UUID][]ExpenseShare, len(expenseIDs))
	if len(expenseIDs) == 0 {
		return out, nil
	}
	var rows []ExpenseShare
	err := r.db.WithContext(ctx).
		Where("expense_id IN ?", expenseIDs).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, s := range rows {
		out[s.ExpenseID] = append(out[s.ExpenseID], s)
	}
	return out, nil
}

// ---- Budget ----

// UpsertBudget upserts by (trip_id, category). The primary key is regenerated
// only on true INSERT; on conflict we update the existing row and keep its ID.
func (r *repository) UpsertBudget(ctx context.Context, b *Budget) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO finance.budget (id, trip_id, category, amount, currency, created_by, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, now(), now())
		 ON CONFLICT (trip_id, category) DO UPDATE
		    SET amount = EXCLUDED.amount,
		        currency = EXCLUDED.currency,
		        updated_at = now()`,
		b.ID, b.TripID, b.Category, b.Amount, b.Currency, b.CreatedBy,
	).Error
}

func (r *repository) FindBudget(ctx context.Context, tripID uuid.UUID, category string) (*Budget, error) {
	var b Budget
	err := r.db.WithContext(ctx).
		Where("trip_id = ? AND category = ?", tripID, category).
		First(&b).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrBudgetNotFound
	}
	if err != nil {
		return nil, err
	}
	return &b, nil
}

func (r *repository) ListBudgets(ctx context.Context, tripID uuid.UUID) ([]Budget, error) {
	var rows []Budget
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("category ASC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) DeleteBudget(ctx context.Context, tripID, id uuid.UUID) error {
	res := r.db.WithContext(ctx).
		Where("trip_id = ? AND id = ?", tripID, id).
		Delete(&Budget{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrBudgetNotFound
	}
	return nil
}

// ---- FX ----

func (r *repository) FindFxRate(ctx context.Context, base, quote string, asOf time.Time) (*FxSnapshot, error) {
	var s FxSnapshot
	// Exact-day lookup first, then fall back to most-recent-before so
	// backdated expenses on days we never fetched still resolve.
	err := r.db.WithContext(ctx).
		Where("base = ? AND quote = ? AND as_of <= ?", base, quote, asOf).
		Order("as_of DESC").
		Limit(1).
		First(&s).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) UpsertFxRate(ctx context.Context, s *FxSnapshot) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Exec(
		`INSERT INTO finance.fx_snapshot (id, base, quote, rate, as_of, source, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, now())
		 ON CONFLICT (base, quote, as_of) DO UPDATE
		    SET rate = EXCLUDED.rate,
		        source = EXCLUDED.source`,
		s.ID, s.Base, s.Quote, s.Rate, s.AsOf, s.Source,
	).Error
}

func (r *repository) ListFxRates(ctx context.Context, base string) ([]FxSnapshot, error) {
	var rows []FxSnapshot
	q := r.db.WithContext(ctx)
	if base != "" {
		q = q.Where("base = ?", base)
	}
	err := q.Order("as_of DESC, quote ASC").Find(&rows).Error
	return rows, err
}

// ---- Settlement ----

func (r *repository) CreateSettlement(ctx context.Context, s *Settlement) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return r.db.WithContext(ctx).Create(s).Error
}

func (r *repository) FindSettlement(ctx context.Context, id uuid.UUID) (*Settlement, error) {
	var s Settlement
	err := r.db.WithContext(ctx).First(&s, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrSettlementNotFound
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *repository) UpdateSettlementStatus(ctx context.Context, id uuid.UUID, status SettlementStatus, at time.Time) error {
	updates := map[string]any{"status": status}
	switch status {
	case SettlementConfirmed:
		updates["confirmed_at"] = at
	case SettlementCancelled:
		updates["cancelled_at"] = at
	}
	res := r.db.WithContext(ctx).Model(&Settlement{}).
		Where("id = ? AND status = ?", id, SettlementProposed).
		Updates(updates)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrSettlementNotFound
	}
	return nil
}

func (r *repository) ListSettlements(ctx context.Context, tripID uuid.UUID) ([]Settlement, error) {
	var rows []Settlement
	err := r.db.WithContext(ctx).
		Where("trip_id = ?", tripID).
		Order("created_at DESC").
		Find(&rows).Error
	return rows, err
}

func (r *repository) DeleteProposedSettlements(ctx context.Context, tripID uuid.UUID) error {
	return r.db.WithContext(ctx).
		Where("trip_id = ? AND status = ?", tripID, SettlementProposed).
		Delete(&Settlement{}).Error
}

// ---- Aggregates ----
// Aggregates deliberately hit the DB rather than reusing the row-level list —
// per-trip expense counts scale to hundreds, and pulling every row + shares
// into memory to sum in Go would waste bandwidth as the trip ages.

// SumPaidByUser returns each user's out-of-pocket total net of settled shares.
// When another participant marks their share paid, that amount reduces the
// payer's outstanding — otherwise the net_balance never moves after settlement.
func (r *repository) SumPaidByUser(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	type row struct {
		PaidBy uuid.UUID
		Total  decimal.Decimal
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`WITH gross AS (
		         SELECT paid_by, COALESCE(SUM(amount_base), 0) AS s
		         FROM finance.expense
		         WHERE trip_id = ? AND deleted_at IS NULL
		         GROUP BY paid_by
		     ),
		     reimbursed AS (
		         SELECT e.paid_by, COALESCE(SUM(sh.amount_base), 0) AS s
		         FROM finance.expense_share sh
		         JOIN finance.expense e ON e.id = sh.expense_id
		         WHERE e.trip_id = ? AND e.deleted_at IS NULL
		           AND sh.paid_at IS NOT NULL
		           AND sh.user_id <> e.paid_by
		         GROUP BY e.paid_by
		     )
		     SELECT g.paid_by, g.s - COALESCE(r.s, 0) AS total
		     FROM gross g
		     LEFT JOIN reimbursed r ON r.paid_by = g.paid_by`, tripID, tripID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]decimal.Decimal, len(rows))
	for _, r := range rows {
		out[r.PaidBy] = r.Total
	}
	return out, nil
}

// SumOwedByUser skips shares whose paid_at is set — a settled share means the
// participant has already reimbursed the payer, so they no longer owe anything
// for that expense.
func (r *repository) SumOwedByUser(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	type row struct {
		UserID uuid.UUID
		Total  decimal.Decimal
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT s.user_id, COALESCE(SUM(s.amount_base), 0) AS total
		     FROM finance.expense_share s
		     JOIN finance.expense e ON e.id = s.expense_id
		     WHERE e.trip_id = ? AND e.deleted_at IS NULL
		       AND s.paid_at IS NULL
		     GROUP BY s.user_id`, tripID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[uuid.UUID]decimal.Decimal, len(rows))
	for _, r := range rows {
		out[r.UserID] = r.Total
	}
	return out, nil
}

func (r *repository) SumSpentByCategory(ctx context.Context, tripID uuid.UUID) (map[string]decimal.Decimal, error) {
	type row struct {
		Category string
		Total    decimal.Decimal
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT category, COALESCE(SUM(amount_base), 0) AS total
		     FROM finance.expense
		     WHERE trip_id = ? AND deleted_at IS NULL
		     GROUP BY category`, tripID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]decimal.Decimal, len(rows))
	for _, r := range rows {
		out[r.Category] = r.Total
	}
	return out, nil
}

func (r *repository) SumUserPaidByCategory(ctx context.Context, tripID, userID uuid.UUID) (map[string]CategoryAgg, error) {
	type row struct {
		Category string
		Total    decimal.Decimal
		N        int
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Raw(`SELECT category, COALESCE(SUM(amount_base), 0) AS total, COUNT(*) AS n
		     FROM finance.expense
		     WHERE trip_id = ? AND paid_by = ? AND deleted_at IS NULL
		     GROUP BY category`, tripID, userID).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	out := make(map[string]CategoryAgg, len(rows))
	for _, r := range rows {
		out[r.Category] = CategoryAgg{AmountBase: r.Total, Count: r.N}
	}
	return out, nil
}

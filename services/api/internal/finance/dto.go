package finance

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---- Requests ----

// ShareInput is one participant's slice on create/update.
// For 'equal' strategy shares are ignored (recomputed from participants list).
// For 'custom' strategy Amount is required.
// For 'percentage' strategy Pct is required (must sum to 100 across all shares).
type ShareInput struct {
	UserID uuid.UUID        `json:"user_id" binding:"required"`
	Amount *decimal.Decimal `json:"amount,omitempty"`
	Pct    *decimal.Decimal `json:"pct,omitempty"`
}

type CreateExpensePayload struct {
	PaidBy         uuid.UUID       `json:"paid_by" binding:"required"`
	Amount         decimal.Decimal `json:"amount" binding:"required"`
	Currency       string          `json:"currency" binding:"required"`
	Description    string          `json:"description"`
	Category       string          `json:"category"`
	SplitStrategy  SplitStrategy   `json:"split_strategy" binding:"required"`
	Participants   []uuid.UUID     `json:"participants" binding:"required"`
	Shares         []ShareInput    `json:"shares,omitempty"`
	OccurredAt     *time.Time      `json:"occurred_at,omitempty"`
	ReceiptAssetID *uuid.UUID      `json:"receipt_asset_id,omitempty"`
}

type UpdateExpensePayload struct {
	PaidBy         *uuid.UUID       `json:"paid_by,omitempty"`
	Amount         *decimal.Decimal `json:"amount,omitempty"`
	Currency       *string          `json:"currency,omitempty"`
	Description    *string          `json:"description,omitempty"`
	Category       *string          `json:"category,omitempty"`
	SplitStrategy  *SplitStrategy   `json:"split_strategy,omitempty"`
	Participants   []uuid.UUID      `json:"participants,omitempty"`
	Shares         []ShareInput     `json:"shares,omitempty"`
	OccurredAt     *time.Time       `json:"occurred_at,omitempty"`
	ReceiptAssetID *uuid.UUID       `json:"receipt_asset_id,omitempty"`
	ClearReceipt   bool             `json:"clear_receipt,omitempty"`
}

type UpsertBudgetPayload struct {
	Category string          `json:"category" binding:"required"`
	Amount   decimal.Decimal `json:"amount" binding:"required"`
	Currency string          `json:"currency" binding:"required"`
}

type UpsertFxRatePayload struct {
	Base  string          `json:"base" binding:"required"`
	Quote string          `json:"quote" binding:"required"`
	Rate  decimal.Decimal `json:"rate" binding:"required"`
	AsOf  *time.Time      `json:"as_of,omitempty"`
}

type ConfirmSettlementPayload struct {
	Note string `json:"note,omitempty"`
}

type SetSharePaidPayload struct {
	Paid bool `json:"paid"`
}

type ProposeSettlementPayload struct {
	// When true, service runs the min-cashflow reducer over the trip's net
	// balances and creates the proposed transfers. When false, caller may
	// pass Manual entries for one-off settlements outside the algorithm.
	Auto   bool                 `json:"auto"`
	Manual []ManualSettlementIn `json:"manual,omitempty"`
}

type ManualSettlementIn struct {
	PayerID uuid.UUID       `json:"payer_id" binding:"required"`
	PayeeID uuid.UUID       `json:"payee_id" binding:"required"`
	Amount  decimal.Decimal `json:"amount" binding:"required"`
	Note    string          `json:"note,omitempty"`
}

// ---- Responses ----

type ExpenseShareDTO struct {
	UserID     uuid.UUID        `json:"user_id"`
	Amount     decimal.Decimal  `json:"amount"`
	AmountBase decimal.Decimal  `json:"amount_base"`
	SharePct   *decimal.Decimal `json:"share_pct,omitempty"`
	PaidAt     *time.Time       `json:"paid_at,omitempty"`
}

type ExpenseDTO struct {
	ID             uuid.UUID         `json:"id"`
	TripID         uuid.UUID         `json:"trip_id"`
	PaidBy         uuid.UUID         `json:"paid_by"`
	Amount         decimal.Decimal   `json:"amount"`
	Currency       string            `json:"currency"`
	AmountBase     decimal.Decimal   `json:"amount_base"`
	BaseCurrency   string            `json:"base_currency"`
	RateToBase     *decimal.Decimal  `json:"rate_to_base,omitempty"`
	Description    string            `json:"description"`
	Category       string            `json:"category"`
	SplitStrategy  SplitStrategy     `json:"split_strategy"`
	OccurredAt     time.Time         `json:"occurred_at"`
	ReceiptAssetID *uuid.UUID        `json:"receipt_asset_id,omitempty"`
	CreatedBy      uuid.UUID         `json:"created_by"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Shares         []ExpenseShareDTO `json:"shares"`
}

type BudgetDTO struct {
	ID            uuid.UUID       `json:"id"`
	TripID        uuid.UUID       `json:"trip_id"`
	Category      string          `json:"category"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	SpentBase     decimal.Decimal `json:"spent_base"`
	RemainingBase decimal.Decimal `json:"remaining_base"`
	OverBudget    bool            `json:"over_budget"`
	CreatedBy     uuid.UUID       `json:"created_by"`
	CreatedAt     time.Time       `json:"created_at"`
	UpdatedAt     time.Time       `json:"updated_at"`
}

type SettlementDTO struct {
	ID          uuid.UUID        `json:"id"`
	TripID      uuid.UUID        `json:"trip_id"`
	PayerID     uuid.UUID        `json:"payer_id"`
	PayeeID     uuid.UUID        `json:"payee_id"`
	Amount      decimal.Decimal  `json:"amount"`
	Currency    string           `json:"currency"`
	Status      SettlementStatus `json:"status"`
	Note        string           `json:"note"`
	CreatedBy   uuid.UUID        `json:"created_by"`
	CreatedAt   time.Time        `json:"created_at"`
	ConfirmedAt *time.Time       `json:"confirmed_at,omitempty"`
	CancelledAt *time.Time       `json:"cancelled_at,omitempty"`
}

type FxRateDTO struct {
	Base   string          `json:"base"`
	Quote  string          `json:"quote"`
	Rate   decimal.Decimal `json:"rate"`
	AsOf   time.Time       `json:"as_of"`
	Source string          `json:"source"`
}

// PersonalStats is the per-user rollup returned by /trips/:id/finance/stats/me.
// All monetary values are in the trip base currency.
type PersonalStats struct {
	TripID       uuid.UUID       `json:"trip_id"`
	UserID       uuid.UUID       `json:"user_id"`
	BaseCurrency string          `json:"base_currency"`
	TotalPaid    decimal.Decimal `json:"total_paid"`
	TotalOwed    decimal.Decimal `json:"total_owed"`
	NetBalance   decimal.Decimal `json:"net_balance"` // paid - owed; +ve = owed to me
	ByCategory   []CategoryStat  `json:"by_category"`
}

type CategoryStat struct {
	Category   string          `json:"category"`
	AmountBase decimal.Decimal `json:"amount_base"`
	Count      int             `json:"count"`
}

// TripBalance is the trip-wide summary returned by /trips/:id/finance/balances.
// Positive net = the trip owes this user; negative = this user owes the trip.
type TripBalance struct {
	UserID     uuid.UUID       `json:"user_id"`
	Paid       decimal.Decimal `json:"paid"`
	Owed       decimal.Decimal `json:"owed"`
	Net        decimal.Decimal `json:"net"`
}

// ---- Broadcast frame ----

type BroadcastKind string

const (
	BroadcastExpenseCreated   BroadcastKind = "FINANCE_EXPENSE_CREATED"
	BroadcastExpenseUpdated   BroadcastKind = "FINANCE_EXPENSE_UPDATED"
	BroadcastExpenseDeleted   BroadcastKind = "FINANCE_EXPENSE_DELETED"
	BroadcastSettlementAdded  BroadcastKind = "FINANCE_SETTLEMENT_PROPOSED"
	BroadcastSettlementDone   BroadcastKind = "FINANCE_SETTLEMENT_CONFIRMED"
	BroadcastBudgetChanged    BroadcastKind = "FINANCE_BUDGET_CHANGED"
)

// ---- Conversions ----

func expenseToDTO(e *Expense, shares []ExpenseShare, baseCurrency string) ExpenseDTO {
	out := ExpenseDTO{
		ID:             e.ID,
		TripID:         e.TripID,
		PaidBy:         e.PaidBy,
		Amount:         e.Amount,
		Currency:       e.Currency,
		AmountBase:     e.AmountBase,
		BaseCurrency:   baseCurrency,
		Description:    e.Description,
		Category:       e.Category,
		SplitStrategy:  e.SplitStrategy,
		OccurredAt:     e.OccurredAt,
		ReceiptAssetID: e.ReceiptAssetID,
		CreatedBy:      e.CreatedBy,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
		Shares:         make([]ExpenseShareDTO, 0, len(shares)),
	}
	if e.RateToBase.Valid {
		r := e.RateToBase.Decimal
		out.RateToBase = &r
	}
	for i := range shares {
		s := shares[i]
		dto := ExpenseShareDTO{
			UserID:     s.UserID,
			Amount:     s.Amount,
			AmountBase: s.AmountBase,
			PaidAt:     s.PaidAt,
		}
		if s.SharePct.Valid {
			p := s.SharePct.Decimal
			dto.SharePct = &p
		}
		out.Shares = append(out.Shares, dto)
	}
	return out
}

func budgetToDTO(b *Budget, spentBase decimal.Decimal) BudgetDTO {
	remaining := b.Amount.Sub(spentBase)
	return BudgetDTO{
		ID:            b.ID,
		TripID:        b.TripID,
		Category:      b.Category,
		Amount:        b.Amount,
		Currency:      b.Currency,
		SpentBase:     spentBase,
		RemainingBase: remaining,
		OverBudget:    spentBase.GreaterThan(b.Amount),
		CreatedBy:     b.CreatedBy,
		CreatedAt:     b.CreatedAt,
		UpdatedAt:     b.UpdatedAt,
	}
}

func settlementToDTO(s *Settlement) SettlementDTO {
	return SettlementDTO{
		ID:          s.ID,
		TripID:      s.TripID,
		PayerID:     s.PayerID,
		PayeeID:     s.PayeeID,
		Amount:      s.Amount,
		Currency:    s.Currency,
		Status:      s.Status,
		Note:        s.Note,
		CreatedBy:   s.CreatedBy,
		CreatedAt:   s.CreatedAt,
		ConfirmedAt: s.ConfirmedAt,
		CancelledAt: s.CancelledAt,
	}
}

func fxSnapshotToDTO(s *FxSnapshot) FxRateDTO {
	return FxRateDTO{
		Base:   s.Base,
		Quote:  s.Quote,
		Rate:   s.Rate,
		AsOf:   s.AsOf,
		Source: s.Source,
	}
}

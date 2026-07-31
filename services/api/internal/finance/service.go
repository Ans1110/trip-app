package finance

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/media"
	tripmod "github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type TripAuthorizer interface {
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)
	FindTripByID(ctx context.Context, id uuid.UUID) (*tripmod.Trip, error)
	FindRoomByTripID(ctx context.Context, tripID uuid.UUID) (*tripmod.Room, error)
	ListMembers(ctx context.Context, roomID uuid.UUID) ([]tripmod.RoomMember, error)
}

type MediaAuthorizer interface {
	AuthorizeAssetForOwner(ctx context.Context, userID, assetID uuid.UUID) (*media.Asset, error)
}

type Broadcaster interface {
	PublishTripEvent(ctx context.Context, tripID uuid.UUID, payload []byte)
}

type IService interface {
	// Expense
	CreateExpense(ctx context.Context, userID, tripID uuid.UUID, p CreateExpensePayload) (*ExpenseDTO, error)
	UpdateExpense(ctx context.Context, userID, expenseID uuid.UUID, p UpdateExpensePayload) (*ExpenseDTO, error)
	DeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error
	GetExpense(ctx context.Context, userID, expenseID uuid.UUID) (*ExpenseDTO, error)
	ListExpenses(ctx context.Context, userID, tripID uuid.UUID) ([]ExpenseDTO, error)
	SetSharePaid(ctx context.Context, userID, expenseID, targetUserID uuid.UUID, paid bool) (*ExpenseDTO, error)

	// Budget
	UpsertBudget(ctx context.Context, userID, tripID uuid.UUID, p UpsertBudgetPayload) (*BudgetDTO, error)
	ListBudgets(ctx context.Context, userID, tripID uuid.UUID) ([]BudgetDTO, error)
	DeleteBudget(ctx context.Context, userID, tripID, budgetID uuid.UUID) error

	// FX
	UpsertFxRate(ctx context.Context, userID uuid.UUID, p UpsertFxRatePayload) (*FxRateDTO, error)
	ListFxRates(ctx context.Context, userID uuid.UUID, base string) ([]FxRateDTO, error)

	// Settlement
	ProposeSettlements(ctx context.Context, userID, tripID uuid.UUID, p ProposeSettlementPayload) ([]SettlementDTO, error)
	ConfirmSettlement(ctx context.Context, userID, settlementID uuid.UUID, p ConfirmSettlementPayload) (*SettlementDTO, error)
	CancelSettlement(ctx context.Context, userID, settlementID uuid.UUID) (*SettlementDTO, error)
	ListSettlements(ctx context.Context, userID, tripID uuid.UUID) ([]SettlementDTO, error)

	// Stats / export
	GetTripBalances(ctx context.Context, userID, tripID uuid.UUID) ([]TripBalance, error)
	GetPersonalStats(ctx context.Context, userID, tripID uuid.UUID) (*PersonalStats, error)
	ExportCSV(ctx context.Context, userID, tripID uuid.UUID, from, to *time.Time) ([]byte, error)
}

type ServiceConfig struct {
	Repo         IRepository
	TripAuth     TripAuthorizer
	MediaAuthz   MediaAuthorizer
	Broadcaster  Broadcaster
	RateProvider RateProvider
	Logger       *zap.Logger
}

type service struct {
	repo   IRepository
	auth   TripAuthorizer
	media  MediaAuthorizer
	bcast  Broadcaster
	rates  RateProvider
	logger *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		auth:   cfg.TripAuth,
		media:  cfg.MediaAuthz,
		bcast:  cfg.Broadcaster,
		rates:  cfg.RateProvider,
		logger: logger.With(zap.String("layer", "finance.service")),
	}
}

const (
	maxDescriptionLen = 500
	maxCategoryLen    = 40
	maxNoteLen        = 500
)

// Rounding for money math. FOUR decimal places matches the schema NUMERIC(18,4)
// so we never store more precision than the column can hold.
const moneyScale = 4

var (
	oneHundred     = decimal.NewFromInt(100)
	splitTolerance = decimal.NewFromFloat(0.01)
)

// ---- Expense create ----

func (s *service) CreateExpense(ctx context.Context, userID, tripID uuid.UUID, p CreateExpensePayload) (*ExpenseDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	trip, err := s.auth.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if trip == nil {
		return nil, ErrForbidden
	}
	baseCurrency := normalizeCurrency(trip.BaseCurrency)
	if baseCurrency == "" {
		baseCurrency = "TWD"
	}

	if err := s.validateCreatePayload(&p); err != nil {
		return nil, err
	}
	if err := s.validateParticipants(ctx, tripID, p.PaidBy, p.Participants); err != nil {
		return nil, err
	}
	if p.ReceiptAssetID != nil {
		if _, err := s.media.AuthorizeAssetForOwner(ctx, userID, *p.ReceiptAssetID); err != nil {
			return nil, fmt.Errorf("receipt: %w", err)
		}
	}

	amount := p.Amount.Round(moneyScale)
	currency := normalizeCurrency(p.Currency)
	occurred := time.Now().UTC()
	if p.OccurredAt != nil {
		occurred = p.OccurredAt.UTC()
	}

	rate, rateNull, err := s.resolveRate(ctx, currency, baseCurrency, occurred)
	if err != nil {
		return nil, err
	}
	amountBase := amount.Mul(rate).Round(moneyScale)

	shares, err := computeShares(p.SplitStrategy, amount, amountBase, p.Participants, p.Shares)
	if err != nil {
		return nil, err
	}

	e := &Expense{
		TripID:         tripID,
		PaidBy:         p.PaidBy,
		Amount:         amount,
		Currency:       currency,
		AmountBase:     amountBase,
		RateToBase:     rateNull,
		Description:    truncate(strings.TrimSpace(p.Description), maxDescriptionLen),
		Category:       normalizeCategory(p.Category),
		SplitStrategy:  p.SplitStrategy,
		OccurredAt:     occurred,
		ReceiptAssetID: p.ReceiptAssetID,
		CreatedBy:      userID,
	}
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreateExpense(ctx, e); err != nil {
			return err
		}
		attachExpenseID(shares, e.ID)
		return tx.ReplaceShares(ctx, e.ID, shares)
	})
	if err != nil {
		return nil, err
	}

	dto := expenseToDTO(e, shares, baseCurrency)
	s.broadcast(ctx, tripID, BroadcastExpenseCreated, dto)
	return &dto, nil
}

// ---- Expense update ----

func (s *service) UpdateExpense(ctx context.Context, userID, expenseID uuid.UUID, p UpdateExpensePayload) (*ExpenseDTO, error) {
	e, err := s.repo.FindExpense(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, e.TripID, userID); err != nil {
		return nil, err
	}
	if e.CreatedBy != userID && e.PaidBy != userID {
		return nil, ErrForbidden
	}
	trip, err := s.auth.FindTripByID(ctx, e.TripID)
	if err != nil {
		return nil, err
	}
	baseCurrency := normalizeCurrency(trip.BaseCurrency)

	// validate the merged state (participants + amount + strategy) as one atomic thing.
	amount := e.Amount
	currency := e.Currency
	strategy := e.SplitStrategy
	description := e.Description
	category := e.Category
	occurred := e.OccurredAt
	paidBy := e.PaidBy
	receiptID := e.ReceiptAssetID

	if p.Amount != nil {
		amount = p.Amount.Round(moneyScale)
	}
	if p.Currency != nil {
		currency = normalizeCurrency(*p.Currency)
	}
	if p.SplitStrategy != nil {
		strategy = *p.SplitStrategy
	}
	if p.Description != nil {
		description = truncate(strings.TrimSpace(*p.Description), maxDescriptionLen)
	}
	if p.Category != nil {
		category = normalizeCategory(*p.Category)
	}
	if p.OccurredAt != nil {
		occurred = p.OccurredAt.UTC()
	}
	if p.PaidBy != nil {
		paidBy = *p.PaidBy
	}
	if p.ClearReceipt {
		receiptID = nil
	} else if p.ReceiptAssetID != nil {
		if _, err := s.media.AuthorizeAssetForOwner(ctx, userID, *p.ReceiptAssetID); err != nil {
			return nil, fmt.Errorf("receipt: %w", err)
		}
		receiptID = p.ReceiptAssetID
	}

	if amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidPayload
	}
	if !isValidStrategy(strategy) {
		return nil, ErrInvalidPayload
	}

	// Resolve participants for the new state.
	participants := p.Participants
	if len(participants) == 0 {
		existing, err := s.repo.ListSharesByExpense(ctx, expenseID)
		if err != nil {
			return nil, err
		}
		for _, sh := range existing {
			participants = append(participants, sh.UserID)
		}
	}
	if err := s.validateParticipants(ctx, e.TripID, paidBy, participants); err != nil {
		return nil, err
	}

	rate, rateNull, err := s.resolveRate(ctx, currency, baseCurrency, occurred)
	if err != nil {
		return nil, err
	}
	amountBase := amount.Mul(rate).Round(moneyScale)

	sharesIn := p.Shares
	// If strategy changed to 'equal' or caller left shares empty on an
	// amount-only edit, recompute from the participant list.
	if len(sharesIn) == 0 {
		sharesIn = defaultSharesForParticipants(strategy, participants, e)
	}
	shares, err := computeShares(strategy, amount, amountBase, participants, sharesIn)
	if err != nil {
		return nil, err
	}

	e.PaidBy = paidBy
	e.Amount = amount
	e.Currency = currency
	e.AmountBase = amountBase
	e.RateToBase = rateNull
	e.Description = description
	e.Category = category
	e.SplitStrategy = strategy
	e.OccurredAt = occurred
	e.ReceiptAssetID = receiptID

	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.UpdateExpense(ctx, e); err != nil {
			return err
		}
		attachExpenseID(shares, e.ID)
		return tx.ReplaceShares(ctx, e.ID, shares)
	})
	if err != nil {
		return nil, err
	}

	dto := expenseToDTO(e, shares, baseCurrency)
	s.broadcast(ctx, e.TripID, BroadcastExpenseUpdated, dto)
	return &dto, nil
}

// ---- Expense delete ----

func (s *service) DeleteExpense(ctx context.Context, userID, expenseID uuid.UUID) error {
	e, err := s.repo.FindExpense(ctx, expenseID)
	if err != nil {
		return err
	}
	if err := s.mustBeMember(ctx, e.TripID, userID); err != nil {
		return err
	}
	if e.CreatedBy != userID && e.PaidBy != userID {
		return ErrForbidden
	}
	if err := s.repo.SoftDeleteExpense(ctx, expenseID); err != nil {
		return err
	}
	payload, _ := json.Marshal(struct {
		Type      BroadcastKind `json:"type"`
		TripID    uuid.UUID     `json:"trip_id"`
		ExpenseID uuid.UUID     `json:"expense_id"`
	}{BroadcastExpenseDeleted, e.TripID, expenseID})
	if s.bcast != nil {
		s.bcast.PublishTripEvent(ctx, e.TripID, payload)
	}
	return nil
}

func (s *service) SetSharePaid(ctx context.Context, userID, expenseID, targetUserID uuid.UUID, paid bool) (*ExpenseDTO, error) {
	e, err := s.repo.FindExpense(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, e.TripID, userID); err != nil {
		return nil, err
	}
	if e.CreatedBy != userID {
		return nil, ErrForbidden
	}
	if targetUserID == e.PaidBy {
		return nil, ErrInvalidPayload
	}

	var paidAt *time.Time
	if paid {
		now := time.Now().UTC()
		paidAt = &now
	}
	if _, err := s.repo.SetSharePaid(ctx, expenseID, targetUserID, paidAt); err != nil {
		return nil, err
	}

	trip, err := s.auth.FindTripByID(ctx, e.TripID)
	if err != nil {
		return nil, err
	}
	shares, err := s.repo.ListSharesByExpense(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	dto := expenseToDTO(e, shares, normalizeCurrency(trip.BaseCurrency))
	s.broadcast(ctx, e.TripID, BroadcastExpenseUpdated, dto)
	return &dto, nil
}

// ---- Expense read ----

func (s *service) GetExpense(ctx context.Context, userID, expenseID uuid.UUID) (*ExpenseDTO, error) {
	e, err := s.repo.FindExpense(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, e.TripID, userID); err != nil {
		return nil, err
	}
	trip, err := s.auth.FindTripByID(ctx, e.TripID)
	if err != nil {
		return nil, err
	}
	shares, err := s.repo.ListSharesByExpense(ctx, expenseID)
	if err != nil {
		return nil, err
	}
	dto := expenseToDTO(e, shares, normalizeCurrency(trip.BaseCurrency))
	return &dto, nil
}

func (s *service) ListExpenses(ctx context.Context, userID, tripID uuid.UUID) ([]ExpenseDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	trip, err := s.auth.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListExpensesByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return []ExpenseDTO{}, nil
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	sharesByExpense, err := s.repo.ListSharesByExpenses(ctx, ids)
	if err != nil {
		return nil, err
	}
	base := normalizeCurrency(trip.BaseCurrency)
	out := make([]ExpenseDTO, 0, len(rows))
	for i := range rows {
		out = append(out, expenseToDTO(&rows[i], sharesByExpense[rows[i].ID], base))
	}
	return out, nil
}

// ---- Budget ----

func (s *service) UpsertBudget(ctx context.Context, userID, tripID uuid.UUID, p UpsertBudgetPayload) (*BudgetDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	cat := normalizeCategory(p.Category)
	if cat == "" {
		return nil, ErrInvalidPayload
	}
	if p.Amount.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidPayload
	}
	b := &Budget{
		TripID:    tripID,
		Category:  cat,
		Amount:    p.Amount.Round(moneyScale),
		Currency:  normalizeCurrency(p.Currency),
		CreatedBy: userID,
	}
	if err := s.repo.UpsertBudget(ctx, b); err != nil {
		return nil, err
	}
	// Re-read so we get server-side timestamps + the row ID after conflict.
	fresh, err := s.repo.FindBudget(ctx, tripID, cat)
	if err != nil {
		return nil, err
	}
	spent, err := s.computeSpentForBudget(ctx, tripID, cat)
	if err != nil {
		return nil, err
	}
	dto := budgetToDTO(fresh, spent)
	s.broadcast(ctx, tripID, BroadcastBudgetChanged, dto)
	return &dto, nil
}

func (s *service) ListBudgets(ctx context.Context, userID, tripID uuid.UUID) ([]BudgetDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListBudgets(ctx, tripID)
	if err != nil {
		return nil, err
	}
	spentByCat, err := s.repo.SumSpentByCategory(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]BudgetDTO, 0, len(rows))
	for i := range rows {
		out = append(out, budgetToDTO(&rows[i], spentByCat[rows[i].Category]))
	}
	return out, nil
}

func (s *service) DeleteBudget(ctx context.Context, userID, tripID, budgetID uuid.UUID) error {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return err
	}
	if err := s.repo.DeleteBudget(ctx, tripID, budgetID); err != nil {
		return err
	}
	s.broadcast(ctx, tripID, BroadcastBudgetChanged, map[string]any{"deleted_id": budgetID})
	return nil
}

// ---- FX ----

func (s *service) UpsertFxRate(ctx context.Context, userID uuid.UUID, p UpsertFxRatePayload) (*FxRateDTO, error) {
	base := normalizeCurrency(p.Base)
	quote := normalizeCurrency(p.Quote)
	if base == "" || quote == "" || base == quote {
		return nil, ErrInvalidPayload
	}
	if p.Rate.LessThanOrEqual(decimal.Zero) {
		return nil, ErrInvalidPayload
	}
	asOf := time.Now().UTC()
	if p.AsOf != nil {
		asOf = *p.AsOf
	}
	snap := &FxSnapshot{
		Base:   base,
		Quote:  quote,
		Rate:   p.Rate,
		AsOf:   truncateToDate(asOf),
		Source: "manual",
	}
	if err := s.repo.UpsertFxRate(ctx, snap); err != nil {
		return nil, err
	}
	dto := fxSnapshotToDTO(snap)
	return &dto, nil
}

func (s *service) ListFxRates(ctx context.Context, userID uuid.UUID, base string) ([]FxRateDTO, error) {
	rows, err := s.repo.ListFxRates(ctx, normalizeCurrency(base))
	if err != nil {
		return nil, err
	}
	out := make([]FxRateDTO, 0, len(rows))
	for i := range rows {
		out = append(out, fxSnapshotToDTO(&rows[i]))
	}
	return out, nil
}

// resolveRate returns the FX rate to convert `from` -> `to` at `at`. If they
// match, returns 1 with a NULL sentinel.
func (s *service) resolveRate(ctx context.Context, from, to string, at time.Time) (decimal.Decimal, decimal.NullDecimal, error) {
	from = normalizeCurrency(from)
	to = normalizeCurrency(to)
	if from == to {
		return decimal.NewFromInt(1), decimal.NullDecimal{}, nil
	}
	day := truncateToDate(at)
	snap, err := s.repo.FindFxRate(ctx, from, to, day)
	if err != nil {
		return decimal.Zero, decimal.NullDecimal{}, err
	}
	if snap != nil {
		return snap.Rate, decimal.NullDecimal{Decimal: snap.Rate, Valid: true}, nil
	}
	if s.rates == nil {
		return decimal.Zero, decimal.NullDecimal{}, fmt.Errorf("%w: %s->%s at %s", ErrRateUnavailable, from, to, day.Format(time.DateOnly))
	}
	rate, source, err := s.rates.GetRate(ctx, from, to, day)
	if err != nil {
		return decimal.Zero, decimal.NullDecimal{}, err
	}
	newSnap := &FxSnapshot{
		Base:   from,
		Quote:  to,
		Rate:   rate,
		AsOf:   day,
		Source: source,
	}
	// Persist best-effort
	if err := s.repo.UpsertFxRate(ctx, newSnap); err != nil {
		s.logger.Warn("finance: fx snapshot cache write failed",
			zap.String("base", from), zap.String("quote", to), zap.Error(err))
	}
	return rate, decimal.NullDecimal{Decimal: rate, Valid: true}, nil
}

// ---- helpers ----

func (s *service) mustBeMember(ctx context.Context, tripID, userID uuid.UUID) error {
	ok, err := s.auth.IsRoomMember(ctx, tripID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

// validateParticipants asserts the payer and every participant is a room
// member, and that the participant list is non-empty and deduplicated.
func (s *service) validateParticipants(ctx context.Context, tripID, payer uuid.UUID, participants []uuid.UUID) error {
	if len(participants) == 0 {
		return ErrInvalidPayload
	}
	seen := make(map[uuid.UUID]struct{}, len(participants))
	for _, u := range participants {
		if u == uuid.Nil {
			return ErrInvalidPayload
		}
		seen[u] = struct{}{}
	}
	if len(seen) != len(participants) {
		return ErrInvalidPayload
	}
	room, err := s.auth.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrForbidden
	}
	members, err := s.auth.ListMembers(ctx, room.ID)
	if err != nil {
		return err
	}
	memberSet := make(map[uuid.UUID]struct{}, len(members))
	for _, m := range members {
		memberSet[m.UserID] = struct{}{}
	}
	if _, ok := memberSet[payer]; !ok {
		return ErrParticipantMissing
	}
	for _, u := range participants {
		if _, ok := memberSet[u]; !ok {
			return ErrParticipantMissing
		}
	}
	return nil
}

func (s *service) validateCreatePayload(p *CreateExpensePayload) error {
	if p.Amount.LessThanOrEqual(decimal.Zero) {
		return ErrInvalidPayload
	}
	if !isValidStrategy(p.SplitStrategy) {
		return ErrInvalidPayload
	}
	if normalizeCurrency(p.Currency) == "" {
		return ErrInvalidPayload
	}
	return nil
}

func (s *service) computeSpentForBudget(ctx context.Context, tripID uuid.UUID, category string) (decimal.Decimal, error) {
	byCat, err := s.repo.SumSpentByCategory(ctx, tripID)
	if err != nil {
		return decimal.Zero, err
	}
	return byCat[category], nil
}

func (s *service) broadcast(ctx context.Context, tripID uuid.UUID, kind BroadcastKind, data any) {
	if s.bcast == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type   BroadcastKind `json:"type"`
		TripID uuid.UUID     `json:"trip_id"`
		Data   any           `json:"data"`
	}{kind, tripID, data})
	if err != nil {
		s.logger.Warn("finance: marshal broadcast", zap.Error(err))
		return
	}
	s.bcast.PublishTripEvent(ctx, tripID, payload)
}

// ---- split math ----

// computeShares turns the incoming split spec into concrete ExpenseShare rows,
// rounding to the money scale and fixing the last-share to absorb rounding
// drift so the sum matches amount exactly.
func computeShares(strategy SplitStrategy, amount, amountBase decimal.Decimal, participants []uuid.UUID, shares []ShareInput) ([]ExpenseShare, error) {
	if len(participants) == 0 {
		return nil, ErrInvalidPayload
	}
	out := make([]ExpenseShare, 0, len(participants))
	switch strategy {
	case SplitEqual:
		n := int64(len(participants))
		each := amount.Div(decimal.NewFromInt(n)).Round(moneyScale)
		eachBase := amountBase.Div(decimal.NewFromInt(n)).Round(moneyScale)
		sum := decimal.Zero
		sumBase := decimal.Zero
		sorted := sortedIDs(participants)
		for i, uid := range sorted {
			amt := each
			amtBase := eachBase
			// Last share absorbs rounding difference so totals reconcile exactly.
			if i == len(sorted)-1 {
				amt = amount.Sub(sum)
				amtBase = amountBase.Sub(sumBase)
			}
			out = append(out, ExpenseShare{UserID: uid, Amount: amt, AmountBase: amtBase})
			sum = sum.Add(amt)
			sumBase = sumBase.Add(amtBase)
		}
	case SplitCustom:
		if len(shares) == 0 {
			return nil, ErrInvalidPayload
		}
		byUser := make(map[uuid.UUID]ShareInput, len(shares))
		for _, s := range shares {
			if s.Amount == nil {
				return nil, ErrInvalidPayload
			}
			byUser[s.UserID] = s
		}
		partSet := make(map[uuid.UUID]struct{}, len(participants))
		for _, u := range participants {
			partSet[u] = struct{}{}
		}
		if len(byUser) != len(partSet) {
			return nil, ErrInvalidPayload
		}
		sum := decimal.Zero
		for uid := range partSet {
			s, ok := byUser[uid]
			if !ok {
				return nil, ErrInvalidPayload
			}
			amt := s.Amount.Round(moneyScale)
			if amt.LessThan(decimal.Zero) {
				return nil, ErrInvalidPayload
			}
			sum = sum.Add(amt)
		}
		if sum.Sub(amount).Abs().GreaterThan(splitTolerance) {
			return nil, ErrShareSumMismatch
		}
		// Materialize in a stable order so tests can compare.
		sorted := sortedIDs(participants)
		accBase := decimal.Zero
		for i, uid := range sorted {
			amt := byUser[uid].Amount.Round(moneyScale)
			// Base amount is scaled proportionally from original amount ratio.
			var amtBase decimal.Decimal
			if amount.IsZero() {
				amtBase = decimal.Zero
			} else if i == len(sorted)-1 {
				amtBase = amountBase.Sub(accBase)
			} else {
				amtBase = amountBase.Mul(amt).Div(amount).Round(moneyScale)
				accBase = accBase.Add(amtBase)
			}
			out = append(out, ExpenseShare{UserID: uid, Amount: amt, AmountBase: amtBase})
		}
	case SplitPercentage:
		if len(shares) == 0 {
			return nil, ErrInvalidPayload
		}
		byUser := make(map[uuid.UUID]ShareInput, len(shares))
		sumPct := decimal.Zero
		for _, s := range shares {
			if s.Pct == nil {
				return nil, ErrInvalidPayload
			}
			pct := s.Pct.Round(6)
			if pct.LessThan(decimal.Zero) {
				return nil, ErrInvalidPayload
			}
			byUser[s.UserID] = ShareInput{UserID: s.UserID, Pct: &pct}
			sumPct = sumPct.Add(pct)
		}
		if sumPct.Sub(oneHundred).Abs().GreaterThan(splitTolerance) {
			return nil, ErrPctSumMismatch
		}
		partSet := make(map[uuid.UUID]struct{}, len(participants))
		for _, u := range participants {
			partSet[u] = struct{}{}
		}
		if len(byUser) != len(partSet) {
			return nil, ErrInvalidPayload
		}
		sorted := sortedIDs(participants)
		accAmt := decimal.Zero
		accBase := decimal.Zero
		for i, uid := range sorted {
			s, ok := byUser[uid]
			if !ok {
				return nil, ErrInvalidPayload
			}
			pct := *s.Pct
			var amt, amtBase decimal.Decimal
			if i == len(sorted)-1 {
				amt = amount.Sub(accAmt)
				amtBase = amountBase.Sub(accBase)
			} else {
				amt = amount.Mul(pct).Div(oneHundred).Round(moneyScale)
				amtBase = amountBase.Mul(pct).Div(oneHundred).Round(moneyScale)
				accAmt = accAmt.Add(amt)
				accBase = accBase.Add(amtBase)
			}
			pctVal := pct
			out = append(out, ExpenseShare{
				UserID:     uid,
				Amount:     amt,
				AmountBase: amtBase,
				SharePct:   decimal.NullDecimal{Decimal: pctVal, Valid: true},
			})
		}
	default:
		return nil, ErrInvalidPayload
	}
	return out, nil
}

// defaultSharesForParticipants supplies a sensible input for update flows
// where the caller changed the amount but not the shares.
func defaultSharesForParticipants(strategy SplitStrategy, participants []uuid.UUID, existing *Expense) []ShareInput {
	_ = existing
	if strategy == SplitEqual {
		return nil
	}
	return nil
}

func isValidStrategy(s SplitStrategy) bool {
	switch s {
	case SplitEqual, SplitCustom, SplitPercentage:
		return true
	}
	return false
}

func normalizeCategory(c string) string {
	c = strings.TrimSpace(strings.ToLower(c))
	if c == "" {
		return "other"
	}
	if len(c) > maxCategoryLen {
		c = c[:maxCategoryLen]
	}
	return c
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max]
}

func attachExpenseID(shares []ExpenseShare, id uuid.UUID) {
	for i := range shares {
		shares[i].ExpenseID = id
	}
}

func sortedIDs(ids []uuid.UUID) []uuid.UUID {
	out := make([]uuid.UUID, len(ids))
	copy(out, ids)
	sort.Slice(out, func(i, j int) bool {
		return out[i].String() < out[j].String()
	})
	return out
}

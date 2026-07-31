package finance_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/finance"
	"github.com/Ans1110/trip-app/internal/media"
	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- Repository mock ----

type repoMock struct {
	// Stored state so tests can assert what was persisted without wiring up
	// per-method assertions.
	expenses    map[uuid.UUID]*finance.Expense
	shares      map[uuid.UUID][]finance.ExpenseShare
	budgets     map[string]*finance.Budget // key: tripID|category
	fxRates     map[string]*finance.FxSnapshot
	settlements map[uuid.UUID]*finance.Settlement

	// Hooks let individual tests override behavior without swapping the whole
	// mock — assign to a hook to intercept, leave nil for default behavior.
	createExpense      func(context.Context, *finance.Expense) error
	upsertFxRate       func(context.Context, *finance.FxSnapshot) error
	sumPaidByUser      func(context.Context, uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)
	sumOwedByUser      func(context.Context, uuid.UUID) (map[uuid.UUID]decimal.Decimal, error)
	sumSpentByCategory func(context.Context, uuid.UUID) (map[string]decimal.Decimal, error)
	listExpensesForExp func(context.Context, uuid.UUID, *time.Time, *time.Time) ([]finance.Expense, error)
}

func newRepoMock() *repoMock {
	return &repoMock{
		expenses:    make(map[uuid.UUID]*finance.Expense),
		shares:      make(map[uuid.UUID][]finance.ExpenseShare),
		budgets:     make(map[string]*finance.Budget),
		fxRates:     make(map[string]*finance.FxSnapshot),
		settlements: make(map[uuid.UUID]*finance.Settlement),
	}
}

func (r *repoMock) WithTx(ctx context.Context, fn func(finance.IRepository) error) error {
	return fn(r)
}

func (r *repoMock) CreateExpense(ctx context.Context, e *finance.Expense) error {
	if r.createExpense != nil {
		return r.createExpense(ctx, e)
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	cp := *e
	r.expenses[e.ID] = &cp
	return nil
}
func (r *repoMock) UpdateExpense(ctx context.Context, e *finance.Expense) error {
	cp := *e
	r.expenses[e.ID] = &cp
	return nil
}
func (r *repoMock) SoftDeleteExpense(ctx context.Context, id uuid.UUID) error {
	if _, ok := r.expenses[id]; !ok {
		return finance.ErrExpenseNotFound
	}
	delete(r.expenses, id)
	delete(r.shares, id)
	return nil
}
func (r *repoMock) FindExpense(ctx context.Context, id uuid.UUID) (*finance.Expense, error) {
	e, ok := r.expenses[id]
	if !ok {
		return nil, finance.ErrExpenseNotFound
	}
	cp := *e
	return &cp, nil
}
func (r *repoMock) ListExpensesByTrip(ctx context.Context, tripID uuid.UUID) ([]finance.Expense, error) {
	out := []finance.Expense{}
	for _, e := range r.expenses {
		if e.TripID == tripID {
			out = append(out, *e)
		}
	}
	return out, nil
}
func (r *repoMock) ListExpensesForExport(ctx context.Context, tripID uuid.UUID, from, to *time.Time) ([]finance.Expense, error) {
	if r.listExpensesForExp != nil {
		return r.listExpensesForExp(ctx, tripID, from, to)
	}
	return r.ListExpensesByTrip(ctx, tripID)
}
func (r *repoMock) ReplaceShares(ctx context.Context, expenseID uuid.UUID, shares []finance.ExpenseShare) error {
	cp := make([]finance.ExpenseShare, len(shares))
	copy(cp, shares)
	r.shares[expenseID] = cp
	return nil
}
func (r *repoMock) ListSharesByExpense(ctx context.Context, expenseID uuid.UUID) ([]finance.ExpenseShare, error) {
	cp := make([]finance.ExpenseShare, len(r.shares[expenseID]))
	copy(cp, r.shares[expenseID])
	return cp, nil
}
func (r *repoMock) ListSharesByExpenses(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID][]finance.ExpenseShare, error) {
	out := make(map[uuid.UUID][]finance.ExpenseShare, len(ids))
	for _, id := range ids {
		if s, ok := r.shares[id]; ok {
			cp := make([]finance.ExpenseShare, len(s))
			copy(cp, s)
			out[id] = cp
		}
	}
	return out, nil
}
func (r *repoMock) SetSharePaid(ctx context.Context, expenseID, userID uuid.UUID, paidAt *time.Time) (*finance.ExpenseShare, error) {
	shares, ok := r.shares[expenseID]
	if !ok {
		return nil, finance.ErrExpenseNotFound
	}
	for i := range shares {
		if shares[i].UserID == userID {
			shares[i].PaidAt = paidAt
			r.shares[expenseID] = shares
			cp := shares[i]
			return &cp, nil
		}
	}
	return nil, finance.ErrExpenseNotFound
}
func (r *repoMock) UpsertBudget(ctx context.Context, b *finance.Budget) error {
	if b.ID == uuid.Nil {
		b.ID = uuid.New()
	}
	key := b.TripID.String() + "|" + b.Category
	if existing, ok := r.budgets[key]; ok {
		b.ID = existing.ID
	}
	now := time.Now().UTC()
	b.CreatedAt = now
	b.UpdatedAt = now
	cp := *b
	r.budgets[key] = &cp
	return nil
}
func (r *repoMock) FindBudget(ctx context.Context, tripID uuid.UUID, category string) (*finance.Budget, error) {
	key := tripID.String() + "|" + category
	b, ok := r.budgets[key]
	if !ok {
		return nil, finance.ErrBudgetNotFound
	}
	cp := *b
	return &cp, nil
}
func (r *repoMock) ListBudgets(ctx context.Context, tripID uuid.UUID) ([]finance.Budget, error) {
	out := []finance.Budget{}
	for _, b := range r.budgets {
		if b.TripID == tripID {
			out = append(out, *b)
		}
	}
	return out, nil
}
func (r *repoMock) DeleteBudget(ctx context.Context, tripID, id uuid.UUID) error {
	for k, b := range r.budgets {
		if b.ID == id && b.TripID == tripID {
			delete(r.budgets, k)
			return nil
		}
	}
	return finance.ErrBudgetNotFound
}
func (r *repoMock) FindFxRate(ctx context.Context, base, quote string, asOf time.Time) (*finance.FxSnapshot, error) {
	// Match on base+quote, return any snapshot on/before asOf.
	var best *finance.FxSnapshot
	for _, s := range r.fxRates {
		if s.Base != base || s.Quote != quote {
			continue
		}
		if s.AsOf.After(asOf) {
			continue
		}
		if best == nil || s.AsOf.After(best.AsOf) {
			cp := *s
			best = &cp
		}
	}
	return best, nil
}
func (r *repoMock) UpsertFxRate(ctx context.Context, s *finance.FxSnapshot) error {
	if r.upsertFxRate != nil {
		return r.upsertFxRate(ctx, s)
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	key := s.Base + ":" + s.Quote + ":" + s.AsOf.Format("2006-01-02")
	cp := *s
	r.fxRates[key] = &cp
	return nil
}
func (r *repoMock) ListFxRates(ctx context.Context, base string) ([]finance.FxSnapshot, error) {
	out := []finance.FxSnapshot{}
	for _, s := range r.fxRates {
		if base == "" || s.Base == base {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (r *repoMock) CreateSettlement(ctx context.Context, s *finance.Settlement) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	s.CreatedAt = time.Now().UTC()
	cp := *s
	r.settlements[s.ID] = &cp
	return nil
}
func (r *repoMock) FindSettlement(ctx context.Context, id uuid.UUID) (*finance.Settlement, error) {
	s, ok := r.settlements[id]
	if !ok {
		return nil, finance.ErrSettlementNotFound
	}
	cp := *s
	return &cp, nil
}
func (r *repoMock) UpdateSettlementStatus(ctx context.Context, id uuid.UUID, status finance.SettlementStatus, at time.Time) error {
	s, ok := r.settlements[id]
	if !ok {
		return finance.ErrSettlementNotFound
	}
	if s.Status != finance.SettlementProposed {
		return finance.ErrSettlementNotFound
	}
	s.Status = status
	switch status {
	case finance.SettlementConfirmed:
		s.ConfirmedAt = &at
	case finance.SettlementCancelled:
		s.CancelledAt = &at
	}
	return nil
}
func (r *repoMock) ListSettlements(ctx context.Context, tripID uuid.UUID) ([]finance.Settlement, error) {
	out := []finance.Settlement{}
	for _, s := range r.settlements {
		if s.TripID == tripID {
			out = append(out, *s)
		}
	}
	return out, nil
}
func (r *repoMock) DeleteProposedSettlements(ctx context.Context, tripID uuid.UUID) error {
	for k, s := range r.settlements {
		if s.TripID == tripID && s.Status == finance.SettlementProposed {
			delete(r.settlements, k)
		}
	}
	return nil
}
func (r *repoMock) SumPaidByUser(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	if r.sumPaidByUser != nil {
		return r.sumPaidByUser(ctx, tripID)
	}
	out := map[uuid.UUID]decimal.Decimal{}
	for _, e := range r.expenses {
		if e.TripID != tripID {
			continue
		}
		out[e.PaidBy] = out[e.PaidBy].Add(e.AmountBase)
	}
	return out, nil
}
func (r *repoMock) SumOwedByUser(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	if r.sumOwedByUser != nil {
		return r.sumOwedByUser(ctx, tripID)
	}
	out := map[uuid.UUID]decimal.Decimal{}
	for id, e := range r.expenses {
		if e.TripID != tripID {
			continue
		}
		for _, sh := range r.shares[id] {
			out[sh.UserID] = out[sh.UserID].Add(sh.AmountBase)
		}
	}
	return out, nil
}
func (r *repoMock) SumSpentByCategory(ctx context.Context, tripID uuid.UUID) (map[string]decimal.Decimal, error) {
	if r.sumSpentByCategory != nil {
		return r.sumSpentByCategory(ctx, tripID)
	}
	out := map[string]decimal.Decimal{}
	for _, e := range r.expenses {
		if e.TripID == tripID {
			out[e.Category] = out[e.Category].Add(e.AmountBase)
		}
	}
	return out, nil
}
func (r *repoMock) SumUserPaidByCategory(ctx context.Context, tripID, userID uuid.UUID) (map[string]finance.CategoryAgg, error) {
	out := map[string]finance.CategoryAgg{}
	for _, e := range r.expenses {
		if e.TripID != tripID || e.PaidBy != userID {
			continue
		}
		agg := out[e.Category]
		agg.AmountBase = agg.AmountBase.Add(e.AmountBase)
		agg.Count++
		out[e.Category] = agg
	}
	return out, nil
}

var _ finance.IRepository = (*repoMock)(nil)

// ---- Trip authorizer mock ----

type tripAuthMock struct {
	isMember       func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	findTrip       func(context.Context, uuid.UUID) (*trip.Trip, error)
	findRoom       func(context.Context, uuid.UUID) (*trip.Room, error)
	listMembers    func(context.Context, uuid.UUID) ([]trip.RoomMember, error)
	defaultMembers []uuid.UUID
	baseCurrency   string
	roomID         uuid.UUID
}

func (t *tripAuthMock) IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error) {
	if t.isMember != nil {
		return t.isMember(ctx, tripID, userID)
	}
	for _, u := range t.defaultMembers {
		if u == userID {
			return true, nil
		}
	}
	return false, nil
}
func (t *tripAuthMock) FindTripByID(ctx context.Context, id uuid.UUID) (*trip.Trip, error) {
	if t.findTrip != nil {
		return t.findTrip(ctx, id)
	}
	base := t.baseCurrency
	if base == "" {
		base = "USD"
	}
	return &trip.Trip{ID: id, BaseCurrency: base}, nil
}
func (t *tripAuthMock) FindRoomByTripID(ctx context.Context, tripID uuid.UUID) (*trip.Room, error) {
	if t.findRoom != nil {
		return t.findRoom(ctx, tripID)
	}
	if t.roomID == uuid.Nil {
		t.roomID = uuid.New()
	}
	return &trip.Room{ID: t.roomID, TripID: tripID}, nil
}
func (t *tripAuthMock) ListMembers(ctx context.Context, roomID uuid.UUID) ([]trip.RoomMember, error) {
	if t.listMembers != nil {
		return t.listMembers(ctx, roomID)
	}
	out := make([]trip.RoomMember, 0, len(t.defaultMembers))
	for _, u := range t.defaultMembers {
		out = append(out, trip.RoomMember{RoomID: roomID, UserID: u})
	}
	return out, nil
}

// ---- Media mock ----

type mediaAuthzMock struct {
	authorize func(context.Context, uuid.UUID, uuid.UUID) (*media.Asset, error)
}

func (m *mediaAuthzMock) AuthorizeAssetForOwner(ctx context.Context, userID, assetID uuid.UUID) (*media.Asset, error) {
	if m.authorize != nil {
		return m.authorize(ctx, userID, assetID)
	}
	return &media.Asset{ID: assetID, OwnerID: userID}, nil
}

// ---- Broadcaster mock ----

type bcastMock struct {
	frames [][]byte
	tripID uuid.UUID
}

func (b *bcastMock) PublishTripEvent(_ context.Context, tripID uuid.UUID, payload []byte) {
	b.tripID = tripID
	b.frames = append(b.frames, payload)
}

// ---- Helpers ----

func newSvc(t *testing.T, repo finance.IRepository, auth finance.TripAuthorizer, mz finance.MediaAuthorizer, bc finance.Broadcaster, rp finance.RateProvider) finance.IService {
	t.Helper()
	return finance.NewService(finance.ServiceConfig{
		Repo:         repo,
		TripAuth:     auth,
		MediaAuthz:   mz,
		Broadcaster:  bc,
		RateProvider: rp,
	})
}

func mustDec(t *testing.T, s string) decimal.Decimal {
	t.Helper()
	d, err := decimal.NewFromString(s)
	require.NoError(t, err)
	return d
}

// ---- Split math ----

func TestCreateExpense_EqualSplit_DividesEvenly(t *testing.T) {
	repo := newRepoMock()
	u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2, u3}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "30.00"),
		Currency:      "USD",
		Description:   "Dinner",
		Category:      "food",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2, u3},
	})
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, "30", dto.Amount.String())
	assert.Equal(t, "30", dto.AmountBase.String())
	require.Len(t, dto.Shares, 3)
	sum := decimal.Zero
	for _, s := range dto.Shares {
		sum = sum.Add(s.Amount)
	}
	assert.True(t, sum.Equal(mustDec(t, "30")), "shares must sum to amount, got %s", sum.String())
}

func TestCreateExpense_EqualSplit_RoundingAbsorbedByLastShare(t *testing.T) {
	repo := newRepoMock()
	u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2, u3}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	// 10 / 3 = 3.3333..., last share absorbs 0.0001 rounding drift
	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "10.00"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2, u3},
	})
	require.NoError(t, err)
	require.Len(t, dto.Shares, 3)
	sum := decimal.Zero
	for _, s := range dto.Shares {
		sum = sum.Add(s.Amount)
	}
	assert.True(t, sum.Equal(mustDec(t, "10")), "shares must reconcile exactly, got %s", sum.String())
}

func TestCreateExpense_CustomSplit_ExactSum(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	a1 := mustDec(t, "70.00")
	a2 := mustDec(t, "30.00")
	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100.00"),
		Currency:      "USD",
		SplitStrategy: finance.SplitCustom,
		Participants:  []uuid.UUID{u1, u2},
		Shares: []finance.ShareInput{
			{UserID: u1, Amount: &a1},
			{UserID: u2, Amount: &a2},
		},
	})
	require.NoError(t, err)
	require.Len(t, dto.Shares, 2)
}

func TestCreateExpense_CustomSplit_MismatchedSum_Rejected(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	a1 := mustDec(t, "50.00")
	a2 := mustDec(t, "40.00")
	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100.00"),
		Currency:      "USD",
		SplitStrategy: finance.SplitCustom,
		Participants:  []uuid.UUID{u1, u2},
		Shares: []finance.ShareInput{
			{UserID: u1, Amount: &a1},
			{UserID: u2, Amount: &a2},
		},
	})
	require.ErrorIs(t, err, finance.ErrShareSumMismatch)
}

func TestCreateExpense_PercentageSplit(t *testing.T) {
	repo := newRepoMock()
	u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2, u3}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	p1 := mustDec(t, "50")
	p2 := mustDec(t, "30")
	p3 := mustDec(t, "20")
	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100.00"),
		Currency:      "USD",
		SplitStrategy: finance.SplitPercentage,
		Participants:  []uuid.UUID{u1, u2, u3},
		Shares: []finance.ShareInput{
			{UserID: u1, Pct: &p1},
			{UserID: u2, Pct: &p2},
			{UserID: u3, Pct: &p3},
		},
	})
	require.NoError(t, err)
	sum := decimal.Zero
	for _, s := range dto.Shares {
		sum = sum.Add(s.Amount)
	}
	assert.True(t, sum.Equal(mustDec(t, "100")), "percentage shares must sum to amount, got %s", sum.String())
}

func TestCreateExpense_PercentageSplit_NotHundred_Rejected(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	p1 := mustDec(t, "40")
	p2 := mustDec(t, "40") // sum = 80, not 100
	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100.00"),
		Currency:      "USD",
		SplitStrategy: finance.SplitPercentage,
		Participants:  []uuid.UUID{u1, u2},
		Shares: []finance.ShareInput{
			{UserID: u1, Pct: &p1},
			{UserID: u2, Pct: &p2},
		},
	})
	require.ErrorIs(t, err, finance.ErrPctSumMismatch)
}

// ---- Authorization ----

func TestCreateExpense_NonMember_Forbidden(t *testing.T) {
	repo := newRepoMock()
	stranger := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{uuid.New()}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), stranger, tripID, finance.CreateExpensePayload{
		PaidBy:        stranger,
		Amount:        mustDec(t, "10"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{stranger},
	})
	require.ErrorIs(t, err, finance.ErrForbidden)
}

func TestCreateExpense_ParticipantNotMember_Rejected(t *testing.T) {
	repo := newRepoMock()
	me := uuid.New()
	outsider := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{me}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), me, tripID, finance.CreateExpensePayload{
		PaidBy:        me,
		Amount:        mustDec(t, "10"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{me, outsider},
	})
	require.ErrorIs(t, err, finance.ErrParticipantMissing)
}

func TestUpdateExpense_NonPayerNonCreator_Forbidden(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	created, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "10"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	newDesc := "hijacked"
	_, err = svc.UpdateExpense(context.Background(), u2, created.ID, finance.UpdateExpensePayload{
		Description: &newDesc,
	})
	require.ErrorIs(t, err, finance.ErrForbidden)
}

// ---- FX ----

func TestCreateExpense_FxSnapshotHit_UsesCachedRate(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}, baseCurrency: "USD"}
	// Pre-seed a JPY->USD rate for today
	today := time.Now().UTC().Truncate(24 * time.Hour)
	repo.fxRates["JPY:USD:"+today.Format("2006-01-02")] = &finance.FxSnapshot{
		Base: "JPY", Quote: "USD", Rate: mustDec(t, "0.007"), AsOf: today, Source: "manual",
	}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "1000"),
		Currency:      "JPY",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)
	assert.Equal(t, "USD", dto.BaseCurrency)
	// 1000 JPY * 0.007 = 7 USD
	assert.True(t, dto.AmountBase.Equal(mustDec(t, "7")), "expected base 7 USD, got %s", dto.AmountBase.String())
}

func TestCreateExpense_NoFxAndNoProvider_ReturnsError(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}, baseCurrency: "USD"}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100"),
		Currency:      "EUR",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1},
	})
	require.Error(t, err)
	assert.True(t, strings.Contains(err.Error(), "fx rate unavailable"), "err = %v", err)
}

func TestCreateExpense_UsesProviderFallbackAndCaches(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}, baseCurrency: "USD"}
	rp := finance.NewStaticRateProvider(map[string]decimal.Decimal{
		"EUR:USD": mustDec(t, "1.10"),
	}, "test-static")
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, rp)

	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100"),
		Currency:      "EUR",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1},
	})
	require.NoError(t, err)
	assert.True(t, dto.AmountBase.Equal(mustDec(t, "110")))
	// Provider hit must have written a snapshot for reuse.
	found, err := repo.FindFxRate(context.Background(), "EUR", "USD", time.Now().UTC())
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "test-static", found.Source)
}

// ---- Delete / Broadcast ----

func TestDeleteExpense_BroadcastsAndRemoves(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	bc := &bcastMock{}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, bc, nil)

	dto, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "20"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)
	require.Len(t, bc.frames, 1)

	require.NoError(t, svc.DeleteExpense(context.Background(), u1, dto.ID))
	// One create + one delete
	assert.Len(t, bc.frames, 2)
	// Deleted expense should be gone from list
	list, err := svc.ListExpenses(context.Background(), u1, tripID)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestBroadcastFrame_HasExpectedShape(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}}
	bc := &bcastMock{}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, bc, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "5"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1},
	})
	require.NoError(t, err)
	require.Len(t, bc.frames, 1)

	var frame struct {
		Type   string          `json:"type"`
		TripID uuid.UUID       `json:"trip_id"`
		Data   json.RawMessage `json:"data"`
	}
	require.NoError(t, json.Unmarshal(bc.frames[0], &frame))
	assert.Equal(t, string(finance.BroadcastExpenseCreated), frame.Type)
	assert.Equal(t, tripID, frame.TripID)
}

// ---- Receipt ----

func TestCreateExpense_ReceiptOwnershipCheckFails_Rejected(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}}
	mz := &mediaAuthzMock{
		authorize: func(_ context.Context, _, _ uuid.UUID) (*media.Asset, error) {
			return nil, media.ErrForbidden
		},
	}
	svc := newSvc(t, repo, auth, mz, &bcastMock{}, nil)

	assetID := uuid.New()
	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:         u1,
		Amount:         mustDec(t, "10"),
		Currency:       "USD",
		SplitStrategy:  finance.SplitEqual,
		Participants:   []uuid.UUID{u1},
		ReceiptAssetID: &assetID,
	})
	require.Error(t, err)
	assert.True(t, errors.Is(err, media.ErrForbidden))
}

// ---- Budgets ----

func TestUpsertBudget_ThenListBudgets_ComputesSpent(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	// Set a food budget of 50
	_, err := svc.UpsertBudget(context.Background(), u1, tripID, finance.UpsertBudgetPayload{
		Category: "food", Amount: mustDec(t, "50"), Currency: "USD",
	})
	require.NoError(t, err)

	// Spend 30 on food
	_, err = svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "30"),
		Currency:      "USD",
		Category:      "food",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	list, err := svc.ListBudgets(context.Background(), u1, tripID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, "food", list[0].Category)
	assert.True(t, list[0].SpentBase.Equal(mustDec(t, "30")))
	assert.True(t, list[0].RemainingBase.Equal(mustDec(t, "20")))
	assert.False(t, list[0].OverBudget)
}

func TestUpsertBudget_OverBudgetFlag(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.UpsertBudget(context.Background(), u1, tripID, finance.UpsertBudgetPayload{
		Category: "food", Amount: mustDec(t, "50"), Currency: "USD",
	})
	require.NoError(t, err)

	_, err = svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "80"),
		Currency:      "USD",
		Category:      "food",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	list, err := svc.ListBudgets(context.Background(), u1, tripID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].OverBudget)
}

func TestDeleteBudget_HappyPath(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	created, err := svc.UpsertBudget(context.Background(), u1, tripID, finance.UpsertBudgetPayload{
		Category: "food", Amount: mustDec(t, "50"), Currency: "USD",
	})
	require.NoError(t, err)
	require.NoError(t, svc.DeleteBudget(context.Background(), u1, tripID, created.ID))

	list, err := svc.ListBudgets(context.Background(), u1, tripID)
	require.NoError(t, err)
	assert.Len(t, list, 0)
}

// ---- Settlement ----

func TestProposeSettlements_Auto_MinCashflow_TwoParty(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	// u1 pays 100, split equally → u1 paid 100 / owed 50 (net +50), u2 paid 0 / owed 50 (net -50)
	// Expected: 1 settlement u2 → u1 for 50
	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "100"),
		Currency:      "USD",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	txfers, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{Auto: true})
	require.NoError(t, err)
	require.Len(t, txfers, 1)
	assert.Equal(t, u2, txfers[0].PayerID)
	assert.Equal(t, u1, txfers[0].PayeeID)
	assert.True(t, txfers[0].Amount.Equal(mustDec(t, "50")))
	assert.Equal(t, finance.SettlementProposed, txfers[0].Status)
}

func TestProposeSettlements_Auto_ReplacesPriorProposals(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy: u1, Amount: mustDec(t, "100"), Currency: "USD",
		SplitStrategy: finance.SplitEqual, Participants: []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	first, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{Auto: true})
	require.NoError(t, err)
	require.Len(t, first, 1)

	second, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{Auto: true})
	require.NoError(t, err)
	require.Len(t, second, 1)

	list, err := svc.ListSettlements(context.Background(), u1, tripID)
	require.NoError(t, err)
	// The first proposal was wiped, only the second survives.
	assert.Len(t, list, 1)
	assert.Equal(t, second[0].ID, list[0].ID)
}

func TestProposeSettlements_Manual_KeepsExistingProposals(t *testing.T) {
	repo := newRepoMock()
	u1, u2, u3 := uuid.New(), uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2, u3}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	// Seed an auto proposal first
	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy: u1, Amount: mustDec(t, "60"), Currency: "USD",
		SplitStrategy: finance.SplitEqual, Participants: []uuid.UUID{u1, u2, u3},
	})
	require.NoError(t, err)
	auto, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{Auto: true})
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(auto), 1)

	// Now add a manual one — auto entries should be preserved
	manual, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{
		Manual: []finance.ManualSettlementIn{
			{PayerID: u2, PayeeID: u3, Amount: mustDec(t, "5"), Note: "coffee"},
		},
	})
	require.NoError(t, err)
	require.Len(t, manual, 1)

	list, err := svc.ListSettlements(context.Background(), u1, tripID)
	require.NoError(t, err)
	assert.Equal(t, len(auto)+1, len(list))
}

func TestConfirmSettlement_PayeeOnly(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy: u1, Amount: mustDec(t, "100"), Currency: "USD",
		SplitStrategy: finance.SplitEqual, Participants: []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)
	txfers, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{Auto: true})
	require.NoError(t, err)
	require.Len(t, txfers, 1)
	sid := txfers[0].ID

	// The payer trying to confirm — payee-only rule kicks in.
	_, err = svc.ConfirmSettlement(context.Background(), u2, sid, finance.ConfirmSettlementPayload{})
	require.ErrorIs(t, err, finance.ErrForbidden)

	// The payee confirms — should succeed and flip status.
	dto, err := svc.ConfirmSettlement(context.Background(), u1, sid, finance.ConfirmSettlementPayload{Note: "received"})
	require.NoError(t, err)
	assert.Equal(t, finance.SettlementConfirmed, dto.Status)
	require.NotNil(t, dto.ConfirmedAt)
}

func TestConfirmSettlement_AlreadyConfirmed_Rejected(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy: u1, Amount: mustDec(t, "20"), Currency: "USD",
		SplitStrategy: finance.SplitEqual, Participants: []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)
	txfers, err := svc.ProposeSettlements(context.Background(), u1, tripID, finance.ProposeSettlementPayload{Auto: true})
	require.NoError(t, err)
	sid := txfers[0].ID
	// Confirm as payee (u1) first — u2 owes u1.
	_, err = svc.ConfirmSettlement(context.Background(), u1, sid, finance.ConfirmSettlementPayload{})
	require.NoError(t, err)
	// Confirm again — should be rejected as invalid state transition.
	_, err = svc.ConfirmSettlement(context.Background(), u1, sid, finance.ConfirmSettlementPayload{})
	require.Error(t, err)
}

// ---- Personal stats + balances ----

func TestGetTripBalances_UnionsPaidAndOwedUsers(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy: u1, Amount: mustDec(t, "100"), Currency: "USD",
		SplitStrategy: finance.SplitEqual, Participants: []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	balances, err := svc.GetTripBalances(context.Background(), u1, tripID)
	require.NoError(t, err)
	require.Len(t, balances, 2)
	// Find u1 in results
	byUser := map[uuid.UUID]finance.TripBalance{}
	for _, b := range balances {
		byUser[b.UserID] = b
	}
	require.Contains(t, byUser, u1)
	require.Contains(t, byUser, u2)
	assert.True(t, byUser[u1].Net.Equal(mustDec(t, "50")))
	assert.True(t, byUser[u2].Net.Equal(mustDec(t, "-50")))
}

// ---- Export ----

func TestExportCSV_HeaderAndRowShape(t *testing.T) {
	repo := newRepoMock()
	u1, u2 := uuid.New(), uuid.New()
	tripID := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1, u2}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.CreateExpense(context.Background(), u1, tripID, finance.CreateExpensePayload{
		PaidBy:        u1,
		Amount:        mustDec(t, "42.50"),
		Currency:      "USD",
		Description:   "Museum",
		Category:      "activities",
		SplitStrategy: finance.SplitEqual,
		Participants:  []uuid.UUID{u1, u2},
	})
	require.NoError(t, err)

	blob, err := svc.ExportCSV(context.Background(), u1, tripID, nil, nil)
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(blob)), "\n")
	require.Len(t, lines, 2, "expected header + 1 row, got %d lines", len(lines))
	assert.True(t, strings.HasPrefix(lines[0], "occurred_at,description,category"))
	assert.Contains(t, lines[1], "Museum")
	assert.Contains(t, lines[1], "activities")
	assert.Contains(t, lines[1], "42.5")
	// Participants column encodes shares as user:amount|user:amount
	assert.Contains(t, lines[1], "|")
}

// ---- FX admin ----

func TestUpsertFxRate_ThenList(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	dto, err := svc.UpsertFxRate(context.Background(), u1, finance.UpsertFxRatePayload{
		Base: "EUR", Quote: "USD", Rate: mustDec(t, "1.05"),
	})
	require.NoError(t, err)
	assert.Equal(t, "EUR", dto.Base)
	assert.Equal(t, "USD", dto.Quote)

	list, err := svc.ListFxRates(context.Background(), u1, "EUR")
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.True(t, list[0].Rate.Equal(mustDec(t, "1.05")))
}

func TestUpsertFxRate_SameCurrency_Rejected(t *testing.T) {
	repo := newRepoMock()
	u1 := uuid.New()
	auth := &tripAuthMock{defaultMembers: []uuid.UUID{u1}}
	svc := newSvc(t, repo, auth, &mediaAuthzMock{}, &bcastMock{}, nil)

	_, err := svc.UpsertFxRate(context.Background(), u1, finance.UpsertFxRatePayload{
		Base: "USD", Quote: "USD", Rate: mustDec(t, "1"),
	})
	require.ErrorIs(t, err, finance.ErrInvalidPayload)
}

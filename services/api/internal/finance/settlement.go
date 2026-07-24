package finance

import (
	"container/heap"
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (s *service) ProposeSettlements(ctx context.Context, userID, tripID uuid.UUID, p ProposeSettlementPayload) ([]SettlementDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	trip, err := s.auth.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	baseCurrency := normalizeCurrency(trip.BaseCurrency)

	var transfers []Settlement
	if p.Auto {
		balances, err := s.computeNetBalances(ctx, tripID)
		if err != nil {
			return nil, err
		}
		transfers = minCashflowReduce(balances, baseCurrency, userID, tripID)
	} else {
		for _, m := range p.Manual {
			if m.PayerID == m.PayeeID {
				return nil, ErrInvalidPayload
			}
			if m.Amount.LessThanOrEqual(decimal.Zero) {
				return nil, ErrInvalidPayload
			}
			if err := s.assertMemberPair(ctx, tripID, m.PayerID, m.PayeeID); err != nil {
				return nil, err
			}
			transfers = append(transfers, Settlement{
				TripID:    tripID,
				PayerID:   m.PayerID,
				PayeeID:   m.PayeeID,
				Amount:    m.Amount.Round(moneyScale),
				Currency:  baseCurrency,
				Status:    SettlementProposed,
				Note:      truncate(m.Note, maxNoteLen),
				CreatedBy: userID,
			})
		}
	}

	out := make([]SettlementDTO, 0, len(transfers))
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		if p.Auto {
			// Only auto-regenerated proposals should wipe the previous set.
			// A manual add appends to whatever's already there.
			if err := tx.DeleteProposedSettlements(ctx, tripID); err != nil {
				return err
			}
		}
		for i := range transfers {
			if err := tx.CreateSettlement(ctx, &transfers[i]); err != nil {
				return err
			}
			out = append(out, settlementToDTO(&transfers[i]))
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	for _, dto := range out {
		s.broadcast(ctx, tripID, BroadcastSettlementAdded, dto)
	}
	return out, nil
}

func (s *service) ConfirmSettlement(ctx context.Context, userID, settlementID uuid.UUID, p ConfirmSettlementPayload) (*SettlementDTO, error) {
	st, err := s.repo.FindSettlement(ctx, settlementID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, st.TripID, userID); err != nil {
		return nil, err
	}
	// Only the payee confirms receipt — the payer "sending" isn't proof
	// of anything; the money-received signal is the payee's.
	if st.PayeeID != userID {
		return nil, ErrForbidden
	}
	if st.Status != SettlementProposed {
		return nil, ErrInvalidPayload
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateSettlementStatus(ctx, settlementID, SettlementConfirmed, now); err != nil {
		return nil, err
	}
	st.Status = SettlementConfirmed
	st.ConfirmedAt = &now
	if p.Note != "" {
		st.Note = truncate(p.Note, maxNoteLen)
	}
	dto := settlementToDTO(st)
	s.broadcast(ctx, st.TripID, BroadcastSettlementDone, dto)
	return &dto, nil
}

func (s *service) CancelSettlement(ctx context.Context, userID, settlementID uuid.UUID) (*SettlementDTO, error) {
	st, err := s.repo.FindSettlement(ctx, settlementID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, st.TripID, userID); err != nil {
		return nil, err
	}
	// Either party in the transfer, or the person who proposed it, may cancel.
	if st.PayerID != userID && st.PayeeID != userID && st.CreatedBy != userID {
		return nil, ErrForbidden
	}
	if st.Status != SettlementProposed {
		return nil, ErrInvalidPayload
	}
	now := time.Now().UTC()
	if err := s.repo.UpdateSettlementStatus(ctx, settlementID, SettlementCancelled, now); err != nil {
		return nil, err
	}
	st.Status = SettlementCancelled
	st.CancelledAt = &now
	dto := settlementToDTO(st)
	return &dto, nil
}

func (s *service) ListSettlements(ctx context.Context, userID, tripID uuid.UUID) ([]SettlementDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListSettlements(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]SettlementDTO, 0, len(rows))
	for i := range rows {
		out = append(out, settlementToDTO(&rows[i]))
	}
	return out, nil
}

// assertMemberPair short-circuits when either party is not a room member.
func (s *service) assertMemberPair(ctx context.Context, tripID, a, b uuid.UUID) error {
	room, err := s.auth.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return err
	}
	members, err := s.auth.ListMembers(ctx, room.ID)
	if err != nil {
		return err
	}
	set := make(map[uuid.UUID]struct{}, len(members))
	for _, m := range members {
		set[m.UserID] = struct{}{}
	}
	if _, ok := set[a]; !ok {
		return ErrParticipantMissing
	}
	if _, ok := set[b]; !ok {
		return ErrParticipantMissing
	}
	return nil
}

func (s *service) computeNetBalances(ctx context.Context, tripID uuid.UUID) (map[uuid.UUID]decimal.Decimal, error) {
	paid, err := s.repo.SumPaidByUser(ctx, tripID)
	if err != nil {
		return nil, err
	}
	owed, err := s.repo.SumOwedByUser(ctx, tripID)
	if err != nil {
		return nil, err
	}
	settlements, err := s.repo.ListSettlements(ctx, tripID)
	if err != nil {
		return nil, err
	}

	net := make(map[uuid.UUID]decimal.Decimal, len(paid)+len(owed))
	for u, v := range paid {
		net[u] = net[u].Add(v)
	}
	for u, v := range owed {
		net[u] = net[u].Sub(v)
	}
	// Confirmed settlements reduce the payer's debt and shrink the payee's
	// credit — apply them as a virtual "extra paid" for the payer and an
	// "extra owed" for the payee. Cancelled/proposed don't count.
	for _, st := range settlements {
		if st.Status != SettlementConfirmed {
			continue
		}
		net[st.PayerID] = net[st.PayerID].Add(st.Amount)
		net[st.PayeeID] = net[st.PayeeID].Sub(st.Amount)
	}
	// Prune zero balances so the reducer doesn't iterate over noise.
	for u, v := range net {
		if v.IsZero() {
			delete(net, u)
		}
	}
	return net, nil
}

func minCashflowReduce(balances map[uuid.UUID]decimal.Decimal, currency string, createdBy, tripID uuid.UUID) []Settlement {
	dust := decimal.New(1, -moneyScale) // 0.0001
	creditors := &partyHeap{max: true}
	debtors := &partyHeap{max: false}
	heap.Init(creditors)
	heap.Init(debtors)

	// Sort user IDs so heap tie-breaks are deterministic — critical for tests.
	ordered := make([]uuid.UUID, 0, len(balances))
	for u := range balances {
		ordered = append(ordered, u)
	}
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].String() < ordered[j].String() })
	for _, u := range ordered {
		v := balances[u].Round(moneyScale)
		if v.Abs().LessThan(dust) {
			continue
		}
		if v.GreaterThan(decimal.Zero) {
			heap.Push(creditors, party{userID: u, amount: v})
		} else {
			heap.Push(debtors, party{userID: u, amount: v.Neg()})
		}
	}

	var out []Settlement
	for creditors.Len() > 0 && debtors.Len() > 0 {
		cr := heap.Pop(creditors).(party)
		de := heap.Pop(debtors).(party)
		transfer := decimal.Min(cr.amount, de.amount).Round(moneyScale)
		if transfer.LessThan(dust) {
			break
		}
		out = append(out, Settlement{
			TripID:    tripID,
			PayerID:   de.userID,
			PayeeID:   cr.userID,
			Amount:    transfer,
			Currency:  currency,
			Status:    SettlementProposed,
			CreatedBy: createdBy,
		})
		crResid := cr.amount.Sub(transfer)
		deResid := de.amount.Sub(transfer)
		if crResid.GreaterThanOrEqual(dust) {
			heap.Push(creditors, party{userID: cr.userID, amount: crResid})
		}
		if deResid.GreaterThanOrEqual(dust) {
			heap.Push(debtors, party{userID: de.userID, amount: deResid})
		}
	}
	return out
}

// party is a heap element: (user, |balance|). Two heaps are used — one max,
// one min — flipped via the `max` field.
type party struct {
	userID uuid.UUID
	amount decimal.Decimal
}

type partyHeap struct {
	items []party
	max   bool
}

func (h *partyHeap) Len() int { return len(h.items) }
func (h *partyHeap) Less(i, j int) bool {
	if h.max || !h.max {
		return h.items[i].amount.GreaterThan(h.items[j].amount)
	}
	return h.items[i].amount.LessThan(h.items[j].amount)
}
func (h *partyHeap) Swap(i, j int) { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *partyHeap) Push(x any)    { h.items = append(h.items, x.(party)) }
func (h *partyHeap) Pop() any {
	old := h.items
	n := len(old)
	x := old[n-1]
	h.items = old[:n-1]
	return x
}

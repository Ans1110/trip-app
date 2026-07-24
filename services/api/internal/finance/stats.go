package finance

import (
	"context"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (s *service) GetTripBalances(ctx context.Context, userID, tripID uuid.UUID) ([]TripBalance, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	paid, err := s.repo.SumPaidByUser(ctx, tripID)
	if err != nil {
		return nil, err
	}
	owed, err := s.repo.SumOwedByUser(ctx, tripID)
	if err != nil {
		return nil, err
	}
	// Union the user sets so a member who only paid (or only owes) still
	// appears with a zero on the other side.
	users := make(map[uuid.UUID]struct{}, len(paid)+len(owed))
	for u := range paid {
		users[u] = struct{}{}
	}
	for u := range owed {
		users[u] = struct{}{}
	}
	out := make([]TripBalance, 0, len(users))
	for u := range users {
		p := paid[u]
		o := owed[u]
		out = append(out, TripBalance{
			UserID: u,
			Paid:   p,
			Owed:   o,
			Net:    p.Sub(o),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].UserID.String() < out[j].UserID.String()
	})
	return out, nil
}

func (s *service) GetPersonalStats(ctx context.Context, userID, tripID uuid.UUID) (*PersonalStats, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	trip, err := s.auth.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	paid, err := s.repo.SumPaidByUser(ctx, tripID)
	if err != nil {
		return nil, err
	}
	owed, err := s.repo.SumOwedByUser(ctx, tripID)
	if err != nil {
		return nil, err
	}
	byCat, err := s.repo.SumUserPaidByCategory(ctx, tripID, userID)
	if err != nil {
		return nil, err
	}

	totalPaid := paid[userID]
	totalOwed := owed[userID]

	cats := make([]CategoryStat, 0, len(byCat))
	for cat, agg := range byCat {
		cats = append(cats, CategoryStat{
			Category:   cat,
			AmountBase: agg.AmountBase,
			Count:      agg.Count,
		})
	}
	sort.Slice(cats, func(i, j int) bool {
		return cats[i].AmountBase.GreaterThan(cats[j].AmountBase)
	})

	return &PersonalStats{
		TripID:       tripID,
		UserID:       userID,
		BaseCurrency: normalizeCurrency(trip.BaseCurrency),
		TotalPaid:    totalPaid,
		TotalOwed:    totalOwed,
		NetBalance:   totalPaid.Sub(totalOwed),
		ByCategory:   cats,
	}, nil
}

// zero is a convenience for cases where we want to guarantee a non-nil
// decimal even when the map lookup missed.
func zero() decimal.Decimal { return decimal.Zero }

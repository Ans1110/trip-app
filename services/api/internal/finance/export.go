package finance

import (
	"bytes"
	"context"
	"encoding/csv"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ExportCSV renders the trip's expenses (optionally date-bounded) into a
// CSV blob.
func (s *service) ExportCSV(ctx context.Context, userID, tripID uuid.UUID, from, to *time.Time) ([]byte, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	trip, err := s.auth.FindTripByID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	rows, err := s.repo.ListExpensesForExport(ctx, tripID, from, to)
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(rows))
	for i := range rows {
		ids = append(ids, rows[i].ID)
	}
	sharesByExpense, err := s.repo.ListSharesByExpenses(ctx, ids)
	if err != nil {
		return nil, err
	}

	buf := &bytes.Buffer{}
	w := csv.NewWriter(buf)
	_ = w.Write([]string{
		"occurred_at",
		"description",
		"category",
		"paid_by",
		"amount",
		"currency",
		"amount_base",
		"base_currency",
		"rate_to_base",
		"split_strategy",
		"participants",
		"receipt_asset_id",
	})
	base := normalizeCurrency(trip.BaseCurrency)
	for i := range rows {
		e := rows[i]
		participants := participantsCSV(sharesByExpense[e.ID])
		rate := ""
		if e.RateToBase.Valid {
			rate = e.RateToBase.Decimal.String()
		}
		receipt := ""
		if e.ReceiptAssetID != nil {
			receipt = e.ReceiptAssetID.String()
		}
		_ = w.Write([]string{
			e.OccurredAt.UTC().Format(time.RFC3339),
			e.Description,
			e.Category,
			e.PaidBy.String(),
			e.Amount.String(),
			e.Currency,
			e.AmountBase.String(),
			base,
			rate,
			string(e.SplitStrategy),
			participants,
			receipt,
		})
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// participantsCSV renders shares as "uuid:amount|uuid:amount" so each row
// keeps its full split visible without exploding into N rows per expense.
func participantsCSV(shares []ExpenseShare) string {
	parts := make([]string, 0, len(shares))
	for _, s := range shares {
		parts = append(parts, s.UserID.String()+":"+s.Amount.String())
	}
	return strings.Join(parts, "|")
}

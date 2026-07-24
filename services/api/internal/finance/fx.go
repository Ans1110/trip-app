package finance

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

// RateProvider is the escape hatch for pulling FX rates from something other
// than the local snapshot table. Production wires an HTTP fetcher; tests wire
// StaticRateProvider. The service falls back to this only on cache miss.
type RateProvider interface {
	GetRate(ctx context.Context, base, quote string, asOf time.Time) (decimal.Decimal, string, error)
}

var ErrRateUnavailable = errors.New("finance: fx rate unavailable")

// StaticRateProvider serves rates from a map. Missing entries return
// ErrRateUnavailable, forcing the service to require a manual snapshot.
type StaticRateProvider struct {
	rates map[string]decimal.Decimal // key = "BASE:QUOTE"
	tag   string
}

func NewStaticRateProvider(rates map[string]decimal.Decimal, tag string) *StaticRateProvider {
	m := make(map[string]decimal.Decimal, len(rates))
	for k, v := range rates {
		m[strings.ToUpper(k)] = v
	}
	if tag == "" {
		tag = "static"
	}
	return &StaticRateProvider{rates: m, tag: tag}
}

func (p *StaticRateProvider) GetRate(_ context.Context, base, quote string, _ time.Time) (decimal.Decimal, string, error) {
	base = normalizeCurrency(base)
	quote = normalizeCurrency(quote)

	if base == quote {
		return decimal.NewFromInt(1), p.tag, nil
	}

	if r, ok := p.rates[base+":"+quote]; ok {
		return r, p.tag, nil
	}

	if r, ok := p.rates[quote+":"+base]; ok && !r.IsZero() {
		return decimal.NewFromInt(1).Div(r), p.tag, nil
	}

	return decimal.Zero, "", ErrRateUnavailable
}

// truncateToDate strips the clock so snapshot lookups collapse onto calendar
// days. Called before both cache lookups and snapshot writes so cache-hit
// keys line up with cache-write keys.
func truncateToDate(t time.Time) time.Time {
	y, m, d := t.UTC().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.UTC)
}

func normalizeCurrency(c string) string {
	return strings.ToUpper(strings.TrimSpace(c))
}

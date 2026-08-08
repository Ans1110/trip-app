package admin

import (
	"context"
	"crypto/subtle"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrElevationDenied      = errors.New("admin credentials invalid")
	ErrElevationUnavailable = errors.New("admin elevation not configured")
	ErrNotElevated          = errors.New("admin elevation required")
)

const (
	elevationKeyPrefix = "admin:elev:"
	elevationTTL       = 30 * time.Minute
)

// ElevationGate carries the admin config + Redis handle for the elevation
// endpoints and middleware. Kept off the main service so read-only admin
// routes can bind the middleware without pulling in the whole service.
type ElevationGate struct {
	rdb           *redis.Client
	adminEmail    string
	adminPassword string
}

func NewElevationGate(rdb *redis.Client, adminEmail, adminPassword string) *ElevationGate {
	return &ElevationGate{
		rdb:           rdb,
		adminEmail:    strings.ToLower(strings.TrimSpace(adminEmail)),
		adminPassword: adminPassword,
	}
}

func (g *ElevationGate) Configured() bool {
	return g != nil && g.adminEmail != "" && g.adminPassword != ""
}

func (g *ElevationGate) IsAdminEmail(email string) bool {
	if !g.Configured() {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(email), g.adminEmail)
}

func (g *ElevationGate) Elevate(ctx context.Context, userID uuid.UUID, email, password string) (time.Time, error) {
	if !g.Configured() {
		return time.Time{}, ErrElevationUnavailable
	}
	if !g.IsAdminEmail(email) {
		return time.Time{}, ErrElevationDenied
	}
	if subtle.ConstantTimeCompare([]byte(password), []byte(g.adminPassword)) != 1 {
		return time.Time{}, ErrElevationDenied
	}
	expires := time.Now().Add(elevationTTL)
	if err := g.rdb.Set(ctx, elevationKey(userID), "1", elevationTTL).Err(); err != nil {
		return time.Time{}, err
	}
	return expires, nil
}

func (g *ElevationGate) Revoke(ctx context.Context, userID uuid.UUID) error {
	if g == nil || g.rdb == nil {
		return nil
	}
	return g.rdb.Del(ctx, elevationKey(userID)).Err()
}

func (g *ElevationGate) IsElevated(ctx context.Context, userID uuid.UUID) (bool, error) {
	if g == nil || g.rdb == nil {
		return false, nil
	}
	n, err := g.rdb.Exists(ctx, elevationKey(userID)).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func elevationKey(userID uuid.UUID) string {
	return fmt.Sprintf("%s%s", elevationKeyPrefix, userID.String())
}

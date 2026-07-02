package ticket

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

var (
	ErrNotFound = errors.New("ticket: not found or already used")
	ErrInvalid  = errors.New("ticket: invalid token")
)

type Payload struct {
	UserID   uuid.UUID `json:"user_id"`
	TripID   uuid.UUID `json:"trip_id"`
	IssuedAt time.Time `json:"issued_at"`
}

type Store struct {
	rdb    *redis.Client
	prefix string
	ttl    time.Duration
}

type Config struct {
	Redis  *redis.Client
	Prefix string
	TTL    time.Duration
}

func NewStore(cfg Config) *Store {
	prefix := cfg.Prefix
	if prefix == "" {
		prefix = "ticket:"
	}
	ttl := cfg.TTL
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &Store{rdb: cfg.Redis, prefix: prefix, ttl: ttl}
}

func (s *Store) TTL() time.Duration { return s.ttl }

func (s *Store) Issue(ctx context.Context, payload Payload) (string, error) {
	token, err := randomToken()
	if err != nil {
		return "", fmt.Errorf("ticket: random: %w", err)
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("ticket: marshal: %w", err)
	}
	if err := s.rdb.Set(ctx, s.prefix+token, raw, s.ttl).Err(); err != nil {
		return "", fmt.Errorf("ticket: redis set: %w", err)
	}
	return token, nil
}

func (s *Store) Consume(ctx context.Context, token string) (Payload, error) {
	if token == "" {
		return Payload{}, ErrInvalid
	}
	raw, err := s.rdb.GetDel(ctx, s.prefix+token).Bytes()
	if err != nil {
		if errors.Is(err, redis.Nil) {
			return Payload{}, ErrNotFound
		}
		return Payload{}, fmt.Errorf("ticket: redis getdel: %w", err)
	}
	var p Payload
	if err := json.Unmarshal(raw, &p); err != nil {
		return Payload{}, fmt.Errorf("ticket: unmarshal: %w", err)
	}

	if p.UserID == uuid.Nil || p.TripID == uuid.Nil {
		return Payload{}, ErrInvalid
	}
	return p, nil
}

func randomToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

package feed

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Ranking constants. score = recencyScale * exp(-hours/decayHours) + follow_boost.
// Half-life is decayHours * ln(2) ≈ 33h at 48. Tune in one place.
const (
	recencyScale = 100.0
	decayHours   = 48.0
	followBoost  = 25.0
)

type Cursor struct {
	RefTime     time.Time `json:"r"`
	Score       float64   `json:"s"`
	PublishedAt time.Time `json:"p"`
	PostID      uuid.UUID `json:"i"`
}

type RankedPost struct {
	ID          uuid.UUID `gorm:"column:id"`
	PublishedAt time.Time `gorm:"column:published_at"`
	Score       float64   `gorm:"column:score"`
}

type IRepository interface {
	ListRanked(ctx context.Context, viewerID uuid.UUID, cursor *Cursor, limit int) (rows []RankedPost, refTime time.Time, err error)
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository { return &repository{db: db} }

var baseSQL = fmt.Sprintf(`
WITH ref AS (SELECT ?::timestamptz AS t, ?::uuid AS viewer),
ranked AS (
    SELECT p.id, p.published_at,
           (%.1f * exp(-EXTRACT(EPOCH FROM (r.t - p.published_at)) / (3600.0 * %.1f))
            + CASE WHEN EXISTS (
                SELECT 1 FROM profile.follows f
                WHERE f.follower_id = r.viewer
                  AND f.followee_id = p.author_id
              ) THEN %.1f ELSE 0.0 END) AS score
    FROM post.posts p
    CROSS JOIN ref r
    WHERE p.deleted_at IS NULL
      AND p.published_at <= r.t
)
SELECT id, published_at, score FROM ranked`, recencyScale, decayHours, followBoost)

func (r *repository) ListRanked(ctx context.Context, viewerID uuid.UUID, cursor *Cursor, limit int) ([]RankedPost, time.Time, error) {
	if limit <= 0 || limit > 50 {
		limit = 20
	}
	refTime := time.Now().UTC()
	if cursor != nil && !cursor.RefTime.IsZero() {
		refTime = cursor.RefTime.UTC()
	}

	var sb strings.Builder
	sb.WriteString(baseSQL)
	args := []any{refTime, viewerID}

	if cursor != nil {
		sb.WriteString(` WHERE (score, published_at, id) < (?::float8, ?::timestamptz, ?::uuid)`)
		args = append(args, cursor.Score, cursor.PublishedAt.UTC(), cursor.PostID)
	}
	sb.WriteString(` ORDER BY score DESC, published_at DESC, id DESC LIMIT ?`)
	args = append(args, limit)

	var rows []RankedPost
	if err := r.db.WithContext(ctx).Raw(sb.String(), args...).Scan(&rows).Error; err != nil {
		return nil, refTime, err
	}
	return rows, refTime, nil
}

func encodeCursor(c Cursor) string {
	b, _ := json.Marshal(c)
	return base64.RawURLEncoding.EncodeToString(b)
}

func DecodeCursor(token string) (*Cursor, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	raw, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		return nil, errors.New("invalid cursor")
	}
	var c Cursor
	if err := json.Unmarshal(raw, &c); err != nil {
		return nil, errors.New("invalid cursor")
	}
	if c.RefTime.IsZero() || c.PostID == uuid.Nil {
		return nil, errors.New("invalid cursor")
	}
	return &c, nil
}

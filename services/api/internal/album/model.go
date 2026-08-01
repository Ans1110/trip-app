package album

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Photo struct {
	ID            uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID        uuid.UUID  `gorm:"type:uuid;column:trip_id;not null"`
	MediaID       uuid.UUID  `gorm:"type:uuid;column:media_id;not null"`
	ThumbSmallID  *uuid.UUID `gorm:"type:uuid;column:thumb_small_id"`
	ThumbMediumID *uuid.UUID `gorm:"type:uuid;column:thumb_medium_id"`
	AddedBy       uuid.UUID  `gorm:"type:uuid;column:added_by;not null"`
	// TakenAt is EXIF DateTimeOriginal when present, else the upload timestamp.
	TakenAt   time.Time        `gorm:"column:taken_at;not null"`
	Latitude  *decimal.Decimal `gorm:"column:latitude;type:numeric(9,6)"`
	Longitude *decimal.Decimal `gorm:"column:longitude;type:numeric(9,6)"`
	Caption   string           `gorm:"column:caption;not null;default:''"`
	CreatedAt time.Time        `gorm:"column:created_at"`
	DeletedAt *time.Time       `gorm:"column:deleted_at"`
}

func (Photo) TableName() string { return "album.album_photo" }

// ShareToken never stores the plaintext token. TokenHash is a 32-byte SHA-256
// of the plaintext; the plaintext is returned to the creator exactly once.
type ShareToken struct {
	ID             uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	TripID         uuid.UUID  `gorm:"type:uuid;column:trip_id;not null"`
	TokenHash      []byte     `gorm:"column:token_hash;type:bytea;not null"`
	CreatedBy      uuid.UUID  `gorm:"type:uuid;column:created_by;not null"`
	ExpiresAt      *time.Time `gorm:"column:expires_at"`
	RevokedAt      *time.Time `gorm:"column:revoked_at"`
	RevokedBy      *uuid.UUID `gorm:"type:uuid;column:revoked_by"`
	LastAccessedAt *time.Time `gorm:"column:last_accessed_at"`
	CreatedAt      time.Time  `gorm:"column:created_at"`
}

func (ShareToken) TableName() string { return "album.share_token" }

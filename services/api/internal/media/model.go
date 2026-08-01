package media

import (
	"time"

	"github.com/google/uuid"
)

// Purpose scopes an upload to a callsite. The service uses it to enforce
// per-purpose size + mime rules and to shape the object key prefix.
type Purpose string

const (
	PurposeChat       Purpose = "chat"
	PurposeAvatar     Purpose = "avatar"
	PurposeCover      Purpose = "cover"
	PurposeTodo       Purpose = "todo"
	PurposeAlbum      Purpose = "album"
	PurposeAlbumThumb Purpose = "album_thumb"
)

type PurposeRule struct {
	MaxBytes    int64
	AllowedMime map[string]struct{}
	KeyPrefix   string // object key prefix, e.g. "chat/"
}

func defaultRules() map[Purpose]PurposeRule {
	imageMimes := map[string]struct{}{
		"image/jpeg": {},
		"image/png":  {},
		"image/webp": {},
		"image/gif":  {},
	}

	fileMimes := map[string]struct{}{
		"image/jpeg":      {},
		"image/png":       {},
		"image/webp":      {},
		"image/gif":       {},
		"video/mp4":       {},
		"video/webm":      {},
		"video/quicktime": {},
		"application/pdf": {},
		"text/plain":      {},
		"application/zip": {},
	}
	return map[Purpose]PurposeRule{
		PurposeAvatar: {MaxBytes: 5 << 20, AllowedMime: imageMimes, KeyPrefix: "avatar/"},
		PurposeCover:  {MaxBytes: 10 << 20, AllowedMime: imageMimes, KeyPrefix: "cover/"},
		// 100MB covers short clips (~1min 1080p) without exposing the bucket to
		// abuse from arbitrarily large uploads.
		PurposeChat: {MaxBytes: 100 << 20, AllowedMime: fileMimes, KeyPrefix: "chat/"},
		PurposeTodo: {MaxBytes: 100 << 20, AllowedMime: fileMimes, KeyPrefix: "todo/"},

		PurposeAlbum:      {MaxBytes: 50 << 20, AllowedMime: imageMimes, KeyPrefix: "album/"},
		PurposeAlbumThumb: {MaxBytes: 2 << 20, AllowedMime: imageMimes, KeyPrefix: "album_thumb/"},
	}
}

type Asset struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID   uuid.UUID  `gorm:"type:uuid;column:owner_id;not null;index"`
	Purpose   Purpose    `gorm:"column:purpose;not null"`
	Bucket    string     `gorm:"column:bucket;not null"`
	ObjectKey string     `gorm:"column:object_key;not null"`
	Mime      string     `gorm:"column:mime;not null"`
	Bytes     int64      `gorm:"column:bytes;not null"`
	Width     *int       `gorm:"column:width"`
	Height    *int       `gorm:"column:height"`
	ETag      string     `gorm:"column:etag;not null"`
	SHA256    []byte     `gorm:"column:sha256;type:bytea"`
	CreatedAt time.Time  `gorm:"column:created_at"`
	DeletedAt *time.Time `gorm:"column:deleted_at"`
}

func (Asset) TableName() string { return "media.assets" }

type UploadSession struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	OwnerID          uuid.UUID  `gorm:"type:uuid;column:owner_id;not null"`
	Purpose          Purpose    `gorm:"column:purpose;not null"`
	Bucket           string     `gorm:"column:bucket;not null"`
	ObjectKey        string     `gorm:"column:object_key;not null"`
	ExpectedMime     string     `gorm:"column:expected_mime;not null"`
	ExpectedMaxBytes int64      `gorm:"column:expected_max_bytes;not null"`
	CreatedAt        time.Time  `gorm:"column:created_at"`
	ExpiresAt        time.Time  `gorm:"column:expires_at;not null"`
	CompletedAt      *time.Time `gorm:"column:completed_at"`
	MediaID          *uuid.UUID `gorm:"type:uuid;column:media_id"`
}

func (UploadSession) TableName() string { return "media.upload_sessions" }

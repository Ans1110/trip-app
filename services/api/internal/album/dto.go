package album

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// ---- Requests ----

type InitUploadItem struct {
	Mime  string `json:"mime" binding:"required"`
	Bytes int64  `json:"bytes" binding:"required"`
}

type InitUploadPayload struct {
	Items []InitUploadItem `json:"items" binding:"required,min=1,max=50"`
}

type InitUploadSlotDTO struct {
	UploadID  uuid.UUID         `json:"upload_id"`
	UploadURL string            `json:"upload_url"`
	Method    string            `json:"method"`
	Headers   map[string]string `json:"headers"`
	ExpiresAt time.Time         `json:"expires_at"`
}

type InitUploadResponse struct {
	Slots []InitUploadSlotDTO `json:"slots"`
}

type CompletePhotoPayload struct {
	UploadID uuid.UUID `json:"upload_id" binding:"required"`
	Caption  string    `json:"caption"`
}

type UpdatePhotoPayload struct {
	Caption *string `json:"caption,omitempty"`
}

type CreateShareTokenPayload struct {
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// ---- Responses ----

type PhotoDTO struct {
	ID            uuid.UUID        `json:"id"`
	TripID        uuid.UUID        `json:"trip_id"`
	MediaID       uuid.UUID        `json:"media_id"`
	ThumbSmallID  *uuid.UUID       `json:"thumb_small_id,omitempty"`
	ThumbMediumID *uuid.UUID       `json:"thumb_medium_id,omitempty"`
	AddedBy       uuid.UUID        `json:"added_by"`
	TakenAt       time.Time        `json:"taken_at"`
	Latitude      *decimal.Decimal `json:"latitude,omitempty"`
	Longitude     *decimal.Decimal `json:"longitude,omitempty"`
	Caption       string           `json:"caption"`
	CreatedAt     time.Time        `json:"created_at"`
}

// PhotoWithURLsDTO carries presigned URLs for the album grid — original +
// thumbnails. Consumers should treat URLs as short-lived (TTL from media cfg).
type PhotoWithURLsDTO struct {
	PhotoDTO
	OriginalURL    string `json:"original_url"`
	ThumbSmallURL  string `json:"thumb_small_url,omitempty"`
	ThumbMediumURL string `json:"thumb_medium_url,omitempty"`
}

type ShareTokenDTO struct {
	ID             uuid.UUID  `json:"id"`
	TripID         uuid.UUID  `json:"trip_id"`
	CreatedBy      uuid.UUID  `json:"created_by"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
	RevokedAt      *time.Time `json:"revoked_at,omitempty"`
	RevokedBy      *uuid.UUID `json:"revoked_by,omitempty"`
	LastAccessedAt *time.Time `json:"last_accessed_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

// CreateShareTokenResponse returns the plaintext token exactly once — it is
// never persisted, only its SHA-256 hash.
type CreateShareTokenResponse struct {
	Token string        `json:"token"`
	Share ShareTokenDTO `json:"share"`
}

type DownloadOriginalResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

// ---- Broadcast frame ----

type BroadcastKind string

const (
	BroadcastPhotoUploaded BroadcastKind = "ALBUM_PHOTO_UPLOADED"
	BroadcastPhotoDeleted  BroadcastKind = "ALBUM_PHOTO_DELETED"
	BroadcastPhotoUpdated  BroadcastKind = "ALBUM_PHOTO_UPDATED"
	BroadcastShareCreated  BroadcastKind = "ALBUM_SHARE_CREATED"
	BroadcastShareRevoked  BroadcastKind = "ALBUM_SHARE_REVOKED"
)

// ---- Conversions ----

func photoToDTO(p *Photo) PhotoDTO {
	return PhotoDTO{
		ID:            p.ID,
		TripID:        p.TripID,
		MediaID:       p.MediaID,
		ThumbSmallID:  p.ThumbSmallID,
		ThumbMediumID: p.ThumbMediumID,
		AddedBy:       p.AddedBy,
		TakenAt:       p.TakenAt,
		Latitude:      p.Latitude,
		Longitude:     p.Longitude,
		Caption:       p.Caption,
		CreatedAt:     p.CreatedAt,
	}
}

func shareTokenToDTO(s *ShareToken) ShareTokenDTO {
	return ShareTokenDTO{
		ID:             s.ID,
		TripID:         s.TripID,
		CreatedBy:      s.CreatedBy,
		ExpiresAt:      s.ExpiresAt,
		RevokedAt:      s.RevokedAt,
		RevokedBy:      s.RevokedBy,
		LastAccessedAt: s.LastAccessedAt,
		CreatedAt:      s.CreatedAt,
	}
}

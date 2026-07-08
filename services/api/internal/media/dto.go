package media

import (
	"time"

	"github.com/google/uuid"
)

type InitUploadRequest struct {
	Purpose string `json:"purpose" binding:"required"` // "chat" | "avatar" | "cover" | "todo"
	Mime    string `json:"mime" binding:"required"`
	Bytes   int64  `json:"bytes" binding:"required"`
}

type InitUploadResponse struct {
	UploadID  uuid.UUID         `json:"upload_id"`
	UploadURL string            `json:"upload_url"`        // browser PUTs the file body directly here
	Method    string            `json:"method"`            // always "PUT"
	Headers   map[string]string `json:"headers,omitempty"` // required PUT headers (e.g. Content-Type)
	ExpiresAt time.Time         `json:"expires_at"`
}

type CompleteUploadRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
}

type AssetDTO struct {
	ID        uuid.UUID `json:"id"`
	Purpose   Purpose   `json:"purpose"`
	Mime      string    `json:"mime"`
	Bytes     int64     `json:"bytes"`
	Width     *int      `json:"width,omitempty"`
	Height    *int      `json:"height,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

type PresignedGetResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func AssetToDTO(a *Asset) AssetDTO {
	return AssetDTO{
		ID:        a.ID,
		Purpose:   a.Purpose,
		Mime:      a.Mime,
		Bytes:     a.Bytes,
		Width:     a.Width,
		Height:    a.Height,
		CreatedAt: a.CreatedAt,
	}
}

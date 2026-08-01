package album

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/media"
	tripmod "github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

func decimalFromFloat(f float64) decimal.Decimal {
	return decimal.NewFromFloat(f).Round(6)
}

var (
	ErrInvalidPayload = errors.New("album: invalid payload")
	ErrCaptionTooLong = errors.New("album: caption too long")
	ErrShareExpired   = errors.New("album: share token expired")
	ErrBadUploadPair  = errors.New("album: upload session does not match this trip")
)

const (
	maxCaptionLen   = 500
	shareTokenBytes = 24
)

type TripAuthorizer interface {
	IsRoomMember(ctx context.Context, tripID, userID uuid.UUID) (bool, error)
	FindTripByID(ctx context.Context, id uuid.UUID) (*tripmod.Trip, error)
	FindRoomByTripID(ctx context.Context, tripID uuid.UUID) (*tripmod.Room, error)
	ListMembers(ctx context.Context, roomID uuid.UUID) ([]tripmod.RoomMember, error)
}

type MediaService interface {
	InitUpload(ctx context.Context, userID uuid.UUID, req media.InitUploadRequest) (*media.InitUploadResponse, error)
	CompleteUpload(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*media.AssetDTO, error)
	PresignRead(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) (*media.PresignedGetResponse, error)
	FetchAssetBytes(ctx context.Context, assetID uuid.UUID) ([]byte, *media.Asset, error)
	CreateInternalAsset(ctx context.Context, ownerID uuid.UUID, purpose media.Purpose, mime string, data []byte) (*media.Asset, error)
}

type Broadcaster interface {
	PublishTripEvent(ctx context.Context, tripID uuid.UUID, payload []byte)
}

type IService interface {
	InitUpload(ctx context.Context, userID, tripID uuid.UUID, p InitUploadPayload) (*InitUploadResponse, error)
	CompletePhoto(ctx context.Context, userID, tripID uuid.UUID, p CompletePhotoPayload) (*PhotoWithURLsDTO, error)
	ListPhotos(ctx context.Context, userID, tripID uuid.UUID) ([]PhotoWithURLsDTO, error)
	UpdatePhoto(ctx context.Context, userID, photoID uuid.UUID, p UpdatePhotoPayload) (*PhotoDTO, error)
	DeletePhoto(ctx context.Context, userID, photoID uuid.UUID) error
	DownloadOriginal(ctx context.Context, userID, photoID uuid.UUID) (*DownloadOriginalResponse, error)

	CreateShareToken(ctx context.Context, userID, tripID uuid.UUID, p CreateShareTokenPayload) (*CreateShareTokenResponse, error)
	ListShareTokens(ctx context.Context, userID, tripID uuid.UUID) ([]ShareTokenDTO, error)
	RevokeShareToken(ctx context.Context, userID, tokenID uuid.UUID) error

	ResolvePublicAlbum(ctx context.Context, plaintextToken string) ([]PhotoWithURLsDTO, *ShareTokenDTO, error)
}

type ServiceConfig struct {
	Repo        IRepository
	TripAuth    TripAuthorizer
	Media       MediaService
	Broadcaster Broadcaster
	Thumbnailer ThumbnailGenerator
	Exif        ExifExtractor
	Logger      *zap.Logger
}

type service struct {
	repo   IRepository
	auth   TripAuthorizer
	media  MediaService
	bcast  Broadcaster
	thumb  ThumbnailGenerator
	exif   ExifExtractor
	logger *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		auth:   cfg.TripAuth,
		media:  cfg.Media,
		bcast:  cfg.Broadcaster,
		thumb:  cfg.Thumbnailer,
		exif:   cfg.Exif,
		logger: logger.With(zap.String("layer", "album.service")),
	}
}

// ---- Upload flow ----

func (s *service) InitUpload(ctx context.Context, userID, tripID uuid.UUID, p InitUploadPayload) (*InitUploadResponse, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	if len(p.Items) == 0 {
		return nil, ErrInvalidPayload
	}
	slots := make([]InitUploadSlotDTO, 0, len(p.Items))
	for _, it := range p.Items {
		res, err := s.media.InitUpload(ctx, userID, media.InitUploadRequest{
			Purpose: string(media.PurposeAlbum),
			Mime:    it.Mime,
			Bytes:   it.Bytes,
		})
		if err != nil {
			return nil, err
		}
		slots = append(slots, InitUploadSlotDTO{
			UploadID:  res.UploadID,
			UploadURL: res.UploadURL,
			Method:    res.Method,
			Headers:   res.Headers,
			ExpiresAt: res.ExpiresAt,
		})
	}
	return &InitUploadResponse{Slots: slots}, nil
}

func (s *service) CompletePhoto(ctx context.Context, userID, tripID uuid.UUID, p CompletePhotoPayload) (*PhotoWithURLsDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	if len(p.Caption) > maxCaptionLen {
		return nil, ErrCaptionTooLong
	}

	asset, err := s.media.CompleteUpload(ctx, userID, p.UploadID)
	if err != nil {
		return nil, err
	}

	raw, _, err := s.media.FetchAssetBytes(ctx, asset.ID)
	if err != nil {
		return nil, fmt.Errorf("album: fetch original: %w", err)
	}

	exif, _ := s.exif.Extract(ctx, raw, asset.Mime)
	takenAt := exif.TakenAt
	if takenAt.IsZero() {
		takenAt = time.Now().UTC()
	}

	photo := &Photo{
		TripID:  tripID,
		MediaID: asset.ID,
		AddedBy: userID,
		TakenAt: takenAt,
		Caption: p.Caption,
	}
	if exif.Latitude != nil && exif.Longitude != nil {
		lat := decimalFromFloat(*exif.Latitude)
		lng := decimalFromFloat(*exif.Longitude)
		photo.Latitude = &lat
		photo.Longitude = &lng
	}

	if small, err := s.generateThumb(ctx, userID, raw, asset.Mime, ThumbSmall); err == nil && small != nil {
		photo.ThumbSmallID = &small.ID
	}
	if medium, err := s.generateThumb(ctx, userID, raw, asset.Mime, ThumbMedium); err == nil && medium != nil {
		photo.ThumbMediumID = &medium.ID
	}

	if err := s.repo.CreatePhoto(ctx, photo); err != nil {
		return nil, fmt.Errorf("album: create photo: %w", err)
	}

	dto, err := s.attachURLs(ctx, userID, photo)
	if err != nil {
		return nil, err
	}
	s.broadcast(ctx, tripID, BroadcastPhotoUploaded, dto)
	return dto, nil
}

func (s *service) generateThumb(ctx context.Context, userID uuid.UUID, src []byte, mime string, size ThumbnailSize) (*media.Asset, error) {
	data, outMime, err := s.thumb.Generate(ctx, src, mime, size)
	if err != nil {
		// Best-effort: a missing thumbnail must not fail the parent upload.
		s.logger.Warn("album: thumb generate failed",
			zap.String("size", size.Name), zap.Error(err))
		return nil, nil
	}
	return s.media.CreateInternalAsset(ctx, userID, media.PurposeAlbumThumb, outMime, data)
}

// ---- Read ----

func (s *service) ListPhotos(ctx context.Context, userID, tripID uuid.UUID) ([]PhotoWithURLsDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	photos, err := s.repo.ListPhotosByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]PhotoWithURLsDTO, 0, len(photos))
	for i := range photos {
		dto, err := s.attachURLs(ctx, userID, &photos[i])
		if err != nil {
			s.logger.Warn("album: skip photo (presign failed)",
				zap.String("photo_id", photos[i].ID.String()), zap.Error(err))
			continue
		}
		out = append(out, *dto)
	}
	return out, nil
}

func (s *service) UpdatePhoto(ctx context.Context, userID, photoID uuid.UUID, p UpdatePhotoPayload) (*PhotoDTO, error) {
	photo, err := s.repo.FindPhoto(ctx, photoID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, photo.TripID, userID); err != nil {
		return nil, err
	}
	patch := map[string]any{}
	if p.Caption != nil {
		c := strings.TrimSpace(*p.Caption)
		if len(c) > maxCaptionLen {
			return nil, ErrCaptionTooLong
		}
		patch["caption"] = c
	}
	if len(patch) == 0 {
		dto := photoToDTO(photo)
		return &dto, nil
	}
	updated, err := s.repo.UpdatePhoto(ctx, photoID, patch)
	if err != nil {
		return nil, err
	}
	dto := photoToDTO(updated)
	s.broadcast(ctx, updated.TripID, BroadcastPhotoUpdated, dto)
	return &dto, nil
}

func (s *service) DeletePhoto(ctx context.Context, userID, photoID uuid.UUID) error {
	photo, err := s.repo.FindPhoto(ctx, photoID)
	if err != nil {
		return err
	}
	if err := s.mustBeOwnerOrAdmin(ctx, photo.TripID, userID, photo.AddedBy); err != nil {
		return err
	}
	if err := s.repo.SoftDeletePhoto(ctx, photoID); err != nil {
		return err
	}
	s.broadcast(ctx, photo.TripID, BroadcastPhotoDeleted, map[string]any{
		"photo_id": photoID,
	})
	return nil
}

func (s *service) DownloadOriginal(ctx context.Context, userID, photoID uuid.UUID) (*DownloadOriginalResponse, error) {
	photo, err := s.repo.FindPhoto(ctx, photoID)
	if err != nil {
		return nil, err
	}
	if err := s.mustBeMember(ctx, photo.TripID, userID); err != nil {
		return nil, err
	}
	pres, err := s.media.PresignRead(ctx, userID, photo.MediaID)
	if err != nil {
		return nil, err
	}
	return &DownloadOriginalResponse{URL: pres.URL, ExpiresAt: pres.ExpiresAt}, nil
}

// ---- Share tokens ----

func (s *service) CreateShareToken(ctx context.Context, userID, tripID uuid.UUID, p CreateShareTokenPayload) (*CreateShareTokenResponse, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	plaintext, hash, err := generateShareToken()
	if err != nil {
		return nil, err
	}
	tok := &ShareToken{
		TripID:    tripID,
		TokenHash: hash,
		CreatedBy: userID,
		ExpiresAt: p.ExpiresAt,
	}
	if err := s.repo.CreateShareToken(ctx, tok); err != nil {
		return nil, err
	}
	dto := shareTokenToDTO(tok)
	s.broadcast(ctx, tripID, BroadcastShareCreated, dto)
	return &CreateShareTokenResponse{Token: plaintext, Share: dto}, nil
}

func (s *service) ListShareTokens(ctx context.Context, userID, tripID uuid.UUID) ([]ShareTokenDTO, error) {
	if err := s.mustBeMember(ctx, tripID, userID); err != nil {
		return nil, err
	}
	rows, err := s.repo.ListShareTokensByTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	out := make([]ShareTokenDTO, 0, len(rows))
	for i := range rows {
		out = append(out, shareTokenToDTO(&rows[i]))
	}
	return out, nil
}

func (s *service) RevokeShareToken(ctx context.Context, userID, tokenID uuid.UUID) error {
	tok, err := s.repo.FindShareToken(ctx, tokenID)
	if err != nil {
		return err
	}
	if err := s.mustBeOwnerOrAdmin(ctx, tok.TripID, userID, tok.CreatedBy); err != nil {
		return err
	}
	if err := s.repo.RevokeShareToken(ctx, tokenID, userID); err != nil {
		return err
	}
	s.broadcast(ctx, tok.TripID, BroadcastShareRevoked, map[string]any{
		"token_id": tokenID,
	})
	return nil
}

func (s *service) ResolvePublicAlbum(ctx context.Context, plaintextToken string) ([]PhotoWithURLsDTO, *ShareTokenDTO, error) {
	if plaintextToken == "" {
		return nil, nil, ErrShareTokenNotFound
	}
	hash := sha256.Sum256([]byte(plaintextToken))
	tok, err := s.repo.FindShareTokenByHash(ctx, hash[:])
	if err != nil {
		return nil, nil, err
	}
	if tok.RevokedAt != nil {
		return nil, nil, ErrShareTokenInactive
	}
	if tok.ExpiresAt != nil && time.Now().UTC().After(*tok.ExpiresAt) {
		return nil, nil, ErrShareExpired
	}

	photos, err := s.repo.ListPhotosByTrip(ctx, tok.TripID)
	if err != nil {
		return nil, nil, err
	}
	out := make([]PhotoWithURLsDTO, 0, len(photos))
	// A valid token authenticates the reader; presign under the creator's id.
	for i := range photos {
		dto, err := s.attachURLs(ctx, tok.CreatedBy, &photos[i])
		if err != nil {
			continue
		}
		out = append(out, *dto)
	}

	// Fire-and-forget: audit update must not block the public read.
	go func(id uuid.UUID) {
		bctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = s.repo.TouchShareTokenAccess(bctx, id, time.Now().UTC())
	}(tok.ID)

	dto := shareTokenToDTO(tok)
	return out, &dto, nil
}

// ---- Helpers ----

func (s *service) mustBeMember(ctx context.Context, tripID, userID uuid.UUID) error {
	ok, err := s.auth.IsRoomMember(ctx, tripID, userID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrForbidden
	}
	return nil
}

func (s *service) mustBeOwnerOrAdmin(ctx context.Context, tripID, actorID, creatorID uuid.UUID) error {
	if actorID == creatorID {
		return s.mustBeMember(ctx, tripID, actorID)
	}
	trip, err := s.auth.FindTripByID(ctx, tripID)
	if err != nil {
		return err
	}
	if trip == nil {
		return ErrForbidden
	}
	if trip.OwnerID == actorID {
		return nil
	}
	room, err := s.auth.FindRoomByTripID(ctx, tripID)
	if err != nil {
		return err
	}
	if room == nil {
		return ErrForbidden
	}
	members, err := s.auth.ListMembers(ctx, room.ID)
	if err != nil {
		return err
	}
	for _, m := range members {
		if m.UserID == actorID && m.Role == tripmod.RoleAdmin {
			return nil
		}
	}
	return ErrForbidden
}

func (s *service) attachURLs(ctx context.Context, userID uuid.UUID, p *Photo) (*PhotoWithURLsDTO, error) {
	dto := PhotoWithURLsDTO{PhotoDTO: photoToDTO(p)}
	orig, err := s.media.PresignRead(ctx, userID, p.MediaID)
	if err != nil {
		return nil, err
	}
	dto.OriginalURL = orig.URL
	if p.ThumbSmallID != nil {
		if u, err := s.media.PresignRead(ctx, userID, *p.ThumbSmallID); err == nil {
			dto.ThumbSmallURL = u.URL
		}
	}
	if p.ThumbMediumID != nil {
		if u, err := s.media.PresignRead(ctx, userID, *p.ThumbMediumID); err == nil {
			dto.ThumbMediumURL = u.URL
		}
	}
	return &dto, nil
}

func (s *service) broadcast(ctx context.Context, tripID uuid.UUID, kind BroadcastKind, data any) {
	if s.bcast == nil {
		return
	}
	payload, err := json.Marshal(struct {
		Type   BroadcastKind `json:"type"`
		TripID uuid.UUID     `json:"trip_id"`
		Data   any           `json:"data"`
	}{kind, tripID, data})
	if err != nil {
		s.logger.Warn("album: broadcast marshal", zap.Error(err))
		return
	}
	s.bcast.PublishTripEvent(ctx, tripID, payload)
}

// generateShareToken returns a URL-safe plaintext and its SHA-256 hash.
// Plaintext is surfaced once; only the hash is persisted.
func generateShareToken() (string, []byte, error) {
	raw := make([]byte, shareTokenBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", nil, err
	}
	plaintext := base64.RawURLEncoding.EncodeToString(raw)
	hash := sha256.Sum256([]byte(plaintext))
	return plaintext, hash[:], nil
}

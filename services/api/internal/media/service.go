package media

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

var (
	ErrInvalidPurpose = errors.New("media: invalid purpose")
	ErrMimeNotAllowed = errors.New("media: mime not allowed for purpose")
	ErrTooLarge       = errors.New("media: declared size exceeds purpose limit")
	ErrSessionExpired = errors.New("media: upload session expired")
	ErrSessionOwned   = errors.New("media: upload session belongs to another user")
	ErrSessionUsed    = errors.New("media: upload session already completed")
	ErrObjectMismatch = errors.New("media: uploaded object violates purpose rules")
	ErrForbidden      = errors.New("media: forbidden")
)

type IService interface {
	InitUpload(ctx context.Context, userID uuid.UUID, req InitUploadRequest) (*InitUploadResponse, error)
	CompleteUpload(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*AssetDTO, error)
	PresignRead(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) (*PresignedGetResponse, error)
	SoftDelete(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) error
	GetAsset(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) (*AssetDTO, error)

	AuthorizeAssetForOwner(ctx context.Context, userID, assetID uuid.UUID) (*Asset, error)
}

type ServiceConfig struct {
	Repo    IRepository
	Storage Storage
	Cfg     config.MediaConfig
	Bus     *event.Bus
	Logger  *zap.Logger
}

type service struct {
	repo    IRepository
	storage Storage
	cfg     config.MediaConfig
	bus     *event.Bus
	logger  *zap.Logger
	rules   map[Purpose]PurposeRule
}

func NewService(c ServiceConfig) IService {
	logger := c.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:    c.Repo,
		storage: c.Storage,
		cfg:     c.Cfg,
		bus:     c.Bus,
		logger:  logger.With(zap.String("layer", "media.service")),
		rules:   defaultRules(),
	}
}

func (s *service) InitUpload(ctx context.Context, userID uuid.UUID, req InitUploadRequest) (*InitUploadResponse, error) {
	purpose := Purpose(strings.ToLower(strings.TrimSpace(req.Purpose)))
	rule, ok := s.rules[purpose]
	if !ok {
		return nil, ErrInvalidPurpose
	}
	declaredMime := strings.ToLower(strings.TrimSpace(req.Mime))
	if _, ok := rule.AllowedMime[declaredMime]; !ok {
		return nil, ErrMimeNotAllowed
	}
	if req.Bytes <= 0 || req.Bytes > rule.MaxBytes {
		return nil, ErrTooLarge
	}

	ext := extForMime(declaredMime)
	key := s.storage.GenerateKey(rule.KeyPrefix, ext)

	now := time.Now().UTC()
	sess := &UploadSession{
		OwnerID:          userID,
		Purpose:          purpose,
		Bucket:           s.storage.Bucket(),
		ObjectKey:        key,
		ExpectedMime:     declaredMime,
		ExpectedMaxBytes: rule.MaxBytes,
		ExpiresAt:        now.Add(s.cfg.UploadSessionTTL),
	}
	if err := s.repo.CreateSession(ctx, sess); err != nil {
		return nil, fmt.Errorf("media: create session: %w", err)
	}

	putURL, err := s.storage.PresignPut(ctx, key, declaredMime, s.cfg.PresignPutTTL)
	if err != nil {
		// Best-effort session cleanup so a broken storage doesn't leave dangling
		// sessions. The GC would sweep this eventually anyway.
		_ = s.repo.DeleteSession(ctx, sess.ID)
		return nil, err
	}

	return &InitUploadResponse{
		UploadID:  sess.ID,
		UploadURL: putURL.String(),
		Method:    "PUT",
		Headers:   map[string]string{"Content-Type": declaredMime},
		ExpiresAt: sess.ExpiresAt,
	}, nil
}

func (s *service) CompleteUpload(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) (*AssetDTO, error) {
	sess, err := s.repo.FindSession(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	if sess.OwnerID != userID {
		return nil, ErrSessionOwned
	}
	if sess.CompletedAt != nil {
		return nil, ErrSessionUsed
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		return nil, ErrSessionExpired
	}

	stat, err := s.storage.StatObject(ctx, sess.ObjectKey)
	if err != nil {
		return nil, fmt.Errorf("media: stat: %w", err)
	}

	// Purpose rule enforced against the SERVER-observed values, not the ones
	// the client declared at InitUpload. If a client uploaded a giant file
	// under a "small-avatar" session, we catch it here.
	rule, ok := s.rules[sess.Purpose]
	if !ok {
		return nil, ErrInvalidPurpose
	}
	serverMime := strings.ToLower(strings.TrimSpace(stat.Mime))
	if _, ok := rule.AllowedMime[serverMime]; !ok {
		return nil, ErrObjectMismatch
	}
	if stat.Size <= 0 || stat.Size > rule.MaxBytes {
		return nil, ErrObjectMismatch
	}

	asset := &Asset{
		OwnerID:   userID,
		Purpose:   sess.Purpose,
		Bucket:    sess.Bucket,
		ObjectKey: sess.ObjectKey,
		Mime:      serverMime,
		Bytes:     stat.Size,
		ETag:      strings.Trim(stat.ETag, `"`),
	}
	if err := s.repo.CreateAsset(ctx, asset); err != nil {
		return nil, fmt.Errorf("media: create asset: %w", err)
	}

	if err := s.repo.CompleteSession(ctx, sess.ID, asset.ID); err != nil {
		s.logger.Warn("media: complete session race",
			zap.String("session_id", sess.ID.String()),
			zap.String("asset_id", asset.ID.String()),
			zap.Error(err),
		)
	}

	if s.bus != nil {
		s.bus.Publish(ctx, event.Event{
			Type:      event.EventMediaUploaded,
			Payload:   AssetToDTO(asset),
			UserID:    userID,
			TargetID:  asset.ID,
			Timestamp: time.Now().UTC(),
		})
	}

	dto := AssetToDTO(asset)
	return &dto, nil
}

// PresignRead: any authenticated caller can read for now.
// Purpose-scoped ACLs layer on top of this in the callsite that resolves the media_id —
// this call itself only checks the row is not soft-deleted.
func (s *service) PresignRead(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) (*PresignedGetResponse, error) {
	a, err := s.repo.FindAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	u, err := s.storage.PresignGet(ctx, a.ObjectKey, s.cfg.PresignGetTTL)
	if err != nil {
		return nil, err
	}
	return &PresignedGetResponse{
		URL:       u.String(),
		ExpiresAt: time.Now().UTC().Add(s.cfg.PresignGetTTL),
	}, nil
}

func (s *service) GetAsset(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) (*AssetDTO, error) {
	a, err := s.repo.FindAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	dto := AssetToDTO(a)
	return &dto, nil
}

func (s *service) SoftDelete(ctx context.Context, userID uuid.UUID, assetID uuid.UUID) error {
	a, err := s.repo.FindAsset(ctx, assetID)
	if err != nil {
		return err
	}
	if a.OwnerID != userID {
		return ErrForbidden
	}
	return s.repo.SoftDeleteAsset(ctx, assetID)
}

func (s *service) AuthorizeAssetForOwner(ctx context.Context, userID, assetID uuid.UUID) (*Asset, error) {
	a, err := s.repo.FindAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}
	if a.OwnerID != userID {
		return nil, ErrForbidden
	}
	return a, nil
}

// extForMime picks a stable extension so the object key stays readable and
// content-type sniffing tools don't get confused. Only the allow-listed mimes
// are mapped — anything else stores without an extension.
func extForMime(m string) string {
	switch m {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "image/gif":
		return ".gif"
	case "application/pdf":
		return ".pdf"
	case "text/plain":
		return ".txt"
	case "application/zip":
		return ".zip"
	}
	return ""
}

package media_test

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/media"
	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	createSession              func(context.Context, *media.UploadSession) error
	findSession                func(context.Context, uuid.UUID) (*media.UploadSession, error)
	completeSession            func(context.Context, uuid.UUID, uuid.UUID) error
	listExpiredPendingSessions func(context.Context, time.Time, int) ([]media.UploadSession, error)
	deleteSession              func(context.Context, uuid.UUID) error

	createAsset     func(context.Context, *media.Asset) error
	findAsset       func(context.Context, uuid.UUID) (*media.Asset, error)
	softDeleteAsset func(context.Context, uuid.UUID) error
}

func (r *repoMock) CreateSession(c context.Context, s *media.UploadSession) error {
	if r.createSession != nil {
		return r.createSession(c, s)
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindSession(c context.Context, id uuid.UUID) (*media.UploadSession, error) {
	if r.findSession != nil {
		return r.findSession(c, id)
	}
	return nil, media.ErrSessionNotFound
}

func (r *repoMock) CompleteSession(c context.Context, sid, mid uuid.UUID) error {
	if r.completeSession != nil {
		return r.completeSession(c, sid, mid)
	}
	return nil
}

func (r *repoMock) ListExpiredPendingSessions(c context.Context, now time.Time, limit int) ([]media.UploadSession, error) {
	if r.listExpiredPendingSessions != nil {
		return r.listExpiredPendingSessions(c, now, limit)
	}
	return nil, nil
}

func (r *repoMock) DeleteSession(c context.Context, id uuid.UUID) error {
	if r.deleteSession != nil {
		return r.deleteSession(c, id)
	}
	return nil
}

func (r *repoMock) CreateAsset(c context.Context, a *media.Asset) error {
	if r.createAsset != nil {
		return r.createAsset(c, a)
	}
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindAsset(c context.Context, id uuid.UUID) (*media.Asset, error) {
	if r.findAsset != nil {
		return r.findAsset(c, id)
	}
	return nil, media.ErrAssetNotFound
}

func (r *repoMock) SoftDeleteAsset(c context.Context, id uuid.UUID) error {
	if r.softDeleteAsset != nil {
		return r.softDeleteAsset(c, id)
	}
	return nil
}

// ---- storageMock ----

type storageMock struct {
	bucketName    string
	ensureBucket  func(context.Context) error
	presignPut    func(context.Context, string, string, time.Duration) (*url.URL, error)
	presignGet    func(context.Context, string, time.Duration) (*url.URL, error)
	statObject    func(context.Context, string) (*media.ObjectStat, error)
	removeObject  func(context.Context, string) error
	putObject     func(context.Context, string, string, []byte) (*media.ObjectStat, error)
	getObject     func(context.Context, string) ([]byte, string, error)
	generateKey   func(string, string) string
	generatedKeys []string
}

func (s *storageMock) Bucket() string {
	if s.bucketName == "" {
		return "test-bucket"
	}
	return s.bucketName
}

func (s *storageMock) EnsureBucket(c context.Context) error {
	if s.ensureBucket != nil {
		return s.ensureBucket(c)
	}
	return nil
}

func (s *storageMock) PresignPut(c context.Context, key, mime string, ttl time.Duration) (*url.URL, error) {
	if s.presignPut != nil {
		return s.presignPut(c, key, mime, ttl)
	}
	u, _ := url.Parse("https://storage.local/put/" + key)
	return u, nil
}

func (s *storageMock) PresignGet(c context.Context, key string, ttl time.Duration) (*url.URL, error) {
	if s.presignGet != nil {
		return s.presignGet(c, key, ttl)
	}
	u, _ := url.Parse("https://storage.local/get/" + key)
	return u, nil
}

func (s *storageMock) StatObject(c context.Context, key string) (*media.ObjectStat, error) {
	if s.statObject != nil {
		return s.statObject(c, key)
	}
	return &media.ObjectStat{}, nil
}

func (s *storageMock) RemoveObject(c context.Context, key string) error {
	if s.removeObject != nil {
		return s.removeObject(c, key)
	}
	return nil
}

func (s *storageMock) PutObject(c context.Context, key, mime string, data []byte) (*media.ObjectStat, error) {
	if s.putObject != nil {
		return s.putObject(c, key, mime, data)
	}
	return &media.ObjectStat{Size: int64(len(data)), Mime: mime}, nil
}

func (s *storageMock) GetObject(c context.Context, key string) ([]byte, string, error) {
	if s.getObject != nil {
		return s.getObject(c, key)
	}
	return nil, "", errors.New("storageMock: GetObject not implemented")
}

func (s *storageMock) GenerateKey(prefix, ext string) string {
	if s.generateKey != nil {
		key := s.generateKey(prefix, ext)
		s.generatedKeys = append(s.generatedKeys, key)
		return key
	}
	key := prefix + uuid.NewString() + ext
	s.generatedKeys = append(s.generatedKeys, key)
	return key
}

// ---- helpers ----

func newSvc(repo *repoMock, store *storageMock) media.IService {
	return media.NewService(media.ServiceConfig{
		Repo:    repo,
		Storage: store,
		Cfg: config.MediaConfig{
			PresignPutTTL:    10 * time.Minute,
			PresignGetTTL:    time.Hour,
			UploadSessionTTL: 15 * time.Minute,
		},
		// Bus intentionally nil — service skips publish when Bus == nil.
	})
}

// ---- InitUpload ----

func TestInitUpload_UnknownPurpose_Rejected(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "not-a-purpose", Mime: "image/png", Bytes: 100,
	})
	assert.ErrorIs(t, err, media.ErrInvalidPurpose)
}

func TestInitUpload_DisallowedMimeForAvatar(t *testing.T) {
	// avatar accepts image mimes only; reject application/pdf
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "avatar", Mime: "application/pdf", Bytes: 100,
	})
	assert.ErrorIs(t, err, media.ErrMimeNotAllowed)
}

func TestInitUpload_ZeroBytes_Rejected(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "avatar", Mime: "image/png", Bytes: 0,
	})
	assert.ErrorIs(t, err, media.ErrTooLarge)
}

func TestInitUpload_ExceedsAvatarLimit(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "avatar", Mime: "image/png", Bytes: (5 << 20) + 1,
	})
	assert.ErrorIs(t, err, media.ErrTooLarge)
}

func TestInitUpload_ChatAcceptsBiggerFile(t *testing.T) {
	// 10MB image is fine for chat (100MB cap) but would fail avatar (5MB cap).
	repo := &repoMock{}
	store := &storageMock{}
	svc := newSvc(repo, store)
	resp, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "chat", Mime: "image/png", Bytes: 10 << 20,
	})
	require.NoError(t, err)
	assert.Equal(t, "PUT", resp.Method)
	assert.Equal(t, "image/png", resp.Headers["Content-Type"])
	assert.NotEqual(t, uuid.Nil, resp.UploadID)
	require.Len(t, store.generatedKeys, 1)
	assert.True(t, strings.HasPrefix(store.generatedKeys[0], "chat/"), "key must live under chat/ prefix")
	assert.True(t, strings.HasSuffix(store.generatedKeys[0], ".png"))
}

func TestInitUpload_NormalizesPurposeAndMimeCase(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	resp, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "  AVATAR ", Mime: " Image/PNG ", Bytes: 1024,
	})
	require.NoError(t, err)
	assert.Equal(t, "image/png", resp.Headers["Content-Type"])
}

func TestInitUpload_PresignFailure_CleansUpSession(t *testing.T) {
	deletedSess := false
	repo := &repoMock{
		deleteSession: func(_ context.Context, _ uuid.UUID) error { deletedSess = true; return nil },
	}
	boom := errors.New("boom")
	store := &storageMock{
		presignPut: func(_ context.Context, _, _ string, _ time.Duration) (*url.URL, error) { return nil, boom },
	}
	svc := newSvc(repo, store)
	_, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "avatar", Mime: "image/png", Bytes: 1024,
	})
	assert.ErrorIs(t, err, boom)
	assert.True(t, deletedSess, "failed presign must schedule session cleanup")
}

func TestInitUpload_CreateSessionError_Wraps(t *testing.T) {
	boom := errors.New("db down")
	repo := &repoMock{
		createSession: func(_ context.Context, _ *media.UploadSession) error { return boom },
	}
	svc := newSvc(repo, &storageMock{})
	_, err := svc.InitUpload(ctx, uuid.New(), media.InitUploadRequest{
		Purpose: "avatar", Mime: "image/png", Bytes: 1024,
	})
	assert.ErrorIs(t, err, boom)
}

// ---- CompleteUpload ----

func TestCompleteUpload_WrongOwner(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	sess := &media.UploadSession{ID: uuid.New(), OwnerID: owner, Purpose: media.PurposeAvatar,
		ExpiresAt: time.Now().Add(10 * time.Minute)}
	repo := &repoMock{findSession: func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil }}
	svc := newSvc(repo, &storageMock{})
	_, err := svc.CompleteUpload(ctx, other, sess.ID)
	assert.ErrorIs(t, err, media.ErrSessionOwned)
}

func TestCompleteUpload_AlreadyCompleted(t *testing.T) {
	owner := uuid.New()
	completed := time.Now().UTC()
	sess := &media.UploadSession{ID: uuid.New(), OwnerID: owner, Purpose: media.PurposeAvatar,
		ExpiresAt: time.Now().Add(10 * time.Minute), CompletedAt: &completed}
	repo := &repoMock{findSession: func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil }}
	svc := newSvc(repo, &storageMock{})
	_, err := svc.CompleteUpload(ctx, owner, sess.ID)
	assert.ErrorIs(t, err, media.ErrSessionUsed)
}

func TestCompleteUpload_Expired(t *testing.T) {
	owner := uuid.New()
	sess := &media.UploadSession{ID: uuid.New(), OwnerID: owner, Purpose: media.PurposeAvatar,
		ExpiresAt: time.Now().Add(-time.Minute)}
	repo := &repoMock{findSession: func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil }}
	svc := newSvc(repo, &storageMock{})
	_, err := svc.CompleteUpload(ctx, owner, sess.ID)
	assert.ErrorIs(t, err, media.ErrSessionExpired)
}

func TestCompleteUpload_ObjectExceedsPurposeSize(t *testing.T) {
	owner := uuid.New()
	sess := &media.UploadSession{ID: uuid.New(), OwnerID: owner, Purpose: media.PurposeAvatar,
		ObjectKey: "avatar/x.png", ExpiresAt: time.Now().Add(10 * time.Minute)}
	repo := &repoMock{findSession: func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil }}
	store := &storageMock{
		statObject: func(_ context.Context, _ string) (*media.ObjectStat, error) {
			return &media.ObjectStat{Size: (5 << 20) + 1, Mime: "image/png", ETag: `"abc"`}, nil
		},
	}
	svc := newSvc(repo, store)
	_, err := svc.CompleteUpload(ctx, owner, sess.ID)
	assert.ErrorIs(t, err, media.ErrObjectMismatch)
}

func TestCompleteUpload_ObjectMimeNotAllowedForPurpose(t *testing.T) {
	owner := uuid.New()
	sess := &media.UploadSession{ID: uuid.New(), OwnerID: owner, Purpose: media.PurposeAvatar,
		ObjectKey: "avatar/x.png", ExpiresAt: time.Now().Add(10 * time.Minute)}
	repo := &repoMock{findSession: func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil }}
	store := &storageMock{
		statObject: func(_ context.Context, _ string) (*media.ObjectStat, error) {
			// Server observed a PDF even though the session was for avatar (image only).
			return &media.ObjectStat{Size: 1024, Mime: "application/pdf", ETag: `"abc"`}, nil
		},
	}
	svc := newSvc(repo, store)
	_, err := svc.CompleteUpload(ctx, owner, sess.ID)
	assert.ErrorIs(t, err, media.ErrObjectMismatch)
}

func TestCompleteUpload_Succeeds_CreatesAssetAndCompletesSession(t *testing.T) {
	owner := uuid.New()
	sessID := uuid.New()
	sess := &media.UploadSession{
		ID: sessID, OwnerID: owner, Purpose: media.PurposeChat,
		Bucket: "test-bucket", ObjectKey: "chat/2026/07/abc.png",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	var savedAsset *media.Asset
	var completedSessArgs struct {
		sessID, mediaID uuid.UUID
	}
	repo := &repoMock{
		findSession: func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil },
		createAsset: func(_ context.Context, a *media.Asset) error {
			a.ID = uuid.New()
			a.CreatedAt = time.Now().UTC()
			savedAsset = a
			return nil
		},
		completeSession: func(_ context.Context, sid, mid uuid.UUID) error {
			completedSessArgs.sessID, completedSessArgs.mediaID = sid, mid
			return nil
		},
	}
	store := &storageMock{
		statObject: func(_ context.Context, key string) (*media.ObjectStat, error) {
			assert.Equal(t, sess.ObjectKey, key)
			return &media.ObjectStat{Size: 2048, Mime: "image/png", ETag: `"abc123"`}, nil
		},
	}
	svc := newSvc(repo, store)
	dto, err := svc.CompleteUpload(ctx, owner, sessID)
	require.NoError(t, err)
	require.NotNil(t, savedAsset)
	assert.Equal(t, owner, savedAsset.OwnerID)
	assert.Equal(t, media.PurposeChat, savedAsset.Purpose)
	assert.Equal(t, "image/png", savedAsset.Mime, "server-observed mime, not the declared one")
	assert.Equal(t, int64(2048), savedAsset.Bytes)
	assert.Equal(t, "abc123", savedAsset.ETag, "quotes stripped from ETag")
	assert.Equal(t, sessID, completedSessArgs.sessID)
	assert.Equal(t, savedAsset.ID, completedSessArgs.mediaID)
	assert.Equal(t, savedAsset.ID, dto.ID)
}

func TestCompleteUpload_CompleteSessionError_IsNonFatal(t *testing.T) {
	// A race where the sweeper or a concurrent complete win must NOT roll back
	// the asset row — the service logs and returns the asset anyway.
	owner := uuid.New()
	sess := &media.UploadSession{
		ID: uuid.New(), OwnerID: owner, Purpose: media.PurposeChat,
		Bucket: "b", ObjectKey: "chat/x.png",
		ExpiresAt: time.Now().Add(10 * time.Minute),
	}
	repo := &repoMock{
		findSession:     func(_ context.Context, _ uuid.UUID) (*media.UploadSession, error) { return sess, nil },
		completeSession: func(_ context.Context, _, _ uuid.UUID) error { return media.ErrSessionNotFound },
	}
	store := &storageMock{
		statObject: func(_ context.Context, _ string) (*media.ObjectStat, error) {
			return &media.ObjectStat{Size: 1024, Mime: "image/png"}, nil
		},
	}
	svc := newSvc(repo, store)
	dto, err := svc.CompleteUpload(ctx, owner, sess.ID)
	require.NoError(t, err, "complete-session race must not fail the caller")
	assert.NotEqual(t, uuid.Nil, dto.ID)
}

// ---- PresignRead ----

func TestPresignRead_ReturnsPresignedURL(t *testing.T) {
	assetID := uuid.New()
	repo := &repoMock{
		findAsset: func(_ context.Context, id uuid.UUID) (*media.Asset, error) {
			assert.Equal(t, assetID, id)
			return &media.Asset{ID: id, ObjectKey: "avatar/foo.png"}, nil
		},
	}
	store := &storageMock{
		presignGet: func(_ context.Context, key string, _ time.Duration) (*url.URL, error) {
			assert.Equal(t, "avatar/foo.png", key)
			u, _ := url.Parse("https://s.example.com/avatar/foo.png?sig=xyz")
			return u, nil
		},
	}
	svc := newSvc(repo, store)
	resp, err := svc.PresignRead(ctx, uuid.New(), assetID)
	require.NoError(t, err)
	assert.Equal(t, "https://s.example.com/avatar/foo.png?sig=xyz", resp.URL)
	assert.WithinDuration(t, time.Now().Add(time.Hour), resp.ExpiresAt, 5*time.Second)
}

func TestPresignRead_AssetNotFound(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.PresignRead(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, media.ErrAssetNotFound)
}

// ---- GetAsset ----

func TestGetAsset_NotFound(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.GetAsset(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, media.ErrAssetNotFound)
}

func TestGetAsset_ReturnsDTO(t *testing.T) {
	assetID, owner := uuid.New(), uuid.New()
	repo := &repoMock{
		findAsset: func(_ context.Context, _ uuid.UUID) (*media.Asset, error) {
			return &media.Asset{ID: assetID, OwnerID: owner, Purpose: media.PurposeChat,
				Mime: "image/png", Bytes: 1024}, nil
		},
	}
	svc := newSvc(repo, &storageMock{})
	dto, err := svc.GetAsset(ctx, uuid.New(), assetID)
	require.NoError(t, err)
	assert.Equal(t, assetID, dto.ID)
	assert.Equal(t, media.PurposeChat, dto.Purpose)
	assert.Equal(t, int64(1024), dto.Bytes)
}

// ---- SoftDelete ----

func TestSoftDelete_NotOwner_Forbidden(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	repo := &repoMock{
		findAsset: func(_ context.Context, id uuid.UUID) (*media.Asset, error) {
			return &media.Asset{ID: id, OwnerID: owner}, nil
		},
	}
	svc := newSvc(repo, &storageMock{})
	err := svc.SoftDelete(ctx, other, uuid.New())
	assert.ErrorIs(t, err, media.ErrForbidden)
}

func TestSoftDelete_ByOwner_Succeeds(t *testing.T) {
	owner := uuid.New()
	deletedID := uuid.Nil
	repo := &repoMock{
		findAsset: func(_ context.Context, id uuid.UUID) (*media.Asset, error) {
			return &media.Asset{ID: id, OwnerID: owner}, nil
		},
		softDeleteAsset: func(_ context.Context, id uuid.UUID) error { deletedID = id; return nil },
	}
	svc := newSvc(repo, &storageMock{})
	target := uuid.New()
	require.NoError(t, svc.SoftDelete(ctx, owner, target))
	assert.Equal(t, target, deletedID)
}

// ---- AuthorizeAssetForOwner ----

func TestAuthorizeAssetForOwner_NotOwner_Forbidden(t *testing.T) {
	owner, other := uuid.New(), uuid.New()
	repo := &repoMock{
		findAsset: func(_ context.Context, id uuid.UUID) (*media.Asset, error) {
			return &media.Asset{ID: id, OwnerID: owner}, nil
		},
	}
	svc := newSvc(repo, &storageMock{})
	_, err := svc.AuthorizeAssetForOwner(ctx, other, uuid.New())
	assert.ErrorIs(t, err, media.ErrForbidden)
}

func TestAuthorizeAssetForOwner_ByOwner_ReturnsAsset(t *testing.T) {
	owner := uuid.New()
	target := uuid.New()
	repo := &repoMock{
		findAsset: func(_ context.Context, id uuid.UUID) (*media.Asset, error) {
			return &media.Asset{ID: id, OwnerID: owner, Mime: "image/png"}, nil
		},
	}
	svc := newSvc(repo, &storageMock{})
	got, err := svc.AuthorizeAssetForOwner(ctx, owner, target)
	require.NoError(t, err)
	assert.Equal(t, target, got.ID)
	assert.Equal(t, owner, got.OwnerID)
}

func TestAuthorizeAssetForOwner_AssetNotFound(t *testing.T) {
	svc := newSvc(&repoMock{}, &storageMock{})
	_, err := svc.AuthorizeAssetForOwner(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, media.ErrAssetNotFound)
}

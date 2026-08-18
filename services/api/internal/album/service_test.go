package album_test

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/album"
	"github.com/Ans1110/trip-app/internal/media"
	tripmod "github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx func(context.Context, func(album.IRepository) error) error

	createPhoto      func(context.Context, *album.Photo) error
	updatePhoto      func(context.Context, uuid.UUID, map[string]any) (*album.Photo, error)
	findPhoto        func(context.Context, uuid.UUID) (*album.Photo, error)
	listPhotosByTrip func(context.Context, uuid.UUID) ([]album.Photo, error)
	softDeletePhoto  func(context.Context, uuid.UUID) error

	createShareToken       func(context.Context, *album.ShareToken) error
	findShareTokenByHash   func(context.Context, []byte) (*album.ShareToken, error)
	findShareToken         func(context.Context, uuid.UUID) (*album.ShareToken, error)
	listShareTokensByTrip  func(context.Context, uuid.UUID) ([]album.ShareToken, error)
	revokeShareToken       func(context.Context, uuid.UUID, uuid.UUID) error
	touchShareTokenAccess  func(context.Context, uuid.UUID, time.Time) error
}

func (r *repoMock) WithTx(c context.Context, fn func(album.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(c, fn)
	}
	return fn(r)
}
func (r *repoMock) CreatePhoto(c context.Context, p *album.Photo) error {
	if r.createPhoto != nil {
		return r.createPhoto(c, p)
	}
	if p.ID == uuid.Nil {
		p.ID = uuid.New()
	}
	return nil
}
func (r *repoMock) UpdatePhoto(c context.Context, id uuid.UUID, patch map[string]any) (*album.Photo, error) {
	if r.updatePhoto != nil {
		return r.updatePhoto(c, id, patch)
	}
	return nil, nil
}
func (r *repoMock) FindPhoto(c context.Context, id uuid.UUID) (*album.Photo, error) {
	if r.findPhoto != nil {
		return r.findPhoto(c, id)
	}
	return nil, nil
}
func (r *repoMock) ListPhotosByTrip(c context.Context, tid uuid.UUID) ([]album.Photo, error) {
	if r.listPhotosByTrip != nil {
		return r.listPhotosByTrip(c, tid)
	}
	return nil, nil
}
func (r *repoMock) SoftDeletePhoto(c context.Context, id uuid.UUID) error {
	if r.softDeletePhoto != nil {
		return r.softDeletePhoto(c, id)
	}
	return nil
}
func (r *repoMock) CreateShareToken(c context.Context, s *album.ShareToken) error {
	if r.createShareToken != nil {
		return r.createShareToken(c, s)
	}
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	return nil
}
func (r *repoMock) FindShareTokenByHash(c context.Context, h []byte) (*album.ShareToken, error) {
	if r.findShareTokenByHash != nil {
		return r.findShareTokenByHash(c, h)
	}
	return nil, album.ErrShareTokenNotFound
}
func (r *repoMock) FindShareToken(c context.Context, id uuid.UUID) (*album.ShareToken, error) {
	if r.findShareToken != nil {
		return r.findShareToken(c, id)
	}
	return nil, album.ErrShareTokenNotFound
}
func (r *repoMock) ListShareTokensByTrip(c context.Context, tid uuid.UUID) ([]album.ShareToken, error) {
	if r.listShareTokensByTrip != nil {
		return r.listShareTokensByTrip(c, tid)
	}
	return nil, nil
}
func (r *repoMock) RevokeShareToken(c context.Context, id, by uuid.UUID) error {
	if r.revokeShareToken != nil {
		return r.revokeShareToken(c, id, by)
	}
	return nil
}
func (r *repoMock) TouchShareTokenAccess(c context.Context, id uuid.UUID, at time.Time) error {
	if r.touchShareTokenAccess != nil {
		return r.touchShareTokenAccess(c, id, at)
	}
	return nil
}

// ---- authMock ----

type authMock struct {
	isRoomMember    func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	findTripByID    func(context.Context, uuid.UUID) (*tripmod.Trip, error)
	findRoomByTrip  func(context.Context, uuid.UUID) (*tripmod.Room, error)
	listMembers     func(context.Context, uuid.UUID) ([]tripmod.RoomMember, error)
}

func (a *authMock) IsRoomMember(c context.Context, t, u uuid.UUID) (bool, error) {
	if a.isRoomMember != nil {
		return a.isRoomMember(c, t, u)
	}
	return true, nil
}
func (a *authMock) FindTripByID(c context.Context, id uuid.UUID) (*tripmod.Trip, error) {
	if a.findTripByID != nil {
		return a.findTripByID(c, id)
	}
	return nil, nil
}
func (a *authMock) FindRoomByTripID(c context.Context, tid uuid.UUID) (*tripmod.Room, error) {
	if a.findRoomByTrip != nil {
		return a.findRoomByTrip(c, tid)
	}
	return nil, nil
}
func (a *authMock) ListMembers(c context.Context, rid uuid.UUID) ([]tripmod.RoomMember, error) {
	if a.listMembers != nil {
		return a.listMembers(c, rid)
	}
	return nil, nil
}

// ---- mediaMock ----

type mediaMock struct {
	initUpload          func(context.Context, uuid.UUID, media.InitUploadRequest) (*media.InitUploadResponse, error)
	completeUpload      func(context.Context, uuid.UUID, uuid.UUID) (*media.AssetDTO, error)
	presignRead         func(context.Context, uuid.UUID, uuid.UUID) (*media.PresignedGetResponse, error)
	fetchAssetBytes     func(context.Context, uuid.UUID) ([]byte, *media.Asset, error)
	createInternalAsset func(context.Context, uuid.UUID, media.Purpose, string, []byte) (*media.Asset, error)
}

func (m *mediaMock) InitUpload(c context.Context, u uuid.UUID, r media.InitUploadRequest) (*media.InitUploadResponse, error) {
	if m.initUpload != nil {
		return m.initUpload(c, u, r)
	}
	return &media.InitUploadResponse{UploadID: uuid.New(), UploadURL: "http://s3/put", Method: "PUT", ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (m *mediaMock) CompleteUpload(c context.Context, u, sess uuid.UUID) (*media.AssetDTO, error) {
	if m.completeUpload != nil {
		return m.completeUpload(c, u, sess)
	}
	return &media.AssetDTO{ID: uuid.New(), Mime: "image/jpeg"}, nil
}
func (m *mediaMock) PresignRead(c context.Context, u, a uuid.UUID) (*media.PresignedGetResponse, error) {
	if m.presignRead != nil {
		return m.presignRead(c, u, a)
	}
	return &media.PresignedGetResponse{URL: "http://s3/get/" + a.String(), ExpiresAt: time.Now().Add(time.Hour)}, nil
}
func (m *mediaMock) FetchAssetBytes(c context.Context, a uuid.UUID) ([]byte, *media.Asset, error) {
	if m.fetchAssetBytes != nil {
		return m.fetchAssetBytes(c, a)
	}
	return []byte("bytes"), &media.Asset{ID: a, Mime: "image/jpeg"}, nil
}
func (m *mediaMock) CreateInternalAsset(c context.Context, u uuid.UUID, p media.Purpose, mime string, d []byte) (*media.Asset, error) {
	if m.createInternalAsset != nil {
		return m.createInternalAsset(c, u, p, mime, d)
	}
	return &media.Asset{ID: uuid.New(), OwnerID: u, Purpose: p, Mime: mime}, nil
}

// ---- broadcasterMock ----

type broadcasterMock struct {
	events []broadcastEvent
}

type broadcastEvent struct {
	tripID  uuid.UUID
	payload []byte
}

func (b *broadcasterMock) PublishTripEvent(_ context.Context, tid uuid.UUID, payload []byte) {
	b.events = append(b.events, broadcastEvent{tid, payload})
}

// ---- thumbMock ----

type thumbMock struct {
	generate func(context.Context, []byte, string, album.ThumbnailSize) ([]byte, string, error)
}

func (t *thumbMock) Generate(c context.Context, src []byte, mime string, size album.ThumbnailSize) ([]byte, string, error) {
	if t.generate != nil {
		return t.generate(c, src, mime, size)
	}
	return []byte("thumb"), "image/jpeg", nil
}

// ---- exifMock ----

type exifMock struct {
	extract func(context.Context, []byte, string) (album.ExifData, error)
}

func (e *exifMock) Extract(c context.Context, src []byte, mime string) (album.ExifData, error) {
	if e.extract != nil {
		return e.extract(c, src, mime)
	}
	return album.ExifData{}, nil
}

// ---- Factory ----

type svcDeps struct {
	repo  *repoMock
	auth  *authMock
	media *mediaMock
	bcast *broadcasterMock
	thumb *thumbMock
	exif  *exifMock
}

func newDeps() svcDeps {
	return svcDeps{
		repo:  &repoMock{},
		auth:  &authMock{},
		media: &mediaMock{},
		bcast: &broadcasterMock{},
		thumb: &thumbMock{},
		exif:  &exifMock{},
	}
}

func newSvc(d svcDeps) album.IService {
	return album.NewService(album.ServiceConfig{
		Repo:        d.repo,
		TripAuth:    d.auth,
		Media:       d.media,
		Broadcaster: d.bcast,
		Thumbnailer: d.thumb,
		Exif:        d.exif,
	})
}

// ---- InitUpload ----

func TestInitUpload_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.auth.isRoomMember = func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	_, err := svc.InitUpload(ctx, uuid.New(), uuid.New(), album.InitUploadPayload{
		Items: []album.InitUploadItem{{Mime: "image/jpeg", Bytes: 100}},
	})
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestInitUpload_EmptyItems(t *testing.T) {
	d := newDeps()
	svc := newSvc(d)
	_, err := svc.InitUpload(ctx, uuid.New(), uuid.New(), album.InitUploadPayload{})
	assert.ErrorIs(t, err, album.ErrInvalidPayload)
}

func TestInitUpload_MintsOneSlotPerItem(t *testing.T) {
	d := newDeps()
	var initCalls int
	d.media.initUpload = func(_ context.Context, _ uuid.UUID, r media.InitUploadRequest) (*media.InitUploadResponse, error) {
		initCalls++
		assert.Equal(t, string(media.PurposeAlbum), r.Purpose)
		return &media.InitUploadResponse{UploadID: uuid.New(), UploadURL: "http://s3/x", Method: "PUT"}, nil
	}
	svc := newSvc(d)
	resp, err := svc.InitUpload(ctx, uuid.New(), uuid.New(), album.InitUploadPayload{
		Items: []album.InitUploadItem{
			{Mime: "image/jpeg", Bytes: 100},
			{Mime: "image/png", Bytes: 200},
		},
	})
	require.NoError(t, err)
	require.Len(t, resp.Slots, 2)
	assert.Equal(t, 2, initCalls)
}

func TestInitUpload_MediaErrorPassthrough(t *testing.T) {
	boom := errors.New("s3 down")
	d := newDeps()
	d.media.initUpload = func(context.Context, uuid.UUID, media.InitUploadRequest) (*media.InitUploadResponse, error) {
		return nil, boom
	}
	svc := newSvc(d)
	_, err := svc.InitUpload(ctx, uuid.New(), uuid.New(), album.InitUploadPayload{
		Items: []album.InitUploadItem{{Mime: "image/jpeg", Bytes: 1}},
	})
	assert.ErrorIs(t, err, boom)
}

// ---- CompletePhoto ----

func TestCompletePhoto_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.auth.isRoomMember = func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	_, err := svc.CompletePhoto(ctx, uuid.New(), uuid.New(), album.CompletePhotoPayload{UploadID: uuid.New()})
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestCompletePhoto_CaptionTooLong(t *testing.T) {
	d := newDeps()
	svc := newSvc(d)
	_, err := svc.CompletePhoto(ctx, uuid.New(), uuid.New(), album.CompletePhotoPayload{
		UploadID: uuid.New(),
		Caption:  strings.Repeat("x", 501),
	})
	assert.ErrorIs(t, err, album.ErrCaptionTooLong)
}

func TestCompletePhoto_HappyPath_UsesExifAndBroadcasts(t *testing.T) {
	viewer := uuid.New()
	trip := uuid.New()
	assetID := uuid.New()
	taken := time.Date(2024, 1, 2, 3, 4, 5, 0, time.UTC)
	lat, lng := 25.0, 121.0

	d := newDeps()
	d.media.completeUpload = func(_ context.Context, _, _ uuid.UUID) (*media.AssetDTO, error) {
		return &media.AssetDTO{ID: assetID, Mime: "image/jpeg"}, nil
	}
	d.exif.extract = func(context.Context, []byte, string) (album.ExifData, error) {
		return album.ExifData{TakenAt: taken, Latitude: &lat, Longitude: &lng}, nil
	}
	var created *album.Photo
	d.repo.createPhoto = func(_ context.Context, p *album.Photo) error {
		p.ID = uuid.New()
		created = p
		return nil
	}
	svc := newSvc(d)

	dto, err := svc.CompletePhoto(ctx, viewer, trip, album.CompletePhotoPayload{
		UploadID: uuid.New(),
		Caption:  "hi",
	})
	require.NoError(t, err)
	require.NotNil(t, created)
	assert.Equal(t, trip, created.TripID)
	assert.Equal(t, assetID, created.MediaID)
	assert.Equal(t, viewer, created.AddedBy)
	assert.True(t, created.TakenAt.Equal(taken))
	require.NotNil(t, created.Latitude)
	require.NotNil(t, created.Longitude)
	assert.Equal(t, "25", created.Latitude.String())
	assert.Equal(t, "121", created.Longitude.String())
	require.NotNil(t, created.ThumbSmallID)
	require.NotNil(t, created.ThumbMediumID)
	assert.NotEmpty(t, dto.OriginalURL)

	require.Len(t, d.bcast.events, 1)
	assert.Equal(t, trip, d.bcast.events[0].tripID)
	var env struct {
		Type   string    `json:"type"`
		TripID uuid.UUID `json:"trip_id"`
	}
	require.NoError(t, json.Unmarshal(d.bcast.events[0].payload, &env))
	assert.Equal(t, string(album.BroadcastPhotoUploaded), env.Type)
	assert.Equal(t, trip, env.TripID)
}

func TestCompletePhoto_FallsBackToNowWhenExifTakenAtZero(t *testing.T) {
	d := newDeps()
	var got *album.Photo
	d.repo.createPhoto = func(_ context.Context, p *album.Photo) error {
		p.ID = uuid.New()
		got = p
		return nil
	}
	svc := newSvc(d)
	_, err := svc.CompletePhoto(ctx, uuid.New(), uuid.New(), album.CompletePhotoPayload{UploadID: uuid.New()})
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.False(t, got.TakenAt.IsZero())
	assert.Nil(t, got.Latitude)
	assert.Nil(t, got.Longitude)
}

func TestCompletePhoto_ThumbFailureDoesNotBreakUpload(t *testing.T) {
	d := newDeps()
	d.thumb.generate = func(context.Context, []byte, string, album.ThumbnailSize) ([]byte, string, error) {
		return nil, "", errors.New("boom")
	}
	var got *album.Photo
	d.repo.createPhoto = func(_ context.Context, p *album.Photo) error {
		p.ID = uuid.New()
		got = p
		return nil
	}
	svc := newSvc(d)
	_, err := svc.CompletePhoto(ctx, uuid.New(), uuid.New(), album.CompletePhotoPayload{UploadID: uuid.New()})
	require.NoError(t, err)
	assert.Nil(t, got.ThumbSmallID)
	assert.Nil(t, got.ThumbMediumID)
}

func TestCompletePhoto_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db down")
	d := newDeps()
	d.repo.createPhoto = func(context.Context, *album.Photo) error { return boom }
	svc := newSvc(d)
	_, err := svc.CompletePhoto(ctx, uuid.New(), uuid.New(), album.CompletePhotoPayload{UploadID: uuid.New()})
	assert.ErrorIs(t, err, boom)
}

// ---- ListPhotos ----

func TestListPhotos_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.auth.isRoomMember = func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	_, err := svc.ListPhotos(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestListPhotos_SkipsPhotosWithPresignFailure(t *testing.T) {
	badMedia := uuid.New()
	d := newDeps()
	d.repo.listPhotosByTrip = func(context.Context, uuid.UUID) ([]album.Photo, error) {
		return []album.Photo{
			{ID: uuid.New(), MediaID: uuid.New()},
			{ID: uuid.New(), MediaID: badMedia},
			{ID: uuid.New(), MediaID: uuid.New()},
		}, nil
	}
	d.media.presignRead = func(_ context.Context, _, a uuid.UUID) (*media.PresignedGetResponse, error) {
		if a == badMedia {
			return nil, errors.New("nope")
		}
		return &media.PresignedGetResponse{URL: "http://s3/" + a.String()}, nil
	}
	svc := newSvc(d)
	got, err := svc.ListPhotos(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

func TestListPhotos_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db")
	d := newDeps()
	d.repo.listPhotosByTrip = func(context.Context, uuid.UUID) ([]album.Photo, error) { return nil, boom }
	svc := newSvc(d)
	_, err := svc.ListPhotos(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, boom)
}

// ---- UpdatePhoto ----

func TestUpdatePhoto_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: uuid.New()}, nil
	}
	d.auth.isRoomMember = func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	cap := "new"
	_, err := svc.UpdatePhoto(ctx, uuid.New(), uuid.New(), album.UpdatePhotoPayload{Caption: &cap})
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestUpdatePhoto_CaptionTooLong(t *testing.T) {
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: uuid.New()}, nil
	}
	svc := newSvc(d)
	cap := strings.Repeat("x", 501)
	_, err := svc.UpdatePhoto(ctx, uuid.New(), uuid.New(), album.UpdatePhotoPayload{Caption: &cap})
	assert.ErrorIs(t, err, album.ErrCaptionTooLong)
}

func TestUpdatePhoto_EmptyPatchReturnsExistingDTO(t *testing.T) {
	photoID := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: uuid.New(), Caption: "same"}, nil
	}
	d.repo.updatePhoto = func(context.Context, uuid.UUID, map[string]any) (*album.Photo, error) {
		t.Fatal("update should not be called for empty patch")
		return nil, nil
	}
	svc := newSvc(d)
	got, err := svc.UpdatePhoto(ctx, uuid.New(), photoID, album.UpdatePhotoPayload{})
	require.NoError(t, err)
	assert.Equal(t, "same", got.Caption)
	assert.Empty(t, d.bcast.events)
}

func TestUpdatePhoto_HappyPathTrimsAndBroadcasts(t *testing.T) {
	photoID := uuid.New()
	trip := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: trip}, nil
	}
	var gotPatch map[string]any
	d.repo.updatePhoto = func(_ context.Context, id uuid.UUID, patch map[string]any) (*album.Photo, error) {
		gotPatch = patch
		return &album.Photo{ID: id, TripID: trip, Caption: patch["caption"].(string)}, nil
	}
	svc := newSvc(d)
	cap := "  hello  "
	dto, err := svc.UpdatePhoto(ctx, uuid.New(), photoID, album.UpdatePhotoPayload{Caption: &cap})
	require.NoError(t, err)
	assert.Equal(t, "hello", gotPatch["caption"])
	assert.Equal(t, "hello", dto.Caption)
	require.Len(t, d.bcast.events, 1)
}

func TestUpdatePhoto_NotFoundPassthrough(t *testing.T) {
	d := newDeps()
	d.repo.findPhoto = func(context.Context, uuid.UUID) (*album.Photo, error) { return nil, album.ErrPhotoNotFound }
	svc := newSvc(d)
	_, err := svc.UpdatePhoto(ctx, uuid.New(), uuid.New(), album.UpdatePhotoPayload{})
	assert.ErrorIs(t, err, album.ErrPhotoNotFound)
}

// ---- DeletePhoto ----

func TestDeletePhoto_CreatorHappyPath(t *testing.T) {
	actor := uuid.New()
	trip := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: trip, AddedBy: actor}, nil
	}
	// Membership check is triggered because actor == creator.
	var memberCalls int
	d.auth.isRoomMember = func(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
		memberCalls++
		return true, nil
	}
	svc := newSvc(d)
	require.NoError(t, svc.DeletePhoto(ctx, actor, uuid.New()))
	assert.Equal(t, 1, memberCalls)
	require.Len(t, d.bcast.events, 1)
	var env struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(d.bcast.events[0].payload, &env))
	assert.Equal(t, string(album.BroadcastPhotoDeleted), env.Type)
}

func TestDeletePhoto_TripOwnerCanDelete(t *testing.T) {
	actor := uuid.New()
	creator := uuid.New()
	trip := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: trip, AddedBy: creator}, nil
	}
	d.auth.findTripByID = func(context.Context, uuid.UUID) (*tripmod.Trip, error) {
		return &tripmod.Trip{ID: trip, OwnerID: actor}, nil
	}
	svc := newSvc(d)
	require.NoError(t, svc.DeletePhoto(ctx, actor, uuid.New()))
	require.Len(t, d.bcast.events, 1)
}

func TestDeletePhoto_AdminCanDelete(t *testing.T) {
	actor := uuid.New()
	creator := uuid.New()
	trip := uuid.New()
	roomID := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: trip, AddedBy: creator}, nil
	}
	d.auth.findTripByID = func(context.Context, uuid.UUID) (*tripmod.Trip, error) {
		return &tripmod.Trip{ID: trip, OwnerID: uuid.New()}, nil
	}
	d.auth.findRoomByTrip = func(context.Context, uuid.UUID) (*tripmod.Room, error) {
		return &tripmod.Room{ID: roomID, TripID: trip}, nil
	}
	d.auth.listMembers = func(context.Context, uuid.UUID) ([]tripmod.RoomMember, error) {
		return []tripmod.RoomMember{{UserID: actor, Role: tripmod.RoleAdmin}}, nil
	}
	svc := newSvc(d)
	require.NoError(t, svc.DeletePhoto(ctx, actor, uuid.New()))
}

func TestDeletePhoto_NonOwnerNonAdminForbidden(t *testing.T) {
	actor := uuid.New()
	creator := uuid.New()
	trip := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: trip, AddedBy: creator}, nil
	}
	d.auth.findTripByID = func(context.Context, uuid.UUID) (*tripmod.Trip, error) {
		return &tripmod.Trip{ID: trip, OwnerID: uuid.New()}, nil
	}
	d.auth.findRoomByTrip = func(context.Context, uuid.UUID) (*tripmod.Room, error) {
		return &tripmod.Room{ID: uuid.New(), TripID: trip}, nil
	}
	d.auth.listMembers = func(context.Context, uuid.UUID) ([]tripmod.RoomMember, error) {
		return []tripmod.RoomMember{{UserID: actor, Role: tripmod.RoleMember}}, nil
	}
	svc := newSvc(d)
	err := svc.DeletePhoto(ctx, actor, uuid.New())
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestDeletePhoto_MissingTripForbidden(t *testing.T) {
	actor := uuid.New()
	creator := uuid.New()
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: uuid.New(), AddedBy: creator}, nil
	}
	d.auth.findTripByID = func(context.Context, uuid.UUID) (*tripmod.Trip, error) { return nil, nil }
	svc := newSvc(d)
	err := svc.DeletePhoto(ctx, actor, uuid.New())
	assert.ErrorIs(t, err, album.ErrForbidden)
}

// ---- DownloadOriginal ----

func TestDownloadOriginal_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, TripID: uuid.New(), MediaID: uuid.New()}, nil
	}
	d.auth.isRoomMember = func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	_, err := svc.DownloadOriginal(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestDownloadOriginal_HappyPath(t *testing.T) {
	mediaID := uuid.New()
	expires := time.Now().Add(15 * time.Minute)
	d := newDeps()
	d.repo.findPhoto = func(_ context.Context, id uuid.UUID) (*album.Photo, error) {
		return &album.Photo{ID: id, MediaID: mediaID}, nil
	}
	d.media.presignRead = func(_ context.Context, _, a uuid.UUID) (*media.PresignedGetResponse, error) {
		assert.Equal(t, mediaID, a)
		return &media.PresignedGetResponse{URL: "http://original", ExpiresAt: expires}, nil
	}
	svc := newSvc(d)
	got, err := svc.DownloadOriginal(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "http://original", got.URL)
	assert.True(t, got.ExpiresAt.Equal(expires))
}

// ---- CreateShareToken ----

func TestCreateShareToken_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.auth.isRoomMember = func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	_, err := svc.CreateShareToken(ctx, uuid.New(), uuid.New(), album.CreateShareTokenPayload{})
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestCreateShareToken_PersistsHashOnly_ReturnsPlaintext(t *testing.T) {
	creator := uuid.New()
	trip := uuid.New()
	expires := time.Now().Add(24 * time.Hour).UTC()

	d := newDeps()
	var saved *album.ShareToken
	d.repo.createShareToken = func(_ context.Context, s *album.ShareToken) error {
		s.ID = uuid.New()
		saved = s
		return nil
	}
	svc := newSvc(d)
	resp, err := svc.CreateShareToken(ctx, creator, trip, album.CreateShareTokenPayload{ExpiresAt: &expires})
	require.NoError(t, err)
	require.NotNil(t, saved)
	require.NotEmpty(t, resp.Token)
	assert.Equal(t, trip, saved.TripID)
	assert.Equal(t, creator, saved.CreatedBy)
	require.NotNil(t, saved.ExpiresAt)
	assert.True(t, saved.ExpiresAt.Equal(expires))

	// Persisted value must be the SHA-256 of plaintext, not the plaintext itself.
	assert.NotContains(t, string(saved.TokenHash), resp.Token)
	sum := sha256.Sum256([]byte(resp.Token))
	assert.Equal(t, sum[:], saved.TokenHash)

	require.Len(t, d.bcast.events, 1)
	var env struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(d.bcast.events[0].payload, &env))
	assert.Equal(t, string(album.BroadcastShareCreated), env.Type)
}

// ---- ListShareTokens ----

func TestListShareTokens_ForbiddenWhenNotMember(t *testing.T) {
	d := newDeps()
	d.auth.isRoomMember = func(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return false, nil }
	svc := newSvc(d)
	_, err := svc.ListShareTokens(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestListShareTokens_HappyPath(t *testing.T) {
	trip := uuid.New()
	d := newDeps()
	d.repo.listShareTokensByTrip = func(_ context.Context, tid uuid.UUID) ([]album.ShareToken, error) {
		assert.Equal(t, trip, tid)
		return []album.ShareToken{{ID: uuid.New(), TripID: tid}, {ID: uuid.New(), TripID: tid}}, nil
	}
	svc := newSvc(d)
	got, err := svc.ListShareTokens(ctx, uuid.New(), trip)
	require.NoError(t, err)
	assert.Len(t, got, 2)
}

// ---- RevokeShareToken ----

func TestRevokeShareToken_CreatorHappyPath(t *testing.T) {
	actor := uuid.New()
	trip := uuid.New()
	tokenID := uuid.New()
	d := newDeps()
	d.repo.findShareToken = func(_ context.Context, id uuid.UUID) (*album.ShareToken, error) {
		return &album.ShareToken{ID: id, TripID: trip, CreatedBy: actor}, nil
	}
	var revoked bool
	d.repo.revokeShareToken = func(_ context.Context, id, by uuid.UUID) error {
		revoked = true
		assert.Equal(t, tokenID, id)
		assert.Equal(t, actor, by)
		return nil
	}
	svc := newSvc(d)
	require.NoError(t, svc.RevokeShareToken(ctx, actor, tokenID))
	assert.True(t, revoked)
	require.Len(t, d.bcast.events, 1)
	var env struct {
		Type string `json:"type"`
	}
	require.NoError(t, json.Unmarshal(d.bcast.events[0].payload, &env))
	assert.Equal(t, string(album.BroadcastShareRevoked), env.Type)
}

func TestRevokeShareToken_NonOwnerForbidden(t *testing.T) {
	d := newDeps()
	d.repo.findShareToken = func(_ context.Context, id uuid.UUID) (*album.ShareToken, error) {
		return &album.ShareToken{ID: id, TripID: uuid.New(), CreatedBy: uuid.New()}, nil
	}
	d.auth.findTripByID = func(context.Context, uuid.UUID) (*tripmod.Trip, error) {
		return &tripmod.Trip{OwnerID: uuid.New()}, nil
	}
	d.auth.findRoomByTrip = func(context.Context, uuid.UUID) (*tripmod.Room, error) {
		return &tripmod.Room{ID: uuid.New()}, nil
	}
	d.auth.listMembers = func(context.Context, uuid.UUID) ([]tripmod.RoomMember, error) { return nil, nil }
	svc := newSvc(d)
	err := svc.RevokeShareToken(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, album.ErrForbidden)
}

func TestRevokeShareToken_NotFound(t *testing.T) {
	d := newDeps()
	d.repo.findShareToken = func(context.Context, uuid.UUID) (*album.ShareToken, error) {
		return nil, album.ErrShareTokenNotFound
	}
	svc := newSvc(d)
	err := svc.RevokeShareToken(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, album.ErrShareTokenNotFound)
}

// ---- ResolvePublicAlbum ----

func TestResolvePublicAlbum_EmptyToken(t *testing.T) {
	d := newDeps()
	svc := newSvc(d)
	_, _, err := svc.ResolvePublicAlbum(ctx, "")
	assert.ErrorIs(t, err, album.ErrShareTokenNotFound)
}

func TestResolvePublicAlbum_HashLookupMiss(t *testing.T) {
	d := newDeps()
	d.repo.findShareTokenByHash = func(context.Context, []byte) (*album.ShareToken, error) {
		return nil, album.ErrShareTokenNotFound
	}
	svc := newSvc(d)
	_, _, err := svc.ResolvePublicAlbum(ctx, "opaque")
	assert.ErrorIs(t, err, album.ErrShareTokenNotFound)
}

func TestResolvePublicAlbum_Revoked(t *testing.T) {
	revokedAt := time.Now().UTC()
	d := newDeps()
	d.repo.findShareTokenByHash = func(context.Context, []byte) (*album.ShareToken, error) {
		return &album.ShareToken{ID: uuid.New(), TripID: uuid.New(), RevokedAt: &revokedAt}, nil
	}
	svc := newSvc(d)
	_, _, err := svc.ResolvePublicAlbum(ctx, "opaque")
	assert.ErrorIs(t, err, album.ErrShareTokenInactive)
}

func TestResolvePublicAlbum_Expired(t *testing.T) {
	past := time.Now().UTC().Add(-time.Hour)
	d := newDeps()
	d.repo.findShareTokenByHash = func(context.Context, []byte) (*album.ShareToken, error) {
		return &album.ShareToken{ID: uuid.New(), TripID: uuid.New(), ExpiresAt: &past}, nil
	}
	svc := newSvc(d)
	_, _, err := svc.ResolvePublicAlbum(ctx, "opaque")
	assert.ErrorIs(t, err, album.ErrShareExpired)
}

func TestResolvePublicAlbum_PresignsUnderCreatorID(t *testing.T) {
	creator := uuid.New()
	trip := uuid.New()
	tokenID := uuid.New()
	d := newDeps()
	d.repo.findShareTokenByHash = func(_ context.Context, hash []byte) (*album.ShareToken, error) {
		want := sha256.Sum256([]byte("opaque"))
		assert.Equal(t, want[:], hash)
		return &album.ShareToken{ID: tokenID, TripID: trip, CreatedBy: creator}, nil
	}
	d.repo.listPhotosByTrip = func(_ context.Context, tid uuid.UUID) ([]album.Photo, error) {
		assert.Equal(t, trip, tid)
		return []album.Photo{{ID: uuid.New(), MediaID: uuid.New()}}, nil
	}
	var presignActor uuid.UUID
	d.media.presignRead = func(_ context.Context, u, a uuid.UUID) (*media.PresignedGetResponse, error) {
		presignActor = u
		return &media.PresignedGetResponse{URL: "http://x/" + a.String()}, nil
	}
	svc := newSvc(d)
	photos, dto, err := svc.ResolvePublicAlbum(ctx, "opaque")
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, tokenID, dto.ID)
	assert.Len(t, photos, 1)
	assert.Equal(t, creator, presignActor)
}

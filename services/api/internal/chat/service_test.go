package chat_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/chat"
	"github.com/Ans1110/trip-app/internal/media"
	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ctx = context.Background()

// ---- chatRepoMock: implements chat.IRepository ----

type chatRepoMock struct {
	findRoomByID          func(context.Context, uuid.UUID) (*chat.Room, error)
	findGroupRoomByTripID func(context.Context, uuid.UUID) (*chat.Room, error)
	ensureGroupRoom       func(context.Context, uuid.UUID, string, []uuid.UUID) (*chat.Room, error)
	findOrCreateDMRoom    func(context.Context, uuid.UUID, uuid.UUID) (*chat.Room, error)
	addMembers            func(context.Context, uuid.UUID, []uuid.UUID) error
	isRoomMember          func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	listRoomMemberIDs     func(context.Context, uuid.UUID) ([]uuid.UUID, error)
	listRoomsForUser      func(context.Context, uuid.UUID) ([]chat.Room, error)
	listDMPeers           func(context.Context, []uuid.UUID, uuid.UUID) (map[uuid.UUID]uuid.UUID, error)
	hideDMForUser         func(context.Context, uuid.UUID, uuid.UUID) error
	unhideRoomForUser     func(context.Context, uuid.UUID, uuid.UUID) error
	unhideRooms           func(context.Context, []uuid.UUID) error
	insertMessages        func(context.Context, []*chat.Message) error
	listMessages          func(context.Context, uuid.UUID, *time.Time, int) ([]chat.Message, error)
	lastMessages          func(context.Context, []uuid.UUID) (map[uuid.UUID]chat.Message, error)
	upsertReadReceipt     func(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (*chat.ReadReceipt, error)
	listReadReceipts      func(context.Context, uuid.UUID) ([]chat.ReadReceipt, error)
	countUnread           func(context.Context, uuid.UUID, uuid.UUID) (int, error)
}

func (r *chatRepoMock) FindRoomByID(c context.Context, id uuid.UUID) (*chat.Room, error) {
	if r.findRoomByID != nil {
		return r.findRoomByID(c, id)
	}
	return nil, chat.ErrRoomNotFound
}

func (r *chatRepoMock) FindGroupRoomByTripID(c context.Context, tid uuid.UUID) (*chat.Room, error) {
	if r.findGroupRoomByTripID != nil {
		return r.findGroupRoomByTripID(c, tid)
	}
	return nil, chat.ErrRoomNotFound
}

func (r *chatRepoMock) EnsureGroupRoom(c context.Context, tid uuid.UUID, name string, ids []uuid.UUID) (*chat.Room, error) {
	if r.ensureGroupRoom != nil {
		return r.ensureGroupRoom(c, tid, name, ids)
	}
	return &chat.Room{ID: uuid.New(), TripID: &tid, Name: name, Type: chat.RoomGroup}, nil
}

func (r *chatRepoMock) FindOrCreateDMRoom(c context.Context, a, b uuid.UUID) (*chat.Room, error) {
	if r.findOrCreateDMRoom != nil {
		return r.findOrCreateDMRoom(c, a, b)
	}
	return &chat.Room{ID: uuid.New(), Type: chat.RoomDM}, nil
}

func (r *chatRepoMock) AddMembers(c context.Context, roomID uuid.UUID, ids []uuid.UUID) error {
	if r.addMembers != nil {
		return r.addMembers(c, roomID, ids)
	}
	return nil
}

func (r *chatRepoMock) IsRoomMember(c context.Context, roomID, userID uuid.UUID) (bool, error) {
	if r.isRoomMember != nil {
		return r.isRoomMember(c, roomID, userID)
	}
	return false, nil
}

func (r *chatRepoMock) ListRoomMemberIDs(c context.Context, roomID uuid.UUID) ([]uuid.UUID, error) {
	if r.listRoomMemberIDs != nil {
		return r.listRoomMemberIDs(c, roomID)
	}
	return nil, nil
}

func (r *chatRepoMock) ListRoomsForUser(c context.Context, userID uuid.UUID) ([]chat.Room, error) {
	if r.listRoomsForUser != nil {
		return r.listRoomsForUser(c, userID)
	}
	return nil, nil
}

func (r *chatRepoMock) ListDMPeers(c context.Context, roomIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
	if r.listDMPeers != nil {
		return r.listDMPeers(c, roomIDs, viewerID)
	}
	return map[uuid.UUID]uuid.UUID{}, nil
}

func (r *chatRepoMock) HideDMForUser(c context.Context, roomID, userID uuid.UUID) error {
	if r.hideDMForUser != nil {
		return r.hideDMForUser(c, roomID, userID)
	}
	return nil
}

func (r *chatRepoMock) UnhideRoomForUser(c context.Context, roomID, userID uuid.UUID) error {
	if r.unhideRoomForUser != nil {
		return r.unhideRoomForUser(c, roomID, userID)
	}
	return nil
}

func (r *chatRepoMock) UnhideRooms(c context.Context, roomIDs []uuid.UUID) error {
	if r.unhideRooms != nil {
		return r.unhideRooms(c, roomIDs)
	}
	return nil
}

func (r *chatRepoMock) InsertMessages(c context.Context, msgs []*chat.Message) error {
	if r.insertMessages != nil {
		return r.insertMessages(c, msgs)
	}
	return nil
}

func (r *chatRepoMock) ListMessages(c context.Context, roomID uuid.UUID, before *time.Time, limit int) ([]chat.Message, error) {
	if r.listMessages != nil {
		return r.listMessages(c, roomID, before, limit)
	}
	return nil, nil
}

func (r *chatRepoMock) LastMessages(c context.Context, roomIDs []uuid.UUID) (map[uuid.UUID]chat.Message, error) {
	if r.lastMessages != nil {
		return r.lastMessages(c, roomIDs)
	}
	return map[uuid.UUID]chat.Message{}, nil
}

func (r *chatRepoMock) UpsertReadReceipt(c context.Context, roomID, userID, msgID uuid.UUID) (*chat.ReadReceipt, error) {
	if r.upsertReadReceipt != nil {
		return r.upsertReadReceipt(c, roomID, userID, msgID)
	}
	return &chat.ReadReceipt{RoomID: roomID, UserID: userID, LastReadID: msgID, UpdatedAt: time.Now()}, nil
}

func (r *chatRepoMock) ListReadReceipts(c context.Context, roomID uuid.UUID) ([]chat.ReadReceipt, error) {
	if r.listReadReceipts != nil {
		return r.listReadReceipts(c, roomID)
	}
	return nil, nil
}

func (r *chatRepoMock) CountUnread(c context.Context, roomID, userID uuid.UUID) (int, error) {
	if r.countUnread != nil {
		return r.countUnread(c, roomID, userID)
	}
	return 0, nil
}

// ---- tripRepoStub: implements trip.IRepository (only exposes hooks the
// chat service actually calls: IsRoomMember, FindTripByID, FindUsersByIDs) ----

type tripRepoStub struct {
	isRoomMember   func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	findTripByID   func(context.Context, uuid.UUID) (*trip.Trip, error)
	findUsersByIDs func(context.Context, []uuid.UUID) (map[uuid.UUID]auth.User, error)
}

func (r *tripRepoStub) WithTx(_ context.Context, fn func(trip.IRepository) error) error {
	return fn(r)
}
func (r *tripRepoStub) Tx() *gorm.DB { return nil }

func (r *tripRepoStub) FindUserByID(_ context.Context, _ uuid.UUID) (*auth.User, error) {
	return nil, nil
}
func (r *tripRepoStub) FindUsersByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	if r.findUsersByIDs != nil {
		return r.findUsersByIDs(c, ids)
	}
	return map[uuid.UUID]auth.User{}, nil
}

func (r *tripRepoStub) CreateTrip(_ context.Context, _ *trip.Trip) error { return nil }
func (r *tripRepoStub) FindTripByID(c context.Context, id uuid.UUID) (*trip.Trip, error) {
	if r.findTripByID != nil {
		return r.findTripByID(c, id)
	}
	return nil, nil
}
func (r *tripRepoStub) UpdateTrip(_ context.Context, _ uuid.UUID, _ map[string]any) error { return nil }
func (r *tripRepoStub) DeleteTrip(_ context.Context, _ uuid.UUID) error                   { return nil }
func (r *tripRepoStub) ListTrips(_ context.Context, _ uuid.UUID, _ trip.ListTripsQuery) ([]trip.Trip, error) {
	return nil, nil
}
func (r *tripRepoStub) CountMembers(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{}, nil
}

func (r *tripRepoStub) CreateRoom(_ context.Context, _ *trip.Room) error       { return nil }
func (r *tripRepoStub) FindRoomByID(_ context.Context, _ uuid.UUID) (*trip.Room, error) {
	return nil, nil
}
func (r *tripRepoStub) FindRoomByTripID(_ context.Context, _ uuid.UUID) (*trip.Room, error) {
	return nil, nil
}
func (r *tripRepoStub) FindRoomByCode(_ context.Context, _ string) (*trip.Room, error) {
	return nil, nil
}
func (r *tripRepoStub) UpdateRoomCode(_ context.Context, _ uuid.UUID, _ string) error { return nil }
func (r *tripRepoStub) RoomCodeExists(_ context.Context, _ string) (bool, error)      { return false, nil }

func (r *tripRepoStub) AddMember(_ context.Context, _ *trip.RoomMember) error { return nil }
func (r *tripRepoStub) RemoveMember(_ context.Context, _, _ uuid.UUID) error  { return nil }
func (r *tripRepoStub) FindMember(_ context.Context, _, _ uuid.UUID) (*trip.RoomMember, error) {
	return nil, nil
}
func (r *tripRepoStub) ListMembers(_ context.Context, _ uuid.UUID) ([]trip.RoomMember, error) {
	return nil, nil
}
func (r *tripRepoStub) FindMemberByTrip(_ context.Context, _, _ uuid.UUID) (*trip.RoomMember, error) {
	return nil, nil
}
func (r *tripRepoStub) CountRoomMembers(_ context.Context, _ uuid.UUID) (int, error) { return 0, nil }
func (r *tripRepoStub) IsRoomMember(c context.Context, tid, uid uuid.UUID) (bool, error) {
	if r.isRoomMember != nil {
		return r.isRoomMember(c, tid, uid)
	}
	return false, nil
}

func (r *tripRepoStub) CreateItinerary(_ context.Context, _ *trip.Itinerary) error { return nil }
func (r *tripRepoStub) FindItineraryByID(_ context.Context, _ uuid.UUID) (*trip.Itinerary, error) {
	return nil, nil
}
func (r *tripRepoStub) UpdateItinerary(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (r *tripRepoStub) UpdateItineraryWithVersion(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *trip.EventMeta) (*trip.Itinerary, error) {
	return nil, nil
}
func (r *tripRepoStub) DeleteItinerary(_ context.Context, _ uuid.UUID) error { return nil }
func (r *tripRepoStub) ListItinerary(_ context.Context, _ uuid.UUID) ([]trip.Itinerary, error) {
	return nil, nil
}
func (r *tripRepoStub) ReorderItinerary(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *tripRepoStub) CreateTodo(_ context.Context, _ *trip.Todo) error { return nil }
func (r *tripRepoStub) FindTodoByID(_ context.Context, _ uuid.UUID) (*trip.Todo, error) {
	return nil, nil
}
func (r *tripRepoStub) UpdateTodo(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (r *tripRepoStub) UpdateTodoWithVersion(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *trip.EventMeta) (*trip.Todo, error) {
	return nil, nil
}
func (r *tripRepoStub) DeleteTodo(_ context.Context, _ uuid.UUID) error { return nil }
func (r *tripRepoStub) ListTodos(_ context.Context, _ uuid.UUID) ([]trip.Todo, error) {
	return nil, nil
}
func (r *tripRepoStub) ReorderTodos(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *tripRepoStub) EnqueueOutbox(_ context.Context, _ *trip.Outbox) error         { return nil }
func (r *tripRepoStub) ClaimOutbox(_ context.Context, _ int) ([]trip.Outbox, error)   { return nil, nil }
func (r *tripRepoStub) MarkOutboxDispatched(_ context.Context, _ []uuid.UUID) error   { return nil }
func (r *tripRepoStub) RecordOutboxFailure(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (r *tripRepoStub) PruneDispatchedOutbox(_ context.Context, _ time.Time, _ int) (int64, error) {
	return 0, nil
}

// ---- mediaAuthzMock ----

type mediaAuthzMock struct {
	authorize func(context.Context, uuid.UUID, uuid.UUID) (*media.Asset, error)
}

func (m *mediaAuthzMock) AuthorizeAssetForOwner(c context.Context, userID, assetID uuid.UUID) (*media.Asset, error) {
	if m.authorize != nil {
		return m.authorize(c, userID, assetID)
	}
	return nil, media.ErrAssetNotFound
}

// ---- helpers ----

// newSvc wires the chat service with a real (empty) Hub and a fast-fail Redis
// client. The empty Hub makes local broadcasts no-ops; the bogus Redis address
// makes Publish return an error which the service discards. Neither can panic.
func newSvc(repo chat.IRepository, tripRepo trip.IRepository, mz chat.MediaAuthorizer) *chat.Service {
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 10 * time.Millisecond,
		MaxRetries:  -1,
	})
	return chat.NewService(chat.ServiceConfig{
		Repo:       repo,
		TripRepo:   tripRepo,
		MediaAuthz: mz,
		Hub:        chat.NewHub(zap.NewNop()),
		Redis:      rdb,
		Logger:     zap.NewNop(),
		InstanceID: "test-instance",
	})
}

// ---- Authorize ----

func TestAuthorize_DelegatesToRepo(t *testing.T) {
	room := uuid.New()
	user := uuid.New()
	var gotRoom, gotUser uuid.UUID
	repo := &chatRepoMock{
		isRoomMember: func(_ context.Context, r, u uuid.UUID) (bool, error) {
			gotRoom, gotUser = r, u
			return true, nil
		},
	}
	svc := newSvc(repo, nil, nil)

	ok, err := svc.Authorize(ctx, user, room)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, room, gotRoom)
	assert.Equal(t, user, gotUser)
}

func TestAuthorize_PropagatesError(t *testing.T) {
	repo := &chatRepoMock{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, errors.New("boom")
		},
	}
	svc := newSvc(repo, nil, nil)

	_, err := svc.Authorize(ctx, uuid.New(), uuid.New())
	assert.EqualError(t, err, "boom")
}

// ---- EnsureTripRoom ----

func TestEnsureTripRoom_NilTripRepo_ReturnsError(t *testing.T) {
	svc := newSvc(&chatRepoMock{}, nil, nil)
	_, err := svc.EnsureTripRoom(ctx, uuid.New(), uuid.New())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "trip repo not configured")
}

func TestEnsureTripRoom_NonMember_Forbidden(t *testing.T) {
	svc := newSvc(&chatRepoMock{}, &tripRepoStub{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}, nil)

	_, err := svc.EnsureTripRoom(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, chat.ErrForbidden)
}

func TestEnsureTripRoom_MembershipCheckError_Propagates(t *testing.T) {
	svc := newSvc(&chatRepoMock{}, &tripRepoStub{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, errors.New("db down")
		},
	}, nil)

	_, err := svc.EnsureTripRoom(ctx, uuid.New(), uuid.New())
	assert.EqualError(t, err, "db down")
}

func TestEnsureTripRoom_HappyPath_CallsEnsureGroupRoomWithTripMeta(t *testing.T) {
	tripID := uuid.New()
	ownerID := uuid.New()
	userID := uuid.New()

	var gotTripID uuid.UUID
	var gotName string
	var gotSeeds []uuid.UUID

	chatRepo := &chatRepoMock{
		ensureGroupRoom: func(_ context.Context, tid uuid.UUID, name string, seeds []uuid.UUID) (*chat.Room, error) {
			gotTripID, gotName, gotSeeds = tid, name, seeds
			return &chat.Room{ID: uuid.New(), TripID: &tid, Name: name, Type: chat.RoomGroup}, nil
		},
	}
	tripRepo := &tripRepoStub{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },
		findTripByID: func(_ context.Context, id uuid.UUID) (*trip.Trip, error) {
			return &trip.Trip{ID: id, OwnerID: ownerID, Title: "Kyoto '26"}, nil
		},
	}
	svc := newSvc(chatRepo, tripRepo, nil)

	room, err := svc.EnsureTripRoom(ctx, tripID, userID)
	require.NoError(t, err)
	require.NotNil(t, room)
	assert.Equal(t, chat.RoomGroup, room.Type)
	assert.Equal(t, tripID, gotTripID)
	assert.Equal(t, "Kyoto '26", gotName)
	assert.Equal(t, []uuid.UUID{ownerID}, gotSeeds)
}

// ---- EnsureDM ----

func TestEnsureDM_SelfDM_Rejected(t *testing.T) {
	me := uuid.New()
	svc := newSvc(&chatRepoMock{}, nil, nil)
	_, err := svc.EnsureDM(ctx, me, me)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cannot DM yourself")
}

func TestEnsureDM_CreatesRoomUnhidesAndAttachesPeer(t *testing.T) {
	me := uuid.New()
	peer := uuid.New()
	roomID := uuid.New()

	var unhidRoom, unhidUser uuid.UUID

	chatRepo := &chatRepoMock{
		findOrCreateDMRoom: func(_ context.Context, a, b uuid.UUID) (*chat.Room, error) {
			return &chat.Room{ID: roomID, Type: chat.RoomDM, Name: ""}, nil
		},
		unhideRoomForUser: func(_ context.Context, rid, uid uuid.UUID) error {
			unhidRoom, unhidUser = rid, uid
			return nil
		},
		listDMPeers: func(_ context.Context, roomIDs []uuid.UUID, viewerID uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
			assert.Equal(t, []uuid.UUID{roomID}, roomIDs)
			assert.Equal(t, me, viewerID)
			return map[uuid.UUID]uuid.UUID{roomID: peer}, nil
		},
	}
	tripRepo := &tripRepoStub{
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{
				peer: {ID: peer, Name: "Peer", Email: "peer@example.com", AvatarURL: "https://cdn/x.png"},
			}, nil
		},
	}
	svc := newSvc(chatRepo, tripRepo, nil)

	dto, err := svc.EnsureDM(ctx, me, peer)
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, roomID, dto.ID)
	assert.Equal(t, chat.RoomDM, dto.Type)
	assert.Equal(t, roomID, unhidRoom)
	assert.Equal(t, me, unhidUser)
	require.NotNil(t, dto.Peer)
	assert.Equal(t, peer, dto.Peer.ID)
	assert.Equal(t, "Peer", dto.Peer.Name)
	assert.Equal(t, "peer@example.com", dto.Peer.Email)
	assert.Equal(t, "https://cdn/x.png", dto.Peer.AvatarURL)
}

func TestEnsureDM_UnhideFailure_IsNonFatal(t *testing.T) {
	me := uuid.New()
	peer := uuid.New()

	chatRepo := &chatRepoMock{
		findOrCreateDMRoom: func(_ context.Context, _, _ uuid.UUID) (*chat.Room, error) {
			return &chat.Room{ID: uuid.New(), Type: chat.RoomDM}, nil
		},
		unhideRoomForUser: func(_ context.Context, _, _ uuid.UUID) error {
			return errors.New("transient")
		},
	}
	svc := newSvc(chatRepo, nil, nil)

	dto, err := svc.EnsureDM(ctx, me, peer)
	require.NoError(t, err, "unhide error must not fail the ensure call")
	require.NotNil(t, dto)
}

func TestEnsureDM_FindOrCreateError_Propagates(t *testing.T) {
	chatRepo := &chatRepoMock{
		findOrCreateDMRoom: func(_ context.Context, _, _ uuid.UUID) (*chat.Room, error) {
			return nil, errors.New("db lock")
		},
	}
	svc := newSvc(chatRepo, nil, nil)

	_, err := svc.EnsureDM(ctx, uuid.New(), uuid.New())
	assert.EqualError(t, err, "db lock")
}

// ---- ListRooms ----

func TestListRooms_Empty_ReturnsEmptySlice(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		listRoomsForUser: func(_ context.Context, _ uuid.UUID) ([]chat.Room, error) {
			return nil, nil
		},
	}, nil, nil)

	out, err := svc.ListRooms(ctx, uuid.New())
	require.NoError(t, err)
	assert.NotNil(t, out, "must return non-nil so JSON is [] not null")
	assert.Len(t, out, 0)
}

func TestListRooms_HydratesPeerLastMessageAndUnread(t *testing.T) {
	me := uuid.New()
	groupID := uuid.New()
	dmID := uuid.New()
	peerID := uuid.New()
	tripID := uuid.New()
	lastMsgID := uuid.New()

	rooms := []chat.Room{
		{ID: groupID, TripID: &tripID, Name: "Trip", Type: chat.RoomGroup, CreatedAt: time.Now()},
		{ID: dmID, Name: "", Type: chat.RoomDM, CreatedAt: time.Now()},
	}
	lastMsg := chat.Message{
		ID:        lastMsgID,
		RoomID:    dmID,
		SenderID:  peerID,
		Content:   "hi",
		Type:      chat.MessageText,
		CreatedAt: time.Now(),
	}

	chatRepo := &chatRepoMock{
		listRoomsForUser: func(_ context.Context, uid uuid.UUID) ([]chat.Room, error) {
			assert.Equal(t, me, uid)
			return rooms, nil
		},
		lastMessages: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]chat.Message, error) {
			assert.ElementsMatch(t, []uuid.UUID{groupID, dmID}, ids)
			return map[uuid.UUID]chat.Message{dmID: lastMsg}, nil
		},
		listDMPeers: func(_ context.Context, dmIDs []uuid.UUID, viewer uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
			assert.Equal(t, []uuid.UUID{dmID}, dmIDs, "only DM rooms should be resolved")
			assert.Equal(t, me, viewer)
			return map[uuid.UUID]uuid.UUID{dmID: peerID}, nil
		},
		countUnread: func(_ context.Context, roomID, uid uuid.UUID) (int, error) {
			if roomID == dmID {
				return 3, nil
			}
			return 0, nil
		},
	}
	tripRepo := &tripRepoStub{
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{peerID: {ID: peerID, Name: "Peer"}}, nil
		},
	}
	svc := newSvc(chatRepo, tripRepo, nil)

	out, err := svc.ListRooms(ctx, me)
	require.NoError(t, err)
	require.Len(t, out, 2)

	// Preserve input order.
	assert.Equal(t, groupID, out[0].ID)
	assert.Equal(t, chat.RoomGroup, out[0].Type)
	assert.Nil(t, out[0].Peer, "group room has no peer")
	assert.Nil(t, out[0].LastMessage)
	assert.Equal(t, 0, out[0].UnreadCount)

	assert.Equal(t, dmID, out[1].ID)
	assert.Equal(t, chat.RoomDM, out[1].Type)
	require.NotNil(t, out[1].Peer)
	assert.Equal(t, peerID, out[1].Peer.ID)
	require.NotNil(t, out[1].LastMessage)
	assert.Equal(t, lastMsgID, out[1].LastMessage.ID)
	assert.Equal(t, 3, out[1].UnreadCount)
}

func TestListRooms_ListError_Propagates(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		listRoomsForUser: func(_ context.Context, _ uuid.UUID) ([]chat.Room, error) {
			return nil, errors.New("boom")
		},
	}, nil, nil)

	_, err := svc.ListRooms(ctx, uuid.New())
	assert.EqualError(t, err, "boom")
}

func TestListRooms_LastMessagesError_Propagates(t *testing.T) {
	rooms := []chat.Room{{ID: uuid.New(), Type: chat.RoomGroup}}
	svc := newSvc(&chatRepoMock{
		listRoomsForUser: func(_ context.Context, _ uuid.UUID) ([]chat.Room, error) {
			return rooms, nil
		},
		lastMessages: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]chat.Message, error) {
			return nil, errors.New("scan fail")
		},
	}, nil, nil)

	_, err := svc.ListRooms(ctx, uuid.New())
	assert.EqualError(t, err, "scan fail")
}

func TestListRooms_PeerResolveError_IsNonFatal(t *testing.T) {
	me := uuid.New()
	dmID := uuid.New()
	rooms := []chat.Room{{ID: dmID, Type: chat.RoomDM}}

	svc := newSvc(&chatRepoMock{
		listRoomsForUser: func(_ context.Context, _ uuid.UUID) ([]chat.Room, error) {
			return rooms, nil
		},
		listDMPeers: func(_ context.Context, _ []uuid.UUID, _ uuid.UUID) (map[uuid.UUID]uuid.UUID, error) {
			return nil, errors.New("peer lookup fail")
		},
	}, &tripRepoStub{}, nil)

	out, err := svc.ListRooms(ctx, me)
	require.NoError(t, err, "peer resolution failure must not fail the list")
	require.Len(t, out, 1)
	assert.Nil(t, out[0].Peer)
}

// ---- HideRoom ----

func TestHideRoom_NonMember_Forbidden(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}, nil, nil)

	err := svc.HideRoom(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, chat.ErrForbidden)
}

func TestHideRoom_MemberCallsHide(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	var hidRoom, hidUser uuid.UUID

	svc := newSvc(&chatRepoMock{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },
		hideDMForUser: func(_ context.Context, r, u uuid.UUID) error {
			hidRoom, hidUser = r, u
			return nil
		},
	}, nil, nil)

	require.NoError(t, svc.HideRoom(ctx, roomID, userID))
	assert.Equal(t, roomID, hidRoom)
	assert.Equal(t, userID, hidUser)
}

func TestHideRoom_MembershipCheckError_Propagates(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, errors.New("db down")
		},
	}, nil, nil)

	err := svc.HideRoom(ctx, uuid.New(), uuid.New())
	assert.EqualError(t, err, "db down")
}

func TestHideRoom_HideNotAllowed_PropagatesFromRepo(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		isRoomMember:  func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },
		hideDMForUser: func(_ context.Context, _, _ uuid.UUID) error { return chat.ErrHideNotAllowed },
	}, nil, nil)

	err := svc.HideRoom(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, chat.ErrHideNotAllowed)
}

// ---- ListMessages ----

func TestListMessages_DelegatesAndConvertsDTOs(t *testing.T) {
	roomID := uuid.New()
	before := time.Now()
	m1 := chat.Message{ID: uuid.New(), RoomID: roomID, SenderID: uuid.New(), Content: "a", Type: chat.MessageText, CreatedAt: time.Now()}
	m2 := chat.Message{ID: uuid.New(), RoomID: roomID, SenderID: uuid.New(), Content: "b", Type: chat.MessageText, CreatedAt: time.Now()}

	var gotBefore *time.Time
	var gotLimit int

	svc := newSvc(&chatRepoMock{
		listMessages: func(_ context.Context, r uuid.UUID, b *time.Time, l int) ([]chat.Message, error) {
			assert.Equal(t, roomID, r)
			gotBefore, gotLimit = b, l
			return []chat.Message{m1, m2}, nil
		},
	}, nil, nil)

	out, err := svc.ListMessages(ctx, roomID, &before, 25)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, m1.ID, out[0].ID)
	assert.Equal(t, m2.ID, out[1].ID)
	assert.Equal(t, &before, gotBefore)
	assert.Equal(t, 25, gotLimit)
}

func TestListMessages_RepoError_Propagates(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		listMessages: func(_ context.Context, _ uuid.UUID, _ *time.Time, _ int) ([]chat.Message, error) {
			return nil, errors.New("query fail")
		},
	}, nil, nil)

	_, err := svc.ListMessages(ctx, uuid.New(), nil, 10)
	assert.EqualError(t, err, "query fail")
}

// ---- MarkRead ----

func TestMarkRead_UpsertsAndReturnsDTO(t *testing.T) {
	roomID := uuid.New()
	userID := uuid.New()
	msgID := uuid.New()
	updated := time.Now().UTC()

	var gotRoom, gotUser, gotMsg uuid.UUID

	svc := newSvc(&chatRepoMock{
		upsertReadReceipt: func(_ context.Context, r, u, m uuid.UUID) (*chat.ReadReceipt, error) {
			gotRoom, gotUser, gotMsg = r, u, m
			return &chat.ReadReceipt{RoomID: r, UserID: u, LastReadID: m, UpdatedAt: updated}, nil
		},
	}, nil, nil)

	dto, err := svc.MarkRead(ctx, roomID, userID, msgID)
	require.NoError(t, err)
	require.NotNil(t, dto)
	assert.Equal(t, roomID, dto.RoomID)
	assert.Equal(t, userID, dto.UserID)
	assert.Equal(t, msgID, dto.LastReadID)
	assert.Equal(t, updated, dto.UpdatedAt)
	assert.Equal(t, roomID, gotRoom)
	assert.Equal(t, userID, gotUser)
	assert.Equal(t, msgID, gotMsg)
}

func TestMarkRead_UpsertError_Propagates(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		upsertReadReceipt: func(_ context.Context, _, _, _ uuid.UUID) (*chat.ReadReceipt, error) {
			return nil, errors.New("conflict")
		},
	}, nil, nil)

	_, err := svc.MarkRead(ctx, uuid.New(), uuid.New(), uuid.New())
	assert.EqualError(t, err, "conflict")
}

// ---- ListReadReceipts ----

func TestListReadReceipts_ConvertsRowsToDTO(t *testing.T) {
	roomID := uuid.New()
	updated := time.Now().UTC()
	rows := []chat.ReadReceipt{
		{RoomID: roomID, UserID: uuid.New(), LastReadID: uuid.New(), UpdatedAt: updated},
		{RoomID: roomID, UserID: uuid.New(), LastReadID: uuid.New(), UpdatedAt: updated},
	}

	svc := newSvc(&chatRepoMock{
		listReadReceipts: func(_ context.Context, r uuid.UUID) ([]chat.ReadReceipt, error) {
			assert.Equal(t, roomID, r)
			return rows, nil
		},
	}, nil, nil)

	out, err := svc.ListReadReceipts(ctx, roomID)
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, rows[0].UserID, out[0].UserID)
	assert.Equal(t, rows[1].LastReadID, out[1].LastReadID)
}

func TestListReadReceipts_Empty(t *testing.T) {
	svc := newSvc(&chatRepoMock{}, nil, nil)
	out, err := svc.ListReadReceipts(ctx, uuid.New())
	require.NoError(t, err)
	assert.Len(t, out, 0)
}

func TestListReadReceipts_RepoError_Propagates(t *testing.T) {
	svc := newSvc(&chatRepoMock{
		listReadReceipts: func(_ context.Context, _ uuid.UUID) ([]chat.ReadReceipt, error) {
			return nil, errors.New("scan fail")
		},
	}, nil, nil)

	_, err := svc.ListReadReceipts(ctx, uuid.New())
	assert.EqualError(t, err, "scan fail")
}

// silence unused-mocks warnings for stubs we intentionally don't exercise.
var _ chat.MediaAuthorizer = (*mediaAuthzMock)(nil)

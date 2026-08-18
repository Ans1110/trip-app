package realtime

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var ctx = context.Background()

// ---- tripRepoStub: implements trip.IRepository with hooks only for the
// methods realtime actually calls: IsRoomMember, FindItineraryByID,
// FindTodoByID, UpdateItineraryWithVersion, UpdateTodoWithVersion ----

type tripRepoStub struct {
	isRoomMember          func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	findItineraryByID     func(context.Context, uuid.UUID) (*trip.Itinerary, error)
	findTodoByID          func(context.Context, uuid.UUID) (*trip.Todo, error)
	updateItineraryWithV  func(context.Context, uuid.UUID, int, map[string]any, *trip.EventMeta) (*trip.Itinerary, error)
	updateTodoWithVersion func(context.Context, uuid.UUID, int, map[string]any, *trip.EventMeta) (*trip.Todo, error)
}

func (r *tripRepoStub) WithTx(_ context.Context, fn func(trip.IRepository) error) error {
	return fn(r)
}
func (r *tripRepoStub) Tx() *gorm.DB { return nil }

func (r *tripRepoStub) FindUserByID(_ context.Context, _ uuid.UUID) (*auth.User, error) {
	return nil, nil
}
func (r *tripRepoStub) FindUsersByIDs(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	return map[uuid.UUID]auth.User{}, nil
}

func (r *tripRepoStub) CreateTrip(_ context.Context, _ *trip.Trip) error { return nil }
func (r *tripRepoStub) FindTripByID(_ context.Context, _ uuid.UUID) (*trip.Trip, error) {
	return nil, nil
}
func (r *tripRepoStub) UpdateTrip(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (r *tripRepoStub) DeleteTrip(_ context.Context, _ uuid.UUID) error { return nil }
func (r *tripRepoStub) ListTrips(_ context.Context, _ uuid.UUID, _ trip.ListTripsQuery) ([]trip.Trip, error) {
	return nil, nil
}
func (r *tripRepoStub) CountMembers(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
	return map[uuid.UUID]int{}, nil
}
func (r *tripRepoStub) TripOwnerID(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (r *tripRepoStub) CreateRoom(_ context.Context, _ *trip.Room) error { return nil }
func (r *tripRepoStub) FindRoomByID(_ context.Context, _ uuid.UUID) (*trip.Room, error) {
	return nil, nil
}
func (r *tripRepoStub) FindRoomByTripID(_ context.Context, _ uuid.UUID) (*trip.Room, error) {
	return nil, nil
}
func (r *tripRepoStub) FindRoomByCode(_ context.Context, _ string) (*trip.Room, error) {
	return nil, nil
}
func (r *tripRepoStub) UpdateRoomCode(_ context.Context, _ uuid.UUID, _ string) error {
	return nil
}
func (r *tripRepoStub) RoomCodeExists(_ context.Context, _ string) (bool, error) {
	return false, nil
}

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
func (r *tripRepoStub) CountRoomMembers(_ context.Context, _ uuid.UUID) (int, error) {
	return 0, nil
}
func (r *tripRepoStub) IsRoomMember(c context.Context, tid, uid uuid.UUID) (bool, error) {
	if r.isRoomMember != nil {
		return r.isRoomMember(c, tid, uid)
	}
	return false, nil
}

func (r *tripRepoStub) CreateItinerary(_ context.Context, _ *trip.Itinerary) error { return nil }
func (r *tripRepoStub) FindItineraryByID(c context.Context, id uuid.UUID) (*trip.Itinerary, error) {
	if r.findItineraryByID != nil {
		return r.findItineraryByID(c, id)
	}
	return nil, nil
}
func (r *tripRepoStub) UpdateItinerary(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (r *tripRepoStub) UpdateItineraryWithVersion(c context.Context, id uuid.UUID, v int, p map[string]any, e *trip.EventMeta) (*trip.Itinerary, error) {
	if r.updateItineraryWithV != nil {
		return r.updateItineraryWithV(c, id, v, p, e)
	}
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
func (r *tripRepoStub) FindTodoByID(c context.Context, id uuid.UUID) (*trip.Todo, error) {
	if r.findTodoByID != nil {
		return r.findTodoByID(c, id)
	}
	return nil, nil
}
func (r *tripRepoStub) UpdateTodo(_ context.Context, _ uuid.UUID, _ map[string]any) error {
	return nil
}
func (r *tripRepoStub) UpdateTodoWithVersion(c context.Context, id uuid.UUID, v int, p map[string]any, e *trip.EventMeta) (*trip.Todo, error) {
	if r.updateTodoWithVersion != nil {
		return r.updateTodoWithVersion(c, id, v, p, e)
	}
	return nil, nil
}
func (r *tripRepoStub) DeleteTodo(_ context.Context, _ uuid.UUID) error { return nil }
func (r *tripRepoStub) ListTodos(_ context.Context, _ uuid.UUID) ([]trip.Todo, error) {
	return nil, nil
}
func (r *tripRepoStub) ReorderTodos(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}
func (r *tripRepoStub) EnqueueTripPublished(_ context.Context, _ uuid.UUID, _ uuid.UUID, _ time.Time, _ string) error {
	return nil
}

// ---- Helpers ----

// newTestSvc spins up an in-memory miniredis, wires a real Hub, and returns
// (service, redis client, teardown). Tests that don't want a working seq/pub
// can use newTestSvcBrokenRedis instead.
func newTestSvc(t *testing.T, repo trip.IRepository) (*Service, *redis.Client, func()) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := NewService(ServiceConfig{
		TripRepo:   repo,
		Hub:        NewHub(zap.NewNop()),
		Redis:      rdb,
		Logger:     zap.NewNop(),
		InstanceID: "test-instance",
	})
	return svc, rdb, func() {
		_ = rdb.Close()
		mr.Close()
	}
}

func newTestSvcBrokenRedis(t *testing.T, repo trip.IRepository) *Service {
	t.Helper()
	rdb := redis.NewClient(&redis.Options{
		Addr:        "127.0.0.1:1",
		DialTimeout: 10 * time.Millisecond,
		MaxRetries:  -1,
	})
	return NewService(ServiceConfig{
		TripRepo:   repo,
		Hub:        NewHub(zap.NewNop()),
		Redis:      rdb,
		Logger:     zap.NewNop(),
		InstanceID: "test-instance",
	})
}

// newTestClient constructs a Client that is safe to use without a live
// WebSocket. Nothing dispatched by HandleClientMsg touches conn as long as the
// send buffer doesn't overflow — assertions read from client.send directly.
func newTestClient(svc *Service, tripID, userID uuid.UUID) *Client {
	return &Client{
		hub:    svc.hub,
		svc:    svc,
		logger: zap.NewNop(),
		conn:   nil,
		send:   make(chan []byte, sendBuffer),
		userID: userID,
		tripID: tripID,
		done:   make(chan struct{}),
	}
}

// nextFrame pops one server frame from the client's send channel, decoded.
func nextFrame(t *testing.T, c *Client) *ServerMsg {
	t.Helper()
	select {
	case raw := <-c.send:
		var m ServerMsg
		require.NoError(t, json.Unmarshal(raw, &m))
		return &m
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected a frame on send channel")
		return nil
	}
}

func drainOne(c *Client) (*ServerMsg, bool) {
	select {
	case raw := <-c.send:
		var m ServerMsg
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, false
		}
		return &m, true
	default:
		return nil, false
	}
}

func stringPtr(s string) *string { return &s }
func intPtr(i int) *int          { return &i }
func boolPtr(b bool) *bool       { return &b }

// ---- Authorize ----

func TestAuthorize_DelegatesToTripRepo(t *testing.T) {
	uid := uuid.New()
	tid := uuid.New()
	var gotTrip, gotUser uuid.UUID
	repo := &tripRepoStub{
		isRoomMember: func(_ context.Context, tripID, userID uuid.UUID) (bool, error) {
			gotTrip, gotUser = tripID, userID
			return true, nil
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()

	ok, err := svc.Authorize(ctx, uid, tid)
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, tid, gotTrip)
	assert.Equal(t, uid, gotUser)
}

func TestAuthorize_PropagatesError(t *testing.T) {
	repo := &tripRepoStub{
		isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			return false, errors.New("boom")
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()

	_, err := svc.Authorize(ctx, uuid.New(), uuid.New())
	assert.EqualError(t, err, "boom")
}

// ---- HandleClientMsg dispatch ----

func TestHandle_UnknownOp_SendsBadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()

	c := newTestClient(svc, uuid.New(), uuid.New())
	svc.HandleClientMsg(ctx, c, &ClientMsg{Type: "MYSTERY", MsgID: "m1", TripID: c.tripID})

	f := nextFrame(t, c)
	assert.Equal(t, srvError, f.Type)
	assert.Equal(t, "m1", f.Ref)
	assert.Equal(t, ErrCodeBadOp, f.Code)
}

func TestHandle_CalendarUpdateAndVoteCast_ReturnUnsupported(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()

	for _, op := range []OpType{OpCalendarUpdate, OpVoteCast} {
		c := newTestClient(svc, uuid.New(), uuid.New())
		svc.HandleClientMsg(ctx, c, &ClientMsg{Type: op, MsgID: "x", TripID: c.tripID})
		f := nextFrame(t, c)
		assert.Equal(t, srvError, f.Type, "op=%s", op)
		assert.Equal(t, ErrCodeUnsupported, f.Code, "op=%s", op)
	}
}

// ---- Cursor ----

func TestHandle_CursorMove_AssignsSeqAndBroadcasts(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()

	tripID := uuid.New()
	sender := newTestClient(svc, tripID, uuid.New())
	observer := newTestClient(svc, tripID, uuid.New())
	svc.hub.Register(observer)
	defer svc.hub.Unregister(observer)

	data, _ := json.Marshal(CursorMoveData{X: 12, Y: 34})
	svc.HandleClientMsg(ctx, sender, &ClientMsg{
		Type: OpCursorMove, MsgID: "c1", TripID: tripID, Data: data,
	})

	// Sender is skipped in the fanout — its buffer should be empty.
	if f, ok := drainOne(sender); ok {
		t.Fatalf("sender should not receive its own cursor: %+v", f)
	}

	// Observer should see the broadcast with a sequence number.
	f := nextFrame(t, observer)
	assert.Equal(t, string(OpCursorMove), f.Type)
	assert.Equal(t, tripID, f.TripID)
	assert.Equal(t, sender.userID, f.From)
	assert.Greater(t, f.Seq, int64(0))
}

func TestHandle_CursorMove_SeqError_SendsInternal(t *testing.T) {
	svc := newTestSvcBrokenRedis(t, &tripRepoStub{})
	c := newTestClient(svc, uuid.New(), uuid.New())

	svc.HandleClientMsg(ctx, c, &ClientMsg{Type: OpCursorMove, MsgID: "m", TripID: c.tripID})
	f := nextFrame(t, c)
	assert.Equal(t, srvError, f.Type)
	assert.Equal(t, ErrCodeInternal, f.Code)
	assert.Equal(t, "seq", f.Message)
}

// ---- ITINERARY_UPDATE ----

func TestHandle_ItineraryUpdate_VersionMissing_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: uuid.New(), Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 0, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "version required", f.Message)
}

func TestHandle_ItineraryUpdate_MissingItemID_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "item_id required", f.Message)
}

func TestHandle_ItineraryUpdate_InvalidDay_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: uuid.New(), Day: intPtr(0)})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "day must be >= 1", f.Message)
}

func TestHandle_ItineraryUpdate_EmptyPatch_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: uuid.New()})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "empty patch", f.Message)
}

func TestHandle_ItineraryUpdate_BadJSON_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1,
		Data: json.RawMessage("not-json"),
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "invalid data", f.Message)
}

func TestHandle_ItineraryUpdate_NotFound(t *testing.T) {
	repo := &tripRepoStub{
		findItineraryByID: func(_ context.Context, _ uuid.UUID) (*trip.Itinerary, error) {
			return nil, nil // service treats nil entity as not found
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: uuid.New(), Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeNotFound, f.Code)
	assert.Equal(t, "itinerary not found", f.Message)
}

func TestHandle_ItineraryUpdate_WrongTrip_Forbidden(t *testing.T) {
	itemID := uuid.New()
	otherTrip := uuid.New()

	repo := &tripRepoStub{
		findItineraryByID: func(_ context.Context, id uuid.UUID) (*trip.Itinerary, error) {
			return &trip.Itinerary{ID: id, TripID: otherTrip}, nil
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: itemID, Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeForbidden, f.Code)
	assert.Contains(t, f.Message, "another trip")
}

func TestHandle_ItineraryUpdate_LookupError_Internal(t *testing.T) {
	repo := &tripRepoStub{
		findItineraryByID: func(_ context.Context, _ uuid.UUID) (*trip.Itinerary, error) {
			return nil, errors.New("db down")
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: uuid.New(), Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeInternal, f.Code)
	assert.Equal(t, "lookup failed", f.Message)
}

func TestHandle_ItineraryUpdate_StaleVersion(t *testing.T) {
	tripID := uuid.New()
	itemID := uuid.New()
	repo := &tripRepoStub{
		findItineraryByID: func(_ context.Context, id uuid.UUID) (*trip.Itinerary, error) {
			return &trip.Itinerary{ID: id, TripID: tripID}, nil
		},
		updateItineraryWithV: func(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *trip.EventMeta) (*trip.Itinerary, error) {
			return nil, trip.ErrStaleVersion
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, tripID, uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: itemID, Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: tripID, Version: 3, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeStaleVersion, f.Code)
	assert.Equal(t, "stale version", f.Message)
}

func TestHandle_ItineraryUpdate_RepoNotFoundError(t *testing.T) {
	tripID := uuid.New()
	repo := &tripRepoStub{
		findItineraryByID: func(_ context.Context, id uuid.UUID) (*trip.Itinerary, error) {
			return &trip.Itinerary{ID: id, TripID: tripID}, nil
		},
		updateItineraryWithV: func(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *trip.EventMeta) (*trip.Itinerary, error) {
			return nil, trip.ErrItineraryNotFound
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, tripID, uuid.New())

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: uuid.New(), Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m", TripID: tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeNotFound, f.Code)
	assert.Equal(t, "itinerary not found", f.Message)
}

func TestHandle_ItineraryUpdate_HappyPath_BroadcastsAndAcks(t *testing.T) {
	tripID := uuid.New()
	itemID := uuid.New()
	userID := uuid.New()

	var gotVersion int
	var gotPatch map[string]any
	var gotEvent *trip.EventMeta

	repo := &tripRepoStub{
		findItineraryByID: func(_ context.Context, id uuid.UUID) (*trip.Itinerary, error) {
			return &trip.Itinerary{ID: id, TripID: tripID}, nil
		},
		updateItineraryWithV: func(_ context.Context, id uuid.UUID, v int, patch map[string]any, event *trip.EventMeta) (*trip.Itinerary, error) {
			gotVersion, gotPatch, gotEvent = v, patch, event
			return &trip.Itinerary{ID: id, TripID: tripID, Title: patch["title"].(string), Version: v + 1}, nil
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()

	sender := newTestClient(svc, tripID, userID)
	observer := newTestClient(svc, tripID, uuid.New())
	svc.hub.Register(observer)
	defer svc.hub.Unregister(observer)

	data, _ := json.Marshal(ItineraryUpdateData{ItemID: itemID, Title: stringPtr("Kyoto Day 1")})
	svc.HandleClientMsg(ctx, sender, &ClientMsg{
		Type: OpItineraryUpdate, MsgID: "m1", TripID: tripID, Version: 5, Data: data,
	})

	// Repo assertions.
	assert.Equal(t, 5, gotVersion)
	assert.Equal(t, "Kyoto Day 1", gotPatch["title"])
	require.NotNil(t, gotEvent)
	assert.Equal(t, string(OpItineraryUpdate), gotEvent.OpType)
	assert.Equal(t, userID, gotEvent.UserID)

	// Observer sees broadcast (sender is skipped there but receives ACK).
	obsFrame := nextFrame(t, observer)
	assert.Equal(t, string(OpItineraryUpdate), obsFrame.Type)
	assert.Equal(t, tripID, obsFrame.TripID)
	assert.Equal(t, userID, obsFrame.From)
	assert.Equal(t, 6, obsFrame.Version)
	assert.Greater(t, obsFrame.Seq, int64(0))

	// Sender receives an ACK carrying the new version.
	ack := nextFrame(t, sender)
	assert.Equal(t, srvAck, ack.Type)
	assert.Equal(t, "m1", ack.Ref)
	assert.Equal(t, 6, ack.Version)
}

// ---- TODO_UPDATE ----

func TestHandle_TodoUpdate_MissingTodoID_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	data, _ := json.Marshal(TodoUpdateData{Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpTodoUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "todo_id required", f.Message)
}

func TestHandle_TodoUpdate_InvalidPriority_BadOp(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()
	c := newTestClient(svc, uuid.New(), uuid.New())

	bad := "urgent"
	data, _ := json.Marshal(TodoUpdateData{TodoID: uuid.New(), Priority: &bad})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpTodoUpdate, MsgID: "m", TripID: c.tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeBadOp, f.Code)
	assert.Equal(t, "invalid priority", f.Message)
}

func TestHandle_TodoUpdate_ValidPriority_Succeeds(t *testing.T) {
	tripID := uuid.New()
	todoID := uuid.New()
	repo := &tripRepoStub{
		findTodoByID: func(_ context.Context, id uuid.UUID) (*trip.Todo, error) {
			return &trip.Todo{ID: id, TripID: tripID}, nil
		},
		updateTodoWithVersion: func(_ context.Context, id uuid.UUID, v int, patch map[string]any, _ *trip.EventMeta) (*trip.Todo, error) {
			assert.Equal(t, "high", patch["priority"])
			return &trip.Todo{ID: id, TripID: tripID, Priority: "high", Version: v + 1}, nil
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, tripID, uuid.New())

	pri := "high"
	data, _ := json.Marshal(TodoUpdateData{TodoID: todoID, Priority: &pri})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpTodoUpdate, MsgID: "m", TripID: tripID, Version: 1, Data: data,
	})

	ack := nextFrame(t, c)
	assert.Equal(t, srvAck, ack.Type)
	assert.Equal(t, 2, ack.Version)
}

func TestHandle_TodoUpdate_HappyPath_TogglesCompletion(t *testing.T) {
	tripID := uuid.New()
	todoID := uuid.New()
	repo := &tripRepoStub{
		findTodoByID: func(_ context.Context, id uuid.UUID) (*trip.Todo, error) {
			return &trip.Todo{ID: id, TripID: tripID}, nil
		},
		updateTodoWithVersion: func(_ context.Context, id uuid.UUID, v int, patch map[string]any, _ *trip.EventMeta) (*trip.Todo, error) {
			assert.Equal(t, true, patch["is_completed"])
			return &trip.Todo{ID: id, TripID: tripID, IsCompleted: true, Version: v + 1}, nil
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, tripID, uuid.New())

	data, _ := json.Marshal(TodoUpdateData{TodoID: todoID, IsCompleted: boolPtr(true)})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpTodoUpdate, MsgID: "m1", TripID: tripID, Version: 4, Data: data,
	})

	ack := nextFrame(t, c)
	assert.Equal(t, srvAck, ack.Type)
	assert.Equal(t, "m1", ack.Ref)
	assert.Equal(t, 5, ack.Version)
}

func TestHandle_TodoUpdate_StaleVersion(t *testing.T) {
	tripID := uuid.New()
	repo := &tripRepoStub{
		findTodoByID: func(_ context.Context, id uuid.UUID) (*trip.Todo, error) {
			return &trip.Todo{ID: id, TripID: tripID}, nil
		},
		updateTodoWithVersion: func(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *trip.EventMeta) (*trip.Todo, error) {
			return nil, trip.ErrStaleVersion
		},
	}
	svc, _, teardown := newTestSvc(t, repo)
	defer teardown()
	c := newTestClient(svc, tripID, uuid.New())

	data, _ := json.Marshal(TodoUpdateData{TodoID: uuid.New(), Title: stringPtr("x")})
	svc.HandleClientMsg(ctx, c, &ClientMsg{
		Type: OpTodoUpdate, MsgID: "m", TripID: tripID, Version: 1, Data: data,
	})
	f := nextFrame(t, c)
	assert.Equal(t, ErrCodeStaleVersion, f.Code)
}

// ---- PublishTripEvent ----

func TestPublishTripEvent_BroadcastsToHub(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()

	tripID := uuid.New()
	c := newTestClient(svc, tripID, uuid.New())
	svc.hub.Register(c)
	defer svc.hub.Unregister(c)

	svc.PublishTripEvent(ctx, tripID, []byte(`{"hello":"world"}`))

	select {
	case raw := <-c.send:
		assert.JSONEq(t, `{"hello":"world"}`, string(raw))
	case <-time.After(200 * time.Millisecond):
		t.Fatal("expected the raw payload to reach the registered client")
	}
}

func TestPublishTripEvent_SkipsUnrelatedTrip(t *testing.T) {
	svc, _, teardown := newTestSvc(t, &tripRepoStub{})
	defer teardown()

	otherClient := newTestClient(svc, uuid.New(), uuid.New())
	svc.hub.Register(otherClient)
	defer svc.hub.Unregister(otherClient)

	svc.PublishTripEvent(ctx, uuid.New(), []byte(`{"payload":1}`))

	if f, ok := drainOne(otherClient); ok {
		t.Fatalf("client registered under a different trip must not receive: %+v", f)
	}
}

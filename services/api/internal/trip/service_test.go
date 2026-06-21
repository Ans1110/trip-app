package trip_test

import (
	"context"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/trip"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx func(context.Context, func(trip.IRepository) error) error

	findUserByID   func(context.Context, uuid.UUID) (*auth.User, error)
	findUsersByIDs func(context.Context, []uuid.UUID) (map[uuid.UUID]auth.User, error)

	createTrip    func(context.Context, *trip.Trip) error
	findTripByID  func(context.Context, uuid.UUID) (*trip.Trip, error)
	updateTrip    func(context.Context, uuid.UUID, map[string]any) error
	deleteTrip    func(context.Context, uuid.UUID) error
	listTrips     func(context.Context, uuid.UUID, trip.ListTripsQuery) ([]trip.Trip, error)
	countMembers  func(context.Context, []uuid.UUID) (map[uuid.UUID]int, error)

	createRoom       func(context.Context, *trip.Room) error
	findRoomByID     func(context.Context, uuid.UUID) (*trip.Room, error)
	findRoomByTripID func(context.Context, uuid.UUID) (*trip.Room, error)
	findRoomByCode   func(context.Context, string) (*trip.Room, error)
	updateRoomCode   func(context.Context, uuid.UUID, string) error
	roomCodeExists   func(context.Context, string) (bool, error)

	addMember        func(context.Context, *trip.RoomMember) error
	removeMember     func(context.Context, uuid.UUID, uuid.UUID) error
	findMember       func(context.Context, uuid.UUID, uuid.UUID) (*trip.RoomMember, error)
	listMembers      func(context.Context, uuid.UUID) ([]trip.RoomMember, error)
	findMemberByTrip func(context.Context, uuid.UUID, uuid.UUID) (*trip.RoomMember, error)
	countRoomMembers func(context.Context, uuid.UUID) (int, error)

	createItinerary   func(context.Context, *trip.Itinerary) error
	findItineraryByID func(context.Context, uuid.UUID) (*trip.Itinerary, error)
	updateItinerary   func(context.Context, uuid.UUID, map[string]any) error
	deleteItinerary   func(context.Context, uuid.UUID) error
	listItinerary     func(context.Context, uuid.UUID) ([]trip.Itinerary, error)

	createTodo   func(context.Context, *trip.Todo) error
	findTodoByID func(context.Context, uuid.UUID) (*trip.Todo, error)
	updateTodo   func(context.Context, uuid.UUID, map[string]any) error
	deleteTodo   func(context.Context, uuid.UUID) error
	listTodos    func(context.Context, uuid.UUID) ([]trip.Todo, error)
}

func (r *repoMock) WithTx(ctx context.Context, fn func(trip.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(ctx, fn)
	}
	return fn(r)
}

func (r *repoMock) FindUserByID(c context.Context, id uuid.UUID) (*auth.User, error) {
	if r.findUserByID != nil {
		return r.findUserByID(c, id)
	}
	return nil, nil
}

func (r *repoMock) FindUsersByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	if r.findUsersByIDs != nil {
		return r.findUsersByIDs(c, ids)
	}
	return map[uuid.UUID]auth.User{}, nil
}

func (r *repoMock) CreateTrip(c context.Context, t *trip.Trip) error {
	if r.createTrip != nil {
		return r.createTrip(c, t)
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindTripByID(c context.Context, id uuid.UUID) (*trip.Trip, error) {
	if r.findTripByID != nil {
		return r.findTripByID(c, id)
	}
	return nil, nil
}

func (r *repoMock) UpdateTrip(c context.Context, id uuid.UUID, patch map[string]any) error {
	if r.updateTrip != nil {
		return r.updateTrip(c, id, patch)
	}
	return nil
}

func (r *repoMock) DeleteTrip(c context.Context, id uuid.UUID) error {
	if r.deleteTrip != nil {
		return r.deleteTrip(c, id)
	}
	return nil
}

func (r *repoMock) ListTrips(c context.Context, uid uuid.UUID, q trip.ListTripsQuery) ([]trip.Trip, error) {
	if r.listTrips != nil {
		return r.listTrips(c, uid, q)
	}
	return nil, nil
}

func (r *repoMock) CountMembers(c context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	if r.countMembers != nil {
		return r.countMembers(c, ids)
	}
	return map[uuid.UUID]int{}, nil
}

func (r *repoMock) CreateRoom(c context.Context, room *trip.Room) error {
	if r.createRoom != nil {
		return r.createRoom(c, room)
	}
	if room.ID == uuid.Nil {
		room.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindRoomByID(c context.Context, id uuid.UUID) (*trip.Room, error) {
	if r.findRoomByID != nil {
		return r.findRoomByID(c, id)
	}
	return nil, nil
}

func (r *repoMock) FindRoomByTripID(c context.Context, tid uuid.UUID) (*trip.Room, error) {
	if r.findRoomByTripID != nil {
		return r.findRoomByTripID(c, tid)
	}
	return nil, nil
}

func (r *repoMock) FindRoomByCode(c context.Context, code string) (*trip.Room, error) {
	if r.findRoomByCode != nil {
		return r.findRoomByCode(c, code)
	}
	return nil, nil
}

func (r *repoMock) UpdateRoomCode(c context.Context, roomID uuid.UUID, code string) error {
	if r.updateRoomCode != nil {
		return r.updateRoomCode(c, roomID, code)
	}
	return nil
}

func (r *repoMock) RoomCodeExists(c context.Context, code string) (bool, error) {
	if r.roomCodeExists != nil {
		return r.roomCodeExists(c, code)
	}
	return false, nil
}

func (r *repoMock) AddMember(c context.Context, m *trip.RoomMember) error {
	if r.addMember != nil {
		return r.addMember(c, m)
	}
	return nil
}

func (r *repoMock) RemoveMember(c context.Context, roomID, userID uuid.UUID) error {
	if r.removeMember != nil {
		return r.removeMember(c, roomID, userID)
	}
	return nil
}

func (r *repoMock) FindMember(c context.Context, roomID, userID uuid.UUID) (*trip.RoomMember, error) {
	if r.findMember != nil {
		return r.findMember(c, roomID, userID)
	}
	return nil, nil
}

func (r *repoMock) ListMembers(c context.Context, roomID uuid.UUID) ([]trip.RoomMember, error) {
	if r.listMembers != nil {
		return r.listMembers(c, roomID)
	}
	return nil, nil
}

func (r *repoMock) FindMemberByTrip(c context.Context, tid, uid uuid.UUID) (*trip.RoomMember, error) {
	if r.findMemberByTrip != nil {
		return r.findMemberByTrip(c, tid, uid)
	}
	return nil, nil
}

func (r *repoMock) CountRoomMembers(c context.Context, roomID uuid.UUID) (int, error) {
	if r.countRoomMembers != nil {
		return r.countRoomMembers(c, roomID)
	}
	return 0, nil
}

func (r *repoMock) CreateItinerary(c context.Context, it *trip.Itinerary) error {
	if r.createItinerary != nil {
		return r.createItinerary(c, it)
	}
	if it.ID == uuid.Nil {
		it.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindItineraryByID(c context.Context, id uuid.UUID) (*trip.Itinerary, error) {
	if r.findItineraryByID != nil {
		return r.findItineraryByID(c, id)
	}
	return nil, nil
}

func (r *repoMock) UpdateItinerary(c context.Context, id uuid.UUID, patch map[string]any) error {
	if r.updateItinerary != nil {
		return r.updateItinerary(c, id, patch)
	}
	return nil
}

func (r *repoMock) DeleteItinerary(c context.Context, id uuid.UUID) error {
	if r.deleteItinerary != nil {
		return r.deleteItinerary(c, id)
	}
	return nil
}

func (r *repoMock) ListItinerary(c context.Context, tid uuid.UUID) ([]trip.Itinerary, error) {
	if r.listItinerary != nil {
		return r.listItinerary(c, tid)
	}
	return nil, nil
}

func (r *repoMock) CreateTodo(c context.Context, t *trip.Todo) error {
	if r.createTodo != nil {
		return r.createTodo(c, t)
	}
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindTodoByID(c context.Context, id uuid.UUID) (*trip.Todo, error) {
	if r.findTodoByID != nil {
		return r.findTodoByID(c, id)
	}
	return nil, nil
}

func (r *repoMock) UpdateTodo(c context.Context, id uuid.UUID, patch map[string]any) error {
	if r.updateTodo != nil {
		return r.updateTodo(c, id, patch)
	}
	return nil
}

func (r *repoMock) DeleteTodo(c context.Context, id uuid.UUID) error {
	if r.deleteTodo != nil {
		return r.deleteTodo(c, id)
	}
	return nil
}

func (r *repoMock) ListTodos(c context.Context, tid uuid.UUID) ([]trip.Todo, error) {
	if r.listTodos != nil {
		return r.listTodos(c, tid)
	}
	return nil, nil
}

func (r *repoMock) ReorderItinerary(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

func (r *repoMock) ReorderTodos(_ context.Context, _ uuid.UUID, _ []uuid.UUID) error {
	return nil
}

// ---- Helpers ----

func newSvc(repo trip.IRepository) trip.IService {
	return trip.NewService(trip.ServiceConfig{Repo: repo})
}

func activeUser(id uuid.UUID, name string) *auth.User {
	return &auth.User{ID: id, Name: name, Email: name + "@example.com", Status: auth.UserStatusActive}
}

// ---- Trip lifecycle ----

func TestCreateTrip_BuildsTripRoomAndAdmin(t *testing.T) {
	owner := uuid.New()
	var savedTrip *trip.Trip
	var savedRoom *trip.Room
	var savedMember *trip.RoomMember

	repo := &repoMock{
		findUserByID: func(_ context.Context, id uuid.UUID) (*auth.User, error) {
			return activeUser(id, "Owner"), nil
		},
		createTrip: func(_ context.Context, tr *trip.Trip) error {
			tr.ID = uuid.New()
			tr.CreatedAt = time.Now()
			tr.UpdatedAt = time.Now()
			savedTrip = tr
			return nil
		},
		createRoom: func(_ context.Context, r *trip.Room) error {
			r.ID = uuid.New()
			r.CreatedAt = time.Now()
			savedRoom = r
			return nil
		},
		addMember: func(_ context.Context, m *trip.RoomMember) error {
			savedMember = m
			return nil
		},
	}
	svc := newSvc(repo)

	resp, err := svc.CreateTrip(ctx, owner, trip.CreateTripPayload{
		Title:     "Kyoto",
		StartDate: "2026-09-01",
		EndDate:   "2026-09-05",
	})
	require.NoError(t, err)
	require.NotNil(t, savedTrip)
	require.NotNil(t, savedRoom)
	require.NotNil(t, savedMember)

	assert.Equal(t, "Kyoto", savedTrip.Title)
	assert.Equal(t, owner, savedTrip.OwnerID)
	assert.Equal(t, trip.TripPlanning, savedTrip.Status)
	assert.Equal(t, savedTrip.ID, savedRoom.TripID)
	assert.Len(t, savedRoom.RoomCode, 8)
	assert.Equal(t, savedRoom.ID, savedMember.RoomID)
	assert.Equal(t, owner, savedMember.UserID)
	assert.Equal(t, trip.RoleAdmin, savedMember.Role)

	assert.Equal(t, savedTrip.ID.String(), resp.ID)
	assert.Equal(t, 1, resp.MemberCount)
	assert.Equal(t, "admin", resp.MyRole)
	require.NotNil(t, resp.Room)
	assert.Equal(t, savedRoom.RoomCode, resp.Room.RoomCode)
}

func TestCreateTrip_RejectsInvalidDateRange(t *testing.T) {
	owner := uuid.New()
	repo := &repoMock{
		findUserByID: func(_ context.Context, id uuid.UUID) (*auth.User, error) {
			return activeUser(id, "Owner"), nil
		},
	}
	svc := newSvc(repo)
	_, err := svc.CreateTrip(ctx, owner, trip.CreateTripPayload{
		Title:     "Bad",
		StartDate: "2026-09-10",
		EndDate:   "2026-09-01",
	})
	assert.ErrorIs(t, err, trip.ErrInvalidDateRange)
}

func TestUpdateTrip_OwnerOnly(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner, StartDate: time.Now(), EndDate: time.Now()}
	repo := &repoMock{
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
	}
	svc := newSvc(repo)
	title := "X"
	_, err := svc.UpdateTrip(ctx, stranger, tr.ID, trip.UpdateTripPayload{Title: &title})
	assert.ErrorIs(t, err, trip.ErrForbidden)
}

func TestDeleteTrip_OwnerOnly(t *testing.T) {
	owner := uuid.New()
	stranger := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	repo := &repoMock{
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
	}
	svc := newSvc(repo)
	err := svc.DeleteTrip(ctx, stranger, tr.ID)
	assert.ErrorIs(t, err, trip.ErrForbidden)
}

// ---- Join ----

func TestJoinRoom_ByCode_Success(t *testing.T) {
	roomID, tripID, user := uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findRoomByCode: func(_ context.Context, c string) (*trip.Room, error) {
			assert.Equal(t, "ABCDEFGH", c) // uppercased
			return &trip.Room{ID: roomID, TripID: tripID, RoomCode: c}, nil
		},
		findMember: func(_ context.Context, _, _ uuid.UUID) (*trip.RoomMember, error) { return nil, nil },
	}
	svc := newSvc(repo)
	out, err := svc.JoinRoom(ctx, user, trip.JoinRoomPayload{Code: "abcdefgh"})
	require.NoError(t, err)
	assert.False(t, out.AlreadyMember)
	assert.Equal(t, tripID.String(), out.TripID)
}

func TestJoinRoom_AlreadyMember_ReturnsFlagNotError(t *testing.T) {
	roomID, tripID, user := uuid.New(), uuid.New(), uuid.New()
	repo := &repoMock{
		findRoomByCode: func(_ context.Context, _ string) (*trip.Room, error) {
			return &trip.Room{ID: roomID, TripID: tripID, RoomCode: "ABCDEFGH"}, nil
		},
		findMember: func(_ context.Context, _, _ uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{RoomID: roomID, UserID: user, Role: trip.RoleMember}, nil
		},
		addMember: func(_ context.Context, _ *trip.RoomMember) error {
			t.Fatalf("AddMember should NOT be called when already a member")
			return nil
		},
	}
	svc := newSvc(repo)
	out, err := svc.JoinRoom(ctx, user, trip.JoinRoomPayload{Code: "ABCDEFGH"})
	require.NoError(t, err)
	assert.True(t, out.AlreadyMember)
}

func TestJoinRoom_RequiresCode(t *testing.T) {
	svc := newSvc(&repoMock{})
	_, err := svc.JoinRoom(ctx, uuid.New(), trip.JoinRoomPayload{})
	assert.ErrorIs(t, err, trip.ErrCodeRequired)
}

// ---- Membership ----

func TestLeaveRoom_OwnerCannotLeave(t *testing.T) {
	owner := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	repo := &repoMock{
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
	}
	svc := newSvc(repo)
	err := svc.LeaveRoom(ctx, owner, tr.ID)
	assert.ErrorIs(t, err, trip.ErrOwnerCannotLeave)
}

func TestLeaveRoom_MemberLeaves(t *testing.T) {
	owner, member := uuid.New(), uuid.New()
	roomID := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	removed := false
	repo := &repoMock{
		findTripByID:     func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findRoomByTripID: func(_ context.Context, _ uuid.UUID) (*trip.Room, error) { return &trip.Room{ID: roomID, TripID: tr.ID}, nil },
		findMember: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{RoomID: roomID, UserID: uid, Role: trip.RoleMember}, nil
		},
		removeMember: func(_ context.Context, _, _ uuid.UUID) error { removed = true; return nil },
	}
	svc := newSvc(repo)
	require.NoError(t, svc.LeaveRoom(ctx, member, tr.ID))
	assert.True(t, removed)
}

func TestRemoveMember_RequiresAdmin(t *testing.T) {
	owner, plain, target := uuid.New(), uuid.New(), uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	repo := &repoMock{
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findMemberByTrip: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{UserID: uid, Role: trip.RoleMember}, nil
		},
	}
	svc := newSvc(repo)
	err := svc.RemoveMember(ctx, plain, tr.ID, target)
	assert.ErrorIs(t, err, trip.ErrForbidden)
}

func TestRemoveMember_CannotRemoveOwner(t *testing.T) {
	owner := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	repo := &repoMock{
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
	}
	svc := newSvc(repo)
	err := svc.RemoveMember(ctx, owner, tr.ID, owner)
	assert.ErrorIs(t, err, trip.ErrCannotRemoveOwner)
}

func TestRemoveMember_AdminSucceeds(t *testing.T) {
	owner, target := uuid.New(), uuid.New()
	roomID := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	removed := false
	repo := &repoMock{
		findTripByID:     func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findMemberByTrip: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			if uid == owner {
				return &trip.RoomMember{UserID: uid, Role: trip.RoleAdmin}, nil
			}
			return nil, nil
		},
		findRoomByTripID: func(_ context.Context, _ uuid.UUID) (*trip.Room, error) { return &trip.Room{ID: roomID, TripID: tr.ID}, nil },
		findMember: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{RoomID: roomID, UserID: uid, Role: trip.RoleMember}, nil
		},
		removeMember: func(_ context.Context, _, _ uuid.UUID) error { removed = true; return nil },
	}
	svc := newSvc(repo)
	require.NoError(t, svc.RemoveMember(ctx, owner, tr.ID, target))
	assert.True(t, removed)
}

// ---- Itinerary permissions ----

func TestUpdateItinerary_NonCreatorMemberForbidden(t *testing.T) {
	owner, creator, other := uuid.New(), uuid.New(), uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	it := &trip.Itinerary{ID: uuid.New(), TripID: tr.ID, CreatedBy: creator}
	repo := &repoMock{
		findItineraryByID: func(_ context.Context, _ uuid.UUID) (*trip.Itinerary, error) { return it, nil },
		findTripByID:      func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findMemberByTrip: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{UserID: uid, Role: trip.RoleMember}, nil
		},
	}
	svc := newSvc(repo)
	title := "Edited"
	_, err := svc.UpdateItinerary(ctx, other, tr.ID, it.ID, trip.UpdateItineraryPayload{Title: &title})
	assert.ErrorIs(t, err, trip.ErrForbidden)
}

func TestDeleteItinerary_CreatorAllowed(t *testing.T) {
	owner, creator := uuid.New(), uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	it := &trip.Itinerary{ID: uuid.New(), TripID: tr.ID, CreatedBy: creator}
	deleted := false
	repo := &repoMock{
		findItineraryByID: func(_ context.Context, _ uuid.UUID) (*trip.Itinerary, error) { return it, nil },
		findTripByID:      func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		deleteItinerary:   func(_ context.Context, _ uuid.UUID) error { deleted = true; return nil },
	}
	svc := newSvc(repo)
	require.NoError(t, svc.DeleteItinerary(ctx, creator, tr.ID, it.ID))
	assert.True(t, deleted)
}

func TestDeleteItinerary_OwnerAllowed(t *testing.T) {
	owner, creator := uuid.New(), uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	it := &trip.Itinerary{ID: uuid.New(), TripID: tr.ID, CreatedBy: creator}
	deleted := false
	repo := &repoMock{
		findItineraryByID: func(_ context.Context, _ uuid.UUID) (*trip.Itinerary, error) { return it, nil },
		findTripByID:      func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		deleteItinerary:   func(_ context.Context, _ uuid.UUID) error { deleted = true; return nil },
	}
	svc := newSvc(repo)
	require.NoError(t, svc.DeleteItinerary(ctx, owner, tr.ID, it.ID))
	assert.True(t, deleted)
}

// ---- Todo permissions ----

func TestUpdateTodo_NonCreatorCanToggleCompletion(t *testing.T) {
	owner, creator, other := uuid.New(), uuid.New(), uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	td := &trip.Todo{ID: uuid.New(), TripID: tr.ID, CreatedBy: creator}
	updated := false
	repo := &repoMock{
		findTodoByID: func(_ context.Context, _ uuid.UUID) (*trip.Todo, error) { return td, nil },
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findMemberByTrip: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{UserID: uid, Role: trip.RoleMember}, nil
		},
		updateTodo: func(_ context.Context, _ uuid.UUID, patch map[string]any) error {
			updated = true
			assert.Contains(t, patch, "is_completed")
			return nil
		},
	}
	svc := newSvc(repo)
	done := true
	_, err := svc.UpdateTodo(ctx, other, tr.ID, td.ID, trip.UpdateTodoPayload{IsCompleted: &done})
	require.NoError(t, err)
	assert.True(t, updated)
}

func TestUpdateTodo_TitleEditRequiresWriter(t *testing.T) {
	owner, creator, other := uuid.New(), uuid.New(), uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner}
	td := &trip.Todo{ID: uuid.New(), TripID: tr.ID, CreatedBy: creator}
	repo := &repoMock{
		findTodoByID: func(_ context.Context, _ uuid.UUID) (*trip.Todo, error) { return td, nil },
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findMemberByTrip: func(_ context.Context, _, uid uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{UserID: uid, Role: trip.RoleMember}, nil
		},
	}
	svc := newSvc(repo)
	title := "Edited"
	_, err := svc.UpdateTodo(ctx, other, tr.ID, td.ID, trip.UpdateTodoPayload{Title: &title})
	assert.ErrorIs(t, err, trip.ErrForbidden)
}

// ---- Preview ----

func TestPreviewByCode_ReturnsAlreadyJoinedFlag(t *testing.T) {
	owner := uuid.New()
	roomID := uuid.New()
	tr := &trip.Trip{ID: uuid.New(), OwnerID: owner, Title: "Kyoto",
		StartDate: time.Now(), EndDate: time.Now(), Status: trip.TripPlanning}
	user := uuid.New()
	repo := &repoMock{
		findRoomByCode: func(_ context.Context, c string) (*trip.Room, error) {
			return &trip.Room{ID: roomID, TripID: tr.ID, RoomCode: c}, nil
		},
		findTripByID: func(_ context.Context, _ uuid.UUID) (*trip.Trip, error) { return tr, nil },
		findUserByID: func(_ context.Context, id uuid.UUID) (*auth.User, error) { return activeUser(id, "Alice"), nil },
		countRoomMembers: func(_ context.Context, _ uuid.UUID) (int, error) { return 3, nil },
		findMember: func(_ context.Context, _, _ uuid.UUID) (*trip.RoomMember, error) {
			return &trip.RoomMember{UserID: user, RoomID: roomID, Role: trip.RoleMember}, nil
		},
	}
	svc := newSvc(repo)
	out, err := svc.PreviewByCode(ctx, user, "ABCDEFGH")
	require.NoError(t, err)
	assert.True(t, out.AlreadyJoined)
	assert.Equal(t, 3, out.MemberCount)
	assert.Equal(t, "Kyoto", out.Title)
	assert.Empty(t, out.Owner.Email, "preview must not leak owner email")
}

// ---- ListTrips ----

func TestListTrips_ForwardsQuery(t *testing.T) {
	user := uuid.New()
	gotQ := trip.ListTripsQuery{}
	repo := &repoMock{
		listTrips: func(_ context.Context, _ uuid.UUID, q trip.ListTripsQuery) ([]trip.Trip, error) {
			gotQ = q
			return nil, nil
		},
	}
	svc := newSvc(repo)
	_, err := svc.ListTrips(ctx, user, trip.ListTripsQuery{
		Status: "planning", Q: "kyo", Role: "owner", Limit: 5, Offset: 10,
	})
	require.NoError(t, err)
	assert.Equal(t, "planning", gotQ.Status)
	assert.Equal(t, "kyo", gotQ.Q)
	assert.Equal(t, "owner", gotQ.Role)
	assert.Equal(t, 5, gotQ.Limit)
	assert.Equal(t, 10, gotQ.Offset)
}

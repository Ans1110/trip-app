package calendar_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/calendar"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx func(context.Context, func(calendar.IRepository) error) error

	createEvent             func(context.Context, *calendar.Event) error
	findEventByID           func(context.Context, uuid.UUID) (*calendar.Event, error)
	findEventBySource       func(context.Context, calendar.SourceType, uuid.UUID) (*calendar.Event, error)
	updateEvent             func(context.Context, uuid.UUID, map[string]any) error
	updateEventWithVersion  func(context.Context, uuid.UUID, int, map[string]any, *calendar.OutboxMeta) (*calendar.Event, error)
	softDeleteEvent         func(context.Context, uuid.UUID) error
	softDeleteEventBySource func(context.Context, calendar.SourceType, uuid.UUID) error

	listEventsForUser       func(context.Context, uuid.UUID, calendar.ListEventsQuery) ([]calendar.Event, error)
	listRoomEventsForTrip   func(context.Context, uuid.UUID, calendar.ListEventsQuery) ([]calendar.Event, error)
	listFriendEventsForUser func(context.Context, uuid.UUID, calendar.ListEventsQuery) ([]calendar.Event, error)

	addMembers           func(context.Context, uuid.UUID, []calendar.EventMember) error
	removeMember         func(context.Context, uuid.UUID, uuid.UUID) error
	removeMemberBySource func(context.Context, calendar.SourceType, uuid.UUID, uuid.UUID) error
	listEventMembers     func(context.Context, uuid.UUID) ([]calendar.EventMember, error)
	isEventMember        func(context.Context, uuid.UUID, uuid.UUID) (bool, error)

	enqueueOutbox func(context.Context, uuid.UUID, []byte, string, string) error
}

func (r *repoMock) WithTx(c context.Context, fn func(calendar.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(c, fn)
	}
	return fn(r)
}

func (r *repoMock) CreateEvent(c context.Context, e *calendar.Event) error {
	if r.createEvent != nil {
		return r.createEvent(c, e)
	}
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	return nil
}

func (r *repoMock) FindEventByID(c context.Context, id uuid.UUID) (*calendar.Event, error) {
	if r.findEventByID != nil {
		return r.findEventByID(c, id)
	}
	return nil, nil
}

func (r *repoMock) FindEventBySource(c context.Context, s calendar.SourceType, id uuid.UUID) (*calendar.Event, error) {
	if r.findEventBySource != nil {
		return r.findEventBySource(c, s, id)
	}
	return nil, nil
}

func (r *repoMock) UpdateEvent(c context.Context, id uuid.UUID, patch map[string]any) error {
	if r.updateEvent != nil {
		return r.updateEvent(c, id, patch)
	}
	return nil
}

func (r *repoMock) UpdateEventWithVersion(c context.Context, id uuid.UUID, v int, patch map[string]any, meta *calendar.OutboxMeta) (*calendar.Event, error) {
	if r.updateEventWithVersion != nil {
		return r.updateEventWithVersion(c, id, v, patch, meta)
	}
	return nil, nil
}

func (r *repoMock) SoftDeleteEvent(c context.Context, id uuid.UUID) error {
	if r.softDeleteEvent != nil {
		return r.softDeleteEvent(c, id)
	}
	return nil
}

func (r *repoMock) SoftDeleteEventBySource(c context.Context, s calendar.SourceType, id uuid.UUID) error {
	if r.softDeleteEventBySource != nil {
		return r.softDeleteEventBySource(c, s, id)
	}
	return nil
}

func (r *repoMock) ListEventsForUser(c context.Context, uid uuid.UUID, q calendar.ListEventsQuery) ([]calendar.Event, error) {
	if r.listEventsForUser != nil {
		return r.listEventsForUser(c, uid, q)
	}
	return nil, nil
}

func (r *repoMock) ListRoomEventsForTrip(c context.Context, tid uuid.UUID, q calendar.ListEventsQuery) ([]calendar.Event, error) {
	if r.listRoomEventsForTrip != nil {
		return r.listRoomEventsForTrip(c, tid, q)
	}
	return nil, nil
}

func (r *repoMock) ListFriendEventsForUser(c context.Context, fid uuid.UUID, q calendar.ListEventsQuery) ([]calendar.Event, error) {
	if r.listFriendEventsForUser != nil {
		return r.listFriendEventsForUser(c, fid, q)
	}
	return nil, nil
}

func (r *repoMock) AddMembers(c context.Context, eid uuid.UUID, ms []calendar.EventMember) error {
	if r.addMembers != nil {
		return r.addMembers(c, eid, ms)
	}
	return nil
}

func (r *repoMock) RemoveMember(c context.Context, eid, uid uuid.UUID) error {
	if r.removeMember != nil {
		return r.removeMember(c, eid, uid)
	}
	return nil
}

func (r *repoMock) RemoveMemberBySource(c context.Context, s calendar.SourceType, sid, uid uuid.UUID) error {
	if r.removeMemberBySource != nil {
		return r.removeMemberBySource(c, s, sid, uid)
	}
	return nil
}

func (r *repoMock) ListEventMembers(c context.Context, eid uuid.UUID) ([]calendar.EventMember, error) {
	if r.listEventMembers != nil {
		return r.listEventMembers(c, eid)
	}
	return nil, nil
}

func (r *repoMock) IsEventMember(c context.Context, eid, uid uuid.UUID) (bool, error) {
	if r.isEventMember != nil {
		return r.isEventMember(c, eid, uid)
	}
	return false, nil
}

func (r *repoMock) EnqueueOutbox(c context.Context, tid uuid.UUID, payload []byte, traceID, stream string) error {
	if r.enqueueOutbox != nil {
		return r.enqueueOutbox(c, tid, payload, traceID, stream)
	}
	return nil
}

// ---- friend / room mocks ----

type friendsMock struct {
	fn func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func (f *friendsMock) AreFriends(c context.Context, a, b uuid.UUID) (bool, error) {
	if f.fn != nil {
		return f.fn(c, a, b)
	}
	return false, nil
}

type roomsMock struct {
	fn func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func (r *roomsMock) IsRoomMember(c context.Context, tid, uid uuid.UUID) (bool, error) {
	if r.fn != nil {
		return r.fn(c, tid, uid)
	}
	return false, nil
}

// ---- helpers ----

func newSvc(repo calendar.IRepository, opts ...func(*calendar.ServiceConfig)) calendar.IService {
	cfg := calendar.ServiceConfig{Repo: repo}
	for _, o := range opts {
		o(&cfg)
	}
	return calendar.NewService(cfg)
}

func withFriends(f calendar.FriendChecker) func(*calendar.ServiceConfig) {
	return func(c *calendar.ServiceConfig) { c.Friends = f }
}

func withRooms(r calendar.RoomMemberChecker) func(*calendar.ServiceConfig) {
	return func(c *calendar.ServiceConfig) { c.RoomMember = r }
}

// ---- Create ----

func TestCreateEvent_PrivateSuccess(t *testing.T) {
	user := uuid.New()
	var saved *calendar.Event
	var savedMembers []calendar.EventMember
	repo := &repoMock{
		createEvent: func(_ context.Context, e *calendar.Event) error {
			e.ID = uuid.New()
			saved = e
			return nil
		},
		addMembers: func(_ context.Context, _ uuid.UUID, ms []calendar.EventMember) error {
			savedMembers = ms
			return nil
		},
	}
	svc := newSvc(repo)

	start := time.Now().UTC()
	end := start.Add(2 * time.Hour)
	resp, err := svc.CreateEvent(ctx, user, calendar.CreateEventPayload{
		Title:      "Dinner",
		StartAt:    start,
		EndAt:      end,
		Visibility: string(calendar.VisibilityPrivate),
	})
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "Dinner", saved.Title)
	assert.Equal(t, calendar.SourceUser, saved.SourceType)
	assert.Equal(t, calendar.EventTypeGeneral, saved.EventType)
	assert.Equal(t, calendar.VisibilityPrivate, saved.Visibility)
	assert.Equal(t, user, saved.CreatedBy)
	assert.Equal(t, 1, saved.Version)

	require.Len(t, savedMembers, 1)
	assert.Equal(t, user, savedMembers[0].UserID)
	assert.Equal(t, calendar.MemberRoleOwner, savedMembers[0].Role)

	assert.Equal(t, saved.ID.String(), resp.ID)
}

func TestCreateEvent_RejectsInvalidDateRange(t *testing.T) {
	svc := newSvc(&repoMock{})
	start := time.Now().UTC()
	_, err := svc.CreateEvent(ctx, uuid.New(), calendar.CreateEventPayload{
		Title:      "Bad",
		StartAt:    start,
		EndAt:      start.Add(-time.Hour),
		Visibility: string(calendar.VisibilityPrivate),
	})
	assert.ErrorIs(t, err, calendar.ErrInvalidDateRange)
}

func TestCreateEvent_RejectsRoomAndPublicVisibility(t *testing.T) {
	svc := newSvc(&repoMock{})
	for _, v := range []string{string(calendar.VisibilityRoom), string(calendar.VisibilityPublic), ""} {
		start := time.Now().UTC()
		_, err := svc.CreateEvent(ctx, uuid.New(), calendar.CreateEventPayload{
			Title:      "Nope",
			StartAt:    start,
			EndAt:      start.Add(time.Hour),
			Visibility: v,
		})
		assert.ErrorIs(t, err, calendar.ErrInvalidPayload, "visibility=%q must be rejected", v)
	}
}

func TestCreateEvent_RejectsInvalidTimeZone(t *testing.T) {
	svc := newSvc(&repoMock{})
	start := time.Now().UTC()
	_, err := svc.CreateEvent(ctx, uuid.New(), calendar.CreateEventPayload{
		Title:      "Tz",
		StartAt:    start,
		EndAt:      start.Add(time.Hour),
		TimeZone:   "Not/A_Real_Zone",
		Visibility: string(calendar.VisibilityPrivate),
	})
	assert.ErrorIs(t, err, calendar.ErrInvalidPayload)
}

// ---- Get / visibility gate ----

func TestGetEvent_CreatorAlwaysSeesIt(t *testing.T) {
	user := uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: user, Visibility: calendar.VisibilityPrivate}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo)
	_, err := svc.GetEvent(ctx, user, e.ID)
	require.NoError(t, err)
}

func TestGetEvent_PrivateRequiresMembership(t *testing.T) {
	creator, viewer := uuid.New(), uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityPrivate}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		isEventMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}
	svc := newSvc(repo)
	_, err := svc.GetEvent(ctx, viewer, e.ID)
	assert.ErrorIs(t, err, calendar.ErrForbidden)

	repo.isEventMember = func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil }
	_, err = svc.GetEvent(ctx, viewer, e.ID)
	assert.NoError(t, err)
}

func TestGetEvent_FriendsUsesFriendChecker(t *testing.T) {
	creator, viewer := uuid.New(), uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityFriends}
	called := false
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	friends := &friendsMock{
		fn: func(_ context.Context, a, b uuid.UUID) (bool, error) {
			called = true
			assert.Equal(t, viewer, a)
			assert.Equal(t, creator, b)
			return true, nil
		},
	}
	svc := newSvc(repo, withFriends(friends))
	_, err := svc.GetEvent(ctx, viewer, e.ID)
	require.NoError(t, err)
	assert.True(t, called)
}

func TestGetEvent_FriendsForbiddenWhenNotFriends(t *testing.T) {
	creator, viewer := uuid.New(), uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityFriends}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo, withFriends(&friendsMock{
		fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}))
	_, err := svc.GetEvent(ctx, viewer, e.ID)
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestGetEvent_RoomRequiresTripAndRoomMembership(t *testing.T) {
	creator, viewer := uuid.New(), uuid.New()
	tripID := uuid.New()
	e := &calendar.Event{
		ID:         uuid.New(),
		CreatedBy:  creator,
		Visibility: calendar.VisibilityRoom,
		SourceType: calendar.SourceTrip,
		SourceID:   &tripID,
	}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	rooms := &roomsMock{
		fn: func(_ context.Context, tid, uid uuid.UUID) (bool, error) {
			assert.Equal(t, tripID, tid)
			assert.Equal(t, viewer, uid)
			return true, nil
		},
	}
	svc := newSvc(repo, withRooms(rooms))
	_, err := svc.GetEvent(ctx, viewer, e.ID)
	require.NoError(t, err)
}

func TestGetEvent_RoomForbiddenWhenNotSourceTrip(t *testing.T) {
	// visibility=room but source_type=user is a corrupt row — must reject.
	creator, viewer := uuid.New(), uuid.New()
	e := &calendar.Event{
		ID:         uuid.New(),
		CreatedBy:  creator,
		Visibility: calendar.VisibilityRoom,
		SourceType: calendar.SourceUser,
	}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo, withRooms(&roomsMock{
		fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			t.Fatalf("IsRoomMember must NOT be consulted for a non-trip source")
			return true, nil
		},
	}))
	_, err := svc.GetEvent(ctx, viewer, e.ID)
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestGetEvent_MissingEvent(t *testing.T) {
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return nil, nil },
	}
	svc := newSvc(repo)
	_, err := svc.GetEvent(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, calendar.ErrEventNotFound)
}

// ---- Update ----

func TestUpdateEvent_NonCreatorForbidden(t *testing.T) {
	creator, stranger := uuid.New(), uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityPrivate, Version: 1}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo)
	title := "hack"
	_, err := svc.UpdateEvent(ctx, stranger, e.ID, calendar.UpdateEventPayload{Title: &title, Version: 1})
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestUpdateEvent_TripSourceSilentlyDropsImmutableFields(t *testing.T) {
	creator := uuid.New()
	tripID := uuid.New()
	e := &calendar.Event{
		ID:         uuid.New(),
		CreatedBy:  creator,
		SourceType: calendar.SourceTrip,
		SourceID:   &tripID,
		Visibility: calendar.VisibilityRoom,
		Version:    2,
	}
	var gotPatch map[string]any
	var gotMeta *calendar.OutboxMeta
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		updateEventWithVersion: func(_ context.Context, _ uuid.UUID, v int, patch map[string]any, meta *calendar.OutboxMeta) (*calendar.Event, error) {
			gotPatch = patch
			gotMeta = meta
			assert.Equal(t, 2, v)
			return e, nil
		},
	}
	svc := newSvc(repo)

	title := "Renamed"
	desc := "New desc"
	color := "#ff00aa"
	newVis := string(calendar.VisibilityPrivate)
	_, err := svc.UpdateEvent(ctx, creator, e.ID, calendar.UpdateEventPayload{
		Title:       &title,
		Description: &desc,
		Color:       &color,
		Visibility:  &newVis,
		Version:     2,
	})
	require.NoError(t, err)
	require.NotNil(t, gotPatch)
	assert.NotContains(t, gotPatch, "title")
	assert.NotContains(t, gotPatch, "description")
	assert.NotContains(t, gotPatch, "visibility")
	assert.Equal(t, "#ff00aa", gotPatch["color"], "color is presentation-only, must pass through")

	// trip-source patches go through the outbox meta with the CALENDAR_UPDATED op
	require.NotNil(t, gotMeta)
	assert.Equal(t, "CALENDAR_UPDATED", gotMeta.OpType)
	assert.Equal(t, tripID, gotMeta.TripID)
}

func TestUpdateEvent_TripSourceEmptyPatchReturnsReadOnly(t *testing.T) {
	creator := uuid.New()
	tripID := uuid.New()
	e := &calendar.Event{
		ID:         uuid.New(),
		CreatedBy:  creator,
		SourceType: calendar.SourceTrip,
		SourceID:   &tripID,
		Visibility: calendar.VisibilityRoom,
		Version:    1,
	}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		updateEventWithVersion: func(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *calendar.OutboxMeta) (*calendar.Event, error) {
			t.Fatalf("update must NOT run when every field was dropped")
			return nil, nil
		},
	}
	svc := newSvc(repo)

	title := "only title, which is immutable"
	_, err := svc.UpdateEvent(ctx, creator, e.ID, calendar.UpdateEventPayload{Title: &title, Version: 1})
	assert.ErrorIs(t, err, calendar.ErrTripSourceReadOnly)
}

func TestUpdateEvent_UserSourceUpdatesFields(t *testing.T) {
	creator := uuid.New()
	e := &calendar.Event{
		ID:         uuid.New(),
		CreatedBy:  creator,
		SourceType: calendar.SourceUser,
		Visibility: calendar.VisibilityPrivate,
		StartAt:    time.Now().UTC(),
		EndAt:      time.Now().UTC().Add(time.Hour),
		Version:    3,
	}
	var gotPatch map[string]any
	var gotMeta *calendar.OutboxMeta
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		updateEventWithVersion: func(_ context.Context, _ uuid.UUID, _ int, patch map[string]any, meta *calendar.OutboxMeta) (*calendar.Event, error) {
			gotPatch = patch
			gotMeta = meta
			return e, nil
		},
	}
	svc := newSvc(repo)

	title := "Renamed"
	_, err := svc.UpdateEvent(ctx, creator, e.ID, calendar.UpdateEventPayload{Title: &title, Version: 3})
	require.NoError(t, err)
	assert.Equal(t, "Renamed", gotPatch["title"])
	assert.Nil(t, gotMeta, "user-source events must not fan out through the trip outbox")
}

func TestUpdateEvent_StaleVersionSurfaces(t *testing.T) {
	creator := uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityPrivate, Version: 5}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		updateEventWithVersion: func(_ context.Context, _ uuid.UUID, _ int, _ map[string]any, _ *calendar.OutboxMeta) (*calendar.Event, error) {
			return nil, calendar.ErrStaleVersion
		},
	}
	svc := newSvc(repo)
	title := "x"
	_, err := svc.UpdateEvent(ctx, creator, e.ID, calendar.UpdateEventPayload{Title: &title, Version: 4})
	assert.ErrorIs(t, err, calendar.ErrStaleVersion)
}

func TestUpdateEvent_RejectsInvertedDateRange(t *testing.T) {
	creator := uuid.New()
	now := time.Now().UTC()
	e := &calendar.Event{
		ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityPrivate,
		StartAt: now, EndAt: now.Add(time.Hour), Version: 1,
	}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo)
	badEnd := now.Add(-time.Hour)
	_, err := svc.UpdateEvent(ctx, creator, e.ID, calendar.UpdateEventPayload{EndAt: &badEnd, Version: 1})
	assert.ErrorIs(t, err, calendar.ErrInvalidDateRange)
}

func TestUpdateEvent_CannotSwitchVisibilityToRoom(t *testing.T) {
	creator := uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, Visibility: calendar.VisibilityPrivate, Version: 1}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo)
	v := string(calendar.VisibilityRoom)
	_, err := svc.UpdateEvent(ctx, creator, e.ID, calendar.UpdateEventPayload{Visibility: &v, Version: 1})
	assert.ErrorIs(t, err, calendar.ErrInvalidPayload)
}

// ---- Delete ----

func TestDeleteEvent_TripSourceRefused(t *testing.T) {
	creator := uuid.New()
	tid := uuid.New()
	e := &calendar.Event{
		ID: uuid.New(), CreatedBy: creator,
		SourceType: calendar.SourceTrip, SourceID: &tid,
	}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		softDeleteEvent: func(_ context.Context, _ uuid.UUID) error {
			t.Fatalf("must not delete trip-source event directly")
			return nil
		},
	}
	svc := newSvc(repo)
	err := svc.DeleteEvent(ctx, creator, e.ID)
	assert.ErrorIs(t, err, calendar.ErrTripSourceReadOnly)
}

func TestDeleteEvent_NonCreatorForbidden(t *testing.T) {
	creator, stranger := uuid.New(), uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, SourceType: calendar.SourceUser}
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
	}
	svc := newSvc(repo)
	err := svc.DeleteEvent(ctx, stranger, e.ID)
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestDeleteEvent_CreatorSucceeds(t *testing.T) {
	creator := uuid.New()
	e := &calendar.Event{ID: uuid.New(), CreatedBy: creator, SourceType: calendar.SourceUser}
	deleted := false
	repo := &repoMock{
		findEventByID:   func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return e, nil },
		softDeleteEvent: func(_ context.Context, _ uuid.UUID) error { deleted = true; return nil },
	}
	svc := newSvc(repo)
	require.NoError(t, svc.DeleteEvent(ctx, creator, e.ID))
	assert.True(t, deleted)
}

// ---- List ----

func TestListMyEvents_ForwardsRangeFilter(t *testing.T) {
	user := uuid.New()
	from := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	to := from.Add(24 * time.Hour)
	var gotQ calendar.ListEventsQuery
	repo := &repoMock{
		listEventsForUser: func(_ context.Context, uid uuid.UUID, q calendar.ListEventsQuery) ([]calendar.Event, error) {
			assert.Equal(t, user, uid)
			gotQ = q
			return nil, nil
		},
	}
	svc := newSvc(repo)
	_, err := svc.ListMyEvents(ctx, user, calendar.ListEventsQuery{From: &from, To: &to})
	require.NoError(t, err)
	require.NotNil(t, gotQ.From)
	assert.Equal(t, from, *gotQ.From)
	require.NotNil(t, gotQ.To)
	assert.Equal(t, to, *gotQ.To)
}

func TestListFriendEvents_SelfShortCircuitsToMyEvents(t *testing.T) {
	user := uuid.New()
	myCalled := false
	friendCalled := false
	repo := &repoMock{
		listEventsForUser: func(_ context.Context, _ uuid.UUID, _ calendar.ListEventsQuery) ([]calendar.Event, error) {
			myCalled = true
			return nil, nil
		},
		listFriendEventsForUser: func(_ context.Context, _ uuid.UUID, _ calendar.ListEventsQuery) ([]calendar.Event, error) {
			friendCalled = true
			return nil, nil
		},
	}
	svc := newSvc(repo, withFriends(&friendsMock{
		fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) {
			t.Fatalf("AreFriends must NOT be called when viewer == friend")
			return true, nil
		},
	}))
	_, err := svc.ListFriendEvents(ctx, user, user, calendar.ListEventsQuery{})
	require.NoError(t, err)
	assert.True(t, myCalled)
	assert.False(t, friendCalled)
}

func TestListFriendEvents_NonFriendForbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, withFriends(&friendsMock{
		fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}))
	_, err := svc.ListFriendEvents(ctx, uuid.New(), uuid.New(), calendar.ListEventsQuery{})
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestListFriendEvents_NoFriendCheckerForbidden(t *testing.T) {
	svc := newSvc(&repoMock{})
	_, err := svc.ListFriendEvents(ctx, uuid.New(), uuid.New(), calendar.ListEventsQuery{})
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestListRoomEvents_NonMemberForbidden(t *testing.T) {
	svc := newSvc(&repoMock{}, withRooms(&roomsMock{
		fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil },
	}))
	_, err := svc.ListRoomEvents(ctx, uuid.New(), uuid.New(), calendar.ListEventsQuery{})
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

func TestListRoomEvents_MemberOK(t *testing.T) {
	user, tripID := uuid.New(), uuid.New()
	called := false
	repo := &repoMock{
		listRoomEventsForTrip: func(_ context.Context, tid uuid.UUID, _ calendar.ListEventsQuery) ([]calendar.Event, error) {
			called = true
			assert.Equal(t, tripID, tid)
			return []calendar.Event{}, nil
		},
	}
	svc := newSvc(repo, withRooms(&roomsMock{
		fn: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil },
	}))
	_, err := svc.ListRoomEvents(ctx, user, tripID, calendar.ListEventsQuery{})
	require.NoError(t, err)
	assert.True(t, called)
}

func TestListRoomEvents_NoRoomCheckerForbidden(t *testing.T) {
	svc := newSvc(&repoMock{})
	_, err := svc.ListRoomEvents(ctx, uuid.New(), uuid.New(), calendar.ListEventsQuery{})
	assert.ErrorIs(t, err, calendar.ErrForbidden)
}

// ---- Misc ----

func TestUpdateEvent_MissingEvent(t *testing.T) {
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return nil, nil },
	}
	svc := newSvc(repo)
	title := "x"
	_, err := svc.UpdateEvent(ctx, uuid.New(), uuid.New(), calendar.UpdateEventPayload{Title: &title, Version: 1})
	assert.ErrorIs(t, err, calendar.ErrEventNotFound)
}

func TestGetEvent_RepoErrorPropagates(t *testing.T) {
	sentinel := errors.New("db down")
	repo := &repoMock{
		findEventByID: func(_ context.Context, _ uuid.UUID) (*calendar.Event, error) { return nil, sentinel },
	}
	svc := newSvc(repo)
	_, err := svc.GetEvent(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, sentinel)
}

package location_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/location"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	createSavedPlace                func(context.Context, *location.SavedPlace) error
	findSavedPlaceByID              func(context.Context, uuid.UUID) (*location.SavedPlace, error)
	findSavedPlaceByUserTripPlaceID func(context.Context, uuid.UUID, *uuid.UUID, string, string) (*location.SavedPlace, error)
	updateSavedPlace                func(context.Context, uuid.UUID, map[string]any) error
	softDeleteSavedPlace            func(context.Context, uuid.UUID) error
	listSavedPlacesForUser          func(context.Context, uuid.UUID, *uuid.UUID, *string) ([]location.SavedPlace, error)
}

func (r *repoMock) CreateSavedPlace(c context.Context, sp *location.SavedPlace) error {
	if r.createSavedPlace != nil {
		return r.createSavedPlace(c, sp)
	}
	if sp.ID == uuid.Nil {
		sp.ID = uuid.New()
	}
	return nil
}
func (r *repoMock) FindSavedPlaceByID(c context.Context, id uuid.UUID) (*location.SavedPlace, error) {
	if r.findSavedPlaceByID != nil {
		return r.findSavedPlaceByID(c, id)
	}
	return nil, nil
}
func (r *repoMock) FindSavedPlaceByUserTripPlaceID(c context.Context, u uuid.UUID, tid *uuid.UUID, pid, cat string) (*location.SavedPlace, error) {
	if r.findSavedPlaceByUserTripPlaceID != nil {
		return r.findSavedPlaceByUserTripPlaceID(c, u, tid, pid, cat)
	}
	return nil, nil
}
func (r *repoMock) UpdateSavedPlace(c context.Context, id uuid.UUID, patch map[string]any) error {
	if r.updateSavedPlace != nil {
		return r.updateSavedPlace(c, id, patch)
	}
	return nil
}
func (r *repoMock) SoftDeleteSavedPlace(c context.Context, id uuid.UUID) error {
	if r.softDeleteSavedPlace != nil {
		return r.softDeleteSavedPlace(c, id)
	}
	return nil
}
func (r *repoMock) ListSavedPlacesForUser(c context.Context, u uuid.UUID, tid *uuid.UUID, cat *string) ([]location.SavedPlace, error) {
	if r.listSavedPlacesForUser != nil {
		return r.listSavedPlacesForUser(c, u, tid, cat)
	}
	return nil, nil
}

// ---- authMock ----

type authMock struct {
	isRoomMember func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
}

func (a *authMock) IsRoomMember(c context.Context, tid, uid uuid.UUID) (bool, error) {
	if a.isRoomMember != nil {
		return a.isRoomMember(c, tid, uid)
	}
	return true, nil
}

// ---- provider mocks ----

type placesMock struct {
	search  func(context.Context, location.PlaceSearchRequest) ([]location.PlaceSummary, error)
	nearby  func(context.Context, location.NearbySearchRequest) ([]location.PlaceSummary, error)
	details func(context.Context, string, string) (*location.PlaceDetails, error)
}

func (p *placesMock) SearchText(c context.Context, req location.PlaceSearchRequest) ([]location.PlaceSummary, error) {
	if p.search != nil {
		return p.search(c, req)
	}
	return nil, nil
}
func (p *placesMock) Nearby(c context.Context, req location.NearbySearchRequest) ([]location.PlaceSummary, error) {
	if p.nearby != nil {
		return p.nearby(c, req)
	}
	return nil, nil
}
func (p *placesMock) Details(c context.Context, id, lang string) (*location.PlaceDetails, error) {
	if p.details != nil {
		return p.details(c, id, lang)
	}
	return nil, nil
}

type routesMock struct {
	compute func(context.Context, location.RouteRequest) (*location.Route, error)
}

func (r *routesMock) ComputeRoute(c context.Context, req location.RouteRequest) (*location.Route, error) {
	if r.compute != nil {
		return r.compute(c, req)
	}
	return nil, nil
}

type weatherMock struct {
	forecast func(context.Context, location.LatLng, string, string) (*location.WeatherForecast, error)
}

func (w *weatherMock) Forecast(c context.Context, at location.LatLng, lang, units string) (*location.WeatherForecast, error) {
	if w.forecast != nil {
		return w.forecast(c, at, lang, units)
	}
	return nil, nil
}

// ---- auditMock ----

type auditMock struct {
	logs []audit.Log
	err  error
}

func (a *auditMock) Create(_ context.Context, l *audit.Log) error {
	a.logs = append(a.logs, *l)
	return a.err
}

// ---- helpers ----

func newRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	c := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = c.Close() })
	return c
}

type svcOpts struct {
	repo      *repoMock
	auth      *authMock
	places    location.PlacesProvider
	routes    location.RouteProvider
	weather   location.WeatherProvider
	cache     *location.Cache
	positions *location.PositionStore
	audit     audit.Writer
}

func newSvc(o svcOpts) location.IService {
	cfg := location.ServiceConfig{
		Repo:      o.repo,
		Places:    o.places,
		Routes:    o.routes,
		Weather:   o.weather,
		Cache:     o.cache,
		Positions: o.positions,
		Audit:     o.audit,
	}
	if o.auth != nil {
		cfg.TripAuth = o.auth
	}
	return location.NewService(cfg)
}

// ---- CreateSavedPlace ----

func TestCreateSavedPlace_HappyPath(t *testing.T) {
	userID := uuid.New()
	var saved *location.SavedPlace
	repo := &repoMock{createSavedPlace: func(_ context.Context, sp *location.SavedPlace) error {
		if sp.ID == uuid.Nil {
			sp.ID = uuid.New()
		}
		saved = sp
		return nil
	}}
	aud := &auditMock{}
	svc := newSvc(svcOpts{repo: repo, audit: aud})
	resp, err := svc.CreateSavedPlace(ctx, userID, location.CreateSavedPlacePayload{
		Name: "  Cafe  ", Address: " 1 Main ", Lat: 1, Lng: 2, PlaceID: " abc ", Category: "food",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "Cafe", saved.Name)
	assert.Equal(t, "1 Main", saved.Address)
	assert.Equal(t, "abc", saved.PlaceID)
	assert.Equal(t, "food", saved.Category)
	require.Len(t, aud.logs, 1)
	assert.Equal(t, location.AuditSavedPlaceCreated, aud.logs[0].Action)
}

func TestCreateSavedPlace_InvalidTripID(t *testing.T) {
	repo := &repoMock{}
	svc := newSvc(svcOpts{repo: repo})
	bad := "not-a-uuid"
	_, err := svc.CreateSavedPlace(ctx, uuid.New(), location.CreateSavedPlacePayload{
		Name: "x", Lat: 1, Lng: 2, TripID: &bad,
	})
	assert.ErrorIs(t, err, location.ErrInvalidPayload)
}

func TestCreateSavedPlace_TripBindingRequiresMembership(t *testing.T) {
	tid := uuid.New().String()
	repo := &repoMock{}
	auth := &authMock{isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }}
	svc := newSvc(svcOpts{repo: repo, auth: auth})
	_, err := svc.CreateSavedPlace(ctx, uuid.New(), location.CreateSavedPlacePayload{
		Name: "x", Lat: 1, Lng: 2, TripID: &tid,
	})
	assert.ErrorIs(t, err, location.ErrForbidden)
}

func TestCreateSavedPlace_TripBindingWithoutAuthorizerFails(t *testing.T) {
	tid := uuid.New().String()
	svc := newSvc(svcOpts{repo: &repoMock{}})
	_, err := svc.CreateSavedPlace(ctx, uuid.New(), location.CreateSavedPlacePayload{
		Name: "x", Lat: 1, Lng: 2, TripID: &tid,
	})
	assert.ErrorIs(t, err, location.ErrForbidden)
}

// ---- UpdateSavedPlace ----

func TestUpdateSavedPlace_NotFound(t *testing.T) {
	repo := &repoMock{findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) { return nil, nil }}
	svc := newSvc(svcOpts{repo: repo})
	_, err := svc.UpdateSavedPlace(ctx, uuid.New(), uuid.New(), location.UpdateSavedPlacePayload{})
	assert.ErrorIs(t, err, location.ErrNotFound)
}

func TestUpdateSavedPlace_Forbidden(t *testing.T) {
	repo := &repoMock{findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) {
		return &location.SavedPlace{UserID: uuid.New()}, nil
	}}
	svc := newSvc(svcOpts{repo: repo})
	_, err := svc.UpdateSavedPlace(ctx, uuid.New(), uuid.New(), location.UpdateSavedPlacePayload{})
	assert.ErrorIs(t, err, location.ErrForbidden)
}

func TestUpdateSavedPlace_EmptyNameRejected(t *testing.T) {
	userID := uuid.New()
	repo := &repoMock{findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) {
		return &location.SavedPlace{UserID: userID, Name: "x"}, nil
	}}
	svc := newSvc(svcOpts{repo: repo})
	empty := "   "
	_, err := svc.UpdateSavedPlace(ctx, userID, uuid.New(), location.UpdateSavedPlacePayload{Name: &empty})
	assert.ErrorIs(t, err, location.ErrInvalidPayload)
}

func TestUpdateSavedPlace_EmptyPatchReturnsExisting(t *testing.T) {
	userID := uuid.New()
	id := uuid.New()
	existing := &location.SavedPlace{ID: id, UserID: userID, Name: "keep"}
	updated := false
	repo := &repoMock{
		findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) { return existing, nil },
		updateSavedPlace: func(_ context.Context, _ uuid.UUID, _ map[string]any) error {
			updated = true
			return nil
		},
	}
	svc := newSvc(svcOpts{repo: repo})
	resp, err := svc.UpdateSavedPlace(ctx, userID, id, location.UpdateSavedPlacePayload{})
	require.NoError(t, err)
	assert.Equal(t, "keep", resp.Name)
	assert.False(t, updated)
}

func TestUpdateSavedPlace_UnsetTrip(t *testing.T) {
	userID := uuid.New()
	id := uuid.New()
	tripID := uuid.New()
	existing := &location.SavedPlace{ID: id, UserID: userID, Name: "n", TripID: &tripID}
	var patch map[string]any
	repo := &repoMock{
		findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) { return existing, nil },
		updateSavedPlace: func(_ context.Context, _ uuid.UUID, p map[string]any) error {
			patch = p
			return nil
		},
	}
	repo.findSavedPlaceByID = func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) { return existing, nil }
	aud := &auditMock{}
	svc := newSvc(svcOpts{repo: repo, audit: aud})
	_, err := svc.UpdateSavedPlace(ctx, userID, id, location.UpdateSavedPlacePayload{UnsetTripID: true})
	require.NoError(t, err)
	require.NotNil(t, patch)
	assert.Nil(t, patch["trip_id"])
	require.Len(t, aud.logs, 1)
	assert.Equal(t, location.AuditSavedPlaceUpdated, aud.logs[0].Action)
}

// ---- DeleteSavedPlace ----

func TestDeleteSavedPlace_NotFoundAndForbidden(t *testing.T) {
	nilRepo := &repoMock{findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) { return nil, nil }}
	svc := newSvc(svcOpts{repo: nilRepo})
	assert.ErrorIs(t, svc.DeleteSavedPlace(ctx, uuid.New(), uuid.New()), location.ErrNotFound)

	otherOwner := &repoMock{findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) {
		return &location.SavedPlace{UserID: uuid.New()}, nil
	}}
	svc = newSvc(svcOpts{repo: otherOwner})
	assert.ErrorIs(t, svc.DeleteSavedPlace(ctx, uuid.New(), uuid.New()), location.ErrForbidden)
}

func TestDeleteSavedPlace_HappyPath(t *testing.T) {
	userID := uuid.New()
	deleted := false
	repo := &repoMock{
		findSavedPlaceByID: func(_ context.Context, _ uuid.UUID) (*location.SavedPlace, error) {
			return &location.SavedPlace{UserID: userID}, nil
		},
		softDeleteSavedPlace: func(_ context.Context, _ uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	aud := &auditMock{}
	svc := newSvc(svcOpts{repo: repo, audit: aud})
	require.NoError(t, svc.DeleteSavedPlace(ctx, userID, uuid.New()))
	assert.True(t, deleted)
	require.Len(t, aud.logs, 1)
	assert.Equal(t, location.AuditSavedPlaceDeleted, aud.logs[0].Action)
}

// ---- ListSavedPlaces ----

func TestListSavedPlaces_InvalidTripID(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	bad := "nope"
	_, err := svc.ListSavedPlaces(ctx, uuid.New(), location.ListSavedPlacesQuery{TripID: &bad})
	assert.ErrorIs(t, err, location.ErrInvalidPayload)
}

func TestListSavedPlaces_TripFilterRequiresMembership(t *testing.T) {
	tid := uuid.New().String()
	auth := &authMock{isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }}
	svc := newSvc(svcOpts{repo: &repoMock{}, auth: auth})
	_, err := svc.ListSavedPlaces(ctx, uuid.New(), location.ListSavedPlacesQuery{TripID: &tid})
	assert.ErrorIs(t, err, location.ErrForbidden)
}

func TestListSavedPlaces_HappyPath(t *testing.T) {
	userID := uuid.New()
	id := uuid.New()
	repo := &repoMock{
		listSavedPlacesForUser: func(_ context.Context, u uuid.UUID, tid *uuid.UUID, cat *string) ([]location.SavedPlace, error) {
			assert.Equal(t, userID, u)
			assert.Nil(t, tid)
			assert.Nil(t, cat)
			return []location.SavedPlace{{ID: id, UserID: userID, Name: "a"}}, nil
		},
	}
	svc := newSvc(svcOpts{repo: repo})
	resp, err := svc.ListSavedPlaces(ctx, userID, location.ListSavedPlacesQuery{})
	require.NoError(t, err)
	require.Len(t, resp, 1)
	assert.Equal(t, id.String(), resp[0].ID)
}

// ---- SearchPlaces / NearbyPlaces / PlaceDetails ----

func TestSearchPlaces_ProviderMissing(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	_, err := svc.SearchPlaces(ctx, location.SearchPlacesQuery{Q: "x"})
	assert.ErrorIs(t, err, location.ErrProviderMissing)
}

func TestSearchPlaces_EmptyQuery(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}, places: &placesMock{}})
	_, err := svc.SearchPlaces(ctx, location.SearchPlacesQuery{Q: "  "})
	assert.ErrorIs(t, err, location.ErrInvalidPayload)
}

func TestSearchPlaces_UsesCache(t *testing.T) {
	rdb := newRedis(t)
	cache := location.NewCache(rdb, "test:")
	calls := 0
	places := &placesMock{search: func(_ context.Context, _ location.PlaceSearchRequest) ([]location.PlaceSummary, error) {
		calls++
		return []location.PlaceSummary{{PlaceID: "p1", Name: "Cafe"}}, nil
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, places: places, cache: cache})

	q := location.SearchPlacesQuery{Q: "cafe"}
	first, err := svc.SearchPlaces(ctx, q)
	require.NoError(t, err)
	assert.Len(t, first, 1)
	assert.Equal(t, 1, calls)

	second, err := svc.SearchPlaces(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, first, second)
	assert.Equal(t, 1, calls, "second call should hit cache")
}

func TestSearchPlaces_ProviderErrorMapped(t *testing.T) {
	places := &placesMock{search: func(_ context.Context, _ location.PlaceSearchRequest) ([]location.PlaceSummary, error) {
		return nil, location.ErrProviderNoResult
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, places: places})
	_, err := svc.SearchPlaces(ctx, location.SearchPlacesQuery{Q: "x"})
	assert.ErrorIs(t, err, location.ErrNoResults)
}

func TestSearchPlaces_ProviderUnavailableMapped(t *testing.T) {
	places := &placesMock{search: func(_ context.Context, _ location.PlaceSearchRequest) ([]location.PlaceSummary, error) {
		return nil, location.ErrProviderUnavailable
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, places: places})
	_, err := svc.SearchPlaces(ctx, location.SearchPlacesQuery{Q: "x"})
	assert.ErrorIs(t, err, location.ErrProviderMissing)
}

func TestNearbyPlaces_ProviderMissing(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	_, err := svc.NearbyPlaces(ctx, location.NearbyPlacesQuery{Lat: 1, Lng: 2})
	assert.ErrorIs(t, err, location.ErrProviderMissing)
}

func TestNearbyPlaces_HappyPath(t *testing.T) {
	places := &placesMock{nearby: func(_ context.Context, req location.NearbySearchRequest) ([]location.PlaceSummary, error) {
		assert.Equal(t, "restaurant", req.Type)
		return []location.PlaceSummary{{PlaceID: "p"}}, nil
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, places: places})
	out, err := svc.NearbyPlaces(ctx, location.NearbyPlacesQuery{Lat: 1, Lng: 2, Type: " restaurant "})
	require.NoError(t, err)
	assert.Len(t, out, 1)
}

func TestPlaceDetails_EmptyID(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}, places: &placesMock{}})
	_, err := svc.PlaceDetails(ctx, "  ", "en")
	assert.ErrorIs(t, err, location.ErrInvalidPayload)
}

func TestPlaceDetails_HappyPath(t *testing.T) {
	places := &placesMock{details: func(_ context.Context, id, lang string) (*location.PlaceDetails, error) {
		assert.Equal(t, "abc", id)
		assert.Equal(t, "en", lang)
		return &location.PlaceDetails{PlaceID: id}, nil
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, places: places})
	d, err := svc.PlaceDetails(ctx, "abc", "en")
	require.NoError(t, err)
	assert.Equal(t, "abc", d.PlaceID)
}

// ---- ComputeRoute ----

func TestComputeRoute_ProviderMissing(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	_, err := svc.ComputeRoute(ctx, location.ComputeRoutePayload{})
	assert.ErrorIs(t, err, location.ErrProviderMissing)
}

func TestComputeRoute_InvalidMode(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}, routes: &routesMock{}})
	_, err := svc.ComputeRoute(ctx, location.ComputeRoutePayload{Mode: "hover"})
	assert.ErrorIs(t, err, location.ErrInvalidPayload)
}

func TestComputeRoute_DefaultsToDriving(t *testing.T) {
	routes := &routesMock{compute: func(_ context.Context, req location.RouteRequest) (*location.Route, error) {
		assert.Equal(t, location.ModeDriving, req.Mode)
		return &location.Route{Mode: req.Mode, DistanceMeters: 100}, nil
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, routes: routes})
	r, err := svc.ComputeRoute(ctx, location.ComputeRoutePayload{
		Origin:      location.LatLngPayload{Lat: 1, Lng: 2},
		Destination: location.LatLngPayload{Lat: 3, Lng: 4},
	})
	require.NoError(t, err)
	assert.Equal(t, 100, r.DistanceMeters)
}

// ---- Forecast ----

func TestForecast_ProviderMissing(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	_, err := svc.Forecast(ctx, location.WeatherQuery{})
	assert.ErrorIs(t, err, location.ErrProviderMissing)
}

func TestForecast_UsesCache(t *testing.T) {
	rdb := newRedis(t)
	cache := location.NewCache(rdb, "wx:")
	calls := 0
	w := &weatherMock{forecast: func(_ context.Context, _ location.LatLng, _, units string) (*location.WeatherForecast, error) {
		calls++
		assert.Equal(t, "metric", units)
		return &location.WeatherForecast{Timezone: "UTC"}, nil
	}}
	svc := newSvc(svcOpts{repo: &repoMock{}, weather: w, cache: cache})
	q := location.WeatherQuery{Lat: 1, Lng: 2, Lang: "en"}
	_, err := svc.Forecast(ctx, q)
	require.NoError(t, err)
	_, err = svc.Forecast(ctx, q)
	require.NoError(t, err)
	assert.Equal(t, 1, calls)
}

// ---- Trip member positions ----

func TestShareTripPosition_ProviderMissing(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	err := svc.ShareTripPosition(ctx, uuid.New(), uuid.New(), location.ShareTripPositionPayload{Lat: 1, Lng: 2})
	assert.ErrorIs(t, err, location.ErrProviderMissing)
}

func TestShareTripPosition_MembershipEnforcedAndListRoundtrip(t *testing.T) {
	rdb := newRedis(t)
	positions := location.NewPositionStore(rdb)
	auth := &authMock{isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return true, nil }}
	svc := newSvc(svcOpts{repo: &repoMock{}, auth: auth, positions: positions})

	userID := uuid.New()
	tripID := uuid.New()
	err := svc.ShareTripPosition(ctx, userID, tripID, location.ShareTripPositionPayload{Lat: 10, Lng: 20, AccuracyM: 5})
	require.NoError(t, err)

	list, err := svc.ListTripPositions(ctx, userID, tripID)
	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, userID.String(), list[0].UserID)
	assert.Equal(t, 10.0, list[0].Lat)
	assert.Equal(t, 20.0, list[0].Lng)

	require.NoError(t, svc.RevokeTripPosition(ctx, userID, tripID))
	list, err = svc.ListTripPositions(ctx, userID, tripID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestShareTripPosition_NonMemberBlocked(t *testing.T) {
	rdb := newRedis(t)
	positions := location.NewPositionStore(rdb)
	auth := &authMock{isRoomMember: func(_ context.Context, _, _ uuid.UUID) (bool, error) { return false, nil }}
	svc := newSvc(svcOpts{repo: &repoMock{}, auth: auth, positions: positions})
	err := svc.ShareTripPosition(ctx, uuid.New(), uuid.New(), location.ShareTripPositionPayload{Lat: 1, Lng: 2})
	assert.ErrorIs(t, err, location.ErrForbidden)
}

func TestRevokeAllUserPositions_NilPositionsIsNoop(t *testing.T) {
	svc := newSvc(svcOpts{repo: &repoMock{}})
	require.NoError(t, svc.RevokeAllUserPositions(ctx, uuid.New()))
}

func TestRevokeAllUserPositions_RemovesAcrossTrips(t *testing.T) {
	rdb := newRedis(t)
	positions := location.NewPositionStore(rdb)
	auth := &authMock{}
	svc := newSvc(svcOpts{repo: &repoMock{}, auth: auth, positions: positions})

	userID := uuid.New()
	trip1 := uuid.New()
	trip2 := uuid.New()
	require.NoError(t, svc.ShareTripPosition(ctx, userID, trip1, location.ShareTripPositionPayload{Lat: 1, Lng: 2}))
	require.NoError(t, svc.ShareTripPosition(ctx, userID, trip2, location.ShareTripPositionPayload{Lat: 3, Lng: 4}))

	require.NoError(t, svc.RevokeAllUserPositions(ctx, userID))
	l1, _ := svc.ListTripPositions(ctx, userID, trip1)
	l2, _ := svc.ListTripPositions(ctx, userID, trip2)
	assert.Empty(t, l1)
	assert.Empty(t, l2)
}

// ---- audit error path (audit failure does not break request) ----

func TestCreate_AuditWriteFailureDoesNotFailRequest(t *testing.T) {
	repo := &repoMock{}
	aud := &auditMock{err: errors.New("audit down")}
	svc := newSvc(svcOpts{repo: repo, audit: aud})
	_, err := svc.CreateSavedPlace(ctx, uuid.New(), location.CreateSavedPlacePayload{Name: "n", Lat: 1, Lng: 2})
	require.NoError(t, err)
	require.Len(t, aud.logs, 1)
}

// Sanity: audit meta context propagates without panicking on empty.
func TestAuditMetaContextRoundtrip(t *testing.T) {
	c := location.WithAuditMeta(ctx, "1.2.3.4", "ua")
	assert.NotNil(t, c)
	c = location.WithAuditMeta(ctx, "", "")
	assert.Equal(t, ctx, c)
	// avoid time-unused warning
	_ = time.Now()
}

package profile_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/profile"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx                func(context.Context, func(profile.IRepository) error) error
	findProfileByUserID   func(context.Context, uuid.UUID) (*profile.Profile, error)
	findProfileByUsername func(context.Context, string) (*profile.Profile, error)
	usernameExists        func(context.Context, string, uuid.UUID) (bool, error)
	upsertProfile         func(context.Context, *profile.Profile) error

	follow        func(context.Context, uuid.UUID, uuid.UUID) error
	unfollow      func(context.Context, uuid.UUID, uuid.UUID) error
	isFollowing   func(context.Context, uuid.UUID, uuid.UUID) (bool, error)
	areFollowing  func(context.Context, uuid.UUID, []uuid.UUID) (map[uuid.UUID]bool, error)
	listFollowers func(context.Context, uuid.UUID, *profile.FollowCursor, int) ([]profile.Follow, error)
	listFollowing func(context.Context, uuid.UUID, *profile.FollowCursor, int) ([]profile.Follow, error)
	getCounts     func(context.Context, []uuid.UUID) (map[uuid.UUID]profile.FollowCounts, error)

	findUsersByIDs func(context.Context, []uuid.UUID) (map[uuid.UUID]auth.User, error)

	insertFeedEntries func(context.Context, []profile.FeedEntry) error
	listFeed          func(context.Context, uuid.UUID, *profile.FeedCursor, int) ([]profile.FeedEntry, error)

	friendsOfFollowees func(context.Context, uuid.UUID, int) ([]uuid.UUID, error)
	trendingUsers      func(context.Context, time.Time, uuid.UUID, int) ([]uuid.UUID, error)
}

func (r *repoMock) WithTx(c context.Context, fn func(profile.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(c, fn)
	}
	return fn(r)
}
func (r *repoMock) FindProfileByUserID(c context.Context, id uuid.UUID) (*profile.Profile, error) {
	if r.findProfileByUserID != nil {
		return r.findProfileByUserID(c, id)
	}
	return nil, nil
}
func (r *repoMock) FindProfileByUsername(c context.Context, u string) (*profile.Profile, error) {
	if r.findProfileByUsername != nil {
		return r.findProfileByUsername(c, u)
	}
	return nil, nil
}
func (r *repoMock) UsernameExists(c context.Context, u string, except uuid.UUID) (bool, error) {
	if r.usernameExists != nil {
		return r.usernameExists(c, u, except)
	}
	return false, nil
}
func (r *repoMock) UpsertProfile(c context.Context, p *profile.Profile) error {
	if r.upsertProfile != nil {
		return r.upsertProfile(c, p)
	}
	return nil
}
func (r *repoMock) Follow(c context.Context, a, b uuid.UUID) error {
	if r.follow != nil {
		return r.follow(c, a, b)
	}
	return nil
}
func (r *repoMock) Unfollow(c context.Context, a, b uuid.UUID) error {
	if r.unfollow != nil {
		return r.unfollow(c, a, b)
	}
	return nil
}
func (r *repoMock) IsFollowing(c context.Context, a, b uuid.UUID) (bool, error) {
	if r.isFollowing != nil {
		return r.isFollowing(c, a, b)
	}
	return false, nil
}
func (r *repoMock) AreFollowing(c context.Context, a uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	if r.areFollowing != nil {
		return r.areFollowing(c, a, ids)
	}
	return map[uuid.UUID]bool{}, nil
}
func (r *repoMock) ListFollowers(c context.Context, id uuid.UUID, cur *profile.FollowCursor, l int) ([]profile.Follow, error) {
	if r.listFollowers != nil {
		return r.listFollowers(c, id, cur, l)
	}
	return nil, nil
}
func (r *repoMock) ListFollowing(c context.Context, id uuid.UUID, cur *profile.FollowCursor, l int) ([]profile.Follow, error) {
	if r.listFollowing != nil {
		return r.listFollowing(c, id, cur, l)
	}
	return nil, nil
}
func (r *repoMock) GetCounts(c context.Context, ids []uuid.UUID) (map[uuid.UUID]profile.FollowCounts, error) {
	if r.getCounts != nil {
		return r.getCounts(c, ids)
	}
	return map[uuid.UUID]profile.FollowCounts{}, nil
}
func (r *repoMock) FindUsersByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	if r.findUsersByIDs != nil {
		return r.findUsersByIDs(c, ids)
	}
	return map[uuid.UUID]auth.User{}, nil
}
func (r *repoMock) InsertFeedEntries(c context.Context, e []profile.FeedEntry) error {
	if r.insertFeedEntries != nil {
		return r.insertFeedEntries(c, e)
	}
	return nil
}
func (r *repoMock) ListFeed(c context.Context, id uuid.UUID, cur *profile.FeedCursor, l int) ([]profile.FeedEntry, error) {
	if r.listFeed != nil {
		return r.listFeed(c, id, cur, l)
	}
	return nil, nil
}
func (r *repoMock) FriendsOfFollowees(c context.Context, id uuid.UUID, l int) ([]uuid.UUID, error) {
	if r.friendsOfFollowees != nil {
		return r.friendsOfFollowees(c, id, l)
	}
	return nil, nil
}
func (r *repoMock) TrendingUsers(c context.Context, since time.Time, ex uuid.UUID, l int) ([]uuid.UUID, error) {
	if r.trendingUsers != nil {
		return r.trendingUsers(c, since, ex, l)
	}
	return nil, nil
}

func newSvc(r *repoMock) profile.IService {
	return profile.NewService(profile.ServiceConfig{Repo: r})
}

// ---- GetMyProfile ----

func TestGetMyProfile_ProvisionsPlaceholderWhenMissing(t *testing.T) {
	userID := uuid.New()
	var upserted *profile.Profile
	r := &repoMock{
		findProfileByUserID: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) { return nil, nil },
		upsertProfile: func(_ context.Context, p *profile.Profile) error {
			upserted = p
			return nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{ids[0]: {ID: ids[0], Name: "Peter"}}, nil
		},
		getCounts: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]profile.FollowCounts, error) {
			return map[uuid.UUID]profile.FollowCounts{userID: {FollowersCount: 3, FollowingCount: 5}}, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.GetMyProfile(ctx, userID)
	require.NoError(t, err)
	require.NotNil(t, upserted)
	assert.Equal(t, userID, upserted.UserID)
	assert.NotEmpty(t, upserted.Username)
	assert.Equal(t, "Peter", resp.Name)
	assert.True(t, resp.IsSelf)
	assert.Equal(t, 3, resp.FollowersCount)
	assert.Equal(t, 5, resp.FollowingCount)
}

func TestGetMyProfile_UsersMissingReturnsError(t *testing.T) {
	userID := uuid.New()
	r := &repoMock{
		findProfileByUserID: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{UserID: userID, Username: "peter"}, nil
		},
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{}, nil
		},
	}
	svc := newSvc(r)
	_, err := svc.GetMyProfile(ctx, userID)
	assert.ErrorIs(t, err, profile.ErrUserNotFound)
}

// ---- GetProfileByUsername ----

func TestGetProfileByUsername_NotFound(t *testing.T) {
	r := &repoMock{
		findProfileByUsername: func(_ context.Context, _ string) (*profile.Profile, error) { return nil, nil },
	}
	svc := newSvc(r)
	_, err := svc.GetProfileByUsername(ctx, uuid.New(), "someone")
	assert.ErrorIs(t, err, profile.ErrProfileNotFound)
}

func TestGetProfileByUsername_SelfSkipsFollowingLookup(t *testing.T) {
	uid := uuid.New()
	areFollowingCalled := false
	r := &repoMock{
		findProfileByUsername: func(_ context.Context, _ string) (*profile.Profile, error) {
			return &profile.Profile{UserID: uid, Username: "peter"}, nil
		},
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{uid: {ID: uid, Name: "Peter"}}, nil
		},
		areFollowing: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			areFollowingCalled = true
			return nil, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.GetProfileByUsername(ctx, uid, "peter")
	require.NoError(t, err)
	assert.True(t, resp.IsSelf)
	assert.False(t, resp.IsFollowing)
	assert.False(t, areFollowingCalled, "AreFollowing should be skipped for self")
}

func TestGetProfileByUsername_ViewerIsFollowing(t *testing.T) {
	target := uuid.New()
	viewer := uuid.New()
	r := &repoMock{
		findProfileByUsername: func(_ context.Context, _ string) (*profile.Profile, error) {
			return &profile.Profile{UserID: target, Username: "peter"}, nil
		},
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{target: {ID: target, Name: "Peter"}}, nil
		},
		areFollowing: func(_ context.Context, follower uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
			assert.Equal(t, viewer, follower)
			assert.Equal(t, []uuid.UUID{target}, ids)
			return map[uuid.UUID]bool{target: true}, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.GetProfileByUsername(ctx, viewer, "peter")
	require.NoError(t, err)
	assert.False(t, resp.IsSelf)
	assert.True(t, resp.IsFollowing)
}

func TestGetProfileByUsername_AnonymousViewerSkipsFollowingLookup(t *testing.T) {
	target := uuid.New()
	r := &repoMock{
		findProfileByUsername: func(_ context.Context, _ string) (*profile.Profile, error) {
			return &profile.Profile{UserID: target, Username: "peter", AvatarURL: "https://a"}, nil
		},
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{target: {ID: target, Name: "Peter", AvatarURL: "https://u"}}, nil
		},
		areFollowing: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) (map[uuid.UUID]bool, error) {
			t.Fatal("should not call AreFollowing")
			return nil, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.GetProfileByUsername(ctx, uuid.Nil, "peter")
	require.NoError(t, err)
	assert.False(t, resp.IsFollowing)
	assert.Equal(t, "https://a", resp.AvatarURL) // profile avatar wins over user avatar
}

// ---- UpdateMyProfile ----

func TestUpdateMyProfile_ValidatesUsername(t *testing.T) {
	uid := uuid.New()
	existing := &profile.Profile{UserID: uid, Username: "existing"}
	r := &repoMock{
		findProfileByUserID: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) { return existing, nil },
	}
	svc := newSvc(r)

	tooShort := "ab"
	_, err := svc.UpdateMyProfile(ctx, uid, profile.UpdateProfilePayload{Username: &tooShort})
	assert.ErrorIs(t, err, profile.ErrInvalidUsername)

	bad := "has space"
	_, err = svc.UpdateMyProfile(ctx, uid, profile.UpdateProfilePayload{Username: &bad})
	assert.ErrorIs(t, err, profile.ErrInvalidUsername)

	reserved := "admin"
	_, err = svc.UpdateMyProfile(ctx, uid, profile.UpdateProfilePayload{Username: &reserved})
	assert.ErrorIs(t, err, profile.ErrReservedUsername)
}

func TestUpdateMyProfile_UsernameTaken(t *testing.T) {
	uid := uuid.New()
	existing := &profile.Profile{UserID: uid, Username: "old"}
	r := &repoMock{
		findProfileByUserID: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) { return existing, nil },
		usernameExists:      func(_ context.Context, _ string, _ uuid.UUID) (bool, error) { return true, nil },
	}
	svc := newSvc(r)
	newName := "newname"
	_, err := svc.UpdateMyProfile(ctx, uid, profile.UpdateProfilePayload{Username: &newName})
	assert.ErrorIs(t, err, profile.ErrUsernameTaken)
}

func TestUpdateMyProfile_SameUsernameCaseInsensitiveSkipsTakenCheck(t *testing.T) {
	uid := uuid.New()
	existing := &profile.Profile{UserID: uid, Username: "Peter"}
	takenChecked := false
	upserted := false
	r := &repoMock{
		findProfileByUserID: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) { return existing, nil },
		usernameExists: func(_ context.Context, _ string, _ uuid.UUID) (bool, error) {
			takenChecked = true
			return false, nil
		},
		upsertProfile: func(_ context.Context, _ *profile.Profile) error { upserted = true; return nil },
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{uid: {ID: uid, Name: "P"}}, nil
		},
	}
	svc := newSvc(r)
	same := "peter"
	_, err := svc.UpdateMyProfile(ctx, uid, profile.UpdateProfilePayload{Username: &same})
	require.NoError(t, err)
	assert.False(t, takenChecked)
	assert.True(t, upserted)
}

func TestUpdateMyProfile_UpdatesFieldsAndNormalizesTags(t *testing.T) {
	uid := uuid.New()
	existing := &profile.Profile{UserID: uid, Username: "peter"}
	var saved *profile.Profile
	r := &repoMock{
		findProfileByUserID: func(_ context.Context, _ uuid.UUID) (*profile.Profile, error) { return existing, nil },
		upsertProfile:       func(_ context.Context, p *profile.Profile) error { saved = p; return nil },
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{uid: {ID: uid, Name: "P"}}, nil
		},
	}
	svc := newSvc(r)
	bio := "wanderer"
	avatar := "https://a"
	tags := []string{" Beach ", "beach", "Mountain", ""}
	ig := "https://ig"
	_, err := svc.UpdateMyProfile(ctx, uid, profile.UpdateProfilePayload{
		Bio:             &bio,
		AvatarURL:       &avatar,
		TravelTags:      &tags,
		SocialInstagram: &ig,
	})
	require.NoError(t, err)
	require.NotNil(t, saved)
	assert.Equal(t, "wanderer", saved.Bio)
	assert.Equal(t, "https://a", saved.AvatarURL)
	assert.Equal(t, "https://ig", saved.SocialInstagram)
	assert.Equal(t, pq.StringArray{"Beach", "Mountain"}, saved.TravelTags)
}

// ---- Follow / Unfollow ----

func TestFollow_SelfRejected(t *testing.T) {
	id := uuid.New()
	svc := newSvc(&repoMock{})
	assert.ErrorIs(t, svc.Follow(ctx, id, id), profile.ErrCannotFollowSelf)
	assert.ErrorIs(t, svc.Unfollow(ctx, id, id), profile.ErrCannotFollowSelf)
}

func TestFollow_TargetUserMissing(t *testing.T) {
	r := &repoMock{
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{}, nil
		},
	}
	svc := newSvc(r)
	err := svc.Follow(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, profile.ErrUserNotFound)
}

func TestFollow_HappyPath(t *testing.T) {
	target := uuid.New()
	called := false
	r := &repoMock{
		findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{target: {ID: target}}, nil
		},
		follow: func(_ context.Context, a, b uuid.UUID) error {
			called = true
			assert.Equal(t, target, b)
			return nil
		},
	}
	svc := newSvc(r)
	require.NoError(t, svc.Follow(ctx, uuid.New(), target))
	assert.True(t, called)
}

func TestUnfollow_HappyPath(t *testing.T) {
	called := false
	r := &repoMock{unfollow: func(_ context.Context, _, _ uuid.UUID) error { called = true; return nil }}
	svc := newSvc(r)
	require.NoError(t, svc.Unfollow(ctx, uuid.New(), uuid.New()))
	assert.True(t, called)
}

// ---- ListFollowers / ListFollowing ----

func TestListFollowers_EmptyReturnsEmpty(t *testing.T) {
	svc := newSvc(&repoMock{listFollowers: func(_ context.Context, _ uuid.UUID, _ *profile.FollowCursor, _ int) ([]profile.Follow, error) {
		return []profile.Follow{}, nil
	}})
	resp, err := svc.ListFollowers(ctx, uuid.New(), uuid.New(), nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []profile.UserSummary{}, resp.Users)
	assert.Empty(t, resp.NextCursor)
}

func TestListFollowers_HydratesAndEncodesCursor(t *testing.T) {
	target := uuid.New()
	follower1 := uuid.New()
	follower2 := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(-time.Hour)
	r := &repoMock{
		listFollowers: func(_ context.Context, _ uuid.UUID, _ *profile.FollowCursor, _ int) ([]profile.Follow, error) {
			return []profile.Follow{
				{FollowerID: follower1, FolloweeID: target, CreatedAt: t1},
				{FollowerID: follower2, FolloweeID: target, CreatedAt: t2},
			}, nil
		},
		findProfileByUserID: func(_ context.Context, id uuid.UUID) (*profile.Profile, error) {
			return &profile.Profile{UserID: id, Username: id.String()[:8]}, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "n"}
			}
			return out, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.ListFollowers(ctx, uuid.New(), target, nil, 20)
	require.NoError(t, err)
	assert.Len(t, resp.Users, 2)
	assert.NotEmpty(t, resp.NextCursor)
	assert.Contains(t, resp.NextCursor, follower2.String())
}

func TestListFollowing_HydratesFolloweeSide(t *testing.T) {
	src := uuid.New()
	followee := uuid.New()
	r := &repoMock{
		listFollowing: func(_ context.Context, id uuid.UUID, _ *profile.FollowCursor, _ int) ([]profile.Follow, error) {
			assert.Equal(t, src, id)
			return []profile.Follow{{FollowerID: src, FolloweeID: followee, CreatedAt: time.Now()}}, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "u"}
			}
			return out, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.ListFollowing(ctx, uuid.New(), src, nil, 20)
	require.NoError(t, err)
	require.Len(t, resp.Users, 1)
	assert.Equal(t, followee.String(), resp.Users[0].UserID)
}

// ---- Feed ----

func TestListFeed_EmptyReturnsEmpty(t *testing.T) {
	svc := newSvc(&repoMock{listFeed: func(_ context.Context, _ uuid.UUID, _ *profile.FeedCursor, _ int) ([]profile.FeedEntry, error) {
		return []profile.FeedEntry{}, nil
	}})
	resp, err := svc.ListFeed(ctx, uuid.New(), nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []profile.FeedItemResponse{}, resp.Items)
}

func TestListFeed_HydratesActorsAndDedupes(t *testing.T) {
	viewer := uuid.New()
	actor1 := uuid.New()
	actor2 := uuid.New()
	tripID := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(-time.Hour)
	t3 := t1.Add(-2 * time.Hour)
	r := &repoMock{
		listFeed: func(_ context.Context, id uuid.UUID, _ *profile.FeedCursor, _ int) ([]profile.FeedEntry, error) {
			assert.Equal(t, viewer, id)
			return []profile.FeedEntry{
				{ID: uuid.New(), UserID: viewer, ActorID: actor1, EventType: profile.EventTripPublished, TripID: &tripID, PublishedAt: t1},
				{ID: uuid.New(), UserID: viewer, ActorID: actor1, EventType: profile.EventTripPublished, PublishedAt: t2},
				{ID: uuid.New(), UserID: viewer, ActorID: actor2, EventType: profile.EventTripPublished, PublishedAt: t3},
			}, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			assert.ElementsMatch(t, []uuid.UUID{actor1, actor2}, ids, "actor IDs should be deduped")
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "u"}
			}
			return out, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.ListFeed(ctx, viewer, nil, 20)
	require.NoError(t, err)
	assert.Len(t, resp.Items, 3)
	assert.Equal(t, tripID.String(), resp.Items[0].TripID)
	assert.Empty(t, resp.Items[1].TripID)
	assert.NotEmpty(t, resp.NextCursor)
}

// ---- Recommendations ----

func TestRecommendations_UsesFriendsOfFolloweesFirst(t *testing.T) {
	uid := uuid.New()
	fofID := uuid.New()
	trendingCalled := false
	r := &repoMock{
		friendsOfFollowees: func(_ context.Context, _ uuid.UUID, limit int) ([]uuid.UUID, error) {
			// return same count as limit so trending is not needed
			out := make([]uuid.UUID, 0, limit)
			for i := 0; i < limit; i++ {
				out = append(out, uuid.New())
			}
			out[0] = fofID
			return out, nil
		},
		trendingUsers: func(_ context.Context, _ time.Time, _ uuid.UUID, _ int) ([]uuid.UUID, error) {
			trendingCalled = true
			return nil, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "u"}
			}
			return out, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.Recommendations(ctx, uid, 10)
	require.NoError(t, err)
	assert.False(t, trendingCalled)
	assert.Len(t, resp.Users, 10)
	assert.Equal(t, fofID.String(), resp.Users[0].UserID)
}

func TestRecommendations_FallsBackToTrendingWhenShort(t *testing.T) {
	uid := uuid.New()
	fofID := uuid.New()
	trendingID := uuid.New()
	r := &repoMock{
		friendsOfFollowees: func(_ context.Context, _ uuid.UUID, _ int) ([]uuid.UUID, error) {
			return []uuid.UUID{fofID}, nil
		},
		trendingUsers: func(_ context.Context, _ time.Time, ex uuid.UUID, _ int) ([]uuid.UUID, error) {
			assert.Equal(t, uid, ex)
			return []uuid.UUID{fofID, trendingID}, nil // fofID should be deduped
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "u"}
			}
			return out, nil
		},
	}
	svc := newSvc(r)
	resp, err := svc.Recommendations(ctx, uid, 5)
	require.NoError(t, err)
	require.Len(t, resp.Users, 2)
	assert.Equal(t, fofID.String(), resp.Users[0].UserID)
	assert.Equal(t, trendingID.String(), resp.Users[1].UserID)
}

func TestRecommendations_EmptyReturnsEmpty(t *testing.T) {
	r := &repoMock{
		friendsOfFollowees: func(_ context.Context, _ uuid.UUID, _ int) ([]uuid.UUID, error) { return nil, nil },
		trendingUsers:      func(_ context.Context, _ time.Time, _ uuid.UUID, _ int) ([]uuid.UUID, error) { return nil, nil },
	}
	svc := newSvc(r)
	resp, err := svc.Recommendations(ctx, uuid.New(), 10)
	require.NoError(t, err)
	assert.Equal(t, []profile.UserSummary{}, resp.Users)
}

func TestRecommendations_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db")
	r := &repoMock{friendsOfFollowees: func(_ context.Context, _ uuid.UUID, _ int) ([]uuid.UUID, error) { return nil, boom }}
	svc := newSvc(r)
	_, err := svc.Recommendations(ctx, uuid.New(), 10)
	assert.ErrorIs(t, err, boom)
}

// ---- cursor decoding ----

func TestDecodeCursors(t *testing.T) {
	id := uuid.New()
	tm := time.Now().UTC().Truncate(time.Nanosecond)
	token := tm.Format(time.RFC3339Nano) + ":" + id.String()

	f, err := profile.DecodeFollowCursor(token)
	require.NoError(t, err)
	require.NotNil(t, f)
	assert.Equal(t, id, f.OtherID)

	fc, err := profile.DecodeFeedCursor(token)
	require.NoError(t, err)
	require.NotNil(t, fc)
	assert.Equal(t, id, fc.ID)

	// empty
	f, err = profile.DecodeFollowCursor("")
	require.NoError(t, err)
	assert.Nil(t, f)

	// invalid
	_, err = profile.DecodeFollowCursor("garbage")
	assert.Error(t, err)
}

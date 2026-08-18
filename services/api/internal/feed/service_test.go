package feed_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/feed"
	"github.com/Ans1110/trip-app/internal/post"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

type repoMock struct {
	listRanked func(context.Context, uuid.UUID, *feed.Cursor, int) ([]feed.RankedPost, time.Time, error)
}

func (r *repoMock) ListRanked(c context.Context, viewer uuid.UUID, cur *feed.Cursor, limit int) ([]feed.RankedPost, time.Time, error) {
	if r.listRanked != nil {
		return r.listRanked(c, viewer, cur, limit)
	}
	return nil, time.Time{}, nil
}

type postsMock struct {
	called       bool
	lastViewerID uuid.UUID
	lastIDs      []uuid.UUID
	get          func(context.Context, uuid.UUID, []uuid.UUID) ([]post.PostResponse, error)
}

func (p *postsMock) GetPostsByIDs(c context.Context, viewerID uuid.UUID, ids []uuid.UUID) ([]post.PostResponse, error) {
	p.called = true
	p.lastViewerID = viewerID
	p.lastIDs = ids
	if p.get != nil {
		return p.get(c, viewerID, ids)
	}
	return []post.PostResponse{}, nil
}

func newSvc(repo *repoMock, posts feed.PostLookup) feed.IService {
	return feed.NewService(feed.ServiceConfig{Repo: repo, Posts: posts})
}

func TestListFeed_EmptyRows(t *testing.T) {
	repo := &repoMock{listRanked: func(_ context.Context, _ uuid.UUID, _ *feed.Cursor, _ int) ([]feed.RankedPost, time.Time, error) {
		return []feed.RankedPost{}, time.Now(), nil
	}}
	posts := &postsMock{}
	svc := newSvc(repo, posts)
	resp, err := svc.ListFeed(ctx, uuid.New(), nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, resp.Posts)
	assert.Empty(t, resp.NextCursor)
	assert.False(t, posts.called)
}

func TestListFeed_HydratesAndEncodesCursor(t *testing.T) {
	viewer := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(-time.Hour)
	ref := t1.Add(time.Minute)
	repo := &repoMock{listRanked: func(_ context.Context, gotViewer uuid.UUID, _ *feed.Cursor, _ int) ([]feed.RankedPost, time.Time, error) {
		assert.Equal(t, viewer, gotViewer)
		return []feed.RankedPost{
			{ID: id1, PublishedAt: t1, Score: 90},
			{ID: id2, PublishedAt: t2, Score: 50},
		}, ref, nil
	}}
	posts := &postsMock{get: func(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]post.PostResponse, error) {
		out := make([]post.PostResponse, len(ids))
		for i, id := range ids {
			out[i] = post.PostResponse{ID: id.String()}
		}
		return out, nil
	}}
	svc := newSvc(repo, posts)
	resp, err := svc.ListFeed(ctx, viewer, nil, 20)
	require.NoError(t, err)
	assert.Equal(t, viewer, posts.lastViewerID)
	assert.Equal(t, []uuid.UUID{id1, id2}, posts.lastIDs)
	require.Len(t, resp.Posts, 2)

	require.NotEmpty(t, resp.NextCursor)
	raw, err := base64.RawURLEncoding.DecodeString(resp.NextCursor)
	require.NoError(t, err)
	var payload struct {
		RefTime     time.Time `json:"r"`
		Score       float64   `json:"s"`
		PublishedAt time.Time `json:"p"`
		PostID      uuid.UUID `json:"i"`
	}
	require.NoError(t, json.Unmarshal(raw, &payload))
	assert.True(t, payload.RefTime.Equal(ref))
	assert.Equal(t, 50.0, payload.Score)
	assert.True(t, payload.PublishedAt.Equal(t2))
	assert.Equal(t, id2, payload.PostID)
}

func TestListFeed_NilLookup(t *testing.T) {
	repo := &repoMock{listRanked: func(_ context.Context, _ uuid.UUID, _ *feed.Cursor, _ int) ([]feed.RankedPost, time.Time, error) {
		return []feed.RankedPost{{ID: uuid.New(), PublishedAt: time.Now(), Score: 1}}, time.Now(), nil
	}}
	svc := newSvc(repo, nil)
	resp, err := svc.ListFeed(ctx, uuid.New(), nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, resp.Posts)
	assert.NotEmpty(t, resp.NextCursor)
}

func TestListFeed_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db down")
	repo := &repoMock{listRanked: func(_ context.Context, _ uuid.UUID, _ *feed.Cursor, _ int) ([]feed.RankedPost, time.Time, error) {
		return nil, time.Time{}, boom
	}}
	svc := newSvc(repo, nil)
	_, err := svc.ListFeed(ctx, uuid.New(), nil, 20)
	assert.ErrorIs(t, err, boom)
}

func TestListFeed_HydrationErrorPassthrough(t *testing.T) {
	boom := errors.New("posts down")
	repo := &repoMock{listRanked: func(_ context.Context, _ uuid.UUID, _ *feed.Cursor, _ int) ([]feed.RankedPost, time.Time, error) {
		return []feed.RankedPost{{ID: uuid.New(), PublishedAt: time.Now(), Score: 1}}, time.Now(), nil
	}}
	posts := &postsMock{get: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]post.PostResponse, error) {
		return nil, boom
	}}
	svc := newSvc(repo, posts)
	_, err := svc.ListFeed(ctx, uuid.New(), nil, 20)
	assert.ErrorIs(t, err, boom)
}

func TestListFeed_CursorForwarded(t *testing.T) {
	want := &feed.Cursor{RefTime: time.Now().UTC(), Score: 1, PublishedAt: time.Now().UTC(), PostID: uuid.New()}
	var got *feed.Cursor
	repo := &repoMock{listRanked: func(_ context.Context, _ uuid.UUID, c *feed.Cursor, limit int) ([]feed.RankedPost, time.Time, error) {
		got = c
		assert.Equal(t, 33, limit)
		return nil, time.Now(), nil
	}}
	svc := newSvc(repo, nil)
	_, err := svc.ListFeed(ctx, uuid.New(), want, 33)
	require.NoError(t, err)
	assert.Same(t, want, got)
}

func TestDecodeCursor(t *testing.T) {
	c, err := feed.DecodeCursor("")
	require.NoError(t, err)
	assert.Nil(t, c)

	_, err = feed.DecodeCursor("!!!not-b64!!!")
	assert.Error(t, err)

	// missing PostID -> invalid
	bad := feed.Cursor{RefTime: time.Now().UTC(), Score: 1, PublishedAt: time.Now().UTC()}
	b, _ := json.Marshal(bad)
	token := base64.RawURLEncoding.EncodeToString(b)
	_, err = feed.DecodeCursor(token)
	assert.Error(t, err)

	// zero RefTime -> invalid
	bad2 := feed.Cursor{PostID: uuid.New(), Score: 1, PublishedAt: time.Now().UTC()}
	b, _ = json.Marshal(bad2)
	token = base64.RawURLEncoding.EncodeToString(b)
	_, err = feed.DecodeCursor(token)
	assert.Error(t, err)

	// valid
	good := feed.Cursor{RefTime: time.Now().UTC(), Score: 5, PublishedAt: time.Now().UTC(), PostID: uuid.New()}
	b, _ = json.Marshal(good)
	token = base64.RawURLEncoding.EncodeToString(b)
	got, err := feed.DecodeCursor(token)
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, good.PostID, got.PostID)
	assert.Equal(t, good.Score, got.Score)
}

package search_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/post"
	"github.com/Ans1110/trip-app/internal/search"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

type repoMock struct {
	searchPosts func(context.Context, string, *search.Cursor, int) ([]search.PostHit, error)
}

func (r *repoMock) SearchPosts(c context.Context, q string, cur *search.Cursor, limit int) ([]search.PostHit, error) {
	if r.searchPosts != nil {
		return r.searchPosts(c, q, cur, limit)
	}
	return nil, nil
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

func newSvc(repo *repoMock, posts search.PostLookup) search.IService {
	return search.NewService(search.ServiceConfig{Repo: repo, Posts: posts})
}

func TestSearchPosts_EmptyQuery(t *testing.T) {
	svc := newSvc(&repoMock{}, nil)
	_, err := svc.SearchPosts(ctx, uuid.New(), "   ", nil, 20)
	assert.ErrorIs(t, err, search.ErrQueryRequired)
}

func TestSearchPosts_EmptyHits(t *testing.T) {
	repo := &repoMock{searchPosts: func(_ context.Context, q string, _ *search.Cursor, _ int) ([]search.PostHit, error) {
		assert.Equal(t, "hello", q)
		return []search.PostHit{}, nil
	}}
	posts := &postsMock{}
	svc := newSvc(repo, posts)
	resp, err := svc.SearchPosts(ctx, uuid.New(), " hello ", nil, 20)
	require.NoError(t, err)
	assert.Equal(t, "hello", resp.Query)
	assert.Equal(t, []post.PostResponse{}, resp.Posts)
	assert.Empty(t, resp.NextCursor)
	assert.False(t, posts.called, "post lookup skipped when no hits")
}

func TestSearchPosts_HydratesAndEncodesCursor(t *testing.T) {
	viewer := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(-time.Hour)
	repo := &repoMock{searchPosts: func(_ context.Context, _ string, _ *search.Cursor, _ int) ([]search.PostHit, error) {
		return []search.PostHit{
			{ID: id1, PublishedAt: t1},
			{ID: id2, PublishedAt: t2},
		}, nil
	}}
	posts := &postsMock{get: func(_ context.Context, _ uuid.UUID, ids []uuid.UUID) ([]post.PostResponse, error) {
		out := make([]post.PostResponse, len(ids))
		for i, id := range ids {
			out[i] = post.PostResponse{ID: id.String()}
		}
		return out, nil
	}}
	svc := newSvc(repo, posts)
	resp, err := svc.SearchPosts(ctx, viewer, "trip", nil, 20)
	require.NoError(t, err)
	assert.Equal(t, viewer, posts.lastViewerID)
	assert.Equal(t, []uuid.UUID{id1, id2}, posts.lastIDs)
	require.Len(t, resp.Posts, 2)
	assert.Equal(t, id1.String(), resp.Posts[0].ID)
	expected := t2.Format(time.RFC3339Nano) + ":" + id2.String()
	assert.Equal(t, expected, resp.NextCursor)
}

func TestSearchPosts_NilLookupReturnsEmptyPosts(t *testing.T) {
	id := uuid.New()
	tm := time.Now().UTC()
	repo := &repoMock{searchPosts: func(_ context.Context, _ string, _ *search.Cursor, _ int) ([]search.PostHit, error) {
		return []search.PostHit{{ID: id, PublishedAt: tm}}, nil
	}}
	svc := newSvc(repo, nil)
	resp, err := svc.SearchPosts(ctx, uuid.New(), "x", nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, resp.Posts)
	assert.NotEmpty(t, resp.NextCursor)
}

func TestSearchPosts_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db down")
	repo := &repoMock{searchPosts: func(_ context.Context, _ string, _ *search.Cursor, _ int) ([]search.PostHit, error) {
		return nil, boom
	}}
	svc := newSvc(repo, nil)
	_, err := svc.SearchPosts(ctx, uuid.New(), "q", nil, 20)
	assert.ErrorIs(t, err, boom)
}

func TestSearchPosts_HydrationErrorPassthrough(t *testing.T) {
	boom := errors.New("posts down")
	repo := &repoMock{searchPosts: func(_ context.Context, _ string, _ *search.Cursor, _ int) ([]search.PostHit, error) {
		return []search.PostHit{{ID: uuid.New(), PublishedAt: time.Now()}}, nil
	}}
	posts := &postsMock{get: func(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]post.PostResponse, error) {
		return nil, boom
	}}
	svc := newSvc(repo, posts)
	_, err := svc.SearchPosts(ctx, uuid.New(), "q", nil, 20)
	assert.ErrorIs(t, err, boom)
}

func TestSearchPosts_CursorForwarded(t *testing.T) {
	want := &search.Cursor{PublishedAt: time.Now().UTC(), ID: uuid.New()}
	var got *search.Cursor
	repo := &repoMock{searchPosts: func(_ context.Context, _ string, c *search.Cursor, limit int) ([]search.PostHit, error) {
		got = c
		assert.Equal(t, 42, limit)
		return nil, nil
	}}
	svc := newSvc(repo, nil)
	_, err := svc.SearchPosts(ctx, uuid.New(), "x", want, 42)
	require.NoError(t, err)
	assert.Same(t, want, got)
}

func TestDecodeCursor(t *testing.T) {
	id := uuid.New()
	tm := time.Now().UTC().Truncate(time.Nanosecond)
	token := tm.Format(time.RFC3339Nano) + ":" + id.String()
	c, err := search.DecodeCursor(token)
	require.NoError(t, err)
	require.NotNil(t, c)
	assert.Equal(t, id, c.ID)
	assert.True(t, c.PublishedAt.Equal(tm))

	c, err = search.DecodeCursor("  ")
	require.NoError(t, err)
	assert.Nil(t, c)

	_, err = search.DecodeCursor("nocolon")
	assert.Error(t, err)

	_, err = search.DecodeCursor("notatime:" + id.String())
	assert.Error(t, err)

	_, err = search.DecodeCursor(tm.Format(time.RFC3339Nano) + ":not-a-uuid")
	assert.Error(t, err)
}

package post_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/outbox"
	"github.com/Ans1110/trip-app/internal/post"
	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	withTx func(context.Context, func(post.IRepository) error) error

	createPost     func(context.Context, *post.Post) error
	updatePost     func(context.Context, *post.Post) error
	softDeletePost func(context.Context, uuid.UUID) error
	findPost       func(context.Context, uuid.UUID) (*post.Post, error)
	findPostsByIDs func(context.Context, []uuid.UUID) (map[uuid.UUID]post.Post, error)
	listPosts      func(context.Context, post.ListPostsFilter, *post.PostCursor, int) ([]post.Post, error)

	insertLike    func(context.Context, uuid.UUID, uuid.UUID) error
	deleteLike    func(context.Context, uuid.UUID, uuid.UUID) error
	areLiked      func(context.Context, uuid.UUID, []uuid.UUID) (map[uuid.UUID]bool, error)
	countLikes    func(context.Context, uuid.UUID) (int, error)
	bumpLikeCount func(context.Context, uuid.UUID, int) error

	insertBookmark func(context.Context, uuid.UUID, uuid.UUID) error
	deleteBookmark func(context.Context, uuid.UUID, uuid.UUID) error
	areBookmarked  func(context.Context, uuid.UUID, []uuid.UUID) (map[uuid.UUID]bool, error)
	listBookmarks  func(context.Context, uuid.UUID, *post.BookmarkCursor, int) ([]post.Bookmark, error)

	createComment        func(context.Context, *post.Comment) error
	updateCommentContent func(context.Context, uuid.UUID, uuid.UUID, string) (*post.Comment, error)
	softDeleteComment    func(context.Context, uuid.UUID) error
	findComment          func(context.Context, uuid.UUID) (*post.Comment, error)
	findCommentAny       func(context.Context, uuid.UUID) (*post.Comment, error)
	listComments         func(context.Context, uuid.UUID, *post.CommentCursor, int) ([]post.Comment, error)
	listReplies          func(context.Context, uuid.UUID, *post.ReplyCursor, int) ([]post.Comment, error)
	bumpCommentCount     func(context.Context, uuid.UUID, int) error
	bumpReplyCount       func(context.Context, uuid.UUID, int) error

	findUsersByIDs func(context.Context, []uuid.UUID) (map[uuid.UUID]auth.User, error)
	insertOutbox   func(context.Context, *outbox.Outbox) error
}

func (r *repoMock) WithTx(c context.Context, fn func(post.IRepository) error) error {
	if r.withTx != nil {
		return r.withTx(c, fn)
	}
	return fn(r)
}
func (r *repoMock) CreatePost(c context.Context, p *post.Post) error {
	if r.createPost != nil {
		return r.createPost(c, p)
	}
	return nil
}
func (r *repoMock) UpdatePost(c context.Context, p *post.Post) error {
	if r.updatePost != nil {
		return r.updatePost(c, p)
	}
	return nil
}
func (r *repoMock) SoftDeletePost(c context.Context, id uuid.UUID) error {
	if r.softDeletePost != nil {
		return r.softDeletePost(c, id)
	}
	return nil
}
func (r *repoMock) FindPost(c context.Context, id uuid.UUID) (*post.Post, error) {
	if r.findPost != nil {
		return r.findPost(c, id)
	}
	return nil, nil
}
func (r *repoMock) FindPostsByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]post.Post, error) {
	if r.findPostsByIDs != nil {
		return r.findPostsByIDs(c, ids)
	}
	return map[uuid.UUID]post.Post{}, nil
}
func (r *repoMock) ListPosts(c context.Context, f post.ListPostsFilter, cur *post.PostCursor, limit int) ([]post.Post, error) {
	if r.listPosts != nil {
		return r.listPosts(c, f, cur, limit)
	}
	return nil, nil
}
func (r *repoMock) InsertLike(c context.Context, p, u uuid.UUID) error {
	if r.insertLike != nil {
		return r.insertLike(c, p, u)
	}
	return nil
}
func (r *repoMock) DeleteLike(c context.Context, p, u uuid.UUID) error {
	if r.deleteLike != nil {
		return r.deleteLike(c, p, u)
	}
	return nil
}
func (r *repoMock) AreLiked(c context.Context, u uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	if r.areLiked != nil {
		return r.areLiked(c, u, ids)
	}
	return map[uuid.UUID]bool{}, nil
}
func (r *repoMock) CountLikes(c context.Context, id uuid.UUID) (int, error) {
	if r.countLikes != nil {
		return r.countLikes(c, id)
	}
	return 0, nil
}
func (r *repoMock) BumpLikeCount(c context.Context, id uuid.UUID, d int) error {
	if r.bumpLikeCount != nil {
		return r.bumpLikeCount(c, id, d)
	}
	return nil
}
func (r *repoMock) InsertBookmark(c context.Context, u, p uuid.UUID) error {
	if r.insertBookmark != nil {
		return r.insertBookmark(c, u, p)
	}
	return nil
}
func (r *repoMock) DeleteBookmark(c context.Context, u, p uuid.UUID) error {
	if r.deleteBookmark != nil {
		return r.deleteBookmark(c, u, p)
	}
	return nil
}
func (r *repoMock) AreBookmarked(c context.Context, u uuid.UUID, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	if r.areBookmarked != nil {
		return r.areBookmarked(c, u, ids)
	}
	return map[uuid.UUID]bool{}, nil
}
func (r *repoMock) ListBookmarks(c context.Context, u uuid.UUID, cur *post.BookmarkCursor, limit int) ([]post.Bookmark, error) {
	if r.listBookmarks != nil {
		return r.listBookmarks(c, u, cur, limit)
	}
	return nil, nil
}
func (r *repoMock) CreateComment(c context.Context, cm *post.Comment) error {
	if r.createComment != nil {
		return r.createComment(c, cm)
	}
	return nil
}
func (r *repoMock) UpdateCommentContent(c context.Context, id, author uuid.UUID, content string) (*post.Comment, error) {
	if r.updateCommentContent != nil {
		return r.updateCommentContent(c, id, author, content)
	}
	return nil, nil
}
func (r *repoMock) SoftDeleteComment(c context.Context, id uuid.UUID) error {
	if r.softDeleteComment != nil {
		return r.softDeleteComment(c, id)
	}
	return nil
}
func (r *repoMock) FindComment(c context.Context, id uuid.UUID) (*post.Comment, error) {
	if r.findComment != nil {
		return r.findComment(c, id)
	}
	return nil, nil
}
func (r *repoMock) FindCommentAny(c context.Context, id uuid.UUID) (*post.Comment, error) {
	if r.findCommentAny != nil {
		return r.findCommentAny(c, id)
	}
	return nil, nil
}
func (r *repoMock) ListComments(c context.Context, id uuid.UUID, cur *post.CommentCursor, limit int) ([]post.Comment, error) {
	if r.listComments != nil {
		return r.listComments(c, id, cur, limit)
	}
	return nil, nil
}
func (r *repoMock) ListReplies(c context.Context, id uuid.UUID, cur *post.ReplyCursor, limit int) ([]post.Comment, error) {
	if r.listReplies != nil {
		return r.listReplies(c, id, cur, limit)
	}
	return nil, nil
}
func (r *repoMock) BumpCommentCount(c context.Context, id uuid.UUID, d int) error {
	if r.bumpCommentCount != nil {
		return r.bumpCommentCount(c, id, d)
	}
	return nil
}
func (r *repoMock) BumpReplyCount(c context.Context, id uuid.UUID, d int) error {
	if r.bumpReplyCount != nil {
		return r.bumpReplyCount(c, id, d)
	}
	return nil
}
func (r *repoMock) FindUsersByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	if r.findUsersByIDs != nil {
		return r.findUsersByIDs(c, ids)
	}
	out := make(map[uuid.UUID]auth.User, len(ids))
	for _, id := range ids {
		out[id] = auth.User{ID: id, Name: "user-" + id.String()[:4]}
	}
	return out, nil
}
func (r *repoMock) InsertOutbox(c context.Context, row *outbox.Outbox) error {
	if r.insertOutbox != nil {
		return r.insertOutbox(c, row)
	}
	return nil
}

// ---- profilesMock ----

type profilesMock struct {
	find func(context.Context, uuid.UUID) (string, string, error)
}

func (p *profilesMock) FindProfileByUserID(c context.Context, id uuid.UUID) (string, string, error) {
	if p.find != nil {
		return p.find(c, id)
	}
	return "", "", nil
}

// ---- Factory ----

func newSvc(repo *repoMock, profiles post.ProfileLookup, rdb *redis.Client) post.IService {
	return post.NewService(post.ServiceConfig{Repo: repo, Profiles: profiles, Redis: rdb})
}

func newRedis(t *testing.T) *redis.Client {
	t.Helper()
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = client.Close() })
	return client
}

// ---- CreatePost ----

func TestCreatePost_DefaultsToPublishedAndWritesOutbox(t *testing.T) {
	author := uuid.New()
	repo := &repoMock{}
	var createdStatus string
	var outboxRow *outbox.Outbox
	repo.createPost = func(_ context.Context, p *post.Post) error {
		createdStatus = p.Status
		return nil
	}
	repo.insertOutbox = func(_ context.Context, row *outbox.Outbox) error {
		outboxRow = row
		return nil
	}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.CreatePost(ctx, author, post.CreatePostPayload{
		Title:   " hi ",
		Content: "body",
		Tags:    []string{"Go", "go", " ", "rust"},
	})
	require.NoError(t, err)
	assert.Equal(t, post.StatusPublished, createdStatus)
	assert.Equal(t, "hi", resp.Title)
	assert.Equal(t, []string{"Go", "rust"}, resp.Tags)
	require.NotNil(t, outboxRow)
	assert.Equal(t, post.OpPostPublished, outboxRow.OpType)
	assert.Equal(t, post.SubjectPost, outboxRow.AggregateType)
}

func TestCreatePost_DraftSkipsOutbox(t *testing.T) {
	draft := post.StatusDraft
	repo := &repoMock{}
	outboxCalled := false
	repo.insertOutbox = func(context.Context, *outbox.Outbox) error {
		outboxCalled = true
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreatePost(ctx, uuid.New(), post.CreatePostPayload{
		Title:   "x",
		Content: "y",
		Status:  &draft,
	})
	require.NoError(t, err)
	assert.False(t, outboxCalled)
}

func TestCreatePost_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db")
	repo := &repoMock{createPost: func(context.Context, *post.Post) error { return boom }}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreatePost(ctx, uuid.New(), post.CreatePostPayload{Title: "t", Content: "c"})
	assert.ErrorIs(t, err, boom)
}

// ---- UpdatePost ----

func TestUpdatePost_ForbiddenWhenNotAuthor(t *testing.T) {
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: uuid.New(), Status: post.StatusPublished}, nil
	}}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdatePost(ctx, uuid.New(), uuid.New(), post.UpdatePostPayload{})
	assert.ErrorIs(t, err, post.ErrForbidden)
}

func TestUpdatePost_NotFound(t *testing.T) {
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdatePost(ctx, uuid.New(), uuid.New(), post.UpdatePostPayload{})
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestUpdatePost_DraftToPublishedEmitsPublishedEvent(t *testing.T) {
	author := uuid.New()
	id := uuid.New()
	pub := post.StatusPublished
	existing := &post.Post{ID: id, AuthorID: author, Status: post.StatusDraft}
	repo := &repoMock{}
	firstCall := true
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) {
		if firstCall {
			firstCall = false
			return existing, nil
		}
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusPublished}, nil
	}
	var op string
	repo.insertOutbox = func(_ context.Context, row *outbox.Outbox) error {
		op = row.OpType
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdatePost(ctx, author, id, post.UpdatePostPayload{Status: &pub})
	require.NoError(t, err)
	assert.Equal(t, post.OpPostPublished, op)
}

func TestUpdatePost_PublishedToArchivedEmitsDeletedEvent(t *testing.T) {
	author := uuid.New()
	id := uuid.New()
	arch := post.StatusArchived
	existing := &post.Post{ID: id, AuthorID: author, Status: post.StatusPublished}
	repo := &repoMock{}
	first := true
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) {
		if first {
			first = false
			return existing, nil
		}
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusArchived}, nil
	}
	var op string
	repo.insertOutbox = func(_ context.Context, row *outbox.Outbox) error {
		op = row.OpType
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdatePost(ctx, author, id, post.UpdatePostPayload{Status: &arch})
	require.NoError(t, err)
	assert.Equal(t, post.OpPostDeleted, op)
}

func TestUpdatePost_PublishedEditEmitsUpdatedEvent(t *testing.T) {
	author := uuid.New()
	id := uuid.New()
	title := "new"
	existing := &post.Post{ID: id, AuthorID: author, Status: post.StatusPublished}
	repo := &repoMock{}
	first := true
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) {
		if first {
			first = false
			return existing, nil
		}
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusPublished, Title: "new"}, nil
	}
	var op string
	repo.insertOutbox = func(_ context.Context, row *outbox.Outbox) error {
		op = row.OpType
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdatePost(ctx, author, id, post.UpdatePostPayload{Title: &title})
	require.NoError(t, err)
	assert.Equal(t, post.OpPostUpdated, op)
}

func TestUpdatePost_DraftToDraftSkipsOutbox(t *testing.T) {
	author := uuid.New()
	id := uuid.New()
	title := "new"
	existing := &post.Post{ID: id, AuthorID: author, Status: post.StatusDraft}
	repo := &repoMock{}
	first := true
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) {
		if first {
			first = false
			return existing, nil
		}
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusDraft, Title: "new"}, nil
	}
	called := false
	repo.insertOutbox = func(context.Context, *outbox.Outbox) error {
		called = true
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdatePost(ctx, author, id, post.UpdatePostPayload{Title: &title})
	require.NoError(t, err)
	assert.False(t, called)
}

// ---- DeletePost ----

func TestDeletePost_ForbiddenWhenNotAuthor(t *testing.T) {
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: uuid.New()}, nil
	}}
	svc := newSvc(repo, nil, nil)
	err := svc.DeletePost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrForbidden)
}

func TestDeletePost_NotFound(t *testing.T) {
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	err := svc.DeletePost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestDeletePost_HappyPath_EmitsDeletedEvent(t *testing.T) {
	author := uuid.New()
	postID := uuid.New()
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: postID, AuthorID: author, Status: post.StatusPublished, PublishedAt: time.Now()}, nil
	}}
	deleted := false
	repo.softDeletePost = func(_ context.Context, id uuid.UUID) error {
		deleted = true
		assert.Equal(t, postID, id)
		return nil
	}
	var op string
	repo.insertOutbox = func(_ context.Context, row *outbox.Outbox) error {
		op = row.OpType
		return nil
	}
	svc := newSvc(repo, nil, nil)
	require.NoError(t, svc.DeletePost(ctx, author, postID))
	assert.True(t, deleted)
	assert.Equal(t, post.OpPostDeleted, op)
}

// ---- GetPost ----

func TestGetPost_HidesDraftFromNonAuthor(t *testing.T) {
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: uuid.New(), Status: post.StatusDraft}, nil
	}}
	svc := newSvc(repo, nil, nil)
	_, err := svc.GetPost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestGetPost_AuthorSeesOwnDraft(t *testing.T) {
	author := uuid.New()
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusDraft}, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.GetPost(ctx, author, uuid.New())
	require.NoError(t, err)
	assert.True(t, resp.IsAuthor)
	assert.Equal(t, post.StatusDraft, resp.Status)
}

// ---- GetPostsByIDs ----

func TestGetPostsByIDs_EmptyIDs(t *testing.T) {
	repo := &repoMock{}
	svc := newSvc(repo, nil, nil)
	out, err := svc.GetPostsByIDs(ctx, uuid.New(), nil)
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, out)
}

func TestGetPostsByIDs_PreservesOrderAndSkipsHiddenDrafts(t *testing.T) {
	viewer := uuid.New()
	author := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()
	id3 := uuid.New()
	repo := &repoMock{findPostsByIDs: func(context.Context, []uuid.UUID) (map[uuid.UUID]post.Post, error) {
		return map[uuid.UUID]post.Post{
			id1: {ID: id1, AuthorID: author, Status: post.StatusPublished},
			id2: {ID: id2, AuthorID: uuid.New(), Status: post.StatusDraft}, // hidden
			id3: {ID: id3, AuthorID: author, Status: post.StatusPublished},
		}, nil
	}}
	svc := newSvc(repo, nil, nil)
	out, err := svc.GetPostsByIDs(ctx, viewer, []uuid.UUID{id1, id2, id3})
	require.NoError(t, err)
	require.Len(t, out, 2)
	assert.Equal(t, id1.String(), out[0].ID)
	assert.Equal(t, id3.String(), out[1].ID)
}

func TestGetPostsByIDs_AllHidden(t *testing.T) {
	repo := &repoMock{findPostsByIDs: func(context.Context, []uuid.UUID) (map[uuid.UUID]post.Post, error) {
		return map[uuid.UUID]post.Post{uuid.New(): {AuthorID: uuid.New(), Status: post.StatusDraft}}, nil
	}}
	svc := newSvc(repo, nil, nil)
	out, err := svc.GetPostsByIDs(ctx, uuid.New(), []uuid.UUID{uuid.New()})
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, out)
}

// ---- ListPosts ----

func TestListPosts_EmptyRows(t *testing.T) {
	repo := &repoMock{listPosts: func(context.Context, post.ListPostsFilter, *post.PostCursor, int) ([]post.Post, error) {
		return []post.Post{}, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListPosts(ctx, uuid.New(), uuid.New(), "", nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, resp.Posts)
	assert.Empty(t, resp.NextCursor)
}

func TestListPosts_PublishedFilterByDefault(t *testing.T) {
	repo := &repoMock{}
	var gotFilter post.ListPostsFilter
	repo.listPosts = func(_ context.Context, f post.ListPostsFilter, _ *post.PostCursor, _ int) ([]post.Post, error) {
		gotFilter = f
		return []post.Post{}, nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.ListPosts(ctx, uuid.New(), uuid.New(), "", nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []string{post.StatusPublished}, gotFilter.Statuses)
}

func TestListPosts_AuthorViewingSelfCanFilterDrafts(t *testing.T) {
	author := uuid.New()
	repo := &repoMock{}
	var gotFilter post.ListPostsFilter
	repo.listPosts = func(_ context.Context, f post.ListPostsFilter, _ *post.PostCursor, _ int) ([]post.Post, error) {
		gotFilter = f
		return []post.Post{}, nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.ListPosts(ctx, author, author, post.StatusDraft, nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []string{post.StatusDraft}, gotFilter.Statuses)
}

func TestListPosts_NonAuthorCannotFilterDrafts(t *testing.T) {
	repo := &repoMock{}
	var gotFilter post.ListPostsFilter
	repo.listPosts = func(_ context.Context, f post.ListPostsFilter, _ *post.PostCursor, _ int) ([]post.Post, error) {
		gotFilter = f
		return []post.Post{}, nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.ListPosts(ctx, uuid.New(), uuid.New(), post.StatusDraft, nil, 20)
	require.NoError(t, err)
	assert.Equal(t, []string{post.StatusPublished}, gotFilter.Statuses)
}

func TestListPosts_HydratesAndEncodesCursor(t *testing.T) {
	viewer := uuid.New()
	author := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(-time.Hour)
	repo := &repoMock{listPosts: func(context.Context, post.ListPostsFilter, *post.PostCursor, int) ([]post.Post, error) {
		return []post.Post{
			{ID: id1, AuthorID: author, Status: post.StatusPublished, PublishedAt: t1},
			{ID: id2, AuthorID: author, Status: post.StatusPublished, PublishedAt: t2},
		}, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListPosts(ctx, viewer, author, "", nil, 20)
	require.NoError(t, err)
	require.Len(t, resp.Posts, 2)
	assert.NotEmpty(t, resp.NextCursor)
	assert.True(t, strings.HasSuffix(resp.NextCursor, ":"+id2.String()))
}

// ---- LikePost / UnlikePost ----

func TestLikePost_PostNotFound(t *testing.T) {
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	err := svc.LikePost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestLikePost_HappyPath_BumpsCounter(t *testing.T) {
	postID := uuid.New()
	rdb := newRedis(t)
	// Seed cache so incr's EXISTS guard bumps it.
	rdb.Set(ctx, "post:"+postID.String()+":likes", "5", 0)

	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: postID, AuthorID: uuid.New(), Status: post.StatusPublished}, nil
	}}
	inserted, bumped := false, false
	repo.insertLike = func(context.Context, uuid.UUID, uuid.UUID) error { inserted = true; return nil }
	repo.bumpLikeCount = func(_ context.Context, _ uuid.UUID, d int) error {
		bumped = true
		assert.Equal(t, +1, d)
		return nil
	}
	svc := newSvc(repo, nil, rdb)
	require.NoError(t, svc.LikePost(ctx, uuid.New(), postID))
	assert.True(t, inserted)
	assert.True(t, bumped)
	val, _ := rdb.Get(ctx, "post:"+postID.String()+":likes").Result()
	assert.Equal(t, "6", val)
}

func TestUnlikePost_DecrementsAndSkipsCacheWhenAbsent(t *testing.T) {
	postID := uuid.New()
	rdb := newRedis(t)
	repo := &repoMock{}
	var deltaSeen int
	repo.deleteLike = func(context.Context, uuid.UUID, uuid.UUID) error { return nil }
	repo.bumpLikeCount = func(_ context.Context, _ uuid.UUID, d int) error {
		deltaSeen = d
		return nil
	}
	svc := newSvc(repo, nil, rdb)
	require.NoError(t, svc.UnlikePost(ctx, uuid.New(), postID))
	assert.Equal(t, -1, deltaSeen)
	// No cache entry => guarded decr is a no-op.
	_, err := rdb.Get(ctx, "post:"+postID.String()+":likes").Result()
	assert.ErrorIs(t, err, redis.Nil)
}

func TestLikePost_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("insert failed")
	repo := &repoMock{
		findPost:   func(context.Context, uuid.UUID) (*post.Post, error) { return &post.Post{Status: post.StatusPublished}, nil },
		insertLike: func(context.Context, uuid.UUID, uuid.UUID) error { return boom },
	}
	svc := newSvc(repo, nil, nil)
	err := svc.LikePost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, boom)
}

// ---- Bookmark ----

func TestBookmarkPost_HidesDraftFromNonAuthor(t *testing.T) {
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) {
		return &post.Post{AuthorID: uuid.New(), Status: post.StatusDraft}, nil
	}}
	svc := newSvc(repo, nil, nil)
	err := svc.BookmarkPost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestBookmarkPost_NotFound(t *testing.T) {
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	err := svc.BookmarkPost(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestBookmarkPost_HappyPath(t *testing.T) {
	user := uuid.New()
	pid := uuid.New()
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: pid, AuthorID: uuid.New(), Status: post.StatusPublished}, nil
	}}
	inserted := false
	repo.insertBookmark = func(_ context.Context, u, p uuid.UUID) error {
		inserted = true
		assert.Equal(t, user, u)
		assert.Equal(t, pid, p)
		return nil
	}
	svc := newSvc(repo, nil, nil)
	require.NoError(t, svc.BookmarkPost(ctx, user, pid))
	assert.True(t, inserted)
}

func TestUnbookmarkPost_Delegates(t *testing.T) {
	called := false
	repo := &repoMock{deleteBookmark: func(context.Context, uuid.UUID, uuid.UUID) error {
		called = true
		return nil
	}}
	svc := newSvc(repo, nil, nil)
	require.NoError(t, svc.UnbookmarkPost(ctx, uuid.New(), uuid.New()))
	assert.True(t, called)
}

func TestListBookmarks_EmptyAndHydrated(t *testing.T) {
	viewer := uuid.New()
	author := uuid.New()

	// Empty rows.
	repo := &repoMock{listBookmarks: func(context.Context, uuid.UUID, *post.BookmarkCursor, int) ([]post.Bookmark, error) {
		return nil, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListBookmarks(ctx, viewer, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []post.PostResponse{}, resp.Posts)

	// Non-empty with cursor.
	postID := uuid.New()
	createdAt := time.Now().UTC().Truncate(time.Second)
	repo2 := &repoMock{
		listBookmarks: func(context.Context, uuid.UUID, *post.BookmarkCursor, int) ([]post.Bookmark, error) {
			return []post.Bookmark{{UserID: viewer, PostID: postID, CreatedAt: createdAt}}, nil
		},
		findPostsByIDs: func(context.Context, []uuid.UUID) (map[uuid.UUID]post.Post, error) {
			return map[uuid.UUID]post.Post{postID: {ID: postID, AuthorID: author, Status: post.StatusPublished}}, nil
		},
	}
	svc2 := newSvc(repo2, nil, nil)
	resp2, err := svc2.ListBookmarks(ctx, viewer, nil, 10)
	require.NoError(t, err)
	require.Len(t, resp2.Posts, 1)
	assert.True(t, strings.HasSuffix(resp2.NextCursor, ":"+postID.String()))
}

// ---- CreateComment ----

func TestCreateComment_PostNotFound(t *testing.T) {
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreateComment(ctx, uuid.New(), uuid.New(), post.CreateCommentPayload{Content: "hi"})
	assert.ErrorIs(t, err, post.ErrPostNotFound)
}

func TestCreateComment_InvalidParentIDFormat(t *testing.T) {
	postID := uuid.New()
	repo := &repoMock{findPost: func(context.Context, uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: postID}, nil
	}}
	svc := newSvc(repo, nil, nil)
	bad := "not-a-uuid"
	_, err := svc.CreateComment(ctx, uuid.New(), postID, post.CreateCommentPayload{Content: "hi", ParentID: &bad})
	assert.ErrorIs(t, err, post.ErrInvalidParent)
}

func TestCreateComment_ParentNotFound(t *testing.T) {
	postID := uuid.New()
	parentStr := uuid.NewString()
	repo := &repoMock{
		findPost:    func(context.Context, uuid.UUID) (*post.Post, error) { return &post.Post{ID: postID}, nil },
		findComment: func(context.Context, uuid.UUID) (*post.Comment, error) { return nil, nil },
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreateComment(ctx, uuid.New(), postID, post.CreateCommentPayload{Content: "hi", ParentID: &parentStr})
	assert.ErrorIs(t, err, post.ErrParentNotFound)
}

func TestCreateComment_ParentOnDifferentPost(t *testing.T) {
	postID := uuid.New()
	parentID := uuid.New()
	parentStr := parentID.String()
	repo := &repoMock{
		findPost: func(context.Context, uuid.UUID) (*post.Post, error) { return &post.Post{ID: postID}, nil },
		findComment: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
			return &post.Comment{ID: id, PostID: uuid.New()}, nil
		},
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreateComment(ctx, uuid.New(), postID, post.CreateCommentPayload{Content: "hi", ParentID: &parentStr})
	assert.ErrorIs(t, err, post.ErrParentNotFound)
}

func TestCreateComment_ReplyToRootStoresRootAsParent(t *testing.T) {
	postID := uuid.New()
	rootID := uuid.New()
	rootStr := rootID.String()
	repo := &repoMock{}
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) { return &post.Post{ID: postID}, nil }
	repo.findComment = func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
		return &post.Comment{ID: id, PostID: postID}, nil
	}
	var created *post.Comment
	repo.createComment = func(_ context.Context, c *post.Comment) error {
		c.ID = uuid.New()
		created = c
		return nil
	}
	var parentBumped bool
	repo.bumpReplyCount = func(_ context.Context, id uuid.UUID, d int) error {
		parentBumped = true
		assert.Equal(t, rootID, id)
		assert.Equal(t, +1, d)
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreateComment(ctx, uuid.New(), postID, post.CreateCommentPayload{Content: "hi", ParentID: &rootStr})
	require.NoError(t, err)
	require.NotNil(t, created.ParentID)
	assert.Equal(t, rootID, *created.ParentID)
	require.NotNil(t, created.InReplyToID)
	assert.Equal(t, rootID, *created.InReplyToID)
	assert.True(t, parentBumped)
}

func TestCreateComment_ReplyToDepth2FlattensToRoot(t *testing.T) {
	postID := uuid.New()
	rootID := uuid.New()
	depth2ID := uuid.New()
	depth2Str := depth2ID.String()
	repo := &repoMock{}
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) { return &post.Post{ID: postID}, nil }
	repo.findComment = func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
		if id == depth2ID {
			return &post.Comment{ID: id, PostID: postID, ParentID: &rootID}, nil
		}
		return &post.Comment{ID: id, PostID: postID}, nil
	}
	var created *post.Comment
	repo.createComment = func(_ context.Context, c *post.Comment) error {
		created = c
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreateComment(ctx, uuid.New(), postID, post.CreateCommentPayload{Content: "hi", ParentID: &depth2Str})
	require.NoError(t, err)
	require.NotNil(t, created.ParentID)
	assert.Equal(t, rootID, *created.ParentID, "reply nests under root, not intermediate")
	require.NotNil(t, created.InReplyToID)
	assert.Equal(t, depth2ID, *created.InReplyToID)
}

func TestCreateComment_TopLevelBumpsCommentCountOnly(t *testing.T) {
	postID := uuid.New()
	repo := &repoMock{}
	repo.findPost = func(context.Context, uuid.UUID) (*post.Post, error) { return &post.Post{ID: postID}, nil }
	commentBumped := false
	replyBumped := false
	repo.bumpCommentCount = func(_ context.Context, id uuid.UUID, d int) error {
		commentBumped = true
		assert.Equal(t, postID, id)
		assert.Equal(t, +1, d)
		return nil
	}
	repo.bumpReplyCount = func(context.Context, uuid.UUID, int) error {
		replyBumped = true
		return nil
	}
	repo.createComment = func(_ context.Context, c *post.Comment) error {
		c.ID = uuid.New()
		return nil
	}
	svc := newSvc(repo, nil, nil)
	_, err := svc.CreateComment(ctx, uuid.New(), postID, post.CreateCommentPayload{Content: " hi "})
	require.NoError(t, err)
	assert.True(t, commentBumped)
	assert.False(t, replyBumped)
}

// ---- UpdateComment ----

func TestUpdateComment_NotFound(t *testing.T) {
	repo := &repoMock{updateCommentContent: func(context.Context, uuid.UUID, uuid.UUID, string) (*post.Comment, error) {
		return nil, nil
	}}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdateComment(ctx, uuid.New(), uuid.New(), post.UpdateCommentPayload{Content: "x"})
	assert.ErrorIs(t, err, post.ErrCommentNotFound)
}

func TestUpdateComment_ForbiddenPassthrough(t *testing.T) {
	repo := &repoMock{updateCommentContent: func(context.Context, uuid.UUID, uuid.UUID, string) (*post.Comment, error) {
		return nil, post.ErrForbidden
	}}
	svc := newSvc(repo, nil, nil)
	_, err := svc.UpdateComment(ctx, uuid.New(), uuid.New(), post.UpdateCommentPayload{Content: "x"})
	assert.ErrorIs(t, err, post.ErrForbidden)
}

func TestUpdateComment_HappyPathTrims(t *testing.T) {
	author := uuid.New()
	repo := &repoMock{}
	var gotContent string
	repo.updateCommentContent = func(_ context.Context, _, _ uuid.UUID, content string) (*post.Comment, error) {
		gotContent = content
		return &post.Comment{ID: uuid.New(), AuthorID: author, Content: content}, nil
	}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.UpdateComment(ctx, author, uuid.New(), post.UpdateCommentPayload{Content: "  hello  "})
	require.NoError(t, err)
	assert.Equal(t, "hello", gotContent)
	assert.True(t, resp.IsAuthor)
}

// ---- DeleteComment ----

func TestDeleteComment_NotFound(t *testing.T) {
	repo := &repoMock{findComment: func(context.Context, uuid.UUID) (*post.Comment, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	err := svc.DeleteComment(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrCommentNotFound)
}

func TestDeleteComment_AuthorCanDelete(t *testing.T) {
	author := uuid.New()
	postID := uuid.New()
	commentID := uuid.New()
	repo := &repoMock{findComment: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
		return &post.Comment{ID: id, PostID: postID, AuthorID: author}, nil
	}}
	deleted := false
	repo.softDeleteComment = func(context.Context, uuid.UUID) error { deleted = true; return nil }
	svc := newSvc(repo, nil, nil)
	require.NoError(t, svc.DeleteComment(ctx, author, commentID))
	assert.True(t, deleted)
}

func TestDeleteComment_PostAuthorCanModerate(t *testing.T) {
	commenter := uuid.New()
	moderator := uuid.New()
	postID := uuid.New()
	repo := &repoMock{
		findComment: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
			return &post.Comment{ID: id, PostID: postID, AuthorID: commenter}, nil
		},
		findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
			return &post.Post{ID: id, AuthorID: moderator}, nil
		},
	}
	deleted := false
	repo.softDeleteComment = func(context.Context, uuid.UUID) error { deleted = true; return nil }
	svc := newSvc(repo, nil, nil)
	require.NoError(t, svc.DeleteComment(ctx, moderator, uuid.New()))
	assert.True(t, deleted)
}

func TestDeleteComment_StrangerForbidden(t *testing.T) {
	repo := &repoMock{
		findComment: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
			return &post.Comment{ID: id, PostID: uuid.New(), AuthorID: uuid.New()}, nil
		},
		findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
			return &post.Post{ID: id, AuthorID: uuid.New()}, nil
		},
	}
	svc := newSvc(repo, nil, nil)
	err := svc.DeleteComment(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, post.ErrForbidden)
}

func TestDeleteComment_ReplyBumpsParentReplyCount(t *testing.T) {
	parentID := uuid.New()
	postID := uuid.New()
	author := uuid.New()
	repo := &repoMock{}
	repo.findComment = func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
		return &post.Comment{ID: id, PostID: postID, AuthorID: author, ParentID: &parentID}, nil
	}
	replyBumped := false
	repo.bumpReplyCount = func(_ context.Context, id uuid.UUID, d int) error {
		replyBumped = true
		assert.Equal(t, parentID, id)
		assert.Equal(t, -1, d)
		return nil
	}
	svc := newSvc(repo, nil, nil)
	require.NoError(t, svc.DeleteComment(ctx, author, uuid.New()))
	assert.True(t, replyBumped)
}

// ---- ListComments ----

func TestListComments_Empty(t *testing.T) {
	repo := &repoMock{listComments: func(context.Context, uuid.UUID, *post.CommentCursor, int) ([]post.Comment, error) {
		return nil, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListComments(ctx, uuid.New(), uuid.New(), nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []post.CommentResponse{}, resp.Comments)
	assert.Empty(t, resp.NextCursor)
}

func TestListComments_HydratesAndEncodesCursor(t *testing.T) {
	viewer := uuid.New()
	author := uuid.New()
	id1 := uuid.New()
	id2 := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	t2 := t1.Add(-time.Hour)
	repo := &repoMock{listComments: func(context.Context, uuid.UUID, *post.CommentCursor, int) ([]post.Comment, error) {
		return []post.Comment{
			{ID: id1, AuthorID: author, Content: "a", CreatedAt: t1},
			{ID: id2, AuthorID: author, Content: "b", CreatedAt: t2},
		}, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListComments(ctx, viewer, uuid.New(), nil, 10)
	require.NoError(t, err)
	require.Len(t, resp.Comments, 2)
	assert.True(t, strings.HasSuffix(resp.NextCursor, ":"+id2.String()))
}

func TestListComments_DeletedContentMasked(t *testing.T) {
	commentID := uuid.New()
	deleted := time.Now()
	repo := &repoMock{listComments: func(context.Context, uuid.UUID, *post.CommentCursor, int) ([]post.Comment, error) {
		return []post.Comment{{ID: commentID, AuthorID: uuid.New(), Content: "was here", DeletedAt: &deleted, ReplyCount: 3}}, nil
	}}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListComments(ctx, uuid.New(), uuid.New(), nil, 10)
	require.NoError(t, err)
	require.Len(t, resp.Comments, 1)
	assert.Empty(t, resp.Comments[0].Content)
	assert.True(t, resp.Comments[0].IsDeleted)
	assert.False(t, resp.Comments[0].IsAuthor)
	assert.Equal(t, 3, resp.Comments[0].ReplyCount)
}

// ---- ListReplies ----

func TestListReplies_ParentNotFound(t *testing.T) {
	repo := &repoMock{findCommentAny: func(context.Context, uuid.UUID) (*post.Comment, error) { return nil, nil }}
	svc := newSvc(repo, nil, nil)
	_, err := svc.ListReplies(ctx, uuid.New(), uuid.New(), nil, 10)
	assert.ErrorIs(t, err, post.ErrCommentNotFound)
}

func TestListReplies_ParentMustBeRoot(t *testing.T) {
	otherRoot := uuid.New()
	repo := &repoMock{findCommentAny: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
		return &post.Comment{ID: id, ParentID: &otherRoot}, nil
	}}
	svc := newSvc(repo, nil, nil)
	_, err := svc.ListReplies(ctx, uuid.New(), uuid.New(), nil, 10)
	assert.ErrorIs(t, err, post.ErrCommentNotFound)
}

func TestListReplies_EmptyAndHydrated(t *testing.T) {
	rootID := uuid.New()
	repo := &repoMock{
		findCommentAny: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
			return &post.Comment{ID: id}, nil
		},
		listReplies: func(context.Context, uuid.UUID, *post.ReplyCursor, int) ([]post.Comment, error) { return nil, nil },
	}
	svc := newSvc(repo, nil, nil)
	resp, err := svc.ListReplies(ctx, uuid.New(), rootID, nil, 10)
	require.NoError(t, err)
	assert.Equal(t, []post.CommentResponse{}, resp.Replies)

	// With rows.
	id1 := uuid.New()
	t1 := time.Now().UTC().Truncate(time.Second)
	repo2 := &repoMock{
		findCommentAny: func(_ context.Context, id uuid.UUID) (*post.Comment, error) {
			return &post.Comment{ID: id}, nil
		},
		listReplies: func(context.Context, uuid.UUID, *post.ReplyCursor, int) ([]post.Comment, error) {
			return []post.Comment{{ID: id1, AuthorID: uuid.New(), Content: "r", CreatedAt: t1}}, nil
		},
	}
	svc2 := newSvc(repo2, nil, nil)
	resp2, err := svc2.ListReplies(ctx, uuid.New(), rootID, nil, 10)
	require.NoError(t, err)
	require.Len(t, resp2.Replies, 1)
	assert.True(t, strings.HasSuffix(resp2.NextCursor, ":"+id1.String()))
}

// ---- Profile lookup fallback ----

func TestBatchProfiles_UsedInResponse(t *testing.T) {
	author := uuid.New()
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusPublished}, nil
	}}
	profiles := &profilesMock{find: func(_ context.Context, id uuid.UUID) (string, string, error) {
		assert.Equal(t, author, id)
		return "alice", "https://cdn/alice.png", nil
	}}
	svc := newSvc(repo, profiles, nil)
	resp, err := svc.GetPost(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Equal(t, "alice", resp.Author.Username)
	assert.Equal(t, "https://cdn/alice.png", resp.Author.AvatarURL)
}

func TestBatchProfiles_LookupErrorDoesNotFailRequest(t *testing.T) {
	author := uuid.New()
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: author, Status: post.StatusPublished}, nil
	}}
	profiles := &profilesMock{find: func(context.Context, uuid.UUID) (string, string, error) {
		return "", "", errors.New("boom")
	}}
	svc := newSvc(repo, profiles, nil)
	resp, err := svc.GetPost(ctx, uuid.New(), uuid.New())
	require.NoError(t, err)
	assert.Empty(t, resp.Author.Username)
}

// ---- Like cache backfill ----

func TestLikeCache_MissBackfillsFromRow(t *testing.T) {
	viewer := uuid.New()
	postID := uuid.New()
	rdb := newRedis(t)
	repo := &repoMock{findPost: func(_ context.Context, id uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: id, AuthorID: uuid.New(), Status: post.StatusPublished, LikeCount: 42}, nil
	}}
	svc := newSvc(repo, nil, rdb)
	resp, err := svc.GetPost(ctx, viewer, postID)
	require.NoError(t, err)
	assert.Equal(t, 42, resp.LikeCount)
	// After a miss the value should be backfilled.
	val, _ := rdb.Get(ctx, "post:"+postID.String()+":likes").Result()
	assert.Equal(t, "42", val)
}

// ---- Cursor decoders ----

func TestDecodeCursors(t *testing.T) {
	id := uuid.New()
	tm := time.Now().UTC().Truncate(time.Nanosecond)
	tok := tm.Format(time.RFC3339Nano) + ":" + id.String()

	pc, err := post.DecodePostCursor(tok)
	require.NoError(t, err)
	require.NotNil(t, pc)
	assert.Equal(t, id, pc.ID)
	assert.True(t, pc.PublishedAt.Equal(tm))

	cc, err := post.DecodeCommentCursor(tok)
	require.NoError(t, err)
	require.NotNil(t, cc)
	assert.Equal(t, id, cc.ID)

	rc, err := post.DecodeReplyCursor(tok)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, id, rc.ID)

	bc, err := post.DecodeBookmarkCursor(tok)
	require.NoError(t, err)
	require.NotNil(t, bc)
	assert.Equal(t, id, bc.PostID)

	// Empty token -> nil, no error.
	pc, err = post.DecodePostCursor("")
	require.NoError(t, err)
	assert.Nil(t, pc)

	// Bad format.
	_, err = post.DecodePostCursor("nocolon")
	assert.Error(t, err)
	_, err = post.DecodePostCursor("bad:" + id.String())
	assert.Error(t, err)
	_, err = post.DecodePostCursor(tm.Format(time.RFC3339Nano) + ":bad-uuid")
	assert.Error(t, err)
}

// ---- IsValidStatus ----

func TestIsValidStatus(t *testing.T) {
	assert.True(t, post.IsValidStatus(post.StatusDraft))
	assert.True(t, post.IsValidStatus(post.StatusPublished))
	assert.True(t, post.IsValidStatus(post.StatusArchived))
	assert.False(t, post.IsValidStatus(""))
	assert.False(t, post.IsValidStatus("hidden"))
}

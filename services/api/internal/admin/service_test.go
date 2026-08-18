package admin_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/admin"
	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/post"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var ctx = context.Background()

// ---- repoMock ----

type repoMock struct {
	createReport          func(context.Context, *admin.Report) error
	findReport            func(context.Context, uuid.UUID) (*admin.Report, error)
	listReports           func(context.Context, admin.ReportFilter, *admin.ListReportsCursor, int) ([]admin.Report, error)
	resolveReport         func(context.Context, uuid.UUID, uuid.UUID, admin.ReportStatus, string) error
	countReportsBySubject func(context.Context, admin.SubjectType, []uuid.UUID) (map[uuid.UUID]int, error)

	findUsersByIDs     func(context.Context, []uuid.UUID) (map[uuid.UUID]auth.User, error)
	listUsers          func(context.Context, admin.UserFilter, *admin.ListUsersCursor, int) ([]auth.User, error)
	setUserBlocked     func(context.Context, uuid.UUID, bool) error
	deactivateUser     func(context.Context, uuid.UUID) error
	reactivateUser     func(context.Context, uuid.UUID) error
	listUserRoles      func(context.Context, []uuid.UUID) (map[uuid.UUID][]string, error)
	countPostsByAuthor func(context.Context, []uuid.UUID) (map[uuid.UUID]int, error)

	listPosts              func(context.Context, admin.PostFilter, *admin.ListPostsCursor, int) ([]post.Post, error)
	findPostAny            func(context.Context, uuid.UUID) (*post.Post, error)
	softDeletePost         func(context.Context, uuid.UUID) error
	restorePost            func(context.Context, uuid.UUID) error
	findCommentAny         func(context.Context, uuid.UUID) (*post.Comment, error)
	softDeleteCommentAdmin func(context.Context, uuid.UUID) error
}

func (r *repoMock) CreateReport(c context.Context, rp *admin.Report) error {
	if r.createReport != nil {
		return r.createReport(c, rp)
	}
	return nil
}
func (r *repoMock) FindReport(c context.Context, id uuid.UUID) (*admin.Report, error) {
	if r.findReport != nil {
		return r.findReport(c, id)
	}
	return nil, nil
}
func (r *repoMock) ListReports(c context.Context, f admin.ReportFilter, cur *admin.ListReportsCursor, l int) ([]admin.Report, error) {
	if r.listReports != nil {
		return r.listReports(c, f, cur, l)
	}
	return nil, nil
}
func (r *repoMock) ResolveReport(c context.Context, id, by uuid.UUID, s admin.ReportStatus, res string) error {
	if r.resolveReport != nil {
		return r.resolveReport(c, id, by, s, res)
	}
	return nil
}
func (r *repoMock) CountReportsBySubject(c context.Context, k admin.SubjectType, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	if r.countReportsBySubject != nil {
		return r.countReportsBySubject(c, k, ids)
	}
	return map[uuid.UUID]int{}, nil
}
func (r *repoMock) FindUsersByIDs(c context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	if r.findUsersByIDs != nil {
		return r.findUsersByIDs(c, ids)
	}
	return map[uuid.UUID]auth.User{}, nil
}
func (r *repoMock) ListUsers(c context.Context, f admin.UserFilter, cur *admin.ListUsersCursor, l int) ([]auth.User, error) {
	if r.listUsers != nil {
		return r.listUsers(c, f, cur, l)
	}
	return nil, nil
}
func (r *repoMock) SetUserBlocked(c context.Context, id uuid.UUID, b bool) error {
	if r.setUserBlocked != nil {
		return r.setUserBlocked(c, id, b)
	}
	return nil
}
func (r *repoMock) DeactivateUser(c context.Context, id uuid.UUID) error {
	if r.deactivateUser != nil {
		return r.deactivateUser(c, id)
	}
	return nil
}
func (r *repoMock) ReactivateUser(c context.Context, id uuid.UUID) error {
	if r.reactivateUser != nil {
		return r.reactivateUser(c, id)
	}
	return nil
}
func (r *repoMock) ListUserRoles(c context.Context, ids []uuid.UUID) (map[uuid.UUID][]string, error) {
	if r.listUserRoles != nil {
		return r.listUserRoles(c, ids)
	}
	return map[uuid.UUID][]string{}, nil
}
func (r *repoMock) CountPostsByAuthor(c context.Context, ids []uuid.UUID) (map[uuid.UUID]int, error) {
	if r.countPostsByAuthor != nil {
		return r.countPostsByAuthor(c, ids)
	}
	return map[uuid.UUID]int{}, nil
}
func (r *repoMock) ListPosts(c context.Context, f admin.PostFilter, cur *admin.ListPostsCursor, l int) ([]post.Post, error) {
	if r.listPosts != nil {
		return r.listPosts(c, f, cur, l)
	}
	return nil, nil
}
func (r *repoMock) FindPostAny(c context.Context, id uuid.UUID) (*post.Post, error) {
	if r.findPostAny != nil {
		return r.findPostAny(c, id)
	}
	return nil, nil
}
func (r *repoMock) SoftDeletePost(c context.Context, id uuid.UUID) error {
	if r.softDeletePost != nil {
		return r.softDeletePost(c, id)
	}
	return nil
}
func (r *repoMock) RestorePost(c context.Context, id uuid.UUID) error {
	if r.restorePost != nil {
		return r.restorePost(c, id)
	}
	return nil
}
func (r *repoMock) FindCommentAny(c context.Context, id uuid.UUID) (*post.Comment, error) {
	if r.findCommentAny != nil {
		return r.findCommentAny(c, id)
	}
	return nil, nil
}
func (r *repoMock) SoftDeleteCommentAdmin(c context.Context, id uuid.UUID) error {
	if r.softDeleteCommentAdmin != nil {
		return r.softDeleteCommentAdmin(c, id)
	}
	return nil
}

// ---- auditMock ----

type auditMock struct {
	logs []audit.Log
}

func (a *auditMock) Create(_ context.Context, l *audit.Log) error {
	a.logs = append(a.logs, *l)
	return nil
}

func newSvc(r *repoMock, aud audit.Writer) admin.IService {
	return admin.NewService(admin.ServiceConfig{Repo: r, Audit: aud})
}

// ---- SubmitReport ----

func TestSubmitReport_InvalidSubjectIDIsRejected(t *testing.T) {
	svc := newSvc(&repoMock{}, nil)
	_, err := svc.SubmitReport(ctx, uuid.New(), admin.CreateReportPayload{
		SubjectType: "post", SubjectID: "not-a-uuid", Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrReportNotFound)
}

func TestSubmitReport_UnknownSubjectTypeRejected(t *testing.T) {
	svc := newSvc(&repoMock{}, nil)
	_, err := svc.SubmitReport(ctx, uuid.New(), admin.CreateReportPayload{
		SubjectType: "trip", SubjectID: uuid.New().String(), Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrReportNotFound)
}

func TestSubmitReport_SelfReportRejected(t *testing.T) {
	reporter := uuid.New()
	postID := uuid.New()
	repo := &repoMock{findPostAny: func(_ context.Context, _ uuid.UUID) (*post.Post, error) {
		return &post.Post{ID: postID, AuthorID: reporter}, nil
	}}
	svc := newSvc(repo, nil)
	_, err := svc.SubmitReport(ctx, reporter, admin.CreateReportPayload{
		SubjectType: "post", SubjectID: postID.String(), Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrInvalidReporter)
}

func TestSubmitReport_MissingSubjectReturnsNotFound(t *testing.T) {
	repo := &repoMock{findPostAny: func(_ context.Context, _ uuid.UUID) (*post.Post, error) { return nil, nil }}
	svc := newSvc(repo, nil)
	_, err := svc.SubmitReport(ctx, uuid.New(), admin.CreateReportPayload{
		SubjectType: "post", SubjectID: uuid.New().String(), Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrPostNotFound)

	repo = &repoMock{findCommentAny: func(_ context.Context, _ uuid.UUID) (*post.Comment, error) { return nil, nil }}
	svc = newSvc(repo, nil)
	_, err = svc.SubmitReport(ctx, uuid.New(), admin.CreateReportPayload{
		SubjectType: "comment", SubjectID: uuid.New().String(), Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrCommentNotFound)

	repo = &repoMock{findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
		return map[uuid.UUID]auth.User{}, nil
	}}
	svc = newSvc(repo, nil)
	_, err = svc.SubmitReport(ctx, uuid.New(), admin.CreateReportPayload{
		SubjectType: "user", SubjectID: uuid.New().String(), Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrUserNotFound)
}

func TestSubmitReport_UserSubjectSelfRejected(t *testing.T) {
	id := uuid.New()
	svc := newSvc(&repoMock{}, nil)
	_, err := svc.SubmitReport(ctx, id, admin.CreateReportPayload{
		SubjectType: "user", SubjectID: id.String(), Reason: "spam",
	})
	assert.ErrorIs(t, err, admin.ErrInvalidReporter)
}

func TestSubmitReport_HappyPathHydrates(t *testing.T) {
	reporter := uuid.New()
	postAuthor := uuid.New()
	postID := uuid.New()
	repo := &repoMock{
		findPostAny: func(_ context.Context, _ uuid.UUID) (*post.Post, error) {
			return &post.Post{ID: postID, AuthorID: postAuthor, Title: "T", Content: "body"}, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "n", Email: "e"}
			}
			return out, nil
		},
	}
	aud := &auditMock{}
	svc := newSvc(repo, aud)
	resp, err := svc.SubmitReport(ctx, reporter, admin.CreateReportPayload{
		SubjectType: "post", SubjectID: postID.String(), Reason: "spam", Description: " abusive ",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, "post", resp.SubjectType)
	require.NotNil(t, resp.Subject)
	assert.Equal(t, "T", resp.Subject.Title)
	assert.Equal(t, "abusive", resp.Description)
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditReportSubmitted, aud.logs[0].Action)
}

// ---- ListReports ----

func TestListReports_EmptyReturnsEmpty(t *testing.T) {
	repo := &repoMock{listReports: func(_ context.Context, _ admin.ReportFilter, _ *admin.ListReportsCursor, _ int) ([]admin.Report, error) {
		return nil, nil
	}}
	svc := newSvc(repo, nil)
	resp, err := svc.ListReports(ctx, admin.ReportFilter{}, nil, 20)
	require.NoError(t, err)
	assert.Empty(t, resp.Reports)
	assert.Empty(t, resp.NextCursor)
}

func TestListReports_HydratesAndEncodesCursor(t *testing.T) {
	reporter := uuid.New()
	subjectUser := uuid.New()
	rpt := admin.Report{
		ID: uuid.New(), ReporterID: reporter, SubjectType: admin.SubjectUser, SubjectID: subjectUser,
		Status: admin.StatusPending, Reason: "spam", CreatedAt: time.Now().UTC().Truncate(time.Second),
	}
	repo := &repoMock{
		listReports: func(_ context.Context, _ admin.ReportFilter, _ *admin.ListReportsCursor, _ int) ([]admin.Report, error) {
			return []admin.Report{rpt}, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "n"}
			}
			return out, nil
		},
	}
	svc := newSvc(repo, nil)
	resp, err := svc.ListReports(ctx, admin.ReportFilter{}, nil, 20)
	require.NoError(t, err)
	require.Len(t, resp.Reports, 1)
	assert.NotEmpty(t, resp.NextCursor)
	assert.Contains(t, resp.NextCursor, rpt.ID.String())
}

// ---- Resolve / Dismiss ----

func TestResolveReport_HappyPath(t *testing.T) {
	actor := uuid.New()
	rid := uuid.New()
	var status admin.ReportStatus
	repo := &repoMock{
		resolveReport: func(_ context.Context, id, by uuid.UUID, s admin.ReportStatus, res string) error {
			assert.Equal(t, rid, id)
			assert.Equal(t, actor, by)
			status = s
			assert.Equal(t, "ok", res)
			return nil
		},
		findReport: func(_ context.Context, id uuid.UUID) (*admin.Report, error) {
			return &admin.Report{
				ID: id, ReporterID: uuid.New(), SubjectType: admin.SubjectPost,
				SubjectID: uuid.New(), Status: admin.StatusResolved, Resolution: "ok",
			}, nil
		},
	}
	aud := &auditMock{}
	svc := newSvc(repo, aud)
	resp, err := svc.ResolveReport(ctx, actor, rid, admin.ResolveReportPayload{Resolution: "  ok "})
	require.NoError(t, err)
	assert.Equal(t, admin.StatusResolved, status)
	assert.Equal(t, "resolved", resp.Status)
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditReportResolved, aud.logs[0].Action)
}

func TestDismissReport_UsesDismissedAction(t *testing.T) {
	actor := uuid.New()
	rid := uuid.New()
	repo := &repoMock{
		resolveReport: func(_ context.Context, _, _ uuid.UUID, s admin.ReportStatus, _ string) error {
			assert.Equal(t, admin.StatusDismissed, s)
			return nil
		},
		findReport: func(_ context.Context, id uuid.UUID) (*admin.Report, error) {
			return &admin.Report{ID: id, ReporterID: uuid.New(), SubjectType: admin.SubjectPost, SubjectID: uuid.New(), Status: admin.StatusDismissed}, nil
		},
	}
	aud := &auditMock{}
	svc := newSvc(repo, aud)
	_, err := svc.DismissReport(ctx, actor, rid, admin.ResolveReportPayload{})
	require.NoError(t, err)
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditReportDismissed, aud.logs[0].Action)
}

func TestResolveReport_MissingAfterUpdateReturnsNotFound(t *testing.T) {
	repo := &repoMock{
		resolveReport: func(_ context.Context, _, _ uuid.UUID, _ admin.ReportStatus, _ string) error { return nil },
		findReport:    func(_ context.Context, _ uuid.UUID) (*admin.Report, error) { return nil, nil },
	}
	svc := newSvc(repo, nil)
	_, err := svc.ResolveReport(ctx, uuid.New(), uuid.New(), admin.ResolveReportPayload{})
	assert.ErrorIs(t, err, admin.ErrReportNotFound)
}

// ---- Users ----

func TestListUsers_HydratesRolesCountsAndReports(t *testing.T) {
	uid := uuid.New()
	users := []auth.User{{ID: uid, Name: "n", Email: "e", CreatedAt: time.Now().UTC().Truncate(time.Second)}}
	repo := &repoMock{
		listUsers: func(_ context.Context, _ admin.UserFilter, _ *admin.ListUsersCursor, _ int) ([]auth.User, error) {
			return users, nil
		},
		listUserRoles: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID][]string, error) {
			return map[uuid.UUID][]string{uid: {"admin"}}, nil
		},
		countPostsByAuthor: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]int, error) {
			return map[uuid.UUID]int{uid: 3}, nil
		},
		countReportsBySubject: func(_ context.Context, k admin.SubjectType, _ []uuid.UUID) (map[uuid.UUID]int, error) {
			assert.Equal(t, admin.SubjectUser, k)
			return map[uuid.UUID]int{uid: 2}, nil
		},
	}
	svc := newSvc(repo, nil)
	resp, err := svc.ListUsers(ctx, admin.UserFilter{}, nil, 20)
	require.NoError(t, err)
	require.Len(t, resp.Users, 1)
	assert.Equal(t, []string{"admin"}, resp.Users[0].Roles)
	assert.Equal(t, 3, resp.Users[0].PostCount)
	assert.Equal(t, 2, resp.Users[0].ReportedCount)
	assert.NotEmpty(t, resp.NextCursor)
}

func TestGetUser_NotFound(t *testing.T) {
	repo := &repoMock{findUsersByIDs: func(_ context.Context, _ []uuid.UUID) (map[uuid.UUID]auth.User, error) {
		return map[uuid.UUID]auth.User{}, nil
	}}
	svc := newSvc(repo, nil)
	_, err := svc.GetUser(ctx, uuid.New())
	assert.ErrorIs(t, err, admin.ErrUserNotFound)
}

func TestGetUser_HappyPath(t *testing.T) {
	uid := uuid.New()
	repo := &repoMock{
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			return map[uuid.UUID]auth.User{ids[0]: {ID: ids[0], Name: "P", Email: "p@x"}}, nil
		},
	}
	svc := newSvc(repo, nil)
	resp, err := svc.GetUser(ctx, uid)
	require.NoError(t, err)
	assert.Equal(t, uid.String(), resp.ID)
	assert.Equal(t, "P", resp.Name)
}

func TestBlockUser_SelfBlockRejected(t *testing.T) {
	id := uuid.New()
	svc := newSvc(&repoMock{}, nil)
	assert.ErrorIs(t, svc.BlockUser(ctx, id, id), admin.ErrInvalidReporter)
}

func TestBlockUser_HappyPath(t *testing.T) {
	target := uuid.New()
	called := false
	repo := &repoMock{setUserBlocked: func(_ context.Context, id uuid.UUID, b bool) error {
		called = true
		assert.Equal(t, target, id)
		assert.True(t, b)
		return nil
	}}
	aud := &auditMock{}
	svc := newSvc(repo, aud)
	require.NoError(t, svc.BlockUser(ctx, uuid.New(), target))
	assert.True(t, called)
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditUserBlocked, aud.logs[0].Action)
}

func TestUnblockUser_WritesAudit(t *testing.T) {
	aud := &auditMock{}
	svc := newSvc(&repoMock{}, aud)
	require.NoError(t, svc.UnblockUser(ctx, uuid.New(), uuid.New()))
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditUserUnblocked, aud.logs[0].Action)
}

func TestDeactivateReactivate_WriteAuditActions(t *testing.T) {
	aud := &auditMock{}
	svc := newSvc(&repoMock{}, aud)
	require.NoError(t, svc.DeactivateUser(ctx, uuid.New(), uuid.New()))
	require.NoError(t, svc.ReactivateUser(ctx, uuid.New(), uuid.New()))
	require.Len(t, aud.logs, 2)
	assert.Equal(t, admin.AuditUserDeactivated, aud.logs[0].Action)
	assert.Equal(t, admin.AuditUserReactivated, aud.logs[1].Action)
}

func TestBlockUser_RepoErrorPassthrough(t *testing.T) {
	boom := errors.New("db")
	repo := &repoMock{setUserBlocked: func(_ context.Context, _ uuid.UUID, _ bool) error { return boom }}
	svc := newSvc(repo, &auditMock{})
	err := svc.BlockUser(ctx, uuid.New(), uuid.New())
	assert.ErrorIs(t, err, boom)
}

// ---- Posts ----

func TestListPosts_HydratesAndCursors(t *testing.T) {
	pid := uuid.New()
	author := uuid.New()
	pub := time.Now().UTC().Truncate(time.Second)
	posts := []post.Post{{ID: pid, AuthorID: author, Title: "T", PublishedAt: pub}}
	repo := &repoMock{
		listPosts: func(_ context.Context, _ admin.PostFilter, _ *admin.ListPostsCursor, _ int) ([]post.Post, error) {
			return posts, nil
		},
		findUsersByIDs: func(_ context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
			out := make(map[uuid.UUID]auth.User)
			for _, id := range ids {
				out[id] = auth.User{ID: id, Name: "a"}
			}
			return out, nil
		},
		countReportsBySubject: func(_ context.Context, k admin.SubjectType, _ []uuid.UUID) (map[uuid.UUID]int, error) {
			assert.Equal(t, admin.SubjectPost, k)
			return map[uuid.UUID]int{pid: 4}, nil
		},
	}
	svc := newSvc(repo, nil)
	resp, err := svc.ListPosts(ctx, admin.PostFilter{}, nil, 20)
	require.NoError(t, err)
	require.Len(t, resp.Posts, 1)
	assert.Equal(t, 4, resp.Posts[0].ReportCount)
	assert.NotEmpty(t, resp.NextCursor)
}

func TestDeletePost_WritesAudit(t *testing.T) {
	aud := &auditMock{}
	svc := newSvc(&repoMock{}, aud)
	require.NoError(t, svc.DeletePost(ctx, uuid.New(), uuid.New()))
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditPostDeleted, aud.logs[0].Action)
}

func TestRestorePost_WritesAudit(t *testing.T) {
	aud := &auditMock{}
	svc := newSvc(&repoMock{}, aud)
	require.NoError(t, svc.RestorePost(ctx, uuid.New(), uuid.New()))
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditPostRestored, aud.logs[0].Action)
}

func TestDeleteComment_WritesAudit(t *testing.T) {
	aud := &auditMock{}
	svc := newSvc(&repoMock{}, aud)
	require.NoError(t, svc.DeleteComment(ctx, uuid.New(), uuid.New()))
	require.Len(t, aud.logs, 1)
	assert.Equal(t, admin.AuditCommentDeleted, aud.logs[0].Action)
}

// ---- Cursor decoders ----

func TestDecodeCursors(t *testing.T) {
	id := uuid.New()
	tm := time.Now().UTC().Truncate(time.Nanosecond)
	token := tm.Format(time.RFC3339Nano) + ":" + id.String()

	rc, err := admin.DecodeReportsCursor(token)
	require.NoError(t, err)
	require.NotNil(t, rc)
	assert.Equal(t, id, rc.ID)

	uc, err := admin.DecodeUsersCursor(token)
	require.NoError(t, err)
	require.NotNil(t, uc)
	assert.Equal(t, id, uc.ID)

	pc, err := admin.DecodePostsCursor(token)
	require.NoError(t, err)
	require.NotNil(t, pc)
	assert.Equal(t, id, pc.ID)

	empty, err := admin.DecodeReportsCursor("")
	require.NoError(t, err)
	assert.Nil(t, empty)

	_, err = admin.DecodeReportsCursor("garbage")
	assert.Error(t, err)
}

func TestIsValidSubjectAndStatus(t *testing.T) {
	assert.True(t, admin.IsValidSubject("post"))
	assert.True(t, admin.IsValidSubject("comment"))
	assert.True(t, admin.IsValidSubject("user"))
	assert.False(t, admin.IsValidSubject("trip"))

	assert.True(t, admin.IsValidStatus("pending"))
	assert.True(t, admin.IsValidStatus("resolved"))
	assert.True(t, admin.IsValidStatus("dismissed"))
	assert.False(t, admin.IsValidStatus("open"))
}

package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/post"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

type IService interface {
	// Reports (user-facing)
	SubmitReport(ctx context.Context, reporterID uuid.UUID, payload CreateReportPayload) (*ReportResponse, error)

	// Reports (admin)
	ListReports(ctx context.Context, filter ReportFilter, cursor *ListReportsCursor, limit int) (ListReportsResponse, error)
	ResolveReport(ctx context.Context, actorID, reportID uuid.UUID, payload ResolveReportPayload) (*ReportResponse, error)
	DismissReport(ctx context.Context, actorID, reportID uuid.UUID, payload ResolveReportPayload) (*ReportResponse, error)

	// Users (admin)
	ListUsers(ctx context.Context, filter UserFilter, cursor *ListUsersCursor, limit int) (ListUsersResponse, error)
	GetUser(ctx context.Context, id uuid.UUID) (*AdminUserResponse, error)
	BlockUser(ctx context.Context, actorID, targetID uuid.UUID) error
	UnblockUser(ctx context.Context, actorID, targetID uuid.UUID) error
	DeactivateUser(ctx context.Context, actorID, targetID uuid.UUID) error

	// Posts (admin)
	ListPosts(ctx context.Context, filter PostFilter, cursor *ListPostsCursor, limit int) (ListPostsResponse, error)
	DeletePost(ctx context.Context, actorID, postID uuid.UUID) error
	RestorePost(ctx context.Context, actorID, postID uuid.UUID) error

	// Comments (admin)
	DeleteComment(ctx context.Context, actorID, commentID uuid.UUID) error
}

type ServiceConfig struct {
	Repo   IRepository
	Audit  audit.Writer
	Logger *zap.Logger
}

type service struct {
	repo   IRepository
	audit  audit.Writer
	logger *zap.Logger
}

func NewService(cfg ServiceConfig) IService {
	logger := cfg.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	return &service{
		repo:   cfg.Repo,
		audit:  cfg.Audit,
		logger: logger.With(zap.String("layer", "admin.service")),
	}
}

// ---- Reports ----

func (s *service) SubmitReport(ctx context.Context, reporterID uuid.UUID, payload CreateReportPayload) (*ReportResponse, error) {
	subjectID, err := uuid.Parse(payload.SubjectID)
	if err != nil {
		return nil, ErrReportNotFound
	}
	if !IsValidSubject(payload.SubjectType) {
		return nil, ErrReportNotFound
	}
	if err := s.validateSubject(ctx, reporterID, SubjectType(payload.SubjectType), subjectID); err != nil {
		return nil, err
	}
	r := &Report{
		ID:          uuid.New(),
		ReporterID:  reporterID,
		SubjectType: SubjectType(payload.SubjectType),
		SubjectID:   subjectID,
		Reason:      strings.TrimSpace(payload.Reason),
		Description: strings.TrimSpace(payload.Description),
		Status:      StatusPending,
	}
	if err := s.repo.CreateReport(ctx, r); err != nil {
		return nil, err
	}
	s.writeAudit(ctx, AuditReportSubmitted, audit.Success, &reporterID, nil, string(r.SubjectType), subjectID.String(), map[string]any{
		"report_id": r.ID.String(),
		"reason":    r.Reason,
	})
	return s.hydrateReport(ctx, r)
}

func (s *service) validateSubject(ctx context.Context, reporterID uuid.UUID, kind SubjectType, subjectID uuid.UUID) error {
	switch kind {
	case SubjectPost:
		p, err := s.repo.FindPostAny(ctx, subjectID)
		if err != nil {
			return err
		}
		if p == nil {
			return ErrPostNotFound
		}
		if p.AuthorID == reporterID {
			return ErrInvalidReporter
		}
	case SubjectComment:
		c, err := s.repo.FindCommentAny(ctx, subjectID)
		if err != nil {
			return err
		}
		if c == nil {
			return ErrCommentNotFound
		}
		if c.AuthorID == reporterID {
			return ErrInvalidReporter
		}
	case SubjectUser:
		if subjectID == reporterID {
			return ErrInvalidReporter
		}
		users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{subjectID})
		if err != nil {
			return err
		}
		if _, ok := users[subjectID]; !ok {
			return ErrUserNotFound
		}
	}
	return nil
}

func (s *service) ListReports(ctx context.Context, filter ReportFilter, cursor *ListReportsCursor, limit int) (ListReportsResponse, error) {
	rows, err := s.repo.ListReports(ctx, filter, cursor, limit)
	if err != nil {
		return ListReportsResponse{}, err
	}
	hydrated, err := s.hydrateReports(ctx, rows)
	if err != nil {
		return ListReportsResponse{}, err
	}
	out := ListReportsResponse{Reports: hydrated}
	if len(rows) > 0 && len(rows) == cap(rows) {
		// pass-through — cap is just the slice; we rely on the caller comparing len == limit
	}
	if len(rows) > 0 {
		last := rows[len(rows)-1]
		out.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return out, nil
}

func (s *service) ResolveReport(ctx context.Context, actorID, reportID uuid.UUID, payload ResolveReportPayload) (*ReportResponse, error) {
	return s.finalizeReport(ctx, actorID, reportID, StatusResolved, payload.Resolution)
}

func (s *service) DismissReport(ctx context.Context, actorID, reportID uuid.UUID, payload ResolveReportPayload) (*ReportResponse, error) {
	return s.finalizeReport(ctx, actorID, reportID, StatusDismissed, payload.Resolution)
}

func (s *service) finalizeReport(ctx context.Context, actorID, reportID uuid.UUID, status ReportStatus, resolution string) (*ReportResponse, error) {
	if err := s.repo.ResolveReport(ctx, reportID, actorID, status, strings.TrimSpace(resolution)); err != nil {
		return nil, err
	}
	r, err := s.repo.FindReport(ctx, reportID)
	if err != nil {
		return nil, err
	}
	if r == nil {
		return nil, ErrReportNotFound
	}
	action := AuditReportResolved
	if status == StatusDismissed {
		action = AuditReportDismissed
	}
	s.writeAudit(ctx, action, audit.Success, &actorID, nil, string(r.SubjectType), r.SubjectID.String(), map[string]any{
		"report_id":  r.ID.String(),
		"resolution": r.Resolution,
	})
	return s.hydrateReport(ctx, r)
}

// ---- Users ----

func (s *service) ListUsers(ctx context.Context, filter UserFilter, cursor *ListUsersCursor, limit int) (ListUsersResponse, error) {
	users, err := s.repo.ListUsers(ctx, filter, cursor, limit)
	if err != nil {
		return ListUsersResponse{}, err
	}
	ids := make([]uuid.UUID, 0, len(users))
	for _, u := range users {
		ids = append(ids, u.ID)
	}
	roles, err := s.repo.ListUserRoles(ctx, ids)
	if err != nil {
		return ListUsersResponse{}, err
	}
	postCounts, err := s.repo.CountPostsByAuthor(ctx, ids)
	if err != nil {
		return ListUsersResponse{}, err
	}
	reportCounts, err := s.repo.CountReportsBySubject(ctx, SubjectUser, ids)
	if err != nil {
		return ListUsersResponse{}, err
	}
	out := ListUsersResponse{Users: make([]AdminUserResponse, 0, len(users))}
	for _, u := range users {
		out.Users = append(out.Users, toAdminUser(u, roles[u.ID], postCounts[u.ID], reportCounts[u.ID]))
	}
	if len(users) > 0 {
		last := users[len(users)-1]
		out.NextCursor = encodeCursor(last.CreatedAt, last.ID)
	}
	return out, nil
}

func (s *service) GetUser(ctx context.Context, id uuid.UUID) (*AdminUserResponse, error) {
	users, err := s.repo.FindUsersByIDs(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	u, ok := users[id]
	if !ok {
		return nil, ErrUserNotFound
	}
	roles, err := s.repo.ListUserRoles(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	postCounts, err := s.repo.CountPostsByAuthor(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	reportCounts, err := s.repo.CountReportsBySubject(ctx, SubjectUser, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}
	resp := toAdminUser(u, roles[id], postCounts[id], reportCounts[id])
	return &resp, nil
}

func (s *service) BlockUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if actorID == targetID {
		return ErrInvalidReporter
	}
	if err := s.repo.SetUserBlocked(ctx, targetID, true); err != nil {
		return err
	}
	s.writeAudit(ctx, AuditUserBlocked, audit.Success, &actorID, &targetID, string(SubjectUser), targetID.String(), nil)
	return nil
}

func (s *service) UnblockUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if err := s.repo.SetUserBlocked(ctx, targetID, false); err != nil {
		return err
	}
	s.writeAudit(ctx, AuditUserUnblocked, audit.Success, &actorID, &targetID, string(SubjectUser), targetID.String(), nil)
	return nil
}

func (s *service) DeactivateUser(ctx context.Context, actorID, targetID uuid.UUID) error {
	if err := s.repo.DeactivateUser(ctx, targetID); err != nil {
		return err
	}
	s.writeAudit(ctx, AuditUserDeactivated, audit.Success, &actorID, &targetID, string(SubjectUser), targetID.String(), nil)
	return nil
}

// ---- Posts / Comments ----

func (s *service) ListPosts(ctx context.Context, filter PostFilter, cursor *ListPostsCursor, limit int) (ListPostsResponse, error) {
	posts, err := s.repo.ListPosts(ctx, filter, cursor, limit)
	if err != nil {
		return ListPostsResponse{}, err
	}
	authorIDs := make([]uuid.UUID, 0, len(posts))
	postIDs := make([]uuid.UUID, 0, len(posts))
	for _, p := range posts {
		authorIDs = append(authorIDs, p.AuthorID)
		postIDs = append(postIDs, p.ID)
	}
	authors, err := s.repo.FindUsersByIDs(ctx, authorIDs)
	if err != nil {
		return ListPostsResponse{}, err
	}
	reportCounts, err := s.repo.CountReportsBySubject(ctx, SubjectPost, postIDs)
	if err != nil {
		return ListPostsResponse{}, err
	}
	out := ListPostsResponse{Posts: make([]AdminPostResponse, 0, len(posts))}
	for _, p := range posts {
		out.Posts = append(out.Posts, toAdminPost(p, authors[p.AuthorID], reportCounts[p.ID]))
	}
	if len(posts) > 0 {
		last := posts[len(posts)-1]
		out.NextCursor = encodeCursor(last.PublishedAt, last.ID)
	}
	return out, nil
}

func (s *service) DeletePost(ctx context.Context, actorID, postID uuid.UUID) error {
	if err := s.repo.SoftDeletePost(ctx, postID); err != nil {
		return err
	}
	s.writeAudit(ctx, AuditPostDeleted, audit.Success, &actorID, nil, string(SubjectPost), postID.String(), nil)
	return nil
}

func (s *service) RestorePost(ctx context.Context, actorID, postID uuid.UUID) error {
	if err := s.repo.RestorePost(ctx, postID); err != nil {
		return err
	}
	s.writeAudit(ctx, AuditPostRestored, audit.Success, &actorID, nil, string(SubjectPost), postID.String(), nil)
	return nil
}

func (s *service) DeleteComment(ctx context.Context, actorID, commentID uuid.UUID) error {
	if err := s.repo.SoftDeleteCommentAdmin(ctx, commentID); err != nil {
		return err
	}
	s.writeAudit(ctx, AuditCommentDeleted, audit.Success, &actorID, nil, string(SubjectComment), commentID.String(), nil)
	return nil
}

// ---- Helpers ----

func (s *service) hydrateReports(ctx context.Context, rows []Report) ([]ReportResponse, error) {
	if len(rows) == 0 {
		return []ReportResponse{}, nil
	}
	userIDs := make(map[uuid.UUID]struct{}, len(rows)*2)
	postIDs := make([]uuid.UUID, 0)
	commentIDs := make([]uuid.UUID, 0)
	subjectUserIDs := make([]uuid.UUID, 0)
	for _, r := range rows {
		userIDs[r.ReporterID] = struct{}{}
		if r.ResolvedBy != nil {
			userIDs[*r.ResolvedBy] = struct{}{}
		}
		switch r.SubjectType {
		case SubjectPost:
			postIDs = append(postIDs, r.SubjectID)
		case SubjectComment:
			commentIDs = append(commentIDs, r.SubjectID)
		case SubjectUser:
			subjectUserIDs = append(subjectUserIDs, r.SubjectID)
			userIDs[r.SubjectID] = struct{}{}
		}
	}
	userIDList := make([]uuid.UUID, 0, len(userIDs))
	for id := range userIDs {
		userIDList = append(userIDList, id)
	}
	users, err := s.repo.FindUsersByIDs(ctx, userIDList)
	if err != nil {
		return nil, err
	}
	postMap := make(map[uuid.UUID]post.Post)
	for _, pid := range postIDs {
		p, err := s.repo.FindPostAny(ctx, pid)
		if err != nil {
			return nil, err
		}
		if p != nil {
			postMap[pid] = *p
		}
	}
	commentMap := make(map[uuid.UUID]post.Comment)
	for _, cid := range commentIDs {
		c, err := s.repo.FindCommentAny(ctx, cid)
		if err != nil {
			return nil, err
		}
		if c != nil {
			commentMap[cid] = *c
		}
	}
	out := make([]ReportResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, buildReportResponse(r, users, postMap, commentMap))
	}
	return out, nil
}

func (s *service) hydrateReport(ctx context.Context, r *Report) (*ReportResponse, error) {
	rows, err := s.hydrateReports(ctx, []Report{*r})
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrReportNotFound
	}
	return &rows[0], nil
}

func buildReportResponse(r Report, users map[uuid.UUID]auth.User, postMap map[uuid.UUID]post.Post, commentMap map[uuid.UUID]post.Comment) ReportResponse {
	resp := ReportResponse{
		ID:          r.ID.String(),
		Reporter:    toUserSummary(users[r.ReporterID]),
		SubjectType: string(r.SubjectType),
		SubjectID:   r.SubjectID.String(),
		Reason:      r.Reason,
		Description: r.Description,
		Status:      string(r.Status),
		Resolution:  r.Resolution,
		ResolvedAt:  r.ResolvedAt,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
	if r.ResolvedBy != nil {
		if u, ok := users[*r.ResolvedBy]; ok {
			s := toUserSummary(u)
			resp.ResolvedBy = &s
		}
	}
	switch r.SubjectType {
	case SubjectPost:
		if p, ok := postMap[r.SubjectID]; ok {
			resp.Subject = &SubjectView{
				Kind:     string(SubjectPost),
				ID:       p.ID.String(),
				Title:    p.Title,
				Excerpt:  excerpt(p.Content, 200),
				AuthorID: p.AuthorID.String(),
				Deleted:  p.DeletedAt != nil,
			}
		}
	case SubjectComment:
		if c, ok := commentMap[r.SubjectID]; ok {
			resp.Subject = &SubjectView{
				Kind:     string(SubjectComment),
				ID:       c.ID.String(),
				Excerpt:  excerpt(c.Content, 200),
				AuthorID: c.AuthorID.String(),
				Deleted:  c.DeletedAt != nil,
			}
		}
	case SubjectUser:
		if u, ok := users[r.SubjectID]; ok {
			resp.Subject = &SubjectView{
				Kind:  string(SubjectUser),
				ID:    u.ID.String(),
				Title: u.Name,
			}
		}
	}
	return resp
}

func toUserSummary(u auth.User) UserSummary {
	return UserSummary{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
	}
}

func toAdminUser(u auth.User, roles []string, postCount, reportCount int) AdminUserResponse {
	return AdminUserResponse{
		ID:            u.ID.String(),
		Email:         u.Email,
		Name:          u.Name,
		AvatarURL:     u.AvatarURL,
		Status:        string(u.Status),
		IsBlocked:     u.IsBlocked,
		IsVerified:    u.IsVerified,
		Roles:         roles,
		PostCount:     postCount,
		ReportedCount: reportCount,
		CreatedAt:     u.CreatedAt,
		DeactivatedAt: u.DeactivatedAt,
	}
}

func toAdminPost(p post.Post, author auth.User, reportCount int) AdminPostResponse {
	return AdminPostResponse{
		ID:           p.ID.String(),
		Author:       toUserSummary(author),
		Title:        p.Title,
		Content:      p.Content,
		Status:       p.Status,
		LikeCount:    p.LikeCount,
		CommentCount: p.CommentCount,
		ReportCount:  reportCount,
		IsDeleted:    p.DeletedAt != nil,
		PublishedAt:  p.PublishedAt,
		CreatedAt:    p.CreatedAt,
		UpdatedAt:    p.UpdatedAt,
	}
}

func excerpt(s string, max int) string {
	s = strings.TrimSpace(s)
	if len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func (s *service) writeAudit(ctx context.Context, action audit.Action, status audit.Status, actorID, targetID *uuid.UUID, resourceType, resourceID string, detail map[string]any) {
	if s.audit == nil {
		return
	}
	log := &audit.Log{
		ID:           uuid.New(),
		ActorUserID:  actorID,
		TargetUserID: targetID,
		Action:       action,
		Status:       status,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		TraceID:      middleware.GetTraceID(ctx),
	}
	if rid := middleware.RequestIDFromContext(ctx); rid != uuid.Nil {
		log.RequestID = &rid
	}
	if detail != nil {
		log.Detail = detail
	}
	if err := s.audit.Create(ctx, log); err != nil {
		s.logger.Warn("audit write failed", zap.Error(err), zap.String("action", string(action)))
	}
}

// ---- Cursor helpers ----

func encodeCursor(t time.Time, id uuid.UUID) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano) + ":" + id.String()
}

func decodeCursor(token string) (*time.Time, uuid.UUID, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, uuid.Nil, nil
	}
	sep := strings.LastIndex(token, ":")
	if sep <= 0 || sep == len(token)-1 {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	t, err := time.Parse(time.RFC3339Nano, token[:sep])
	if err != nil {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	id, err := uuid.Parse(token[sep+1:])
	if err != nil {
		return nil, uuid.Nil, errors.New("invalid cursor")
	}
	return &t, id, nil
}

func DecodeReportsCursor(token string) (*ListReportsCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &ListReportsCursor{CreatedAt: *t, ID: id}, nil
}

func DecodeUsersCursor(token string) (*ListUsersCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &ListUsersCursor{CreatedAt: *t, ID: id}, nil
}

func DecodePostsCursor(token string) (*ListPostsCursor, error) {
	t, id, err := decodeCursor(token)
	if err != nil || t == nil {
		return nil, err
	}
	return &ListPostsCursor{PublishedAt: *t, ID: id}, nil
}

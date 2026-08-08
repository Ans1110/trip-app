package admin

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/internal/post"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

var (
	ErrReportNotFound   = errors.New("report not found")
	ErrDuplicateReport  = errors.New("report already pending")
	ErrUserNotFound     = errors.New("user not found")
	ErrPostNotFound     = errors.New("post not found")
	ErrCommentNotFound  = errors.New("comment not found")
	ErrInvalidReporter  = errors.New("cannot report own content")
)

type UserFilter struct {
	Keyword   string
	Status    string
	IsBlocked *bool
}

type ListUsersCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type PostFilter struct {
	Keyword    string
	Status     string
	AuthorID   uuid.UUID
	IncludeDel bool
}

type ListPostsCursor struct {
	PublishedAt time.Time
	ID          uuid.UUID
}

type ReportFilter struct {
	Status      string
	SubjectType string
}

type ListReportsCursor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type IRepository interface {
	// Reports
	CreateReport(ctx context.Context, r *Report) error
	FindReport(ctx context.Context, id uuid.UUID) (*Report, error)
	ListReports(ctx context.Context, filter ReportFilter, cursor *ListReportsCursor, limit int) ([]Report, error)
	ResolveReport(ctx context.Context, id, resolvedBy uuid.UUID, status ReportStatus, resolution string) error
	CountReportsBySubject(ctx context.Context, subjectType SubjectType, subjectIDs []uuid.UUID) (map[uuid.UUID]int, error)

	// Users
	FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error)
	ListUsers(ctx context.Context, filter UserFilter, cursor *ListUsersCursor, limit int) ([]auth.User, error)
	SetUserBlocked(ctx context.Context, id uuid.UUID, blocked bool) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	ListUserRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]string, error)
	CountPostsByAuthor(ctx context.Context, authorIDs []uuid.UUID) (map[uuid.UUID]int, error)

	// Posts / Comments
	ListPosts(ctx context.Context, filter PostFilter, cursor *ListPostsCursor, limit int) ([]post.Post, error)
	FindPostAny(ctx context.Context, id uuid.UUID) (*post.Post, error)
	SoftDeletePost(ctx context.Context, id uuid.UUID) error
	RestorePost(ctx context.Context, id uuid.UUID) error
	FindCommentAny(ctx context.Context, id uuid.UUID) (*post.Comment, error)
	SoftDeleteCommentAdmin(ctx context.Context, id uuid.UUID) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

// ---- Reports ----

func (r *repository) CreateReport(ctx context.Context, rp *Report) error {
	if rp.ID == uuid.Nil {
		rp.ID = uuid.New()
	}
	if rp.Status == "" {
		rp.Status = StatusPending
	}
	err := r.db.WithContext(ctx).Create(rp).Error
	if err != nil && isUniqueViolation(err) {
		return ErrDuplicateReport
	}
	return err
}

func (r *repository) FindReport(ctx context.Context, id uuid.UUID) (*Report, error) {
	var rp Report
	err := r.db.WithContext(ctx).First(&rp, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &rp, nil
}

func (r *repository) ListReports(ctx context.Context, filter ReportFilter, cursor *ListReportsCursor, limit int) ([]Report, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&Report{})
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.SubjectType != "" {
		q = q.Where("subject_type = ?", filter.SubjectType)
	}
	if cursor != nil {
		q = q.Where("(created_at, id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	var rows []Report
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) ResolveReport(ctx context.Context, id, resolvedBy uuid.UUID, status ReportStatus, resolution string) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&Report{}).
		Where("id = ? AND status = ?", id, StatusPending).
		Updates(map[string]any{
			"status":      status,
			"resolution":  resolution,
			"resolved_by": resolvedBy,
			"resolved_at": now,
			"updated_at":  now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrReportNotFound
	}
	return nil
}

func (r *repository) CountReportsBySubject(ctx context.Context, subjectType SubjectType, subjectIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(subjectIDs))
	if len(subjectIDs) == 0 {
		return out, nil
	}
	type row struct {
		SubjectID uuid.UUID
		N         int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&Report{}).
		Select("subject_id, COUNT(*) AS n").
		Where("subject_type = ? AND subject_id IN ?", subjectType, subjectIDs).
		Group("subject_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.SubjectID] = int(r.N)
	}
	return out, nil
}

// ---- Users ----

func (r *repository) FindUsersByIDs(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]auth.User, error) {
	out := make(map[uuid.UUID]auth.User, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	var users []auth.User
	if err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error; err != nil {
		return nil, err
	}
	for _, u := range users {
		out[u.ID] = u
	}
	return out, nil
}

func (r *repository) ListUsers(ctx context.Context, filter UserFilter, cursor *ListUsersCursor, limit int) ([]auth.User, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&auth.User{})
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		like := "%" + strings.ToLower(kw) + "%"
		q = q.Where("LOWER(email) LIKE ? OR LOWER(name) LIKE ?", like, like)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.IsBlocked != nil {
		q = q.Where("is_blocked = ?", *filter.IsBlocked)
	}
	if cursor != nil {
		q = q.Where("(created_at, id) < (?, ?)", cursor.CreatedAt, cursor.ID)
	}
	var rows []auth.User
	err := q.Order("created_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) SetUserBlocked(ctx context.Context, id uuid.UUID, blocked bool) error {
	res := r.db.WithContext(ctx).Model(&auth.User{}).
		Where("id = ?", id).
		Update("is_blocked", blocked)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&auth.User{}).
		Where("id = ?", id).
		Updates(map[string]any{
			"status":         auth.UserStatusDeactivated,
			"deactivated_at": now,
		})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrUserNotFound
	}
	return nil
}

func (r *repository) ListUserRoles(ctx context.Context, userIDs []uuid.UUID) (map[uuid.UUID][]string, error) {
	out := make(map[uuid.UUID][]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	type row struct {
		UserID uuid.UUID
		Name   string
	}
	var rows []row
	err := r.db.WithContext(ctx).
		Table("auth.user_roles AS ur").
		Select("ur.user_id, r.name").
		Joins("JOIN auth.roles AS r ON r.id = ur.role_id").
		Where("ur.user_id IN ?", userIDs).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.UserID] = append(out[r.UserID], r.Name)
	}
	return out, nil
}

func (r *repository) CountPostsByAuthor(ctx context.Context, authorIDs []uuid.UUID) (map[uuid.UUID]int, error) {
	out := make(map[uuid.UUID]int, len(authorIDs))
	if len(authorIDs) == 0 {
		return out, nil
	}
	type row struct {
		AuthorID uuid.UUID
		N        int64
	}
	var rows []row
	err := r.db.WithContext(ctx).Model(&post.Post{}).
		Select("author_id, COUNT(*) AS n").
		Where("author_id IN ? AND deleted_at IS NULL", authorIDs).
		Group("author_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		out[r.AuthorID] = int(r.N)
	}
	return out, nil
}

// ---- Posts / Comments ----

func (r *repository) ListPosts(ctx context.Context, filter PostFilter, cursor *ListPostsCursor, limit int) ([]post.Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	q := r.db.WithContext(ctx).Model(&post.Post{})
	if !filter.IncludeDel {
		q = q.Where("deleted_at IS NULL")
	}
	if kw := strings.TrimSpace(filter.Keyword); kw != "" {
		like := "%" + strings.ToLower(kw) + "%"
		q = q.Where("LOWER(title) LIKE ? OR LOWER(content) LIKE ?", like, like)
	}
	if filter.Status != "" {
		q = q.Where("status = ?", filter.Status)
	}
	if filter.AuthorID != uuid.Nil {
		q = q.Where("author_id = ?", filter.AuthorID)
	}
	if cursor != nil {
		q = q.Where("(published_at, id) < (?, ?)", cursor.PublishedAt, cursor.ID)
	}
	var rows []post.Post
	err := q.Order("published_at DESC, id DESC").Limit(limit).Find(&rows).Error
	return rows, err
}

func (r *repository) FindPostAny(ctx context.Context, id uuid.UUID) (*post.Post, error) {
	var p post.Post
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *repository) SoftDeletePost(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&post.Post{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *repository) RestorePost(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&post.Post{}).
		Where("id = ? AND deleted_at IS NOT NULL", id).
		Update("deleted_at", nil)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrPostNotFound
	}
	return nil
}

func (r *repository) FindCommentAny(ctx context.Context, id uuid.UUID) (*post.Comment, error) {
	var c post.Comment
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&c).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (r *repository) SoftDeleteCommentAdmin(ctx context.Context, id uuid.UUID) error {
	res := r.db.WithContext(ctx).Model(&post.Comment{}).
		Where("id = ? AND deleted_at IS NULL", id).
		Update("deleted_at", time.Now())
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return ErrCommentNotFound
	}
	return nil
}

// isUniqueViolation covers both pgconn's SQLSTATE 23505 and generic driver-side
// error strings so we can keep the dependency graph narrow.
func isUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "SQLSTATE 23505") ||
		strings.Contains(msg, "duplicate key value")
}

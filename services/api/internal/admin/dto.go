package admin

import "time"

type ElevatePayload struct {
	Password string `json:"password" binding:"required,min=1,max=128"`
}

type ElevationResponse struct {
	Elevated  bool      `json:"elevated"`
	ExpiresAt time.Time `json:"expires_at"`
}

type CreateReportPayload struct {
	SubjectType string `json:"subject_type" binding:"required,oneof=post comment user"`
	SubjectID   string `json:"subject_id"   binding:"required,uuid"`
	Reason      string `json:"reason"       binding:"required,min=1,max=32"`
	Description string `json:"description"  binding:"omitempty,max=1000"`
}

type ResolveReportPayload struct {
	Resolution string `json:"resolution" binding:"omitempty,max=1000"`
}

type UserSummary struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	Name      string `json:"name"`
	AvatarURL string `json:"avatar_url,omitempty"`
}

type ReportResponse struct {
	ID          string       `json:"id"`
	Reporter    UserSummary  `json:"reporter"`
	SubjectType string       `json:"subject_type"`
	SubjectID   string       `json:"subject_id"`
	Subject     *SubjectView `json:"subject,omitempty"`
	Reason      string       `json:"reason"`
	Description string       `json:"description,omitempty"`
	Status      string       `json:"status"`
	Resolution  string       `json:"resolution,omitempty"`
	ResolvedBy  *UserSummary `json:"resolved_by,omitempty"`
	ResolvedAt  *time.Time   `json:"resolved_at,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
	UpdatedAt   time.Time    `json:"updated_at"`
}

// SubjectView is a compact snapshot of what was reported. Populated only when
// available (soft-deleted rows still exist; hard-deleted references stay nil).
type SubjectView struct {
	Kind      string `json:"kind"`
	ID        string `json:"id"`
	Title     string `json:"title,omitempty"`
	Excerpt   string `json:"excerpt,omitempty"`
	AuthorID  string `json:"author_id,omitempty"`
	Deleted   bool   `json:"deleted,omitempty"`
}

type ListReportsResponse struct {
	Reports    []ReportResponse `json:"reports"`
	NextCursor string           `json:"next_cursor,omitempty"`
}

type AdminUserResponse struct {
	ID            string     `json:"id"`
	Email         string     `json:"email"`
	Name          string     `json:"name"`
	AvatarURL     string     `json:"avatar_url,omitempty"`
	Status        string     `json:"status"`
	IsBlocked     bool       `json:"is_blocked"`
	IsVerified    bool       `json:"is_verified"`
	Roles         []string   `json:"roles"`
	PostCount     int        `json:"post_count"`
	ReportedCount int        `json:"reported_count"`
	CreatedAt     time.Time  `json:"created_at"`
	DeactivatedAt *time.Time `json:"deactivated_at,omitempty"`
}

type ListUsersResponse struct {
	Users      []AdminUserResponse `json:"users"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type AdminPostResponse struct {
	ID           string      `json:"id"`
	Author       UserSummary `json:"author"`
	Title        string      `json:"title"`
	Content      string      `json:"content"`
	Status       string      `json:"status"`
	LikeCount    int         `json:"like_count"`
	CommentCount int         `json:"comment_count"`
	ReportCount  int         `json:"report_count"`
	IsDeleted    bool        `json:"is_deleted"`
	PublishedAt  time.Time   `json:"published_at"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type ListPostsResponse struct {
	Posts      []AdminPostResponse `json:"posts"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

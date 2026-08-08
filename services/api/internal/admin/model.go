package admin

import (
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/google/uuid"
)

type SubjectType string

const (
	SubjectPost    SubjectType = "post"
	SubjectComment SubjectType = "comment"
	SubjectUser    SubjectType = "user"
)

func IsValidSubject(s string) bool {
	switch SubjectType(s) {
	case SubjectPost, SubjectComment, SubjectUser:
		return true
	}
	return false
}

type ReportStatus string

const (
	StatusPending   ReportStatus = "pending"
	StatusResolved  ReportStatus = "resolved"
	StatusDismissed ReportStatus = "dismissed"
)

func IsValidStatus(s string) bool {
	switch ReportStatus(s) {
	case StatusPending, StatusResolved, StatusDismissed:
		return true
	}
	return false
}

type Report struct {
	ID          uuid.UUID    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReporterID  uuid.UUID    `gorm:"type:uuid;column:reporter_id;not null"`
	SubjectType SubjectType  `gorm:"column:subject_type;size:16;not null"`
	SubjectID   uuid.UUID    `gorm:"type:uuid;column:subject_id;not null"`
	Reason      string       `gorm:"column:reason;size:32;not null"`
	Description string       `gorm:"column:description;not null;default:''"`
	Status      ReportStatus `gorm:"column:status;size:16;not null;default:'pending'"`
	Resolution  string       `gorm:"column:resolution;not null;default:''"`
	ResolvedBy  *uuid.UUID   `gorm:"type:uuid;column:resolved_by"`
	ResolvedAt  *time.Time   `gorm:"column:resolved_at"`
	CreatedAt   time.Time    `gorm:"column:created_at"`
	UpdatedAt   time.Time    `gorm:"column:updated_at"`
}

func (Report) TableName() string { return "admin.reports" }

// Admin-domain audit actions.
const (
	AuditUserBlocked      audit.Action = "admin_user_blocked"
	AuditUserUnblocked    audit.Action = "admin_user_unblocked"
	AuditUserDeactivated  audit.Action = "admin_user_deactivated"
	AuditUserReactivated  audit.Action = "admin_user_reactivated"
	AuditPostDeleted      audit.Action = "admin_post_deleted"
	AuditPostRestored     audit.Action = "admin_post_restored"
	AuditCommentDeleted   audit.Action = "admin_comment_deleted"
	AuditReportResolved   audit.Action = "admin_report_resolved"
	AuditReportDismissed  audit.Action = "admin_report_dismissed"
	AuditReportSubmitted  audit.Action = "report_submitted"
)

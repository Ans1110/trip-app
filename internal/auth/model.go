package auth

import (
	"time"

	"github.com/google/uuid"
)

type UserStatus string
type DeviceType string

const (
	UserStatusActive      UserStatus = "active"
	UserStatusDeactivated UserStatus = "deactivated"
	UserStatusDeleted     UserStatus = "deleted"
)

const (
	DeviceWeb     DeviceType = "web"
	DeviceIOS     DeviceType = "ios"
	DeviceAndroid DeviceType = "android"
)

type User struct {
	ID                  uuid.UUID  `gorm:"type:uuid;primaryKey"`
	Email               string     `gorm:"size:320;uniqueIndex;not null"`
	Name                string     `gorm:"size:255;not null"`
	AvatarURL           string     `gorm:"column:avatar_url"`
	PasswordHash        *string    `gorm:"column:password_hash"`
	IsBlocked           bool       `gorm:"default:false"`
	IsVerified          bool       `gorm:"column:is_verified;default:false"`
	Status              UserStatus `gorm:"default:'active'"` // active | deactivated | deleted
	DeactivatedAt       *time.Time `gorm:"column:deactivated_at"`
	DeletionScheduledAt *time.Time `gorm:"column:deletion_scheduled_at"`
	CreatedAt           time.Time
	UpdatedAt           time.Time
}

func (User) TableName() string {
	return "auth.users"
}

type Provider struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey"`
	UserID     uuid.UUID `gorm:"type:uuid;index;not null"`
	User       User      `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	Provider   string    `gorm:"not null;uniqueIndex:idx_provider_pid"` // google | github | facebook
	ProviderID string    `gorm:"column:provider_id;not null;uniqueIndex:idx_provider_pid"`
	CreatedAt  time.Time
}

func (Provider) TableName() string {
	return "auth.providers"
}

type EmailVerification struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null"`
	User      User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	TokenHash string     `gorm:"size:64;not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"not null"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time
}

func (EmailVerification) TableName() string {
	return "auth.email_verifications"
}

type PasswordReset struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID    uuid.UUID  `gorm:"type:uuid;index;not null"`
	User      User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	TokenHash string     `gorm:"not null;uniqueIndex"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time
}

func (PasswordReset) TableName() string {
	return "auth.password_resets"
}

type MFAConfig struct {
	UserID     uuid.UUID `gorm:"type:uuid;primaryKey"`
	TOTPSecret string    `gorm:"column:totp_secret;not null"` // AES-GCM ciphertext (hex)
	IsEnabled  bool      `gorm:"column:is_enabled;default:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (MFAConfig) TableName() string {
	return "auth.mfa_configs"
}

type UserSession struct {
	ID                uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID            uuid.UUID  `gorm:"type:uuid;index;not null"`
	User              User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	DeviceName        string     `gorm:"column:device_name"`
	DeviceType        DeviceType `gorm:"column:device_type"` // web | ios | android
	DeviceFingerprint string     `gorm:"column:device_fingerprint;size:64;index"`
	IPAddress         string     `gorm:"column:ip_address"`
	UserAgent         string     `gorm:"type:text;column:user_agent"`
	RefreshTokenHash  string     `gorm:"column:refresh_token_hash;not null;uniqueIndex"`
	LastActiveAt      time.Time  `gorm:"column:last_active_at"`
	ExpiresAt         time.Time  `gorm:"column:expires_at"`
	RevokedAt         *time.Time `gorm:"column:revoked_at"`
	CreatedAt         time.Time
}

func (UserSession) TableName() string {
	return "auth.user_sessions"
}

type Role struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey"`
	Name      string    `gorm:"size:64;uniqueIndex;not null"`
	CreatedAt time.Time
}

func (Role) TableName() string {
	return "auth.roles"
}

type UserRole struct {
	UserID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	RoleID    uuid.UUID `gorm:"type:uuid;primaryKey"`
	CreatedAt time.Time
}

func (UserRole) TableName() string {
	return "auth.user_roles"
}

type AuditAction string

const (
	AuditLogin              AuditAction = "login"
	AuditLoginFailed        AuditAction = "login_failed"
	AuditRegister           AuditAction = "register"
	AuditOAuthLogin         AuditAction = "oauth_login"
	AuditRefresh            AuditAction = "token_refresh"
	AuditLogout             AuditAction = "logout"
	AuditPasswordReset      AuditAction = "password_reset"
	AuditPasswordChange     AuditAction = "password_change"
	AuditTOTPEnabled        AuditAction = "totp_enabled"
	AuditTOTPDisabled       AuditAction = "totp_disabled"
	AuditAccountDeactivated AuditAction = "account_deactivated"
	AuditAccountDeleted     AuditAction = "account_deleted"
	AuditRateLimited        AuditAction = "rate_limited"
)

type AuditStatus string

const (
	AuditSuccess AuditStatus = "success"
	AuditFailure AuditStatus = "failure"
)

type AuditLog struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	ActorUserID  *uuid.UUID     `gorm:"type:uuid;column:actor_user_id;index"`
	TargetUserID *uuid.UUID     `gorm:"type:uuid;column:target_user_id;index"`
	Action       AuditAction    `gorm:"size:64;not null;index"`
	Status       AuditStatus    `gorm:"size:32;not null"`
	ResourceType string         `gorm:"size:64;column:resource_type"`
	ResourceID   string         `gorm:"type:text;column:resource_id"`
	IPAddress    *string        `gorm:"type:inet;column:ip_address"`
	UserAgent    string         `gorm:"type:text;column:user_agent"`
	RequestID    *uuid.UUID     `gorm:"type:uuid;column:request_id"`
	TraceID      string         `gorm:"type:text;column:trace_id"`
	Detail       map[string]any `gorm:"type:jsonb;serializer:json"`
	CreatedAt    time.Time
}

func (AuditLog) TableName() string {
	return "auth.audit_logs"
}

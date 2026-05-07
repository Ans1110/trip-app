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
	TOTPSecret string    `gorm:"column:totp_secret;not null"`
	IsEnabled  bool      `gorm:"column:is_enabled;default:false"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (MFAConfig) TableName() string {
	return "auth.mfa_configs"
}

type UserSession struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey"`
	UserID           uuid.UUID  `gorm:"type:uuid;index;not null"`
	User             User       `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
	DeviceName       string     `gorm:"column:device_name"`
	DeviceType       DeviceType `gorm:"column:device_type"` // web | ios | android
	IPAddress        string     `gorm:"column:ip_address"`
	UserAgent        string     `gorm:"type:text;column:user_agent"`
	RefreshTokenHash string     `gorm:"column:refresh_token_hash;not null;uniqueIndex"`
	LastActiveAt     time.Time  `gorm:"column:last_active_at"`
	ExpiresAt        time.Time  `gorm:"column:expires_at"`
	RevokedAt        *time.Time `gorm:"column:revoked_at"`
	CreatedAt        time.Time
}

func (UserSession) TableName() string {
	return "auth.user_sessions"
}

package auth

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type IRepository interface {
	WithTx(ctx context.Context, fn func(IRepository) error) error

	CreateUser(ctx context.Context, user *User) error
	FindUserByEmail(ctx context.Context, email string) (*User, error)
	FindUserByID(ctx context.Context, id uuid.UUID) (*User, error)
	FindUserByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error)
	UpdateUserFields(ctx context.Context, id uuid.UUID, updates map[string]any) error
	DeactivateUser(ctx context.Context, id uuid.UUID) error
	MarkUserDeleted(ctx context.Context, id uuid.UUID) error
	BlockUser(ctx context.Context, id uuid.UUID) error
	CreateProvider(ctx context.Context, provider *Provider) error
	FindProviderByProviderID(ctx context.Context, provider, providerID string) (*Provider, error)

	CreateEmailVerification(ctx context.Context, ev *EmailVerification) error
	FindEmailVerificationByTokenHash(ctx context.Context, tokenHash string) (*EmailVerification, error)
	MarkEmailVerificationUsed(ctx context.Context, id uuid.UUID) error

	CreatePasswordReset(ctx context.Context, pr *PasswordReset) error
	FindPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*PasswordReset, error)
	MarkPasswordResetUsed(ctx context.Context, id uuid.UUID) error
	InvalidatePendingPasswordResets(ctx context.Context, userID uuid.UUID) error

	UpsertMFAConfig(ctx context.Context, config *MFAConfig) error
	FindMFAConfig(ctx context.Context, userID uuid.UUID) (*MFAConfig, error)

	CreateUserSession(ctx context.Context, session *UserSession) error
	FindUserSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*UserSession, error)
	RevokeUserSession(ctx context.Context, id uuid.UUID) error
	RevokeUserSessionIfActive(ctx context.Context, id uuid.UUID) (bool, error)
	ListSessionByUserID(ctx context.Context, userID uuid.UUID) ([]UserSession, error)
	DeleteUserSession(ctx context.Context, id uuid.UUID) error
	DeleteAllSessions(ctx context.Context, userID uuid.UUID) error

	ListUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error)

	CreateAuditLog(ctx context.Context, log *AuditLog) error
}

type repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) IRepository {
	return &repository{db: db}
}

func (r *repository) WithTx(ctx context.Context, fn func(IRepository) error) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		return fn(&repository{db: tx})
	})
}

func (r *repository) CreateUser(ctx context.Context, user *User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *repository) FindUserByEmail(ctx context.Context, email string) (*User, error) {
	email = strings.ToLower(strings.TrimSpace(email))
	var user User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *repository) FindUserByID(ctx context.Context, id uuid.UUID) (*User, error) {
	var user User
	err := r.db.WithContext(ctx).First(&user, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &user, err
}

func (r *repository) FindUserByIDs(ctx context.Context, ids []uuid.UUID) ([]User, error) {
	var users []User
	err := r.db.WithContext(ctx).Where("id IN ?", ids).Find(&users).Error
	return users, err
}

func (r *repository) UpdateUserFields(ctx context.Context, id uuid.UUID, updates map[string]any) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(updates).Error
}

func (r *repository) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(map[string]any{
		"status":         UserStatusDeactivated,
		"deactivated_at": now,
	}).Error
}

func (r *repository) MarkUserDeleted(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(map[string]any{
		"status":                UserStatusDeleted,
		"deletion_scheduled_at": now,
	}).Error
}

func (r *repository) BlockUser(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Update("is_blocked", true).Error
}

func (r *repository) CreateProvider(ctx context.Context, provider *Provider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

func (r *repository) FindProviderByProviderID(ctx context.Context, provider, providerID string) (*Provider, error) {
	var p Provider
	err := r.db.WithContext(ctx).Where("provider = ? AND provider_id = ?", provider, providerID).First(&p).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &p, err
}

func (r *repository) CreateEmailVerification(ctx context.Context, ev *EmailVerification) error {
	return r.db.WithContext(ctx).Create(ev).Error
}

func (r *repository) FindEmailVerificationByTokenHash(ctx context.Context, tokenHash string) (*EmailVerification, error) {
	var ev EmailVerification
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&ev).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &ev, err
}

func (r *repository) MarkEmailVerificationUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&EmailVerification{}).Where("id = ?", id).Update("used_at", now).Error
}

func (r *repository) CreatePasswordReset(ctx context.Context, pr *PasswordReset) error {
	return r.db.WithContext(ctx).Create(pr).Error
}

func (r *repository) FindPasswordResetByTokenHash(ctx context.Context, tokenHash string) (*PasswordReset, error) {
	var pr PasswordReset
	err := r.db.WithContext(ctx).
		Where("token_hash = ? AND used_at IS NULL AND expires_at > ?", tokenHash, time.Now()).
		First(&pr).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &pr, err
}

func (r *repository) MarkPasswordResetUsed(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&PasswordReset{}).Where("id = ?", id).Update("used_at", now).Error
}

func (r *repository) InvalidatePendingPasswordResets(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&PasswordReset{}).
		Where("user_id = ? AND used_at IS NULL", userID).
		Update("used_at", now).Error
}

func (r *repository) UpsertMFAConfig(ctx context.Context, config *MFAConfig) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "user_id"}},
		DoUpdates: clause.AssignmentColumns([]string{"totp_secret", "is_enabled", "updated_at"}),
	}).Create(config).Error
}

func (r *repository) FindMFAConfig(ctx context.Context, userID uuid.UUID) (*MFAConfig, error) {
	var config MFAConfig
	err := r.db.WithContext(ctx).First(&config, "user_id = ?", userID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &config, err
}

func (r *repository) CreateUserSession(ctx context.Context, session *UserSession) error {
	return r.db.WithContext(ctx).Create(session).Error
}

func (r *repository) FindUserSessionByRefreshTokenHash(ctx context.Context, refreshTokenHash string) (*UserSession, error) {
	var session UserSession
	err := r.db.WithContext(ctx).
		Where("refresh_token_hash = ? AND revoked_at IS NULL AND expires_at > ?", refreshTokenHash, time.Now()).
		First(&session).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	return &session, err
}

func (r *repository) RevokeUserSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&UserSession{}).Where("id = ?", id).Update("revoked_at", now).Error
}

// RevokeUserSessionIfActive atomically marks a session revoked only when it is
// not already revoked, returning whether the caller's update won the race.
// Used to detect concurrent refresh attempts that would otherwise duplicate sessions.
func (r *repository) RevokeUserSessionIfActive(ctx context.Context, id uuid.UUID) (bool, error) {
	now := time.Now()
	res := r.db.WithContext(ctx).Model(&UserSession{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", now)
	if res.Error != nil {
		return false, res.Error
	}
	return res.RowsAffected > 0, nil
}

func (r *repository) ListSessionByUserID(ctx context.Context, userID uuid.UUID) ([]UserSession, error) {
	var sessions []UserSession
	err := r.db.WithContext(ctx).Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, time.Now()).
		Order("last_active_at DESC").Find(&sessions).Error
	return sessions, err
}

func (r *repository) DeleteUserSession(ctx context.Context, id uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&UserSession{}).
		Where("id = ? AND revoked_at IS NULL", id).Update("revoked_at", now).Error
}

func (r *repository) DeleteAllSessions(ctx context.Context, userID uuid.UUID) error {
	now := time.Now()
	return r.db.WithContext(ctx).Model(&UserSession{}).
		Where("user_id = ? AND revoked_at IS NULL", userID).Update("revoked_at", now).Error
}

func (r *repository) ListUserRoles(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var names []string
	err := r.db.WithContext(ctx).
		Table("auth.roles AS r").
		Joins("JOIN auth.user_roles AS ur ON ur.role_id = r.id").
		Where("ur.user_id = ?", userID).
		Pluck("r.name", &names).Error
	return names, err
}

func (r *repository) CreateAuditLog(ctx context.Context, log *AuditLog) error {
	return r.db.WithContext(ctx).Create(log).Error
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/internal/audit"
	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/Ans1110/trip-app/pkg/event"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists         = errors.New("email already registered")
	ErrInvalidCredentials  = errors.New("invalid email or password")
	ErrUserNotFound        = errors.New("user not found")
	ErrUserBlocked         = errors.New("user is blocked")
	ErrInvalidToken        = errors.New("invalid or expired token")
	ErrInvalidMFACode      = errors.New("invalid mfa code")
	ErrMFANotConfigured    = errors.New("mfa not configured")
	ErrMFAAlreadyEnabled   = errors.New("mfa already enabled")
	ErrPasswordNotSet      = errors.New("password not set")
	ErrSessionNotFound     = errors.New("session not found")
	ErrOAuthNotConfigured  = errors.New("oauth verifier not configured")
	ErrInvalidOAuth        = errors.New("invalid oauth identity")
	ErrMFAStoreUnavailable = errors.New("mfa code store unavailable")
)

const (
	defaultIssuer    = "tripapp"
	emailTokenTTL    = 24 * time.Hour
	passwordResetTTL = time.Hour

	jwtBlacklistKeyPrefix = "jwt_blacklist:"
	mfaCodeKeyPrefix      = "auth:mfa:code:"
	mfaCodeTTL            = 10 * time.Minute
	mfaCodeDigits         = 6
)

type DeviceInfo struct {
	DeviceName string
	DeviceType DeviceType
	IPAddress  string
	UserAgent  string
}

type OAuthIdentity struct {
	Provider      string
	ProviderID    string
	Email         string
	EmailVerified bool
	Name          string
	AvatarURL     string
}

type Mailer interface {
	SendVerificationEmail(ctx context.Context, to, name, token string) error
	SendPasswordResetEmail(ctx context.Context, to, name, token string) error
	SendMFACodeEmail(ctx context.Context, to, name, code string) error
}

type OAuthVerifier interface {
	VerifyGoogle(ctx context.Context, idToken string) (*OAuthIdentity, error)
	VerifyGithub(ctx context.Context, code string) (*OAuthIdentity, error)
}

type IService interface {
	Register(ctx context.Context, req RegisterRequest, device DeviceInfo) (*SessionResponse, error)
	Login(ctx context.Context, req LoginRequest, device DeviceInfo) (*SessionResponse, error)
	OAuthLogin(ctx context.Context, identity OAuthIdentity, device DeviceInfo) (*SessionResponse, error)
	OAuthGoogle(ctx context.Context, idToken string, device DeviceInfo) (*SessionResponse, error)
	OAuthGithub(ctx context.Context, code string, device DeviceInfo) (*SessionResponse, error)
	Refresh(ctx context.Context, refreshToken string, device DeviceInfo) (*SessionResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	VerifyEmail(ctx context.Context, token string, device DeviceInfo) (*SessionResponse, error)
	ResendVerification(ctx context.Context, email string) error

	ForgotPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error

	RequestMFAEnableCode(ctx context.Context, userID uuid.UUID) error
	EnableMFA(ctx context.Context, userID uuid.UUID, code string) error
	RequestMFADisableCode(ctx context.Context, userID uuid.UUID) error
	DisableMFA(ctx context.Context, userID uuid.UUID, code string) error

	ListSessions(ctx context.Context, userID uuid.UUID) ([]UserSession, error)
	DeleteSession(ctx context.Context, userID, sessionID uuid.UUID) error

	GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error)
	DeactivateAccount(ctx context.Context, userID uuid.UUID) error
	DeleteAccount(ctx context.Context, userID uuid.UUID) error

	JWKS() *JWKResponse
}

type ServiceConfig struct {
	Repo       IRepository
	Logger     *zap.Logger
	PrivateKey *rsa.PrivateKey
	JWT        config.JWTConfig
	Security   config.SecurityConfig
	Mailer     Mailer
	OAuth      OAuthVerifier
	Redis      *redis.Client
	Issuer     string
	Audit      audit.Writer
	Bus        *event.Bus
	AdminEmail string
}

type Service struct {
	repo       IRepository
	logger     *zap.Logger
	privateKey *rsa.PrivateKey
	publicKey  *rsa.PublicKey
	jwtCfg     config.JWTConfig
	mailer     Mailer
	oauth      OAuthVerifier
	rdb        *redis.Client
	issuer     string
	audience   []string
	opTimeout  time.Duration
	rl         *rateLimiter
	auditW     audit.Writer
	bus        *event.Bus
	adminEmail string
}

func NewService(cfg ServiceConfig) IService {
	issuer := cfg.Issuer
	if issuer == "" {
		issuer = defaultIssuer
	}
	return &Service{
		repo:       cfg.Repo,
		logger:     cfg.Logger.With(zap.String("service", "auth")),
		privateKey: cfg.PrivateKey,
		publicKey:  &cfg.PrivateKey.PublicKey,
		jwtCfg:     cfg.JWT,
		mailer:     cfg.Mailer,
		oauth:      cfg.OAuth,
		rdb:        cfg.Redis,
		issuer:     issuer,
		audience:   cfg.JWT.Audience,
		opTimeout:  cfg.Security.OperationTimeout,
		rl:         newRateLimiter(cfg.Redis, cfg.Security.RateLimit),
		auditW:     cfg.Audit,
		bus:        cfg.Bus,
		adminEmail: normalizeEmail(cfg.AdminEmail),
	}
}

func (s *Service) publish(ctx context.Context, evtType string, userID uuid.UUID, targetID uuid.UUID, payload map[string]any) {
	if s.bus == nil {
		return
	}
	s.bus.Publish(ctx, event.Event{
		Type:      evtType,
		Payload:   payload,
		UserID:    userID,
		TargetID:  targetID,
		Timestamp: time.Now(),
	})
}

func (s *Service) Register(ctx context.Context, req RegisterRequest, device DeviceInfo) (*SessionResponse, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	email := normalizeEmail(req.Email)

	if err := s.checkRate(ctx, rateRegister, rateKey(device.IPAddress, email), nil, device); err != nil {
		return nil, err
	}

	existing, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		s.logger.Warn("register rejected: email already exists",
			zap.String("email", email),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditRegister, audit.Failure, nil, device, "duplicate email")
		return nil, ErrEmailExists
	}

	pwHash, err := hashPassword(req.Password)
	if err != nil {
		return nil, err
	}
	user := &User{
		ID:           uuid.New(),
		Email:        email,
		Name:         strings.TrimSpace(req.Name),
		PasswordHash: &pwHash,
		Status:       UserStatusActive,
	}

	var verificationToken string
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.CreateUser(ctx, user); err != nil {
			return err
		}
		raw, err := s.createEmailVerificationToken(ctx, tx, user.ID)
		if err != nil {
			return err
		}
		verificationToken = raw
		return nil
	}); err != nil {
		return nil, err
	}

	go s.sendVerificationEmail(context.WithoutCancel(ctx), user, verificationToken)

	s.logger.Info("user registered (verification pending)",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
		zap.String("ip", device.IPAddress),
	)
	s.audit(ctx, AuditRegister, audit.Success, &user.ID, device, "verification pending")
	s.publish(ctx, event.EventUserRegistered, user.ID, user.ID, map[string]any{
		"email":      user.Email,
		"name":       user.Name,
		"ip_address": device.IPAddress,
	})
	return &SessionResponse{
		User:                 s.toUserResponse(user),
		RequiresVerification: true,
	}, nil
}

func (s *Service) Login(ctx context.Context, req LoginRequest, device DeviceInfo) (*SessionResponse, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	email := normalizeEmail(req.Email)
	if err := s.checkRate(ctx, rateLogin, rateKey(email, device.IPAddress), nil, device); err != nil {
		return nil, err
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if user == nil || user.PasswordHash == nil {
		s.logger.Warn("login failed: invalid credentials",
			zap.String("email", email),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, audit.Failure, nil, device, "invalid credentials")
		return nil, ErrInvalidCredentials
	}
	if !checkPassword(*user.PasswordHash, req.Password) {
		s.logger.Warn("login failed: wrong password",
			zap.String("user_id", user.ID.String()),
			zap.String("email", email),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, audit.Failure, &user.ID, device, "wrong password")
		return nil, ErrInvalidCredentials
	}
	if user.IsBlocked {
		s.logger.Warn("login rejected: user is blocked",
			zap.String("user_id", user.ID.String()),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, audit.Failure, &user.ID, device, "blocked")
		return nil, ErrUserBlocked
	}
	if user.Status == UserStatusDeleted {
		s.logger.Warn("login rejected: user is deleted",
			zap.String("user_id", user.ID.String()),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, audit.Failure, &user.ID, device, "deleted")
		return nil, ErrUserNotFound
	}

	challenge, err := s.verifyMFAIfEnable(ctx, user, req.MFACode, device)
	if err != nil {
		if errors.Is(err, ErrInvalidMFACode) {
			s.logger.Warn("login failed: invalid mfa code",
				zap.String("user_id", user.ID.String()),
				zap.String("ip", device.IPAddress),
			)
			s.audit(ctx, AuditLoginFailed, audit.Failure, &user.ID, device, "invalid mfa")
		}
		return nil, err
	}
	if challenge {
		s.logger.Info("login: mfa challenge issued",
			zap.String("user_id", user.ID.String()),
			zap.String("ip", device.IPAddress),
		)
		mfaUser := s.toUserResponse(user)
		mfaUser.MFAEnabled = true
		return &SessionResponse{
			User:        mfaUser,
			RequiresMFA: true,
		}, nil
	}

	if err := s.reactiveIfDeactivated(ctx, user); err != nil {
		return nil, err
	}

	resp, _, err := s.createSession(ctx, user, device, "")
	if err != nil {
		return nil, err
	}
	s.logger.Info("user logged in",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
		zap.String("ip", device.IPAddress),
		zap.String("device_type", string(device.DeviceType)),
	)
	s.audit(ctx, AuditLogin, audit.Success, &user.ID, device, "")
	s.publish(ctx, event.EventUserLoggedIn, user.ID, user.ID, map[string]any{
		"email":       user.Email,
		"ip_address":  device.IPAddress,
		"device_type": string(device.DeviceType),
		"user_agent":  device.UserAgent,
	})
	if s.rl != nil {
		s.rl.resetWindow(ctx, rateLogin, rateKey(email, device.IPAddress))
	}
	return resp, nil
}

func (s *Service) OAuthLogin(ctx context.Context, identity OAuthIdentity, device DeviceInfo) (*SessionResponse, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if identity.Provider == "" || identity.ProviderID == "" {
		return nil, ErrInvalidOAuth
	}

	user, created, err := s.resolveOAuthUser(ctx, identity)
	if err != nil {
		return nil, err
	}
	if user.IsBlocked {
		s.logger.Warn("oauth login rejected: user is blocked",
			zap.String("user_id", user.ID.String()),
			zap.String("provider", identity.Provider),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, audit.Failure, &user.ID, device, "oauth blocked")
		return nil, ErrUserBlocked
	}
	if user.Status == UserStatusDeleted {
		s.logger.Warn("oauth login rejected: user is deleted",
			zap.String("user_id", user.ID.String()),
			zap.String("provider", identity.Provider),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, audit.Failure, &user.ID, device, "oauth deleted")
		return nil, ErrUserNotFound
	}
	if err := s.reactiveIfDeactivated(ctx, user); err != nil {
		return nil, err
	}

	resp, _, err := s.createSession(ctx, user, device, identity.Provider)
	if err != nil {
		return nil, err
	}
	if !user.IsVerified {
		resp.RequiresVerification = true
	}
	s.logger.Info("user logged in via oauth",
		zap.String("user_id", user.ID.String()),
		zap.String("provider", identity.Provider),
		zap.String("ip", device.IPAddress),
		zap.String("device_type", string(device.DeviceType)),
	)
	if created {
		s.publish(ctx, event.EventUserRegistered, user.ID, user.ID, map[string]any{
			"email":      user.Email,
			"name":       user.Name,
			"provider":   identity.Provider,
			"ip_address": device.IPAddress,
		})
	}
	s.publish(ctx, event.EventUserLoggedIn, user.ID, user.ID, map[string]any{
		"email":       user.Email,
		"provider":    identity.Provider,
		"ip_address":  device.IPAddress,
		"device_type": string(device.DeviceType),
		"user_agent":  device.UserAgent,
	})
	return resp, nil
}

func (s *Service) OAuthGithub(ctx context.Context, code string, device DeviceInfo) (*SessionResponse, error) {
	if s.oauth == nil {
		return nil, ErrOAuthNotConfigured
	}
	id, err := s.oauth.VerifyGithub(ctx, code)
	if err != nil {
		return nil, err
	}
	return s.OAuthLogin(ctx, *id, device)
}

func (s *Service) OAuthGoogle(ctx context.Context, idToken string, device DeviceInfo) (*SessionResponse, error) {
	if s.oauth == nil {
		return nil, ErrOAuthNotConfigured
	}
	id, err := s.oauth.VerifyGoogle(ctx, idToken)
	if err != nil {
		return nil, err
	}
	return s.OAuthLogin(ctx, *id, device)
}

func (s *Service) Refresh(ctx context.Context, refreshToken string, device DeviceInfo) (*SessionResponse, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	hash := hashToken(refreshToken)
	session, err := s.repo.FindUserSessionByRefreshTokenHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if session == nil {
		s.logger.Warn("token refresh rejected: session not found",
			zap.String("ip", device.IPAddress),
		)
		return nil, ErrInvalidToken
	}

	user, err := s.repo.FindUserByID(ctx, session.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil || user.IsBlocked || user.Status == UserStatusDeleted {
		s.logger.Warn("token refresh rejected: user unavailable",
			zap.String("session_id", session.ID.String()),
			zap.String("user_id", session.UserID.String()),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditRefresh, audit.Failure, &session.UserID, device, "user unavailable")
		return nil, ErrInvalidToken
	}

	won, err := s.repo.RevokeUserSessionIfActive(ctx, session.ID)
	if err != nil {
		return nil, err
	}
	if !won {
		s.logger.Warn("token refresh rejected: session already revoked",
			zap.String("session_id", session.ID.String()),
			zap.String("user_id", session.UserID.String()),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditRefresh, audit.Failure, &user.ID, device, "concurrent rotation")
		return nil, ErrInvalidToken
	}

	resp, _, err := s.createSession(ctx, user, device, "")
	if err != nil {
		return nil, err
	}
	s.logger.Info("session refreshed",
		zap.String("user_id", user.ID.String()),
		zap.String("old_session_id", session.ID.String()),
		zap.String("ip", device.IPAddress),
	)
	s.audit(ctx, AuditRefresh, audit.Success, &user.ID, device, "")
	return resp, nil
}

func (s *Service) Logout(ctx context.Context, refreshToken string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	hash := hashToken(refreshToken)
	session, err := s.repo.FindUserSessionByRefreshTokenHash(ctx, hash)
	if err != nil {
		return err
	}
	if session == nil {
		return nil
	}
	if err := s.repo.RevokeUserSession(ctx, session.ID); err != nil {
		return err
	}
	s.logger.Info("session revoked",
		zap.String("session_id", session.ID.String()),
		zap.String("user_id", session.UserID.String()),
	)
	s.audit(ctx, AuditLogout, audit.Success, &session.UserID, DeviceInfo{}, "")
	s.publish(ctx, event.EventUserLoggedOut, session.UserID, session.UserID, map[string]any{
		"session_id": session.ID.String(),
	})
	return nil
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.repo.RevokeAllSessions(ctx, userID); err != nil {
		return err
	}
	s.logger.Info("all session revoked", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditLogin, audit.Success, &userID, DeviceInfo{}, "all")
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string, device DeviceInfo) (*SessionResponse, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	ev, err := s.repo.FindEmailVerificationByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, err
	}
	if ev == nil {
		s.logger.Warn("email verification failed: invalid or expired token")
		return nil, ErrInvalidToken
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.MarkEmailVerificationUsed(ctx, ev.ID); err != nil {
			return err
		}
		return tx.UpdateUserFields(ctx, ev.UserID, map[string]any{"is_verified": true})
	}); err != nil {
		return nil, err
	}
	user, err := s.repo.FindUserByID(ctx, ev.UserID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	user.IsVerified = true
	resp, _, err := s.createSession(ctx, user, device, "")
	if err != nil {
		return nil, err
	}
	s.logger.Info("email verified",
		zap.String("user_id", ev.UserID.String()),
		zap.String("ip", device.IPAddress),
	)
	s.publish(ctx, event.EventUserVerified, user.ID, user.ID, map[string]any{
		"email":      user.Email,
		"ip_address": device.IPAddress,
	})
	return resp, nil
}

func (s *Service) ResendVerification(ctx context.Context, email string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	user, err := s.repo.FindUserByEmail(ctx, normalizeEmail(email))
	if err != nil {
		return err
	}
	if user == nil || user.IsVerified {
		return nil
	}
	raw, err := s.createEmailVerificationToken(ctx, s.repo, user.ID)
	if err != nil {
		return err
	}
	s.sendVerificationEmail(ctx, user, raw)
	s.logger.Info("resend verification email", zap.String("user_id", user.ID.String()), zap.String("email", email))
	return nil
}

func (s *Service) ForgotPassword(ctx context.Context, email string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	email = normalizeEmail(email)
	if err := s.checkRate(ctx, rateForgot, rateKey(email), nil, DeviceInfo{}); err != nil {
		return err
	}

	user, err := s.repo.FindUserByEmail(ctx, email)
	if err != nil {
		return err
	}
	if user == nil {
		return nil
	}
	raw, hash, err := generateOpaqueToken()
	if err != nil {
		return err
	}
	pr := &PasswordReset{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(passwordResetTTL),
	}

	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.InvalidatePendingPasswordResets(ctx, user.ID); err != nil {
			return err
		}
		return tx.CreatePasswordReset(ctx, pr)
	}); err != nil {
		return err
	}
	if s.mailer != nil {
		return s.mailer.SendPasswordResetEmail(ctx, user.Email, user.Name, raw)
	}
	return nil
}

func (s *Service) ResetPassword(ctx context.Context, token string, newPassword string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	pr, err := s.repo.FindPasswordResetByTokenHash(ctx, hashToken(token))
	if err != nil {
		return err
	}
	if pr == nil {
		s.logger.Warn("password reset failed: invalid or expired token")
		return ErrInvalidToken
	}
	pwHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.MarkPasswordResetUsed(ctx, pr.ID); err != nil {
			return err
		}
		if err := tx.UpdateUserFields(ctx, pr.UserID, map[string]any{"password_hash": pwHash}); err != nil {
			return err
		}
		return tx.RevokeAllSessions(ctx, pr.UserID)
	}); err != nil {
		return err
	}
	s.logger.Info("password_reset completed",
		zap.String("user_id", pr.UserID.String()),
	)
	s.audit(ctx, AuditPasswordReset, audit.Success, &pr.UserID, DeviceInfo{}, "")
	s.publish(ctx, event.EventPasswordReset, pr.UserID, pr.UserID, nil)
	return nil
}

func (s *Service) BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error {
	if s.rdb == nil || jti == "" || ttl < 0 {
		return nil
	}
	return s.rdb.Set(ctx, jwtBlacklistKeyPrefix+jti, "1", ttl).Err()
}

func (s *Service) IsBlacklisted(ctx context.Context, jti string) (bool, error) {
	if s.rdb == nil || jti == "" {
		return false, nil
	}
	n, err := s.rdb.Exists(ctx, jwtBlacklistKeyPrefix+jti).Result()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword string, newPassword string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}

	if user.PasswordHash != nil {
		if oldPassword == "" {
			return ErrPasswordNotSet
		}
		if !checkPassword(*user.PasswordHash, oldPassword) {
			s.logger.Warn("password change failed: wrong current password",
				zap.String("user_id", userID.String()),
			)
			s.audit(ctx, AuditPasswordChange, audit.Failure, &userID, DeviceInfo{}, "wrong password")
			return ErrInvalidCredentials
		}
	}
	pwHash, err := hashPassword(newPassword)
	if err != nil {
		return err
	}
	if err := s.repo.UpdateUserFields(ctx, userID, map[string]any{"password_hash": pwHash}); err != nil {
		return err
	}

	// Boot every other session after a password change
	currentSID := middleware.SessionIDFromContext(ctx)
	if currentSID != uuid.Nil {
		if err := s.repo.RevokeOtherSessions(ctx, userID, currentSID); err != nil {
			s.logger.Warn("revoke other sessions after password change failed",
				zap.Error(err),
				zap.String("user_id", userID.String()),
			)
		}
	} else {
		if err := s.repo.RevokeAllSessions(ctx, userID); err != nil {
			s.logger.Warn("revoke all sessions after password change failed",
				zap.Error(err),
				zap.String("user_id", userID.String()),
			)
		}
	}

	s.logger.Info("password changed", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditPasswordChange, audit.Success, &userID, DeviceInfo{}, "")
	s.publish(ctx, event.EventPasswordChanged, userID, userID, nil)
	return nil
}

func (s *Service) RequestMFAEnableCode(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.checkRate(ctx, rateMFA, userID.String(), &userID, DeviceInfo{}); err != nil {
		return err
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	mfa, err := s.repo.FindMFAConfig(ctx, userID)
	if err != nil {
		return err
	}
	if mfa != nil && mfa.IsEnabled {
		return ErrMFAAlreadyEnabled
	}
	return s.issueMFACode(ctx, user, "enable")
}

func (s *Service) EnableMFA(ctx context.Context, userID uuid.UUID, code string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.checkRate(ctx, rateMFA, userID.String(), &userID, DeviceInfo{}); err != nil {
		return err
	}

	ok, err := s.consumeMFACode(ctx, userID, code)
	if err != nil {
		return err
	}
	if !ok {
		s.logger.Warn("mfa enable failed: invalid code", zap.String("user_id", userID.String()))
		s.audit(ctx, AuditMFAEnabled, audit.Failure, &userID, DeviceInfo{}, "invalid_code")
		return ErrInvalidMFACode
	}
	if err := s.repo.UpsertMFAConfig(ctx, &MFAConfig{
		UserID:    userID,
		Method:    "email",
		IsEnabled: true,
	}); err != nil {
		return err
	}
	s.logger.Info("mfa enabled", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditMFAEnabled, audit.Success, &userID, DeviceInfo{}, "")
	return nil
}

func (s *Service) RequestMFADisableCode(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.checkRate(ctx, rateMFA, userID.String(), &userID, DeviceInfo{}); err != nil {
		return err
	}

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if user == nil {
		return ErrUserNotFound
	}
	mfa, err := s.repo.FindMFAConfig(ctx, userID)
	if err != nil {
		return err
	}
	if mfa == nil || !mfa.IsEnabled {
		return ErrMFANotConfigured
	}
	return s.issueMFACode(ctx, user, "disable")
}

func (s *Service) DisableMFA(ctx context.Context, userID uuid.UUID, code string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.checkRate(ctx, rateMFA, userID.String(), &userID, DeviceInfo{}); err != nil {
		return err
	}

	mfa, err := s.repo.FindMFAConfig(ctx, userID)
	if err != nil {
		return err
	}
	if mfa == nil || !mfa.IsEnabled {
		return ErrMFANotConfigured
	}
	ok, err := s.consumeMFACode(ctx, userID, code)
	if err != nil {
		return err
	}
	if !ok {
		s.logger.Warn("mfa disable failed: invalid code", zap.String("user_id", userID.String()))
		s.audit(ctx, AuditMFADisabled, audit.Failure, &userID, DeviceInfo{}, "invalid_code")
		return ErrInvalidMFACode
	}
	if err := s.repo.DeleteMFAConfig(ctx, userID); err != nil {
		return err
	}
	s.logger.Info("mfa disabled", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditMFADisabled, audit.Success, &userID, DeviceInfo{}, "")
	return nil
}

func (s *Service) ListSessions(ctx context.Context, userID uuid.UUID) ([]UserSession, error) {
	return s.repo.ListSessionByUserID(ctx, userID)
}

func (s *Service) DeleteSession(ctx context.Context, userID uuid.UUID, sessionID uuid.UUID) error {
	sessions, err := s.repo.ListSessionByUserID(ctx, userID)
	if err != nil {
		return err
	}
	for _, session := range sessions {
		if session.ID == sessionID {
			return s.repo.DeleteUserSession(ctx, sessionID)
		}
	}
	return ErrSessionNotFound
}

func (s *Service) GetUser(ctx context.Context, userID uuid.UUID) (*UserResponse, error) {
	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	out := s.toUserResponse(user)
	if mfa, err := s.repo.FindMFAConfig(ctx, userID); err == nil && mfa != nil {
		out.MFAEnabled = mfa.IsEnabled
	}
	return &out, nil
}

func (s *Service) DeactivateAccount(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.DeactivateUser(ctx, userID); err != nil {
			return err
		}
		return tx.RevokeAllSessions(ctx, userID)
	}); err != nil {
		return err
	}
	s.logger.Info("account deactivated", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditAccountDeactivated, audit.Success, &userID, DeviceInfo{}, "")
	s.publish(ctx, event.EventUserDeactivated, userID, userID, nil)
	return nil
}

func (s *Service) DeleteAccount(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.MarkUserDeleted(ctx, userID); err != nil {
			return err
		}
		return tx.RevokeAllSessions(ctx, userID)
	}); err != nil {
		return err
	}
	s.logger.Info("account deletion scheduled", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditAccountDeleted, audit.Success, &userID, DeviceInfo{}, "")
	s.publish(ctx, event.EventUserDeleted, userID, userID, nil)
	return nil
}

func (s *Service) JWKS() *JWKResponse {
	return buildJWKS(s.publicKey)
}

func (s *Service) withTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if s.opTimeout <= 0 {
		return context.WithCancel(ctx)
	}
	return context.WithTimeout(ctx, s.opTimeout)
}

// concatenates non-empty parts with ":" for use as the rate-limit
func rateKey(parts ...string) string {
	clean := make([]string, 0, len(parts))

	for _, p := range parts {
		if p != "" {
			clean = append(clean, p)
		}
	}

	return strings.Join(clean, ":")
}

func (s *Service) toUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:         u.ID.String(),
		Email:      u.Email,
		Name:       u.Name,
		AvatarURL:  u.AvatarURL,
		IsVerified: u.IsVerified,
		IsAdmin:    s.isAdminEmail(u.Email),
		CreatedAt:  u.CreatedAt,
	}
}

func (s *Service) isAdminEmail(email string) bool {
	if s.adminEmail == "" {
		return false
	}
	return strings.EqualFold(strings.TrimSpace(email), s.adminEmail)
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

func generateOpaqueToken() (raw, hashed string, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", "", err
	}
	raw = hex.EncodeToString(b)
	return raw, hashToken(raw), nil
}

func hashPassword(p string) (string, error) {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func checkPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func (s *Service) audit(ctx context.Context, action audit.Action, status audit.Status, userID *uuid.UUID, device DeviceInfo, detail string) {
	if s.auditW == nil {
		return
	}
	log := &audit.Log{
		ID:           uuid.New(),
		ActorUserID:  userID,
		TargetUserID: userID,
		Action:       action,
		Status:       status,
		UserAgent:    device.UserAgent,
		TraceID:      middleware.GetTraceID(ctx),
	}

	if device.IPAddress != "" {
		ip := device.IPAddress
		log.IPAddress = &ip
	}
	if rid := middleware.RequestIDFromContext(ctx); rid != uuid.Nil {
		log.RequestID = &rid
	}
	if detail != "" {
		log.Detail = map[string]any{"message": detail}
	}
	if err := s.auditW.Create(ctx, log); err != nil {
		s.logger.Warn("audit log write failed",
			zap.Error(err),
			zap.String("action", string(action)),
		)
	}
}

func (s *Service) checkRate(ctx context.Context, action rateAction, identifier string, userID *uuid.UUID, device DeviceInfo) error {
	if s.rl == nil {
		return nil
	}
	if err := s.rl.allow(ctx, action, identifier); err != nil {
		if errors.Is(err, ErrRateLimited) {
			s.logger.Warn("rate limit exceeded",
				zap.String("action", string(action)),
				zap.String("identifier", identifier),
			)
			s.audit(ctx, AuditRateLimited, audit.Failure, userID, device, string(action))
			return err
		}
		// fail-open strategy for rate limiter errors
		s.logger.Warn("rate limiter err; allowing request",
			zap.Error(err),
			zap.String("action", string(action)),
		)
	}
	return nil
}

func (s *Service) createEmailVerificationToken(ctx context.Context, repo IRepository, userID uuid.UUID) (string, error) {
	raw, hash, err := generateOpaqueToken()
	if err != nil {
		return "", err
	}
	ev := &EmailVerification{
		ID:        uuid.New(),
		UserID:    userID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(emailTokenTTL),
	}
	if err := repo.CreateEmailVerification(ctx, ev); err != nil {
		return "", err
	}
	return raw, nil
}

func (s *Service) sendVerificationEmail(ctx context.Context, user *User, raw string) {
	if s.mailer == nil {
		return
	}
	if err := s.mailer.SendVerificationEmail(ctx, user.Email, user.Name, raw); err != nil {
		s.logger.Warn("send verification email failed",
			zap.Error(err),
			zap.String("user_id", user.ID.String()),
		)
	}
}

func (s *Service) reactiveIfDeactivated(ctx context.Context, user *User) error {
	if user.Status != UserStatusDeactivated {
		return nil
	}
	if err := s.repo.UpdateUserFields(ctx, user.ID, map[string]any{
		"status":         UserStatusActive,
		"deactivated_at": nil,
	}); err != nil {
		return err
	}
	user.Status = UserStatusActive
	user.DeactivatedAt = nil
	s.logger.Info("account reactivated on login", zap.String("user_id", user.ID.String()))
	return nil
}

func (s *Service) verifyMFAIfEnable(ctx context.Context, user *User, code string, device DeviceInfo) (bool, error) {
	mfa, err := s.repo.FindMFAConfig(ctx, user.ID)
	if err != nil {
		return false, err
	}
	if mfa == nil || !mfa.IsEnabled {
		return false, nil
	}
	if code == "" {
		if err := s.issueMFACode(ctx, user, "login"); err != nil {
			s.logger.Warn("mfa: issue login code failed",
				zap.Error(err),
				zap.String("user_id", user.ID.String()),
				zap.String("ip", device.IPAddress),
			)
		}
		return true, nil
	}
	ok, err := s.consumeMFACode(ctx, user.ID, code)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, ErrInvalidMFACode
	}
	return false, nil
}

func (s *Service) issueMFACode(ctx context.Context, user *User, purpose string) error {
	if s.rdb == nil {
		return ErrMFAStoreUnavailable
	}
	code, err := generateNumericCode(mfaCodeDigits)
	if err != nil {
		return err
	}
	if err := s.rdb.Set(ctx, mfaCodeKey(user.ID), hashToken(code), mfaCodeTTL).Err(); err != nil {
		return err
	}
	if s.mailer != nil {
		if err := s.mailer.SendMFACodeEmail(ctx, user.Email, user.Name, code); err != nil {
			s.logger.Warn("mfa code email send failed",
				zap.Error(err),
				zap.String("user_id", user.ID.String()),
				zap.String("purpose", purpose),
			)
		}
	}
	s.audit(ctx, AuditMFACodeSent, audit.Success, &user.ID, DeviceInfo{}, purpose)
	return nil
}

func (s *Service) consumeMFACode(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	if s.rdb == nil {
		return false, ErrMFAStoreUnavailable
	}
	key := mfaCodeKey(userID)
	stored, err := s.rdb.GetDel(ctx, key).Result()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if subtleCompare(stored, hashToken(code)) {
		return true, nil
	}
	return false, nil
}

func mfaCodeKey(userID uuid.UUID) string {
	return mfaCodeKeyPrefix + userID.String()
}

func generateNumericCode(digits int) (string, error) {
	max := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(digits)), nil)
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%0*d", digits, n.Int64()), nil
}

func subtleCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := 0; i < len(a); i++ {
		v |= a[i] ^ b[i]
	}
	return v == 0
}

func (s *Service) resolveOAuthUser(ctx context.Context, identity OAuthIdentity) (*User, bool, error) {
	provider, err := s.repo.FindProviderByProviderID(ctx, identity.Provider, identity.ProviderID)
	if err != nil {
		return nil, false, err
	}
	if provider != nil {
		user, err := s.repo.FindUserByID(ctx, provider.UserID)
		if err != nil {
			return nil, false, err
		}
		if user == nil {
			return nil, false, ErrUserNotFound
		}
		return user, false, nil
	}

	var (
		user              *User
		created           bool
		verificationToken string
	)
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		u, isNew, err := s.findOrCreateUserByEmailTx(ctx, tx, identity)
		if err != nil {
			return err
		}
		user = u
		created = isNew
		if err := tx.CreateProvider(ctx, &Provider{
			ID:         uuid.New(),
			UserID:     u.ID,
			Provider:   identity.Provider,
			ProviderID: identity.ProviderID,
		}); err != nil {
			return err
		}
		if isNew && !u.IsVerified && u.Email != "" {
			raw, err := s.createEmailVerificationToken(ctx, tx, u.ID)
			if err != nil {
				return err
			}
			verificationToken = raw
		}
		return nil
	})
	if err != nil {
		return nil, false, err
	}
	if verificationToken != "" {
		go s.sendVerificationEmail(context.WithoutCancel(ctx), user, verificationToken)
	}
	return user, created, nil
}

func (s *Service) findOrCreateUserByEmailTx(ctx context.Context, tx IRepository, identity OAuthIdentity) (*User, bool, error) {
	email := normalizeEmail(identity.Email)
	if email != "" {
		existing, err := tx.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, false, err
		}
		if existing != nil {
			return existing, false, nil
		}
	}
	user := &User{
		ID:         uuid.New(),
		Email:      email,
		Name:       strings.TrimSpace(identity.Name),
		AvatarURL:  identity.AvatarURL,
		Status:     UserStatusActive,
		IsVerified: identity.EmailVerified,
	}
	if err := tx.CreateUser(ctx, user); err != nil {
		return nil, false, err
	}
	s.logger.Info("oauth: new user created",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
		zap.String("provider", identity.Provider),
		zap.Bool("email_verified", identity.EmailVerified),
	)
	return user, true, nil
}

func (s *Service) signAccessToken(ctx context.Context, user *User, sessionID uuid.UUID, provider string) (string, time.Time, error) {
	now := time.Now()
	exp := now.Add(s.jwtCfg.AccessTokenTTL)

	roles, err := s.repo.ListUserRoles(ctx, user.ID)
	if err != nil {
		s.logger.Warn("list user role failed", zap.Error(err), zap.String("user_id", user.ID.String()))
		roles = nil
	}
	if roles == nil {
		roles = []string{}
	}

	claims := middleware.Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			ID:        uuid.NewString(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(exp),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    s.issuer,
			Audience:  jwt.ClaimStrings(s.audience),
		},
		Email:     user.Email,
		Roles:     roles,
		Provider:  provider,
		SessionID: sessionID.String(),
	}
	tok := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	tok.Header["kid"] = jwkKeyID
	signed, err := tok.SignedString(s.privateKey)
	return signed, exp, err
}

func (s *Service) createSession(ctx context.Context, user *User, device DeviceInfo, provider string) (*SessionResponse, *UserSession, error) {
	refreshRaw, refreshHash, err := generateOpaqueToken()
	if err != nil {
		return nil, nil, err
	}
	now := time.Now()
	session := &UserSession{
		ID:                uuid.New(),
		UserID:            user.ID,
		DeviceName:        device.DeviceName,
		DeviceType:        device.DeviceType,
		DeviceFingerprint: deviceFingerprint(device),
		IPAddress:         device.IPAddress,
		UserAgent:         device.UserAgent,
		RefreshTokenHash:  refreshHash,
		LastActiveAt:      now,
		ExpiresAt:         now.Add(s.jwtCfg.RefreshTokenTTL),
	}
	if err := s.repo.CreateUserSession(ctx, session); err != nil {
		return nil, nil, err
	}

	accessTok, accessEcp, err := s.signAccessToken(ctx, user, session.ID, provider)
	if err != nil {
		return nil, nil, err
	}

	userResp := s.toUserResponse(user)
	if mfa, err := s.repo.FindMFAConfig(ctx, user.ID); err == nil && mfa != nil {
		userResp.MFAEnabled = mfa.IsEnabled
	}
	return &SessionResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshRaw,
		ExpiresIn:    int64(time.Until(accessEcp).Seconds()),
		TokenType:    "Bearer",
		User:         userResp,
	}, session, nil
}

package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrEmailExists          = errors.New("email already registered")
	ErrInvalidCredentials   = errors.New("invalid email or password")
	ErrUserNotFound         = errors.New("user not found")
	ErrUserBlocked          = errors.New("user is blocked")
	ErrInvalidToken         = errors.New("invalid or expired token")
	ErrInvalidTOTP          = errors.New("invalid totp code")
	ErrTOTPNotConfigured    = errors.New("totp not configured")
	ErrTOTPAlreadyEnabled   = errors.New("totp already enabled")
	ErrPasswordNotSet       = errors.New("password not set")
	ErrSessionNotFound      = errors.New("session not found")
	ErrOAuthNotConfigured   = errors.New("oauth verifier not configured")
	ErrInvalidOAuth         = errors.New("invalid oauth identity")
	ErrTOTPStoreUnavailable = errors.New("totp pending store unavailable")
)

const (
	defaultIssuer    = "tripapp"
	emailTokenTTL    = 24 * time.Hour
	passwordResetTTL = time.Hour

	jwtBlacklistKeyPrefix = "jwt_blacklist:"
	totpPendingKeyPrefix  = "auth:totp:pending:"
	totpUsedKeyPrefix     = "auth:totp:used:"
	totpPendingTTL        = 10 * time.Minute
	totpUsedTTL           = 2 * totpStep * time.Second
)

type DeviceInfo struct {
	DeviceName string
	DeviceType DeviceType
	IPAddress  string
	UserAgent  string
}

type OAuthIdentity struct {
	Provider   string
	ProviderID string
	Email      string
	Name       string
	AvatarURL  string
}

type Mailer interface {
	SendVerificationEmail(ctx context.Context, to, name, token string) error
	SendPasswordResetEmail(ctx context.Context, to, name, token string) error
}

type OAuthVerifier interface {
	VerifyGoogle(ctx context.Context, idToken string) (*OAuthIdentity, error)
	VerifyGithub(ctx context.Context, code string) (*OAuthIdentity, error)
	VerifyFacebook(ctx context.Context, accessToken string) (*OAuthIdentity, error)
}

type IService interface {
	Register(ctx context.Context, req RegisterRequest, device DeviceInfo) (*SessionResponse, error)
	Login(ctx context.Context, req LoginRequest, device DeviceInfo) (*SessionResponse, error)
	OAuthLogin(ctx context.Context, identity OAuthIdentity, device DeviceInfo) (*SessionResponse, error)
	OAuthGoogle(ctx context.Context, idToken string, device DeviceInfo) (*SessionResponse, error)
	OAuthGithub(ctx context.Context, code string, device DeviceInfo) (*SessionResponse, error)
	OAuthFacebook(ctx context.Context, accessToken string, device DeviceInfo) (*SessionResponse, error)
	Refresh(ctx context.Context, refreshToken string, device DeviceInfo) (*SessionResponse, error)
	Logout(ctx context.Context, refreshToken string) error
	LogoutAll(ctx context.Context, userID uuid.UUID) error
	BlacklistJTI(ctx context.Context, jti string, ttl time.Duration) error
	IsBlacklisted(ctx context.Context, jti string) (bool, error)

	VerifyEmail(ctx context.Context, token string) error
	ResendVerification(ctx context.Context, email string) error

	ForgetPassword(ctx context.Context, email string) error
	ResetPassword(ctx context.Context, token, newPassword string) error
	ChangePassword(ctx context.Context, userID uuid.UUID, oldPassword, newPassword string) error

	SetupTOTP(ctx context.Context, userID uuid.UUID) (*TOTPSetupResponse, error)
	EnableTOTP(ctx context.Context, userID uuid.UUID, code string) error
	DisableTOTP(ctx context.Context, userID uuid.UUID, code string) error

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
	totpKey    string
	opTimeout  time.Duration
	rl         *rateLimiter
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
		totpKey:    cfg.Security.TOTPEncryptionKey,
		opTimeout:  cfg.Security.OperationTimeout,
		rl:         newRateLimiter(cfg.Redis, cfg.Security.RateLimit),
	}
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
		s.audit(ctx, AuditRegister, AuditFailure, nil, device, "duplicate email")
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

	s.sendVerificationEmail(ctx, user, verificationToken)

	resp, _, err := s.createSession(ctx, user, device, "")
	if err != nil {
		return nil, err
	}
	s.logger.Info("user registered",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
		zap.String("ip", device.IPAddress),
	)
	s.audit(ctx, AuditRegister, AuditSuccess, &user.ID, device, "")
	return resp, nil
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
		s.audit(ctx, AuditLoginFailed, AuditFailure, nil, device, "invalid credentials")
		return nil, ErrInvalidCredentials
	}
	if !checkPassword(*user.PasswordHash, req.Password) {
		s.logger.Warn("login failed: wrong password",
			zap.String("user_id", user.ID.String()),
			zap.String("email", email),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, AuditFailure, &user.ID, device, "wrong password")
		return nil, ErrInvalidCredentials
	}
	if user.IsBlocked {
		s.logger.Warn("login rejected: user is blocked",
			zap.String("user_id", user.ID.String()),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, AuditFailure, &user.ID, device, "blocked")
		return nil, ErrUserBlocked
	}
	if user.Status == UserStatusDeleted {
		s.logger.Warn("login rejected: user is deleted",
			zap.String("user_id", user.ID.String()),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, AuditFailure, &user.ID, device, "deleted")
		return nil, ErrUserNotFound
	}

	challenge, err := s.verifyMFAIfEnable(ctx, user.ID, req.TOTPCode)
	if err != nil {
		if errors.Is(err, ErrInvalidTOTP) {
			s.logger.Warn("login failed: invalid totp code",
				zap.String("user_id", user.ID.String()),
				zap.String("ip", device.IPAddress),
			)
			s.audit(ctx, AuditLoginFailed, AuditFailure, &user.ID, device, "invalid totp")
		}
		return nil, err
	}
	if challenge {
		s.logger.Info("login: mfa challenge issued",
			zap.String("user_id", user.ID.String()),
			zap.String("ip", device.IPAddress),
		)
		return &SessionResponse{RequiresTOTP: true}, nil
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
	s.audit(ctx, AuditLogin, AuditSuccess, &user.ID, device, "")
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

	user, err := s.resolveOAuthUser(ctx, identity)
	if err != nil {
		return nil, err
	}
	if user.IsBlocked {
		s.logger.Warn("oauth login rejected: user is blocked",
			zap.String("user_id", user.ID.String()),
			zap.String("provider", identity.Provider),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, AuditFailure, &user.ID, device, "oauth blocked")
		return nil, ErrUserBlocked
	}
	if user.Status == UserStatusDeleted {
		s.logger.Warn("oauth login rejected: user is deleted",
			zap.String("user_id", user.ID.String()),
			zap.String("provider", identity.Provider),
			zap.String("ip", device.IPAddress),
		)
		s.audit(ctx, AuditLoginFailed, AuditFailure, &user.ID, device, "oauth deleted")
		return nil, ErrUserNotFound
	}
	if err := s.reactiveIfDeactivated(ctx, user); err != nil {
		return nil, err
	}

	resp, _, err := s.createSession(ctx, user, device, identity.Provider)
	if err != nil {
		return nil, err
	}
	s.logger.Info("user logged in via oauth",
		zap.String("user_id", user.ID.String()),
		zap.String("provider", identity.Provider),
		zap.String("ip", device.IPAddress),
		zap.String("device_type", string(device.DeviceType)),
	)
	return resp, nil
}

func (s *Service) OAuthFacebook(ctx context.Context, accessToken string, device DeviceInfo) (*SessionResponse, error) {
	if s.oauth == nil {
		return nil, ErrOAuthNotConfigured
	}
	id, err := s.oauth.VerifyFacebook(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	return s.OAuthLogin(ctx, *id, device)
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
		s.audit(ctx, AuditRefresh, AuditFailure, &session.UserID, device, "user unavailable")
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
		s.audit(ctx, AuditRefresh, AuditFailure, &user.ID, device, "concurrent rotation")
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
	s.audit(ctx, AuditRefresh, AuditSuccess, &user.ID, device, "")
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
	s.audit(ctx, AuditLogout, AuditSuccess, &session.UserID, DeviceInfo{}, "")
	return nil
}

func (s *Service) LogoutAll(ctx context.Context, userID uuid.UUID) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.repo.RevokeAllSessions(ctx, userID); err != nil {
		return err
	}
	s.logger.Info("all session revoked", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditLogin, AuditSuccess, &userID, DeviceInfo{}, "all")
	return nil
}

func (s *Service) VerifyEmail(ctx context.Context, token string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	ev, err := s.repo.FindEmailVerificationByTokenHash(ctx, hashToken(token))
	if err != nil {
		return err
	}
	if ev == nil {
		s.logger.Warn("email verification failed: invalid or expired token")
		return ErrInvalidToken
	}
	if err := s.repo.WithTx(ctx, func(tx IRepository) error {
		if err := tx.MarkEmailVerificationUsed(ctx, ev.ID); err != nil {
			return err
		}
		return tx.UpdateUserFields(ctx, ev.UserID, map[string]any{"is_verified": true})
	}); err != nil {
		return err
	}
	s.logger.Info("email verified", zap.String("user_id", ev.UserID.String()))
	return nil
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

func (s *Service) ForgetPassword(ctx context.Context, email string) error {
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
	s.audit(ctx, AuditPasswordReset, AuditSuccess, &pr.UserID, DeviceInfo{}, "")
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
	if user.PasswordHash == nil {
		return ErrPasswordNotSet
	}
	if !checkPassword(*user.PasswordHash, oldPassword) {
		s.logger.Warn("password change failed: wrong current password",
			zap.String("user_id", userID.String()),
		)
		s.audit(ctx, AuditPasswordChange, AuditFailure, &userID, DeviceInfo{}, "wrong password")
		return ErrInvalidCredentials
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
	s.audit(ctx, AuditPasswordChange, AuditSuccess, &userID, DeviceInfo{}, "")
	return nil
}

func (s *Service) SetupTOTP(ctx context.Context, userID uuid.UUID) (*TOTPSetupResponse, error) {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	user, err := s.repo.FindUserByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	mfa, err := s.repo.FindMFAConfig(ctx, userID)
	if err != nil {
		return nil, err
	}
	if mfa != nil && mfa.IsEnabled {
		return nil, ErrTOTPAlreadyEnabled
	}
	secret, err := generateTOTPSecret()
	if err != nil {
		return nil, err
	}
	stored, err := s.protectTOTPSecret(secret)
	if err != nil {
		return nil, err
	}
	if err := s.storePendingTOTP(ctx, userID, stored); err != nil {
		return nil, err
	}
	return &TOTPSetupResponse{
		Secret:          secret,
		ProvisioningURL: totpProvisioningURL(secret, user.Email, s.issuer),
	}, nil
}

func (s *Service) EnableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.checkRate(ctx, rateTOTP, userID.String(), &userID, DeviceInfo{}); err != nil {
		return err
	}

	stored, err := s.loadPendingTOTP(ctx, userID)
	if err != nil {
		return err
	}
	if stored == "" {
		return ErrTOTPNotConfigured
	}
	secret, err := s.unprotectTOTPSecret(stored)
	if err != nil {
		return err
	}
	counter, ok := verifyTOTP(secret, code)
	if !ok {
		s.logger.Warn("totp enable failed: invalid code", zap.String("user_id", userID.String()))
		s.audit(ctx, AuditTOTPEnabled, AuditFailure, &userID, DeviceInfo{}, "invalid_code")
		return ErrInvalidTOTP
	}
	if err := s.consumeTOTPStep(ctx, userID, counter); err != nil {
		s.audit(ctx, AuditTOTPEnabled, AuditFailure, &userID, DeviceInfo{}, "replay")
		return err
	}
	if err := s.repo.UpsertMFAConfig(ctx, &MFAConfig{
		UserID:     userID,
		TOTPSecret: stored,
		IsEnabled:  true,
	}); err != nil {
		return err
	}
	s.clearPendingTOTP(ctx, userID)
	s.logger.Info("totp enabled", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditTOTPEnabled, AuditSuccess, &userID, DeviceInfo{}, "")
	return nil
}

func (s *Service) DisableTOTP(ctx context.Context, userID uuid.UUID, code string) error {
	ctx, cancel := s.withTimeout(ctx)
	defer cancel()

	if err := s.checkRate(ctx, rateTOTP, userID.String(), &userID, DeviceInfo{}); err != nil {
		return err
	}

	mfa, err := s.repo.FindMFAConfig(ctx, userID)
	if err != nil {
		return err
	}
	if mfa == nil || !mfa.IsEnabled {
		return ErrTOTPNotConfigured
	}
	secret, err := s.unprotectTOTPSecret(mfa.TOTPSecret)
	if err != nil {
		return err
	}
	counter, ok := verifyTOTP(secret, code)
	if !ok {
		s.logger.Warn("totp disable failed: invalid code", zap.String("user_id", userID.String()))
		s.audit(ctx, AuditTOTPDisabled, AuditFailure, &userID, DeviceInfo{}, "invalid_code")
		return ErrInvalidTOTP
	}
	if err := s.consumeTOTPStep(ctx, userID, counter); err != nil {
		s.audit(ctx, AuditTOTPDisabled, AuditFailure, &userID, DeviceInfo{}, "replay")
		return err
	}
	if err := s.repo.DeleteMFAConfig(ctx, userID); err != nil {
		return err
	}
	s.clearPendingTOTP(ctx, userID)
	s.logger.Info("totp disabled", zap.String("user_id", userID.String()))
	s.audit(ctx, AuditTOTPDisabled, AuditSuccess, &userID, DeviceInfo{}, "")
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
	out := toUserResponse(user)
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
	s.audit(ctx, AuditAccountDeactivated, AuditSuccess, &userID, DeviceInfo{}, "")
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
	s.audit(ctx, AuditAccountDeleted, AuditSuccess, &userID, DeviceInfo{}, "")
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

func toUserResponse(u *User) UserResponse {
	return UserResponse{
		ID:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		AvatarURL: u.AvatarURL,
		CreatedAt: u.CreatedAt,
	}
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

func (s *Service) protectTOTPSecret(plain string) (string, error) {
	if s.totpKey == "" {
		return plain, nil
	}
	return encryptSecret(plain, s.totpKey)
}

func (s *Service) unprotectTOTPSecret(stored string) (string, error) {
	if s.totpKey == "" {
		return stored, nil
	}
	return decryptSecret(stored, s.totpKey)
}

func (s *Service) audit(ctx context.Context, action AuditAction, status AuditStatus, userID *uuid.UUID, device DeviceInfo, detail string) {
	if s.repo == nil {
		return
	}
	log := &AuditLog{
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
	if err := s.repo.CreateAuditLog(ctx, log); err != nil {
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
			s.audit(ctx, AuditRateLimited, AuditFailure, userID, device, string(action))
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

func (s *Service) verifyMFAIfEnable(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	mfa, err := s.repo.FindMFAConfig(ctx, userID)
	if err != nil {
		return false, err
	}
	if mfa == nil || !mfa.IsEnabled {
		return false, nil
	}
	if code == "" {
		return true, nil
	}
	secret, err := s.unprotectTOTPSecret(mfa.TOTPSecret)
	if err != nil {
		return false, nil
	}
	counter, ok := verifyTOTP(secret, code)
	if !ok {
		return false, ErrInvalidTOTP
	}
	if err := s.consumeTOTPStep(ctx, userID, counter); err != nil {
		return false, err
	}
	return false, nil
}

func (s *Service) storePendingTOTP(ctx context.Context, userID uuid.UUID, ciphertext string) error {
	if s.rdb == nil {
		return ErrTOTPStoreUnavailable
	}
	return s.rdb.Set(ctx, totpPendingKeyPrefix+userID.String(), ciphertext, totpPendingTTL).Err()
}

func (s *Service) loadPendingTOTP(ctx context.Context, userID uuid.UUID) (string, error) {
	if s.rdb == nil {
		return "", ErrTOTPStoreUnavailable
	}
	v, err := s.rdb.Get(ctx, totpPendingKeyPrefix+userID.String()).Result()
	if errors.Is(err, redis.Nil) {
		return "", nil
	}
	return v, err
}

func (s *Service) clearPendingTOTP(ctx context.Context, userID uuid.UUID) {
	if s.rdb == nil {
		return
	}
	if err := s.rdb.Del(ctx, totpPendingKeyPrefix+userID.String()).Err(); err != nil {
		s.logger.Warn("totp clear pending failed", zap.Error(err), zap.String("user_id", userID.String()))
	}
}

func (s *Service) consumeTOTPStep(ctx context.Context, userID uuid.UUID, counter int64) error {
	if s.rdb == nil {
		return nil
	}
	key := totpUsedKeyPrefix + userID.String() + ":" + strconv.FormatInt(counter, 10)
	ok, err := s.rdb.SetNX(ctx, key, "1", totpUsedTTL).Result()
	if err != nil {
		s.logger.Warn("totp replay check failed; allowing", zap.Error(err))
		return nil
	}
	if !ok {
		s.logger.Warn("totp replay detected",
			zap.String("user_id", userID.String()),
			zap.Int64("counter", counter),
		)
		return ErrInvalidTOTP
	}
	return nil
}

func (s *Service) resolveOAuthUser(ctx context.Context, identity OAuthIdentity) (*User, error) {
	provider, err := s.repo.FindProviderByProviderID(ctx, identity.Provider, identity.ProviderID)
	if err != nil {
		return nil, err
	}
	if provider != nil {
		user, err := s.repo.FindUserByID(ctx, provider.UserID)
		if err != nil {
			return nil, err
		}
		if user == nil {
			return nil, ErrUserNotFound
		}
		return user, nil
	}

	var user *User
	err = s.repo.WithTx(ctx, func(tx IRepository) error {
		u, err := s.findOrCreateUserByEmailTx(ctx, tx, identity)
		if err != nil {
			return err
		}
		user = u
		return tx.CreateProvider(ctx, &Provider{
			ID:         uuid.New(),
			UserID:     u.ID,
			Provider:   identity.Provider,
			ProviderID: identity.ProviderID,
		})
	})
	if err != nil {
		return nil, err
	}
	return user, nil
}

func (s *Service) findOrCreateUserByEmailTx(ctx context.Context, tx IRepository, identity OAuthIdentity) (*User, error) {
	email := normalizeEmail(identity.Email)
	if email != "" {
		existing, err := tx.FindUserByEmail(ctx, email)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			return existing, nil
		}
	}
	user := &User{
		ID:         uuid.New(),
		Email:      email,
		Name:       strings.TrimSpace(identity.Name),
		AvatarURL:  identity.AvatarURL,
		Status:     UserStatusActive,
		IsVerified: true,
	}
	if err := tx.CreateUser(ctx, user); err != nil {
		return nil, err
	}
	s.logger.Info("oauth: new user created",
		zap.String("user_id", user.ID.String()),
		zap.String("email", email),
		zap.String("provider", identity.Provider),
	)
	return user, nil
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

	return &SessionResponse{
		AccessToken:  accessTok,
		RefreshToken: refreshRaw,
		ExpiresIn:    int64(time.Until(accessEcp).Seconds()),
		TokenType:    "Bearer",
		User:         toUserResponse(user),
	}, session, nil
}

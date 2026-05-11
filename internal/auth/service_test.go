package auth_test

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"errors"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

// ---- package-level test fixtures ----

var (
	testPrivKey *rsa.PrivateKey
	testJWTCfg  = config.JWTConfig{
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 30 * 24 * time.Hour,
	}
	ctx = context.Background()
)

func init() {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		panic("generate RSA key: " + err.Error())
	}
	testPrivKey = key
}

// ---- mock repository ----

type repoMock struct {
	withTx                            func(context.Context, func(auth.IRepository) error) error
	createUser                        func(context.Context, *auth.User) error
	findUserByEmail                   func(context.Context, string) (*auth.User, error)
	findUserByID                      func(context.Context, uuid.UUID) (*auth.User, error)
	findUserByIDs                     func(context.Context, []uuid.UUID) ([]auth.User, error)
	updateUserFields                  func(context.Context, uuid.UUID, map[string]any) error
	deactivateUser                    func(context.Context, uuid.UUID) error
	markUserDeleted                   func(context.Context, uuid.UUID) error
	blockUser                         func(context.Context, uuid.UUID) error
	createProvider                    func(context.Context, *auth.Provider) error
	findProviderByProviderID          func(context.Context, string, string) (*auth.Provider, error)
	createEmailVerification           func(context.Context, *auth.EmailVerification) error
	findEmailVerificationByTokenHash  func(context.Context, string) (*auth.EmailVerification, error)
	markEmailVerificationUsed         func(context.Context, uuid.UUID) error
	createPasswordReset               func(context.Context, *auth.PasswordReset) error
	findPasswordResetByTokenHash      func(context.Context, string) (*auth.PasswordReset, error)
	markPasswordResetUsed             func(context.Context, uuid.UUID) error
	invalidatePendingPasswordResets   func(context.Context, uuid.UUID) error
	upsertMFAConfig                   func(context.Context, *auth.MFAConfig) error
	findMFAConfig                     func(context.Context, uuid.UUID) (*auth.MFAConfig, error)
	createUserSession                 func(context.Context, *auth.UserSession) error
	findUserSessionByRefreshTokenHash func(context.Context, string) (*auth.UserSession, error)
	revokeUserSession                 func(context.Context, uuid.UUID) error
	revokeUserSessionIfActive         func(context.Context, uuid.UUID) (bool, error)
	listSessionByUserID               func(context.Context, uuid.UUID) ([]auth.UserSession, error)
	deleteUserSession                 func(context.Context, uuid.UUID) error
	revokeAllSessions                 func(context.Context, uuid.UUID) error
	revokeOtherSessions               func(context.Context, uuid.UUID, uuid.UUID) error
	listUserRoles                     func(context.Context, uuid.UUID) ([]string, error)
	createAuditLog                    func(context.Context, *auth.AuditLog) error
}

func (m *repoMock) WithTx(c context.Context, fn func(auth.IRepository) error) error {
	if m.withTx != nil {
		return m.withTx(c, fn)
	}
	return fn(m)
}

func (m *repoMock) CreateUser(c context.Context, u *auth.User) error {
	if m.createUser != nil {
		return m.createUser(c, u)
	}
	return nil
}
func (m *repoMock) FindUserByEmail(c context.Context, e string) (*auth.User, error) {
	if m.findUserByEmail != nil {
		return m.findUserByEmail(c, e)
	}
	return nil, nil
}
func (m *repoMock) FindUserByID(c context.Context, id uuid.UUID) (*auth.User, error) {
	if m.findUserByID != nil {
		return m.findUserByID(c, id)
	}
	return nil, nil
}
func (m *repoMock) FindUserByIDs(c context.Context, ids []uuid.UUID) ([]auth.User, error) {
	if m.findUserByIDs != nil {
		return m.findUserByIDs(c, ids)
	}
	return nil, nil
}
func (m *repoMock) UpdateUserFields(c context.Context, id uuid.UUID, u map[string]any) error {
	if m.updateUserFields != nil {
		return m.updateUserFields(c, id, u)
	}
	return nil
}
func (m *repoMock) DeactivateUser(c context.Context, id uuid.UUID) error {
	if m.deactivateUser != nil {
		return m.deactivateUser(c, id)
	}
	return nil
}
func (m *repoMock) MarkUserDeleted(c context.Context, id uuid.UUID) error {
	if m.markUserDeleted != nil {
		return m.markUserDeleted(c, id)
	}
	return nil
}
func (m *repoMock) BlockUser(c context.Context, id uuid.UUID) error {
	if m.blockUser != nil {
		return m.blockUser(c, id)
	}
	return nil
}
func (m *repoMock) CreateProvider(c context.Context, p *auth.Provider) error {
	if m.createProvider != nil {
		return m.createProvider(c, p)
	}
	return nil
}
func (m *repoMock) FindProviderByProviderID(c context.Context, prov, id string) (*auth.Provider, error) {
	if m.findProviderByProviderID != nil {
		return m.findProviderByProviderID(c, prov, id)
	}
	return nil, nil
}
func (m *repoMock) CreateEmailVerification(c context.Context, ev *auth.EmailVerification) error {
	if m.createEmailVerification != nil {
		return m.createEmailVerification(c, ev)
	}
	return nil
}
func (m *repoMock) FindEmailVerificationByTokenHash(c context.Context, h string) (*auth.EmailVerification, error) {
	if m.findEmailVerificationByTokenHash != nil {
		return m.findEmailVerificationByTokenHash(c, h)
	}
	return nil, nil
}
func (m *repoMock) MarkEmailVerificationUsed(c context.Context, id uuid.UUID) error {
	if m.markEmailVerificationUsed != nil {
		return m.markEmailVerificationUsed(c, id)
	}
	return nil
}
func (m *repoMock) CreatePasswordReset(c context.Context, pr *auth.PasswordReset) error {
	if m.createPasswordReset != nil {
		return m.createPasswordReset(c, pr)
	}
	return nil
}
func (m *repoMock) FindPasswordResetByTokenHash(c context.Context, h string) (*auth.PasswordReset, error) {
	if m.findPasswordResetByTokenHash != nil {
		return m.findPasswordResetByTokenHash(c, h)
	}
	return nil, nil
}
func (m *repoMock) MarkPasswordResetUsed(c context.Context, id uuid.UUID) error {
	if m.markPasswordResetUsed != nil {
		return m.markPasswordResetUsed(c, id)
	}
	return nil
}
func (m *repoMock) InvalidatePendingPasswordResets(c context.Context, id uuid.UUID) error {
	if m.invalidatePendingPasswordResets != nil {
		return m.invalidatePendingPasswordResets(c, id)
	}
	return nil
}
func (m *repoMock) UpsertMFAConfig(c context.Context, cfg *auth.MFAConfig) error {
	if m.upsertMFAConfig != nil {
		return m.upsertMFAConfig(c, cfg)
	}
	return nil
}
func (m *repoMock) FindMFAConfig(c context.Context, id uuid.UUID) (*auth.MFAConfig, error) {
	if m.findMFAConfig != nil {
		return m.findMFAConfig(c, id)
	}
	return nil, nil
}
func (m *repoMock) CreateUserSession(c context.Context, s *auth.UserSession) error {
	if m.createUserSession != nil {
		return m.createUserSession(c, s)
	}
	return nil
}
func (m *repoMock) FindUserSessionByRefreshTokenHash(c context.Context, h string) (*auth.UserSession, error) {
	if m.findUserSessionByRefreshTokenHash != nil {
		return m.findUserSessionByRefreshTokenHash(c, h)
	}
	return nil, nil
}
func (m *repoMock) RevokeUserSession(c context.Context, id uuid.UUID) error {
	if m.revokeUserSession != nil {
		return m.revokeUserSession(c, id)
	}
	return nil
}
func (m *repoMock) RevokeUserSessionIfActive(c context.Context, id uuid.UUID) (bool, error) {
	if m.revokeUserSessionIfActive != nil {
		return m.revokeUserSessionIfActive(c, id)
	}
	return true, nil
}
func (m *repoMock) ListSessionByUserID(c context.Context, id uuid.UUID) ([]auth.UserSession, error) {
	if m.listSessionByUserID != nil {
		return m.listSessionByUserID(c, id)
	}
	return nil, nil
}
func (m *repoMock) DeleteUserSession(c context.Context, id uuid.UUID) error {
	if m.deleteUserSession != nil {
		return m.deleteUserSession(c, id)
	}
	return nil
}
func (m *repoMock) RevokeAllSessions(c context.Context, id uuid.UUID) error {
	if m.revokeAllSessions != nil {
		return m.revokeAllSessions(c, id)
	}
	return nil
}
func (m *repoMock) RevokeOtherSessions(c context.Context, userID, exceptID uuid.UUID) error {
	if m.revokeOtherSessions != nil {
		return m.revokeOtherSessions(c, userID, exceptID)
	}
	return nil
}
func (m *repoMock) ListUserRoles(c context.Context, id uuid.UUID) ([]string, error) {
	if m.listUserRoles != nil {
		return m.listUserRoles(c, id)
	}
	return nil, nil
}
func (m *repoMock) CreateAuditLog(c context.Context, log *auth.AuditLog) error {
	if m.createAuditLog != nil {
		return m.createAuditLog(c, log)
	}
	return nil
}

// ---- mock mailer ----

type mockMailer struct {
	verificationCalled bool
	resetCalled        bool
	err                error
}

func (m *mockMailer) SendVerificationEmail(_ context.Context, _, _, _ string) error {
	m.verificationCalled = true
	return m.err
}
func (m *mockMailer) SendPasswordResetEmail(_ context.Context, _, _, _ string) error {
	m.resetCalled = true
	return m.err
}

// ---- mock oauth verifier ----

type mockOAuth struct {
	identity *auth.OAuthIdentity
	err      error
}

func (m *mockOAuth) VerifyGoogle(_ context.Context, _ string) (*auth.OAuthIdentity, error) {
	return m.identity, m.err
}
func (m *mockOAuth) VerifyGithub(_ context.Context, _ string) (*auth.OAuthIdentity, error) {
	return m.identity, m.err
}
func (m *mockOAuth) VerifyFacebook(_ context.Context, _ string) (*auth.OAuthIdentity, error) {
	return m.identity, m.err
}

// ---- helpers ----

func newSvc(repo auth.IRepository, opts ...func(*auth.ServiceConfig)) auth.IService {
	cfg := auth.ServiceConfig{
		Repo:       repo,
		Logger:     zap.NewNop(),
		PrivateKey: testPrivKey,
		JWT:        testJWTCfg,
	}
	for _, o := range opts {
		o(&cfg)
	}
	return auth.NewService(cfg)
}

func mustHashPwd(p string) string {
	h, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.MinCost)
	if err != nil {
		panic(err)
	}
	return string(h)
}

func newActiveUser() *auth.User {
	pw := mustHashPwd("password123")
	return &auth.User{
		ID:           uuid.New(),
		Email:        "alice@example.com",
		Name:         "Alice",
		PasswordHash: &pw,
		Status:       auth.UserStatusActive,
		IsVerified:   true,
		CreatedAt:    time.Now(),
	}
}

func assertValidSession(t *testing.T, resp *auth.SessionResponse) {
	t.Helper()
	require.NotNil(t, resp)
	assert.NotEmpty(t, resp.AccessToken)
	assert.NotEmpty(t, resp.RefreshToken)
	assert.Positive(t, resp.ExpiresIn)
	assert.Equal(t, "Bearer", resp.TokenType)

	// verify the JWT is correctly signed
	_, err := jwt.Parse(resp.AccessToken, func(tok *jwt.Token) (any, error) {
		return &testPrivKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	assert.NoError(t, err, "access token should be a valid RS256 JWT")
}

var noDevice = auth.DeviceInfo{}

// ---- Register ----

func TestRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := &repoMock{}
		svc := newSvc(repo)

		resp, err := svc.Register(ctx, auth.RegisterRequest{
			Email:    "bob@example.com",
			Password: "password123",
			Name:     "Bob",
		}, noDevice)

		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.Equal(t, "bob@example.com", resp.User.Email)
	})

	t.Run("duplicate_email", func(t *testing.T) {
		existing := newActiveUser()
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) {
				return existing, nil
			},
		}
		_, err := newSvc(repo).Register(ctx, auth.RegisterRequest{
			Email:    existing.Email,
			Password: "password123",
			Name:     "Bob",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrEmailExists)
	})

	t.Run("repo_error_propagates", func(t *testing.T) {
		repoErr := errors.New("db down")
		repo := &repoMock{
			createUser: func(_ context.Context, _ *auth.User) error { return repoErr },
		}
		_, err := newSvc(repo).Register(ctx, auth.RegisterRequest{
			Email:    "new@example.com",
			Password: "password123",
			Name:     "New",
		}, noDevice)
		require.ErrorIs(t, err, repoErr)
	})

	t.Run("mailer_failure_is_non_fatal", func(t *testing.T) {
		mailer := &mockMailer{err: errors.New("smtp down")}
		repo := &repoMock{}
		svc := newSvc(repo, func(c *auth.ServiceConfig) { c.Mailer = mailer })

		resp, err := svc.Register(ctx, auth.RegisterRequest{
			Email:    "bob@example.com",
			Password: "password123",
			Name:     "Bob",
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.True(t, mailer.verificationCalled)
	})
}

// ---- Login ----

func TestLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		resp, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email:    user.Email,
			Password: "password123",
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.Equal(t, user.ID.String(), resp.User.ID)
	})

	t.Run("unknown_email", func(t *testing.T) {
		repo := &repoMock{}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: "nobody@example.com", Password: "password123",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("oauth_only_user_no_password", func(t *testing.T) {
		user := newActiveUser()
		user.PasswordHash = nil
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("wrong_password", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "wrong-password",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("blocked_user", func(t *testing.T) {
		user := newActiveUser()
		user.IsBlocked = true
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrUserBlocked)
	})

	t.Run("deleted_user", func(t *testing.T) {
		user := newActiveUser()
		user.Status = auth.UserStatusDeleted
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrUserNotFound)
	})

	t.Run("mfa_required_when_no_code", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: user.ID, TOTPSecret: "SECRET", IsEnabled: true}, nil
			},
		}
		resp, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.NoError(t, err)
		assert.True(t, resp.RequiresTOTP)
		assert.Empty(t, resp.AccessToken)
	})

	t.Run("mfa_wrong_code", func(t *testing.T) {
		user := newActiveUser()
		secret := "JBSWY3DPEHPK3PXP"
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: user.ID, TOTPSecret: secret, IsEnabled: true}, nil
			},
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123", TOTPCode: "000000",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidTOTP)
	})

	t.Run("mfa_valid_code", func(t *testing.T) {
		user := newActiveUser()
		secret := "JBSWY3DPEHPK3PXP"
		code, err := auth.TotpNow(secret)
		require.NoError(t, err)
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: user.ID, TOTPSecret: secret, IsEnabled: true}, nil
			},
		}
		resp, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123", TOTPCode: code,
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
	})

	t.Run("deactivated_user_reactivates_on_login", func(t *testing.T) {
		user := newActiveUser()
		user.Status = auth.UserStatusDeactivated
		reactivated := false
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			updateUserFields: func(_ context.Context, _ uuid.UUID, updates map[string]any) error {
				if updates["status"] == auth.UserStatusActive {
					reactivated = true
				}
				return nil
			},
		}
		resp, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.True(t, reactivated)
	})
}

// ---- OAuthLogin ----

func TestOAuthLogin(t *testing.T) {
	t.Run("empty_provider_rejected", func(t *testing.T) {
		repo := &repoMock{}
		_, err := newSvc(repo).OAuthLogin(ctx, auth.OAuthIdentity{}, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidOAuth)
	})

	t.Run("existing_provider_link_logs_in", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findProviderByProviderID: func(_ context.Context, _, _ string) (*auth.Provider, error) {
				return &auth.Provider{ID: uuid.New(), UserID: user.ID, Provider: "google", ProviderID: "gid-1"}, nil
			},
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
		}
		resp, err := newSvc(repo).OAuthLogin(ctx, auth.OAuthIdentity{
			Provider: "google", ProviderID: "gid-1",
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
	})

	t.Run("new_user_created_when_no_match", func(t *testing.T) {
		created := false
		repo := &repoMock{
			createUser: func(_ context.Context, u *auth.User) error {
				created = true
				return nil
			},
		}
		resp, err := newSvc(repo).OAuthLogin(ctx, auth.OAuthIdentity{
			Provider:   "github",
			ProviderID: "gh-1",
			Email:      "newuser@github.com",
			Name:       "New User",
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.True(t, created)
		assert.Equal(t, "newuser@github.com", resp.User.Email)
	})

	t.Run("existing_email_links_provider", func(t *testing.T) {
		user := newActiveUser()
		providerLinked := false
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			createProvider: func(_ context.Context, _ *auth.Provider) error {
				providerLinked = true
				return nil
			},
		}
		resp, err := newSvc(repo).OAuthLogin(ctx, auth.OAuthIdentity{
			Provider: "github", ProviderID: "gh-1", Email: user.Email,
		}, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.True(t, providerLinked)
	})

	t.Run("blocked_oauth_user", func(t *testing.T) {
		user := newActiveUser()
		user.IsBlocked = true
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			findProviderByProviderID: func(_ context.Context, _, _ string) (*auth.Provider, error) {
				return &auth.Provider{UserID: user.ID, Provider: "google", ProviderID: "g1"}, nil
			},
		}
		_, err := newSvc(repo).OAuthLogin(ctx, auth.OAuthIdentity{
			Provider: "google", ProviderID: "g1",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrUserBlocked)
	})
}

// ---- OAuthGoogle / OAuthGithub / OAuthFacebook ----

func TestOAuthProviderWrappers(t *testing.T) {
	t.Run("no_verifier_returns_error", func(t *testing.T) {
		svc := newSvc(&repoMock{})
		_, err := svc.OAuthGoogle(ctx, "id-token", noDevice)
		require.ErrorIs(t, err, auth.ErrOAuthNotConfigured)
		_, err = svc.OAuthGithub(ctx, "code", noDevice)
		require.ErrorIs(t, err, auth.ErrOAuthNotConfigured)
		_, err = svc.OAuthFacebook(ctx, "token", noDevice)
		require.ErrorIs(t, err, auth.ErrOAuthNotConfigured)
	})

	t.Run("verifier_error_propagates", func(t *testing.T) {
		verifierErr := errors.New("invalid token")
		oauthMock := &mockOAuth{err: verifierErr}
		svc := newSvc(&repoMock{}, func(c *auth.ServiceConfig) { c.OAuth = oauthMock })
		_, err := svc.OAuthGoogle(ctx, "bad-token", noDevice)
		require.ErrorIs(t, err, verifierErr)
	})

	t.Run("verifier_success_calls_oauth_login", func(t *testing.T) {
		identity := &auth.OAuthIdentity{
			Provider:   "google",
			ProviderID: "g-123",
			Email:      "user@google.com",
			Name:       "Google User",
		}
		oauthMock := &mockOAuth{identity: identity}
		created := false
		repo := &repoMock{
			createUser: func(_ context.Context, _ *auth.User) error { created = true; return nil },
		}
		svc := newSvc(repo, func(c *auth.ServiceConfig) { c.OAuth = oauthMock })
		resp, err := svc.OAuthGoogle(ctx, "valid-id-token", noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.True(t, created)
	})
}

// ---- Refresh ----

func TestRefresh(t *testing.T) {
	t.Run("invalid_token_rejected", func(t *testing.T) {
		repo := &repoMock{}
		_, err := newSvc(repo).Refresh(ctx, "bad-token", noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("blocked_user_rejected", func(t *testing.T) {
		raw := "raw-refresh-token"
		hash := auth.HashToken(raw)
		user := newActiveUser()
		user.IsBlocked = true
		repo := &repoMock{
			findUserSessionByRefreshTokenHash: func(_ context.Context, h string) (*auth.UserSession, error) {
				if h == hash {
					return &auth.UserSession{ID: uuid.New(), UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
				}
				return nil, nil
			},
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
		}
		_, err := newSvc(repo).Refresh(ctx, raw, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("success_rotates_session", func(t *testing.T) {
		raw := "raw-refresh-token"
		hash := auth.HashToken(raw)
		user := newActiveUser()
		oldSessID := uuid.New()
		revokedID := uuid.Nil
		repo := &repoMock{
			findUserSessionByRefreshTokenHash: func(_ context.Context, h string) (*auth.UserSession, error) {
				if h == hash {
					return &auth.UserSession{ID: oldSessID, UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
				}
				return nil, nil
			},
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			revokeUserSessionIfActive: func(_ context.Context, id uuid.UUID) (bool, error) {
				revokedID = id
				return true, nil
			},
		}
		resp, err := newSvc(repo).Refresh(ctx, raw, noDevice)
		require.NoError(t, err)
		assertValidSession(t, resp)
		assert.Equal(t, oldSessID, revokedID, "old session should be revoked")
		assert.NotEqual(t, raw, resp.RefreshToken, "new refresh token should differ")
	})

	t.Run("concurrent_rotation_loses_race", func(t *testing.T) {
		raw := "raw-refresh-token"
		hash := auth.HashToken(raw)
		user := newActiveUser()
		sessID := uuid.New()
		createCalled := false
		repo := &repoMock{
			findUserSessionByRefreshTokenHash: func(_ context.Context, h string) (*auth.UserSession, error) {
				if h == hash {
					return &auth.UserSession{ID: sessID, UserID: user.ID, ExpiresAt: time.Now().Add(time.Hour)}, nil
				}
				return nil, nil
			},
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			revokeUserSessionIfActive: func(_ context.Context, _ uuid.UUID) (bool, error) {
				// Lose the race: another concurrent refresh already revoked the session.
				return false, nil
			},
			createUserSession: func(_ context.Context, _ *auth.UserSession) error {
				createCalled = true
				return nil
			},
		}
		_, err := newSvc(repo).Refresh(ctx, raw, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidToken)
		assert.False(t, createCalled, "no new session should be issued when race lost")
	})
}

// ---- Logout / LogoutAll ----

func TestLogout(t *testing.T) {
	t.Run("unknown_token_is_noop", func(t *testing.T) {
		revokeCalled := false
		repo := &repoMock{
			revokeUserSession: func(_ context.Context, _ uuid.UUID) error { revokeCalled = true; return nil },
		}
		err := newSvc(repo).Logout(ctx, "unknown-token")
		require.NoError(t, err)
		assert.False(t, revokeCalled)
	})

	t.Run("success", func(t *testing.T) {
		raw := "logout-token"
		hash := auth.HashToken(raw)
		sessID := uuid.New()
		revokedID := uuid.Nil
		repo := &repoMock{
			findUserSessionByRefreshTokenHash: func(_ context.Context, h string) (*auth.UserSession, error) {
				if h == hash {
					return &auth.UserSession{ID: sessID}, nil
				}
				return nil, nil
			},
			revokeUserSession: func(_ context.Context, id uuid.UUID) error { revokedID = id; return nil },
		}
		err := newSvc(repo).Logout(ctx, raw)
		require.NoError(t, err)
		assert.Equal(t, sessID, revokedID)
	})
}

func TestLogoutAll(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		deletedFor := uuid.Nil
		repo := &repoMock{
			revokeAllSessions: func(_ context.Context, id uuid.UUID) error { deletedFor = id; return nil },
		}
		err := newSvc(repo).LogoutAll(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, userID, deletedFor)
	})
}

// ---- BlacklistJTI / IsBlacklisted ----

func TestBlacklistJTI(t *testing.T) {
	t.Run("nil_redis_is_noop", func(t *testing.T) {
		svc := newSvc(&repoMock{})
		err := svc.BlacklistJTI(ctx, "some-jti", time.Minute)
		require.NoError(t, err)
	})
}

func TestIsBlacklisted(t *testing.T) {
	t.Run("nil_redis_returns_false", func(t *testing.T) {
		svc := newSvc(&repoMock{})
		got, err := svc.IsBlacklisted(ctx, "some-jti")
		require.NoError(t, err)
		assert.False(t, got)
	})

	t.Run("empty_jti_returns_false", func(t *testing.T) {
		svc := newSvc(&repoMock{})
		got, err := svc.IsBlacklisted(ctx, "")
		require.NoError(t, err)
		assert.False(t, got)
	})
}

// ---- VerifyEmail ----

func TestVerifyEmail(t *testing.T) {
	t.Run("invalid_token_rejected", func(t *testing.T) {
		repo := &repoMock{}
		err := newSvc(repo).VerifyEmail(ctx, "bad-token")
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("success", func(t *testing.T) {
		raw := "verify-email-token"
		hash := auth.HashToken(raw)
		evID := uuid.New()
		userID := uuid.New()
		markedUsed := false
		updatedVerified := false

		repo := &repoMock{
			findEmailVerificationByTokenHash: func(_ context.Context, h string) (*auth.EmailVerification, error) {
				if h == hash {
					return &auth.EmailVerification{ID: evID, UserID: userID}, nil
				}
				return nil, nil
			},
			markEmailVerificationUsed: func(_ context.Context, id uuid.UUID) error {
				if id == evID {
					markedUsed = true
				}
				return nil
			},
			updateUserFields: func(_ context.Context, _ uuid.UUID, updates map[string]any) error {
				if updates["is_verified"] == true {
					updatedVerified = true
				}
				return nil
			},
		}
		err := newSvc(repo).VerifyEmail(ctx, raw)
		require.NoError(t, err)
		assert.True(t, markedUsed)
		assert.True(t, updatedVerified)
	})
}

// ---- ResendVerification ----

func TestResendVerification(t *testing.T) {
	t.Run("unknown_email_is_noop", func(t *testing.T) {
		evCreated := false
		repo := &repoMock{
			createEmailVerification: func(_ context.Context, _ *auth.EmailVerification) error {
				evCreated = true
				return nil
			},
		}
		err := newSvc(repo).ResendVerification(ctx, "nobody@example.com")
		require.NoError(t, err)
		assert.False(t, evCreated)
	})

	t.Run("already_verified_is_noop", func(t *testing.T) {
		user := newActiveUser()
		user.IsVerified = true
		evCreated := false
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			createEmailVerification: func(_ context.Context, _ *auth.EmailVerification) error {
				evCreated = true
				return nil
			},
		}
		err := newSvc(repo).ResendVerification(ctx, user.Email)
		require.NoError(t, err)
		assert.False(t, evCreated)
	})

	t.Run("success_sends_email", func(t *testing.T) {
		user := newActiveUser()
		user.IsVerified = false
		mailer := &mockMailer{}
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		svc := newSvc(repo, func(c *auth.ServiceConfig) { c.Mailer = mailer })
		err := svc.ResendVerification(ctx, user.Email)
		require.NoError(t, err)
		assert.True(t, mailer.verificationCalled)
	})
}

// ---- ForgotPassword ----

func TestForgotPassword(t *testing.T) {
	t.Run("unknown_email_is_noop", func(t *testing.T) {
		prCreated := false
		repo := &repoMock{
			createPasswordReset: func(_ context.Context, _ *auth.PasswordReset) error {
				prCreated = true
				return nil
			},
		}
		err := newSvc(repo).ForgetPassword(ctx, "nobody@example.com")
		require.NoError(t, err)
		assert.False(t, prCreated)
	})

	t.Run("success_no_mailer", func(t *testing.T) {
		user := newActiveUser()
		prCreated := false
		repo := &repoMock{
			findUserByEmail:     func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			createPasswordReset: func(_ context.Context, _ *auth.PasswordReset) error { prCreated = true; return nil },
		}
		err := newSvc(repo).ForgetPassword(ctx, user.Email)
		require.NoError(t, err)
		assert.True(t, prCreated)
	})

	t.Run("success_sends_email", func(t *testing.T) {
		user := newActiveUser()
		mailer := &mockMailer{}
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		}
		svc := newSvc(repo, func(c *auth.ServiceConfig) { c.Mailer = mailer })
		err := svc.ForgetPassword(ctx, user.Email)
		require.NoError(t, err)
		assert.True(t, mailer.resetCalled)
	})
}

// ---- ResetPassword ----

func TestResetPassword(t *testing.T) {
	t.Run("invalid_token_rejected", func(t *testing.T) {
		repo := &repoMock{}
		err := newSvc(repo).ResetPassword(ctx, "bad-token", "newpass123")
		require.ErrorIs(t, err, auth.ErrInvalidToken)
	})

	t.Run("success_updates_password_and_revokes_sessions", func(t *testing.T) {
		raw := "reset-token"
		hash := auth.HashToken(raw)
		prID := uuid.New()
		userID := uuid.New()
		sessionsRevoked := false
		passwordUpdated := false

		repo := &repoMock{
			findPasswordResetByTokenHash: func(_ context.Context, h string) (*auth.PasswordReset, error) {
				if h == hash {
					return &auth.PasswordReset{ID: prID, UserID: userID}, nil
				}
				return nil, nil
			},
			updateUserFields: func(_ context.Context, _ uuid.UUID, updates map[string]any) error {
				if _, ok := updates["password_hash"]; ok {
					passwordUpdated = true
				}
				return nil
			},
			revokeAllSessions: func(_ context.Context, _ uuid.UUID) error { sessionsRevoked = true; return nil },
		}
		err := newSvc(repo).ResetPassword(ctx, raw, "newpassword123")
		require.NoError(t, err)
		assert.True(t, passwordUpdated)
		assert.True(t, sessionsRevoked)
	})
}

// ---- ChangePassword ----

func TestChangePassword(t *testing.T) {
	t.Run("user_not_found", func(t *testing.T) {
		repo := &repoMock{}
		err := newSvc(repo).ChangePassword(ctx, uuid.New(), "old", "new")
		require.ErrorIs(t, err, auth.ErrUserNotFound)
	})

	t.Run("oauth_user_without_password", func(t *testing.T) {
		user := newActiveUser()
		user.PasswordHash = nil
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
		}
		err := newSvc(repo).ChangePassword(ctx, user.ID, "old", "new")
		require.ErrorIs(t, err, auth.ErrPasswordNotSet)
	})

	t.Run("wrong_current_password", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
		}
		err := newSvc(repo).ChangePassword(ctx, user.ID, "wrong-password", "newpass123")
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	})

	t.Run("success_with_session_id_revokes_others_only", func(t *testing.T) {
		user := newActiveUser()
		currentSID := uuid.New()
		updated := false
		var revokedOtherFor, revokedExcept uuid.UUID
		allRevokedCalled := false
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			updateUserFields: func(_ context.Context, _ uuid.UUID, updates map[string]any) error {
				if _, ok := updates["password_hash"]; ok {
					updated = true
				}
				return nil
			},
			revokeOtherSessions: func(_ context.Context, userID, except uuid.UUID) error {
				revokedOtherFor = userID
				revokedExcept = except
				return nil
			},
			revokeAllSessions: func(_ context.Context, _ uuid.UUID) error {
				allRevokedCalled = true
				return nil
			},
		}
		reqCtx := middleware.WithSessionID(ctx, currentSID)
		err := newSvc(repo).ChangePassword(reqCtx, user.ID, "password123", "newpass123")
		require.NoError(t, err)
		assert.True(t, updated)
		assert.Equal(t, user.ID, revokedOtherFor)
		assert.Equal(t, currentSID, revokedExcept)
		assert.False(t, allRevokedCalled, "should not revoke caller's session when sid is on ctx")
	})

	t.Run("success_without_session_id_revokes_all", func(t *testing.T) {
		user := newActiveUser()
		var revokedAllFor uuid.UUID
		otherCalled := false
		repo := &repoMock{
			findUserByID:     func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			updateUserFields: func(_ context.Context, _ uuid.UUID, _ map[string]any) error { return nil },
			revokeOtherSessions: func(_ context.Context, _, _ uuid.UUID) error {
				otherCalled = true
				return nil
			},
			revokeAllSessions: func(_ context.Context, id uuid.UUID) error {
				revokedAllFor = id
				return nil
			},
		}
		err := newSvc(repo).ChangePassword(ctx, user.ID, "password123", "newpass123")
		require.NoError(t, err)
		assert.Equal(t, user.ID, revokedAllFor)
		assert.False(t, otherCalled, "no sid on ctx should fall back to revoke-all")
	})
}

// ---- TOTP ----

func TestSetupTOTP(t *testing.T) {
	t.Run("user_not_found", func(t *testing.T) {
		repo := &repoMock{}
		_, err := newSvc(repo).SetupTOTP(ctx, uuid.New())
		require.ErrorIs(t, err, auth.ErrUserNotFound)
	})

	t.Run("already_enabled", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: user.ID, TOTPSecret: "S", IsEnabled: true}, nil
			},
		}
		_, err := newSvc(repo).SetupTOTP(ctx, user.ID)
		require.ErrorIs(t, err, auth.ErrTOTPAlreadyEnabled)
	})

	t.Run("success_returns_secret_and_url", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
		}
		resp, err := newSvc(repo).SetupTOTP(ctx, user.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.Secret)
		assert.Contains(t, resp.ProvisioningURL, "otpauth://totp/")
		assert.Contains(t, resp.ProvisioningURL, resp.Secret)
	})
}

func TestEnableTOTP(t *testing.T) {
	t.Run("not_configured", func(t *testing.T) {
		repo := &repoMock{}
		err := newSvc(repo).EnableTOTP(ctx, uuid.New(), "123456")
		require.ErrorIs(t, err, auth.ErrTOTPNotConfigured)
	})

	t.Run("wrong_code", func(t *testing.T) {
		userID := uuid.New()
		repo := &repoMock{
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: userID, TOTPSecret: "JBSWY3DPEHPK3PXP", IsEnabled: false}, nil
			},
		}
		err := newSvc(repo).EnableTOTP(ctx, userID, "000000")
		require.ErrorIs(t, err, auth.ErrInvalidTOTP)
	})

	t.Run("success_enables_mfa", func(t *testing.T) {
		userID := uuid.New()
		secret := "JBSWY3DPEHPK3PXP"
		code, err := auth.TotpNow(secret)
		require.NoError(t, err)

		var upserted *auth.MFAConfig
		repo := &repoMock{
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: userID, TOTPSecret: secret, IsEnabled: false}, nil
			},
			upsertMFAConfig: func(_ context.Context, cfg *auth.MFAConfig) error {
				upserted = cfg
				return nil
			},
		}
		err = newSvc(repo).EnableTOTP(ctx, userID, code)
		require.NoError(t, err)
		require.NotNil(t, upserted)
		assert.True(t, upserted.IsEnabled)
	})
}

func TestDisableTOTP(t *testing.T) {
	t.Run("not_configured", func(t *testing.T) {
		repo := &repoMock{}
		err := newSvc(repo).DisableTOTP(ctx, uuid.New(), "123456")
		require.ErrorIs(t, err, auth.ErrTOTPNotConfigured)
	})

	t.Run("disabled_config_treated_as_not_configured", func(t *testing.T) {
		userID := uuid.New()
		repo := &repoMock{
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: userID, IsEnabled: false}, nil
			},
		}
		err := newSvc(repo).DisableTOTP(ctx, userID, "000000")
		require.ErrorIs(t, err, auth.ErrTOTPNotConfigured)
	})

	t.Run("wrong_code", func(t *testing.T) {
		userID := uuid.New()
		repo := &repoMock{
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: userID, TOTPSecret: "JBSWY3DPEHPK3PXP", IsEnabled: true}, nil
			},
		}
		err := newSvc(repo).DisableTOTP(ctx, userID, "000000")
		require.ErrorIs(t, err, auth.ErrInvalidTOTP)
	})

	t.Run("success_disables_mfa", func(t *testing.T) {
		userID := uuid.New()
		secret := "JBSWY3DPEHPK3PXP"
		code, err := auth.TotpNow(secret)
		require.NoError(t, err)

		var upserted *auth.MFAConfig
		repo := &repoMock{
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: userID, TOTPSecret: secret, IsEnabled: true}, nil
			},
			upsertMFAConfig: func(_ context.Context, cfg *auth.MFAConfig) error {
				upserted = cfg
				return nil
			},
		}
		err = newSvc(repo).DisableTOTP(ctx, userID, code)
		require.NoError(t, err)
		require.NotNil(t, upserted)
		assert.False(t, upserted.IsEnabled)
	})
}

// ---- Sessions ----

func TestListSessions(t *testing.T) {
	t.Run("returns_sessions", func(t *testing.T) {
		userID := uuid.New()
		sessions := []auth.UserSession{
			{ID: uuid.New(), UserID: userID},
			{ID: uuid.New(), UserID: userID},
		}
		repo := &repoMock{
			listSessionByUserID: func(_ context.Context, id uuid.UUID) ([]auth.UserSession, error) {
				return sessions, nil
			},
		}
		got, err := newSvc(repo).ListSessions(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, got, 2)
	})
}

func TestDeleteSession(t *testing.T) {
	t.Run("session_not_owned_by_user", func(t *testing.T) {
		userID := uuid.New()
		otherSessID := uuid.New()
		repo := &repoMock{
			listSessionByUserID: func(_ context.Context, _ uuid.UUID) ([]auth.UserSession, error) {
				return []auth.UserSession{{ID: uuid.New(), UserID: userID}}, nil
			},
		}
		err := newSvc(repo).DeleteSession(ctx, userID, otherSessID)
		require.ErrorIs(t, err, auth.ErrSessionNotFound)
	})

	t.Run("success", func(t *testing.T) {
		userID := uuid.New()
		sessID := uuid.New()
		deleted := uuid.Nil
		repo := &repoMock{
			listSessionByUserID: func(_ context.Context, _ uuid.UUID) ([]auth.UserSession, error) {
				return []auth.UserSession{{ID: sessID, UserID: userID}}, nil
			},
			deleteUserSession: func(_ context.Context, id uuid.UUID) error { deleted = id; return nil },
		}
		err := newSvc(repo).DeleteSession(ctx, userID, sessID)
		require.NoError(t, err)
		assert.Equal(t, sessID, deleted)
	})
}

// ---- GetUser ----

func TestGetUser(t *testing.T) {
	t.Run("not_found", func(t *testing.T) {
		repo := &repoMock{}
		_, err := newSvc(repo).GetUser(ctx, uuid.New())
		require.ErrorIs(t, err, auth.ErrUserNotFound)
	})

	t.Run("success", func(t *testing.T) {
		user := newActiveUser()
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
		}
		got, err := newSvc(repo).GetUser(ctx, user.ID)
		require.NoError(t, err)
		assert.Equal(t, user.ID.String(), got.ID)
		assert.Equal(t, user.Email, got.Email)
		assert.Equal(t, user.Name, got.Name)
	})
}

// ---- DeactivateAccount / DeleteAccount ----

func TestDeactivateAccount(t *testing.T) {
	t.Run("success_deactivates_and_revokes_sessions", func(t *testing.T) {
		userID := uuid.New()
		deactivated := false
		sessionsRevoked := false
		repo := &repoMock{
			deactivateUser:    func(_ context.Context, _ uuid.UUID) error { deactivated = true; return nil },
			revokeAllSessions: func(_ context.Context, _ uuid.UUID) error { sessionsRevoked = true; return nil },
		}
		err := newSvc(repo).DeactivateAccount(ctx, userID)
		require.NoError(t, err)
		assert.True(t, deactivated)
		assert.True(t, sessionsRevoked)
	})
}

func TestDeleteAccount(t *testing.T) {
	t.Run("success_marks_deleted_and_revokes_sessions", func(t *testing.T) {
		userID := uuid.New()
		marked := false
		sessionsRevoked := false
		repo := &repoMock{
			markUserDeleted:   func(_ context.Context, _ uuid.UUID) error { marked = true; return nil },
			revokeAllSessions: func(_ context.Context, _ uuid.UUID) error { sessionsRevoked = true; return nil },
		}
		err := newSvc(repo).DeleteAccount(ctx, userID)
		require.NoError(t, err)
		assert.True(t, marked)
		assert.True(t, sessionsRevoked)
	})
}

// ---- JWKS ----

func TestJWKS(t *testing.T) {
	t.Run("returns_rsa_public_key", func(t *testing.T) {
		svc := newSvc(&repoMock{})
		jwks := svc.JWKS()
		require.Len(t, jwks.Keys, 1)
		k := jwks.Keys[0]
		assert.Equal(t, "RSA", k.Kty)
		assert.Equal(t, "RS256", k.Alg)
		assert.Equal(t, "sig", k.Use)
		assert.NotEmpty(t, k.N)
		assert.NotEmpty(t, k.E)
	})
}

// ---- Hardening: audience, roles, audit log, single-active password reset ----

func TestAccessTokenIncludesAudienceAndRoles(t *testing.T) {
	user := newActiveUser()
	repo := &repoMock{
		findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		listUserRoles: func(_ context.Context, _ uuid.UUID) ([]string, error) {
			return []string{"admin", "trip_creator"}, nil
		},
	}
	svc := newSvc(repo, func(c *auth.ServiceConfig) {
		c.JWT.Audience = []string{"trip-app", "mobile"}
	})
	resp, err := svc.Login(ctx, auth.LoginRequest{
		Email: user.Email, Password: "password123",
	}, noDevice)
	require.NoError(t, err)

	parsed, err := jwt.Parse(resp.AccessToken, func(_ *jwt.Token) (any, error) {
		return &testPrivKey.PublicKey, nil
	}, jwt.WithValidMethods([]string{"RS256"}))
	require.NoError(t, err)

	claims := parsed.Claims.(jwt.MapClaims)
	aud, _ := claims["aud"].([]any)
	require.Len(t, aud, 2)
	assert.Equal(t, "trip-app", aud[0])
	assert.Equal(t, "mobile", aud[1])

	roles, _ := claims["roles"].([]any)
	require.Len(t, roles, 2)
	assert.Equal(t, "admin", roles[0])
	assert.Equal(t, "trip_creator", roles[1])

	sid, _ := claims["sid"].(string)
	require.NotEmpty(t, sid)
	_, parseErr := uuid.Parse(sid)
	require.NoError(t, parseErr, "sid must be a valid UUID")
}

func TestAuditLogEmitted(t *testing.T) {
	t.Run("login_success", func(t *testing.T) {
		user := newActiveUser()
		var actions []auth.AuditAction
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			createAuditLog: func(_ context.Context, log *auth.AuditLog) error {
				actions = append(actions, log.Action)
				return nil
			},
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.NoError(t, err)
		assert.Contains(t, actions, auth.AuditLogin)
	})

	t.Run("login_failure_logs_failed", func(t *testing.T) {
		user := newActiveUser()
		var statuses []auth.AuditStatus
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			createAuditLog: func(_ context.Context, log *auth.AuditLog) error {
				if log.Action == auth.AuditLoginFailed {
					statuses = append(statuses, log.Status)
				}
				return nil
			},
		}
		_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
			Email: user.Email, Password: "wrong",
		}, noDevice)
		require.ErrorIs(t, err, auth.ErrInvalidCredentials)
		assert.Contains(t, statuses, auth.AuditFailure)
	})

	t.Run("propagates_request_id_and_trace_id_from_context", func(t *testing.T) {
		user := newActiveUser()
		var captured *auth.AuditLog
		repo := &repoMock{
			findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
			createAuditLog: func(_ context.Context, log *auth.AuditLog) error {
				if log.Action == auth.AuditLogin {
					captured = log
				}
				return nil
			},
		}
		reqID := uuid.New()
		reqCtx := middleware.WithRequestID(ctx, reqID)
		reqCtx = middleware.WithTraceID(reqCtx, "trace-xyz")

		_, err := newSvc(repo).Login(reqCtx, auth.LoginRequest{
			Email: user.Email, Password: "password123",
		}, noDevice)
		require.NoError(t, err)
		require.NotNil(t, captured, "expected an audit log for login success")
		require.NotNil(t, captured.RequestID)
		assert.Equal(t, reqID, *captured.RequestID)
		assert.Equal(t, "trace-xyz", captured.TraceID)
	})
}

func TestForgotPasswordSingleActive(t *testing.T) {
	user := newActiveUser()
	invalidatedFor := uuid.Nil
	createdFor := uuid.Nil
	callOrder := []string{}
	repo := &repoMock{
		findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		invalidatePendingPasswordResets: func(_ context.Context, id uuid.UUID) error {
			invalidatedFor = id
			callOrder = append(callOrder, "invalidate")
			return nil
		},
		createPasswordReset: func(_ context.Context, pr *auth.PasswordReset) error {
			createdFor = pr.UserID
			callOrder = append(callOrder, "create")
			return nil
		},
	}
	err := newSvc(repo).ForgetPassword(ctx, user.Email)
	require.NoError(t, err)
	assert.Equal(t, user.ID, invalidatedFor, "should invalidate prior tokens for user")
	assert.Equal(t, user.ID, createdFor, "should create new token for user")
	assert.Equal(t, []string{"invalidate", "create"}, callOrder, "must invalidate before create")
}

func TestEncryptedTOTPRoundTrip(t *testing.T) {
	t.Run("setup_stores_ciphertext", func(t *testing.T) {
		user := newActiveUser()
		var stored *auth.MFAConfig
		repo := &repoMock{
			findUserByID: func(_ context.Context, _ uuid.UUID) (*auth.User, error) { return user, nil },
			upsertMFAConfig: func(_ context.Context, cfg *auth.MFAConfig) error {
				stored = cfg
				return nil
			},
		}
		key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		svc := newSvc(repo, func(c *auth.ServiceConfig) { c.Security.TOTPEncryptionKey = key })
		resp, err := svc.SetupTOTP(ctx, user.ID)
		require.NoError(t, err)
		require.NotNil(t, stored)
		assert.NotEqual(t, resp.Secret, stored.TOTPSecret, "stored secret must differ from plaintext")

		// Round-trip: decrypted ciphertext should equal original plaintext.
		plain, err := auth.DecryptSecret(stored.TOTPSecret, key)
		require.NoError(t, err)
		assert.Equal(t, resp.Secret, plain)
	})

	t.Run("enable_decrypts_and_verifies", func(t *testing.T) {
		userID := uuid.New()
		key := "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
		secret := "JBSWY3DPEHPK3PXP"
		ct, err := auth.EncryptSecret(secret, key)
		require.NoError(t, err)
		code, err := auth.TotpNow(secret)
		require.NoError(t, err)

		var enabled *auth.MFAConfig
		repo := &repoMock{
			findMFAConfig: func(_ context.Context, _ uuid.UUID) (*auth.MFAConfig, error) {
				return &auth.MFAConfig{UserID: userID, TOTPSecret: ct, IsEnabled: false}, nil
			},
			upsertMFAConfig: func(_ context.Context, cfg *auth.MFAConfig) error {
				enabled = cfg
				return nil
			},
		}
		svc := newSvc(repo, func(c *auth.ServiceConfig) { c.Security.TOTPEncryptionKey = key })
		err = svc.EnableTOTP(ctx, userID, code)
		require.NoError(t, err)
		require.NotNil(t, enabled)
		assert.True(t, enabled.IsEnabled)
	})
}

func TestDeviceFingerprintWrittenToSession(t *testing.T) {
	user := newActiveUser()
	var captured *auth.UserSession
	repo := &repoMock{
		findUserByEmail: func(_ context.Context, _ string) (*auth.User, error) { return user, nil },
		createUserSession: func(_ context.Context, s *auth.UserSession) error {
			captured = s
			return nil
		},
	}
	device := auth.DeviceInfo{
		DeviceName: "Pixel-7", DeviceType: auth.DeviceAndroid,
		IPAddress: "10.0.0.5", UserAgent: "okhttp/4",
	}
	_, err := newSvc(repo).Login(ctx, auth.LoginRequest{
		Email: user.Email, Password: "password123",
	}, device)
	require.NoError(t, err)
	require.NotNil(t, captured)
	assert.Equal(t, auth.DeviceFingerprint(device), captured.DeviceFingerprint)
	assert.NotEmpty(t, captured.DeviceFingerprint)
}

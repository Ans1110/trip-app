package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// ---- mock service implementing auth.IService ----

type svcMock struct {
	register           func(context.Context, auth.RegisterRequest, auth.DeviceInfo) (*auth.SessionResponse, error)
	login              func(context.Context, auth.LoginRequest, auth.DeviceInfo) (*auth.SessionResponse, error)
	oauthLogin         func(context.Context, auth.OAuthIdentity, auth.DeviceInfo) (*auth.SessionResponse, error)
	oauthGoogle        func(context.Context, string, auth.DeviceInfo) (*auth.SessionResponse, error)
	oauthGithub        func(context.Context, string, auth.DeviceInfo) (*auth.SessionResponse, error)
	oauthFacebook      func(context.Context, string, auth.DeviceInfo) (*auth.SessionResponse, error)
	refresh            func(context.Context, string, auth.DeviceInfo) (*auth.SessionResponse, error)
	logout             func(context.Context, string) error
	logoutAll          func(context.Context, uuid.UUID) error
	blacklistJTI       func(context.Context, string, time.Duration) error
	isBlacklisted      func(context.Context, string) (bool, error)
	verifyEmail        func(context.Context, string) error
	resendVerification func(context.Context, string) error
	forgotPassword     func(context.Context, string) error
	resetPassword      func(context.Context, string, string) error
	changePassword     func(context.Context, uuid.UUID, string, string) error
	setupTOTP          func(context.Context, uuid.UUID) (*auth.TOTPSetupResponse, error)
	enableTOTP         func(context.Context, uuid.UUID, string) error
	disableTOTP        func(context.Context, uuid.UUID, string) error
	listSessions       func(context.Context, uuid.UUID) ([]auth.UserSession, error)
	deleteSession      func(context.Context, uuid.UUID, uuid.UUID) error
	getUser            func(context.Context, uuid.UUID) (*auth.UserResponse, error)
	deactivateAccount  func(context.Context, uuid.UUID) error
	deleteAccount      func(context.Context, uuid.UUID) error
	jwks               func() *auth.JWKResponse
}

func (s *svcMock) Register(c context.Context, r auth.RegisterRequest, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.register != nil {
		return s.register(c, r, d)
	}
	return &auth.SessionResponse{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 60, TokenType: "Bearer"}, nil
}
func (s *svcMock) Login(c context.Context, r auth.LoginRequest, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.login != nil {
		return s.login(c, r, d)
	}
	return &auth.SessionResponse{AccessToken: "at", RefreshToken: "rt", ExpiresIn: 60, TokenType: "Bearer"}, nil
}
func (s *svcMock) OAuthLogin(c context.Context, id auth.OAuthIdentity, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.oauthLogin != nil {
		return s.oauthLogin(c, id, d)
	}
	return &auth.SessionResponse{AccessToken: "at"}, nil
}
func (s *svcMock) OAuthGoogle(c context.Context, tok string, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.oauthGoogle != nil {
		return s.oauthGoogle(c, tok, d)
	}
	return &auth.SessionResponse{AccessToken: "at"}, nil
}
func (s *svcMock) OAuthGithub(c context.Context, code string, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.oauthGithub != nil {
		return s.oauthGithub(c, code, d)
	}
	return &auth.SessionResponse{AccessToken: "at"}, nil
}
func (s *svcMock) OAuthFacebook(c context.Context, tok string, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.oauthFacebook != nil {
		return s.oauthFacebook(c, tok, d)
	}
	return &auth.SessionResponse{AccessToken: "at"}, nil
}
func (s *svcMock) Refresh(c context.Context, tok string, d auth.DeviceInfo) (*auth.SessionResponse, error) {
	if s.refresh != nil {
		return s.refresh(c, tok, d)
	}
	return &auth.SessionResponse{AccessToken: "at2", RefreshToken: "rt2"}, nil
}
func (s *svcMock) Logout(c context.Context, tok string) error {
	if s.logout != nil {
		return s.logout(c, tok)
	}
	return nil
}
func (s *svcMock) LogoutAll(c context.Context, uid uuid.UUID) error {
	if s.logoutAll != nil {
		return s.logoutAll(c, uid)
	}
	return nil
}
func (s *svcMock) BlacklistJTI(c context.Context, jti string, ttl time.Duration) error {
	if s.blacklistJTI != nil {
		return s.blacklistJTI(c, jti, ttl)
	}
	return nil
}
func (s *svcMock) IsBlacklisted(c context.Context, jti string) (bool, error) {
	if s.isBlacklisted != nil {
		return s.isBlacklisted(c, jti)
	}
	return false, nil
}
func (s *svcMock) VerifyEmail(c context.Context, tok string) error {
	if s.verifyEmail != nil {
		return s.verifyEmail(c, tok)
	}
	return nil
}
func (s *svcMock) ResendVerification(c context.Context, email string) error {
	if s.resendVerification != nil {
		return s.resendVerification(c, email)
	}
	return nil
}
func (s *svcMock) ForgotPassword(c context.Context, email string) error {
	if s.forgotPassword != nil {
		return s.forgotPassword(c, email)
	}
	return nil
}
func (s *svcMock) ResetPassword(c context.Context, tok, pw string) error {
	if s.resetPassword != nil {
		return s.resetPassword(c, tok, pw)
	}
	return nil
}
func (s *svcMock) ChangePassword(c context.Context, uid uuid.UUID, old, newPw string) error {
	if s.changePassword != nil {
		return s.changePassword(c, uid, old, newPw)
	}
	return nil
}
func (s *svcMock) SetupTOTP(c context.Context, uid uuid.UUID) (*auth.TOTPSetupResponse, error) {
	if s.setupTOTP != nil {
		return s.setupTOTP(c, uid)
	}
	return &auth.TOTPSetupResponse{Secret: "S3CR3T", ProvisioningURL: "otpauth://totp/x"}, nil
}
func (s *svcMock) EnableTOTP(c context.Context, uid uuid.UUID, code string) error {
	if s.enableTOTP != nil {
		return s.enableTOTP(c, uid, code)
	}
	return nil
}
func (s *svcMock) DisableTOTP(c context.Context, uid uuid.UUID, code string) error {
	if s.disableTOTP != nil {
		return s.disableTOTP(c, uid, code)
	}
	return nil
}
func (s *svcMock) ListSessions(c context.Context, uid uuid.UUID) ([]auth.UserSession, error) {
	if s.listSessions != nil {
		return s.listSessions(c, uid)
	}
	return []auth.UserSession{{ID: uuid.New(), UserID: uid}}, nil
}
func (s *svcMock) DeleteSession(c context.Context, uid, sid uuid.UUID) error {
	if s.deleteSession != nil {
		return s.deleteSession(c, uid, sid)
	}
	return nil
}
func (s *svcMock) GetUser(c context.Context, uid uuid.UUID) (*auth.UserResponse, error) {
	if s.getUser != nil {
		return s.getUser(c, uid)
	}
	return &auth.UserResponse{ID: uid.String(), Email: "u@example.com", Name: "U"}, nil
}
func (s *svcMock) DeactivateAccount(c context.Context, uid uuid.UUID) error {
	if s.deactivateAccount != nil {
		return s.deactivateAccount(c, uid)
	}
	return nil
}
func (s *svcMock) DeleteAccount(c context.Context, uid uuid.UUID) error {
	if s.deleteAccount != nil {
		return s.deleteAccount(c, uid)
	}
	return nil
}
func (s *svcMock) JWKS() *auth.JWKResponse {
	if s.jwks != nil {
		return s.jwks()
	}
	return &auth.JWKResponse{Keys: []auth.JWK{{Kty: "RSA", Kid: "test"}}}
}

// ---- helpers ----

// newRouter mounts the handler with a fresh gin engine. If userID is non-nil
// the protected group is wrapped with a middleware that injects it, simulating
// JWTAuth having already run.
func newRouter(svc auth.IService, userID *uuid.UUID) *gin.Engine {
	r := gin.New()
	api := r.Group("/api/v1")
	pub := api.Group("/")
	pri := api.Group("/")
	if userID != nil {
		uid := *userID
		pri.Use(func(c *gin.Context) {
			c.Set(middleware.ContextUserID, uid)
			c.Next()
		})
	}
	h := auth.NewHandler(svc, zap.NewNop(), auth.CookieConfig{})
	h.RegisterRoutes(pub, pri)
	return r
}

func doJSON(t *testing.T, r *gin.Engine, method, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		require.NoError(t, err)
		buf = bytes.NewReader(raw)
	}
	req := httptest.NewRequest(method, path, buf)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func doRaw(t *testing.T, r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewReader([]byte(body)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

type apiResp struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data"`
}

func parseEnvelope(t *testing.T, w *httptest.ResponseRecorder) apiResp {
	t.Helper()
	var resp apiResp
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &resp))
	return resp
}

// ---- public endpoints ----

func TestHandlerRegister(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		called := false
		svc := &svcMock{register: func(_ context.Context, r auth.RegisterRequest, _ auth.DeviceInfo) (*auth.SessionResponse, error) {
			called = true
			assert.Equal(t, "bob@example.com", r.Email)
			return &auth.SessionResponse{AccessToken: "AT", RefreshToken: "RT", ExpiresIn: 60, TokenType: "Bearer"}, nil
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/register", auth.RegisterRequest{
			Email: "bob@example.com", Password: "password123", Name: "Bob",
		})
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.True(t, called)
	})

	t.Run("duplicate_email_returns_409", func(t *testing.T) {
		svc := &svcMock{register: func(context.Context, auth.RegisterRequest, auth.DeviceInfo) (*auth.SessionResponse, error) {
			return nil, auth.ErrEmailExists
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/register", auth.RegisterRequest{
			Email: "x@x.com", Password: "password123", Name: "X",
		})
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("invalid_body_returns_400", func(t *testing.T) {
		w := doRaw(t, newRouter(&svcMock{}, nil), http.MethodPost, "/api/v1/auth/register", `{"email":"not-an-email"}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlerLogin(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &svcMock{login: func(_ context.Context, r auth.LoginRequest, _ auth.DeviceInfo) (*auth.SessionResponse, error) {
			assert.Equal(t, "u@x.com", r.Email)
			return &auth.SessionResponse{AccessToken: "AT"}, nil
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/login", auth.LoginRequest{
			Email: "u@x.com", Password: "password123",
		})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid_credentials_returns_401", func(t *testing.T) {
		svc := &svcMock{login: func(context.Context, auth.LoginRequest, auth.DeviceInfo) (*auth.SessionResponse, error) {
			return nil, auth.ErrInvalidCredentials
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/login", auth.LoginRequest{
			Email: "u@x.com", Password: "password123",
		})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("blocked_returns_403", func(t *testing.T) {
		svc := &svcMock{login: func(context.Context, auth.LoginRequest, auth.DeviceInfo) (*auth.SessionResponse, error) {
			return nil, auth.ErrUserBlocked
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/login", auth.LoginRequest{
			Email: "u@x.com", Password: "password123",
		})
		assert.Equal(t, http.StatusForbidden, w.Code)
	})

	t.Run("requires_totp_passthrough", func(t *testing.T) {
		svc := &svcMock{login: func(context.Context, auth.LoginRequest, auth.DeviceInfo) (*auth.SessionResponse, error) {
			return &auth.SessionResponse{RequiresTOTP: true}, nil
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/login", auth.LoginRequest{
			Email: "u@x.com", Password: "password123",
		})
		assert.Equal(t, http.StatusOK, w.Code)
		env := parseEnvelope(t, w)
		var sr auth.SessionResponse
		require.NoError(t, json.Unmarshal(env.Data, &sr))
		assert.True(t, sr.RequiresTOTP)
	})
}

func TestHandlerOAuthProviders(t *testing.T) {
	t.Run("google_success", func(t *testing.T) {
		svc := &svcMock{oauthGoogle: func(_ context.Context, tok string, _ auth.DeviceInfo) (*auth.SessionResponse, error) {
			assert.Equal(t, "id-token", tok)
			return &auth.SessionResponse{AccessToken: "AT"}, nil
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/oauth/google",
			auth.GoogleOAuthRequest{IDToken: "id-token"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("github_success", func(t *testing.T) {
		svc := &svcMock{oauthGithub: func(_ context.Context, c string, _ auth.DeviceInfo) (*auth.SessionResponse, error) {
			assert.Equal(t, "code", c)
			return &auth.SessionResponse{AccessToken: "AT"}, nil
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/oauth/github",
			auth.GithubOAuthRequest{Code: "code"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("facebook_oauth_not_configured_returns_503", func(t *testing.T) {
		svc := &svcMock{oauthFacebook: func(context.Context, string, auth.DeviceInfo) (*auth.SessionResponse, error) {
			return nil, auth.ErrOAuthNotConfigured
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/oauth/facebook",
			auth.FacebookOAuthRequest{AccessToken: "tok"})
		assert.Equal(t, http.StatusServiceUnavailable, w.Code)
	})
}

func TestHandlerRefresh(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc := &svcMock{refresh: func(_ context.Context, tok string, _ auth.DeviceInfo) (*auth.SessionResponse, error) {
			assert.Equal(t, "RT", tok)
			return &auth.SessionResponse{AccessToken: "AT2", RefreshToken: "RT2"}, nil
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/refresh",
			auth.RefreshTokenRequest{RefreshToken: "RT"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("invalid_token_returns_401", func(t *testing.T) {
		svc := &svcMock{refresh: func(context.Context, string, auth.DeviceInfo) (*auth.SessionResponse, error) {
			return nil, auth.ErrInvalidToken
		}}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/refresh",
			auth.RefreshTokenRequest{RefreshToken: "bad"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlerEmailFlows(t *testing.T) {
	t.Run("verify_email_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, nil), http.MethodPost, "/api/v1/auth/verify-email",
			auth.VerifyEmailRequest{Token: "tok"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("verify_email_invalid_returns_401", func(t *testing.T) {
		svc := &svcMock{verifyEmail: func(context.Context, string) error { return auth.ErrInvalidToken }}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/verify-email",
			auth.VerifyEmailRequest{Token: "x"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("resend_verification_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, nil), http.MethodPost, "/api/v1/auth/resend-verification",
			auth.ResendVerificationRequest{Email: "u@x.com"})
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandlerPasswordFlows(t *testing.T) {
	t.Run("forgot_password_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, nil), http.MethodPost, "/api/v1/auth/forgot-password",
			auth.ForgotPasswordRequest{Email: "u@x.com"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("forgot_password_rate_limited_returns_429", func(t *testing.T) {
		svc := &svcMock{forgotPassword: func(context.Context, string) error { return auth.ErrRateLimited }}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/forgot-password",
			auth.ForgotPasswordRequest{Email: "u@x.com"})
		assert.Equal(t, http.StatusTooManyRequests, w.Code)
	})

	t.Run("reset_password_invalid_token_returns_401", func(t *testing.T) {
		svc := &svcMock{resetPassword: func(context.Context, string, string) error { return auth.ErrInvalidToken }}
		w := doJSON(t, newRouter(svc, nil), http.MethodPost, "/api/v1/auth/reset-password",
			auth.ResetPasswordRequest{Token: "t", Password: "password123"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlerJWKS(t *testing.T) {
	svc := &svcMock{jwks: func() *auth.JWKResponse {
		return &auth.JWKResponse{Keys: []auth.JWK{{Kty: "RSA", Kid: "key-1", Alg: "RS256"}}}
	}}
	w := doJSON(t, newRouter(svc, nil), http.MethodGet, "/api/v1/auth/.well-known/jwks.json", nil)
	assert.Equal(t, http.StatusOK, w.Code)
	var jwks auth.JWKResponse
	require.NoError(t, json.Unmarshal(w.Body.Bytes(), &jwks))
	require.Len(t, jwks.Keys, 1)
	assert.Equal(t, "key-1", jwks.Keys[0].Kid)
}

// ---- protected endpoints ----

func TestHandlerLogout(t *testing.T) {
	uid := uuid.New()
	t.Run("success", func(t *testing.T) {
		called := false
		svc := &svcMock{logout: func(_ context.Context, tok string) error {
			called = true
			assert.Equal(t, "RT", tok)
			return nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/logout",
			auth.LogoutRequest{RefreshToken: "RT"})
		assert.Equal(t, http.StatusOK, w.Code)
		assert.True(t, called)
	})

	t.Run("invalid_body_returns_400", func(t *testing.T) {
		w := doRaw(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/auth/logout", `{}`)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestHandlerLogoutAll(t *testing.T) {
	uid := uuid.New()
	t.Run("success", func(t *testing.T) {
		var seen uuid.UUID
		svc := &svcMock{logoutAll: func(_ context.Context, id uuid.UUID) error { seen = id; return nil }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/logout-all", nil)
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, uid, seen)
	})

	t.Run("no_user_returns_401", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, nil), http.MethodPost, "/api/v1/auth/logout-all", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlerChangePassword(t *testing.T) {
	uid := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &svcMock{changePassword: func(_ context.Context, id uuid.UUID, old, n string) error {
			assert.Equal(t, uid, id)
			assert.Equal(t, "oldpassword", old)
			assert.Equal(t, "newpassword", n)
			return nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/password",
			auth.ChangePasswordRequest{CurrentPassword: "oldpassword", NewPassword: "newpassword"})
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("wrong_password_returns_401", func(t *testing.T) {
		svc := &svcMock{changePassword: func(context.Context, uuid.UUID, string, string) error {
			return auth.ErrInvalidCredentials
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/password",
			auth.ChangePasswordRequest{CurrentPassword: "oldpassword", NewPassword: "newpassword"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlerTOTPFlow(t *testing.T) {
	uid := uuid.New()

	t.Run("setup_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/auth/totp/setup", nil)
		assert.Equal(t, http.StatusOK, w.Code)
		env := parseEnvelope(t, w)
		var setup auth.TOTPSetupResponse
		require.NoError(t, json.Unmarshal(env.Data, &setup))
		assert.NotEmpty(t, setup.Secret)
	})

	t.Run("setup_already_enabled_returns_409", func(t *testing.T) {
		svc := &svcMock{setupTOTP: func(context.Context, uuid.UUID) (*auth.TOTPSetupResponse, error) {
			return nil, auth.ErrTOTPAlreadyEnabled
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/totp/setup", nil)
		assert.Equal(t, http.StatusConflict, w.Code)
	})

	t.Run("enable_invalid_code_returns_401", func(t *testing.T) {
		svc := &svcMock{enableTOTP: func(context.Context, uuid.UUID, string) error { return auth.ErrInvalidTOTP }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/totp/enable",
			auth.VerifyTOTPRequest{TOTPCode: "000000"})
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("enable_not_configured_returns_404", func(t *testing.T) {
		svc := &svcMock{enableTOTP: func(context.Context, uuid.UUID, string) error { return auth.ErrTOTPNotConfigured }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/totp/enable",
			auth.VerifyTOTPRequest{TOTPCode: "123456"})
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("disable_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/auth/totp/disable",
			auth.VerifyTOTPRequest{TOTPCode: "123456"})
		assert.Equal(t, http.StatusOK, w.Code)
	})
}

func TestHandlerSessions(t *testing.T) {
	uid := uuid.New()

	t.Run("list_success", func(t *testing.T) {
		svc := &svcMock{listSessions: func(_ context.Context, id uuid.UUID) ([]auth.UserSession, error) {
			assert.Equal(t, uid, id)
			return []auth.UserSession{{ID: uuid.New(), UserID: id}}, nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/auth/sessions", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete_success", func(t *testing.T) {
		sid := uuid.New()
		svc := &svcMock{deleteSession: func(_ context.Context, u, s uuid.UUID) error {
			assert.Equal(t, uid, u)
			assert.Equal(t, sid, s)
			return nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/auth/sessions/"+sid.String(), nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete_bad_id_returns_400", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodDelete, "/api/v1/auth/sessions/not-a-uuid", nil)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("delete_not_found_returns_404", func(t *testing.T) {
		sid := uuid.New()
		svc := &svcMock{deleteSession: func(context.Context, uuid.UUID, uuid.UUID) error { return auth.ErrSessionNotFound }}
		w := doJSON(t, newRouter(svc, &uid), http.MethodDelete, "/api/v1/auth/sessions/"+sid.String(), nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})
}

func TestHandlerMe(t *testing.T) {
	uid := uuid.New()

	t.Run("success", func(t *testing.T) {
		svc := &svcMock{getUser: func(_ context.Context, id uuid.UUID) (*auth.UserResponse, error) {
			return &auth.UserResponse{ID: id.String(), Email: "u@x.com", Name: "U"}, nil
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/auth/me", nil)
		assert.Equal(t, http.StatusOK, w.Code)
		env := parseEnvelope(t, w)
		var user auth.UserResponse
		require.NoError(t, json.Unmarshal(env.Data, &user))
		assert.Equal(t, uid.String(), user.ID)
	})

	t.Run("not_found_returns_404", func(t *testing.T) {
		svc := &svcMock{getUser: func(context.Context, uuid.UUID) (*auth.UserResponse, error) {
			return nil, auth.ErrUserNotFound
		}}
		w := doJSON(t, newRouter(svc, &uid), http.MethodGet, "/api/v1/auth/me", nil)
		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("no_user_returns_401", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, nil), http.MethodGet, "/api/v1/auth/me", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

func TestHandlerAccountLifecycle(t *testing.T) {
	uid := uuid.New()

	t.Run("deactivate_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodPost, "/api/v1/auth/deactivate", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("delete_success", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, &uid), http.MethodDelete, "/api/v1/auth/account", nil)
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("deactivate_no_user_returns_401", func(t *testing.T) {
		w := doJSON(t, newRouter(&svcMock{}, nil), http.MethodPost, "/api/v1/auth/deactivate", nil)
		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})
}

// ---- error-mapping table ----

func TestHandlerErrorMapping(t *testing.T) {
	uid := uuid.New()
	cases := []struct {
		name     string
		svcErr   error
		wantCode int
	}{
		{"invalid_credentials_to_401", auth.ErrInvalidCredentials, http.StatusUnauthorized},
		{"invalid_token_to_401", auth.ErrInvalidToken, http.StatusUnauthorized},
		{"invalid_totp_to_401", auth.ErrInvalidTOTP, http.StatusUnauthorized},
		{"blocked_to_403", auth.ErrUserBlocked, http.StatusForbidden},
		{"not_found_to_404", auth.ErrUserNotFound, http.StatusNotFound},
		{"rate_limited_to_429", auth.ErrRateLimited, http.StatusTooManyRequests},
		{"oauth_unavailable_to_503", auth.ErrOAuthNotConfigured, http.StatusServiceUnavailable},
		{"password_not_set_to_400", auth.ErrPasswordNotSet, http.StatusBadRequest},
		{"unknown_to_500", errors.New("kaboom"), http.StatusInternalServerError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			svc := &svcMock{login: func(context.Context, auth.LoginRequest, auth.DeviceInfo) (*auth.SessionResponse, error) {
				return nil, tc.svcErr
			}}
			w := doJSON(t, newRouter(svc, &uid), http.MethodPost, "/api/v1/auth/login",
				auth.LoginRequest{Email: "u@x.com", Password: "password123"})
			assert.Equal(t, tc.wantCode, w.Code, "got body=%s", w.Body.String())
		})
	}
}

// ---- access-log middleware ----

func TestHandlerAccessLogMiddleware(t *testing.T) {
	uid := uuid.New()
	core, logs := observer.New(zapcore.InfoLevel)
	logger := zap.New(core)

	r := gin.New()
	api := r.Group("/api/v1")
	pub := api.Group("/")
	pri := api.Group("/")
	pri.Use(func(c *gin.Context) {
		c.Set(middleware.ContextUserID, uid)
		c.Next()
	})
	h := auth.NewHandler(&svcMock{}, logger, auth.CookieConfig{})
	h.RegisterRoutes(pub, pri)

	w := doJSON(t, r, http.MethodPost, "/api/v1/auth/logout", auth.LogoutRequest{RefreshToken: "RT"})
	require.Equal(t, http.StatusOK, w.Code)

	entries := logs.FilterMessage("auth api access").All()
	require.NotEmpty(t, entries, "expected an access-log entry")
	fields := entries[0].ContextMap()
	assert.Equal(t, "/api/v1/auth/logout", fields["route"])
	assert.Equal(t, "POST", fields["method"])
	assert.EqualValues(t, http.StatusOK, fields["status"])
	assert.Equal(t, uid.String(), fields["user_id"])
}

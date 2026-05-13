package auth

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// IHandler is the HTTP-layer contract for the auth module. Consumers (the
// server's router wiring, tests, alternate transports) should depend on this
// interface rather than the concrete Handler.
type IHandler interface {
	RegisterRoutes(public, protected *gin.RouterGroup)

	// Public endpoints (no JWT required).
	Register(c *gin.Context)
	Login(c *gin.Context)
	OAuthGoogle(c *gin.Context)
	OAuthGithub(c *gin.Context)
	OAuthFacebook(c *gin.Context)
	Refresh(c *gin.Context)
	VerifyEmail(c *gin.Context)
	ResendVerification(c *gin.Context)
	ForgotPassword(c *gin.Context)
	ResetPassword(c *gin.Context)
	JWKS(c *gin.Context)

	// Protected endpoints (JWT required).
	Logout(c *gin.Context)
	LogoutAll(c *gin.Context)
	ChangePassword(c *gin.Context)
	SetupTOTP(c *gin.Context)
	EnableTOTP(c *gin.Context)
	DisableTOTP(c *gin.Context)
	ListSessions(c *gin.Context)
	DeleteSession(c *gin.Context)
	Me(c *gin.Context)
	Deactivate(c *gin.Context)
	DeleteAccount(c *gin.Context)
}

type Handler struct {
	svc    IService
	logger *zap.Logger
	cookie CookieConfig
}

func NewHandler(svc IService, logger *zap.Logger, cookieCfg CookieConfig) IHandler {
	return &Handler{
		svc:    svc,
		logger: logger.With(zap.String("layer", "auth.handler")),
		cookie: cookieCfg.withDefaults(),
	}
}

func (h *Handler) RegisterRoutes(public *gin.RouterGroup, protected *gin.RouterGroup) {
	access := h.accessLog()

	pub := public.Group("/auth")
	pub.Use(access)
	{
		pub.POST("/register", h.Register)
		pub.POST("/login", h.Login)
		pub.POST("/oauth/google", h.OAuthGoogle)
		pub.POST("/oauth/github", h.OAuthGithub)
		pub.POST("/oauth/facebook", h.OAuthFacebook)
		pub.POST("/refresh", h.Refresh)
		pub.POST("/verify-email", h.VerifyEmail)
		pub.POST("/resend-verification", h.ResendVerification)
		pub.POST("/forgot-password", h.ForgotPassword)
		pub.POST("/reset-password", h.ResetPassword)
		pub.GET("/.well-known/jwks.json", h.JWKS)
	}

	pri := protected.Group("/auth")
	pri.Use(access)
	{
		pri.POST("/logout", h.Logout)
		pri.POST("/logout-all", h.LogoutAll)
		pri.POST("/password", h.ChangePassword)
		pri.POST("/totp/setup", h.SetupTOTP)
		pri.POST("/totp/enable", h.EnableTOTP)
		pri.POST("/totp/disable", h.DisableTOTP)
		pri.GET("/sessions", h.ListSessions)
		pri.DELETE("/sessions/:id", h.DeleteSession)
		pri.GET("/me", h.Me)
		pri.POST("/deactivate", h.Deactivate)
		pri.DELETE("/account", h.DeleteAccount)
	}
}

// Register godoc
// @Summary      Register a new user
// @Description  Creates a user account and returns an initial session. Sends an email verification token.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      RegisterRequest  true  "Registration payload"
// @Success      201   {object}  response.Response{data=SessionResponse}
// @Failure      400   {object}  response.Response
// @Failure      409   {object}  response.Response  "email already registered"
// @Failure      429   {object}  response.Response
// @Failure      500   {object}  response.Response
// @Router       /auth/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	resp, err := h.svc.Register(c.Request.Context(), req, deviceFromCtx(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.issueSession(c, resp, http.StatusCreated)
}

// Login godoc
// @Summary      Log in with email + password
// @Description  Returns access/refresh tokens. If TOTP is enabled and no code is provided, responds with requires_totp=true.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      LoginRequest  true  "Login payload"
// @Success      200   {object}  response.Response{data=SessionResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response  "invalid credentials or TOTP"
// @Failure      403   {object}  response.Response  "user blocked"
// @Failure      429   {object}  response.Response
// @Router       /auth/login [post]
func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	resp, err := h.svc.Login(c.Request.Context(), req, deviceFromCtx(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.issueSession(c, resp, http.StatusOK)
}

// OAuthFacebook godoc
// @Summary      Sign in with Facebook
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      FacebookOAuthRequest  true  "Facebook access token"
// @Success      200   {object}  response.Response{data=SessionResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /auth/oauth/facebook [post]
func (h *Handler) OAuthFacebook(c *gin.Context) {
	var req FacebookOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	resp, err := h.svc.OAuthFacebook(c.Request.Context(), req.AccessToken, deviceFromCtx(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.issueSession(c, resp, http.StatusOK)
}

// OAuthGithub godoc
// @Summary      Sign in with GitHub
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      GithubOAuthRequest  true  "GitHub OAuth code"
// @Success      200   {object}  response.Response{data=SessionResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /auth/oauth/github [post]
func (h *Handler) OAuthGithub(c *gin.Context) {
	var req GithubOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	resp, err := h.svc.OAuthGithub(c.Request.Context(), req.Code, deviceFromCtx(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.issueSession(c, resp, http.StatusOK)
}

// OAuthGoogle godoc
// @Summary      Sign in with Google
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      GoogleOAuthRequest  true  "Google ID token"
// @Success      200   {object}  response.Response{data=SessionResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /auth/oauth/google [post]
func (h *Handler) OAuthGoogle(c *gin.Context) {
	var req GoogleOAuthRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	resp, err := h.svc.OAuthGoogle(c.Request.Context(), req.IDToken, deviceFromCtx(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.issueSession(c, resp, http.StatusOK)
}

// Refresh godoc
// @Summary      Rotate refresh token
// @Description  Revokes the supplied refresh token and issues a fresh access/refresh pair. The refresh token is read from the httpOnly cookie when present; otherwise the request body is used.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      RefreshTokenRequest  false  "Refresh payload (omit if using cookie)"
// @Success      200   {object}  response.Response{data=SessionResponse}
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response  "invalid or rotated refresh token"
// @Router       /auth/refresh [post]
func (h *Handler) Refresh(c *gin.Context) {
	token, ok := h.extractRefreshToken(c)
	if !ok {
		return
	}
	resp, err := h.svc.Refresh(c.Request.Context(), token, deviceFromCtx(c))
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.issueSession(c, resp, http.StatusOK)
}

// VerifyEmail godoc
// @Summary      Verify email via token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      VerifyEmailRequest  true  "Verification token"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response  "invalid or expired token"
// @Router       /auth/verify-email [post]
func (h *Handler) VerifyEmail(c *gin.Context) {
	var req VerifyEmailRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.VerifyEmail(c.Request.Context(), req.Token); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ResendVerification godoc
// @Summary      Resend the email verification message
// @Description  Always responds 200 to avoid leaking which addresses have an account.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      ResendVerificationRequest  true  "Email"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Router       /auth/resend-verification [post]
func (h *Handler) ResendVerification(c *gin.Context) {
	var req ResendVerificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.ResendVerification(c.Request.Context(), req.Email); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ForgotPassword godoc
// @Summary      Start password reset flow
// @Description  Always responds 200 to avoid account enumeration.
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      ForgotPasswordRequest  true  "Email"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      429   {object}  response.Response
// @Router       /auth/forgot-password [post]
func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.ForgotPassword(c.Request.Context(), req.Email); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ResetPassword godoc
// @Summary      Complete password reset using emailed token
// @Tags         auth
// @Accept       json
// @Produce      json
// @Param        body  body      ResetPasswordRequest  true  "Reset payload"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response  "invalid or expired token"
// @Router       /auth/reset-password [post]
func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.ResetPassword(c.Request.Context(), req.Token, req.Password); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// JWKS godoc
// @Summary      Public JWKS for token verification
// @Tags         auth
// @Produce      json
// @Success      200  {object}  JWKResponse
// @Router       /auth/.well-known/jwks.json [get]
func (h *Handler) JWKS(c *gin.Context) {
	c.JSON(http.StatusOK, h.svc.JWKS())
}

// Logout godoc
// @Summary      Revoke the current refresh token
// @Description  Reads the refresh token from the httpOnly cookie when present; otherwise from the request body. Clears the cookie on success.
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      LogoutRequest  false  "Refresh token (omit if using cookie)"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /auth/logout [post]
func (h *Handler) Logout(c *gin.Context) {
	token, ok := h.extractRefreshToken(c)
	if !ok {
		return
	}
	if err := h.svc.Logout(c.Request.Context(), token); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.OK(c, nil)
}

// LogoutAll godoc
// @Summary      Revoke every active session for the caller
// @Description  Revokes all sessions belonging to the caller and clears the refresh cookie on success.
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/logout-all [post]
func (h *Handler) LogoutAll(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	if err := h.svc.LogoutAll(c.Request.Context(), userID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	h.clearRefreshCookie(c)
	response.OK(c, nil)
}

// ChangePassword godoc
// @Summary      Change the current user's password
// @Description  Revokes other sessions on success; the calling session is kept.
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      ChangePasswordRequest  true  "Old + new password"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Router       /auth/password [post]
func (h *Handler) ChangePassword(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.ChangePassword(c.Request.Context(), userID, req.CurrentPassword, req.NewPassword); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// SetupTOTP godoc
// @Summary      Generate a TOTP secret + provisioning URL
// @Description  Secret is stored unverified; call /auth/totp/enable with a code to activate.
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=TOTPSetupResponse}
// @Failure      401  {object}  response.Response
// @Failure      409  {object}  response.Response  "totp already enabled"
func (h *Handler) SetupTOTP(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	resp, err := h.svc.SetupTOTP(c.Request.Context(), userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, resp)
}

// EnableTOTP godoc
// @Summary      Activate TOTP after verifying a code
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      VerifyTOTPRequest  true  "TOTP code"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response  "invalid code"
// @Failure      404   {object}  response.Response  "totp not configured"
// @Router       /auth/totp/enable [post]
func (h *Handler) EnableTOTP(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.EnableTOTP(c.Request.Context(), userID, req.TOTPCode); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// DisableTOTP godoc
// @Summary      Disable TOTP after verifying a code
// @Tags         auth
// @Security     BearerAuth
// @Accept       json
// @Produce      json
// @Param        body  body      VerifyTOTPRequest  true  "TOTP code"
// @Success      200   {object}  response.Response
// @Failure      400   {object}  response.Response
// @Failure      401   {object}  response.Response
// @Failure      404   {object}  response.Response  "totp not configured"
// @Router       /auth/totp/disable [post]
func (h *Handler) DisableTOTP(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	var req VerifyTOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	if err := h.svc.DisableTOTP(c.Request.Context(), userID, req.TOTPCode); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// ListSessions godoc
// @Summary      List active sessions for the current user
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=[]UserSession}
// @Failure      401  {object}  response.Response
// @Router       /auth/sessions [get]
func (h *Handler) ListSessions(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	sessions, err := h.svc.ListSessions(c.Request.Context(), userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, sessions)
}

// DeleteSession godoc
// @Summary      Revoke a specific session by ID
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Param        id   path      string  true  "Session UUID"
// @Success      200  {object}  response.Response
// @Failure      400  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response  "session not found"
// @Router       /auth/sessions/{id} [delete]
func (h *Handler) DeleteSession(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	sessionID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "invalid session id")
		return
	}
	if err := h.svc.DeleteSession(c.Request.Context(), userID, sessionID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// Me godoc
// @Summary      Get the current user's profile
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response{data=UserResponse}
// @Failure      401  {object}  response.Response
// @Failure      404  {object}  response.Response
// @Router       /auth/me [get]
func (h *Handler) Me(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	user, err := h.svc.GetUser(c.Request.Context(), userID)
	if err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, user)
}

// Deactivate godoc
// @Summary      Soft-deactivate the current account
// @Description  All sessions are revoked. Logging in again reactivates the account.
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/deactivate [post]
func (h *Handler) Deactivate(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	if err := h.svc.DeactivateAccount(c.Request.Context(), userID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

// DeleteAccount godoc
// @Summary      Schedule the current account for deletion
// @Tags         auth
// @Security     BearerAuth
// @Produce      json
// @Success      200  {object}  response.Response
// @Failure      401  {object}  response.Response
// @Router       /auth/account [delete]
func (h *Handler) DeleteAccount(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c)
		return
	}
	if err := h.svc.DeleteAccount(c.Request.Context(), userID); err != nil {
		h.respondServiceError(c, err)
		return
	}
	response.OK(c, nil)
}

func (h *Handler) accessLog() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		latency := time.Since(start)

		fields := []zap.Field{
			zap.String("route", c.FullPath()),
			zap.String("method", c.Request.Method),
			zap.Int("status", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
		}

		if uid := middleware.GetUserID(c); uid != uuid.Nil {
			fields = append(fields, zap.String("user_id", uid.String()))
		}

		if rid := middleware.RequestIDFromContext(c.Request.Context()); rid != uuid.Nil {
			fields = append(fields, zap.String("request_id", rid.String()))
		}

		if len(c.Errors) > 0 {
			err := make([]error, 0, len(c.Errors))
			for _, e := range c.Errors {
				err = append(err, e.Err)
			}
			fields = append(fields, zap.Errors("errors", err))
		}

		logger := h.logger.Info

		switch status := c.Writer.Status(); {
		case status >= 500:
			logger = h.logger.Error
		case status >= 400:
			logger = h.logger.Warn
		}

		logger("auth api access", fields...)
	}
}

func (h *Handler) setRefreshCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    token,
		Path:     h.cookie.Path,
		Domain:   h.cookie.Domain,
		MaxAge:   int(h.cookie.MaxAge.Seconds()),
		Secure:   h.cookie.Secure,
		HttpOnly: true,
		SameSite: h.cookie.SameSite,
	})
}

func (h *Handler) clearRefreshCookie(c *gin.Context) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     h.cookie.Name,
		Value:    "",
		Path:     h.cookie.Path,
		Domain:   h.cookie.Domain,
		MaxAge:   -1,
		Secure:   h.cookie.Secure,
		HttpOnly: true,
		SameSite: h.cookie.SameSite,
	})
}

func (h *Handler) refreshTokenFromCookie(c *gin.Context) string {
	v, err := c.Cookie(h.cookie.Name)
	if err != nil {
		return ""
	}
	return v
}

func (h *Handler) extractRefreshToken(c *gin.Context) (string, bool) {
	if tok := h.refreshTokenFromCookie(c); tok != "" {
		return tok, true
	}
	var req struct {
		RefreshToken string `json:"refresh_token"`
	}
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			response.BadRequest(c, err.Error())
			return "", false
		}
	}
	if req.RefreshToken == "" {
		response.BadRequest(c, "refresh token is required")
		return "", false
	}
	return req.RefreshToken, true
}

func (h *Handler) issueSession(c *gin.Context, resp *SessionResponse, status int) {
	if resp != nil && resp.RefreshToken != "" {
		h.setRefreshCookie(c, resp.RefreshToken)
	}
	switch status {
	case http.StatusCreated:
		response.Created(c, resp)
	default:
		// mfa skip
		response.OK(c, resp)
	}
}

func (h *Handler) respondServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, ErrEmailExists):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrInvalidCredentials),
		errors.Is(err, ErrInvalidToken),
		errors.Is(err, ErrInvalidTOTP),
		errors.Is(err, ErrInvalidOAuth):
		response.Unauthorized(c)
	case errors.Is(err, ErrUserBlocked):
		response.Forbidden(c)
	case errors.Is(err, ErrUserNotFound),
		errors.Is(err, ErrSessionNotFound),
		errors.Is(err, ErrTOTPNotConfigured):
		response.NotFound(c, err.Error())
	case errors.Is(err, ErrTOTPAlreadyEnabled):
		response.Conflict(c, err.Error())
	case errors.Is(err, ErrPasswordNotSet):
		response.BadRequest(c, err.Error())
	case errors.Is(err, ErrOAuthNotConfigured):
		response.Error(c, http.StatusServiceUnavailable, "oauth_unavailable", err.Error())
	case errors.Is(err, ErrTOTPStoreUnavailable):
		response.Error(c, http.StatusServiceUnavailable, "totp_unavailable", err.Error())
	case errors.Is(err, ErrRateLimited):
		response.TooManyRequests(c)
	default:
		h.logger.Error("auth handler: internal error", zap.Error(err))
		response.InternalError(c, "internal error")
	}
}

func deviceFromCtx(c *gin.Context) DeviceInfo {
	return DeviceInfo{
		DeviceName: c.GetHeader("X-Device-Name"),
		DeviceType: parseDeviceType(c.GetHeader("X-Device-Type")),
		IPAddress:  c.ClientIP(),
		UserAgent:  c.Request.UserAgent(),
	}
}

func parseDeviceType(s string) DeviceType {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "ios":
		return DeviceIOS
	case "android":
		return DeviceAndroid
	default:
		return DeviceWeb
	}
}

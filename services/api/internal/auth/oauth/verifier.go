package oauth

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"crypto/rsa"

	"github.com/Ans1110/trip-app/internal/auth"
	"github.com/Ans1110/trip-app/pkg/config"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"
)

const httpTimeout = 10 * time.Second

const (
	googleJWKSURL   = "https://www.googleapis.com/oauth2/v3/certs"
	googleIssuer1   = "accounts.google.com"
	googleIssuer2   = "https://accounts.google.com"
	githubTokenURL  = "https://github.com/login/oauth/access_token"
	githubUserURL   = "https://api.github.com/user"
	githubEmailsURL = "https://api.github.com/user/emails"
)

type Verifier struct {
	cfg    config.OAuthConfig
	http   *http.Client
	logger *zap.Logger
	jwks   *jwksCache
}

func New(cfg config.OAuthConfig, logger *zap.Logger) auth.OAuthVerifier {
	client := &http.Client{Timeout: httpTimeout}
	return &Verifier{
		cfg:    cfg,
		http:   client,
		logger: logger.With(zap.String("component", "oauth")),
		jwks:   newJWKSCache(client, googleJWKSURL),
	}
}

// ---- Google ----

type googleClaims struct {
	jwt.RegisteredClaims
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Picture       string `json:"picture"`
	HD            string `json:"hd"`
}

func (v *Verifier) VerifyGoogle(ctx context.Context, idToken string) (*auth.OAuthIdentity, error) {
	if v.cfg.Google.ClientID == "" {
		return nil, auth.ErrOAuthNotConfigured
	}
	claims := &googleClaims{}
	tok, err := jwt.ParseWithClaims(idToken, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method %q", t.Method.Alg())
		}
		kid, _ := t.Header["kid"].(string)
		if kid == "" {
			return nil, errors.New("id token missing kid")
		}
		return v.jwks.key(ctx, kid)
	}, jwt.WithAudience(v.cfg.Google.ClientID), jwt.WithValidMethods([]string{"RS256"}))
	if err != nil || !tok.Valid {
		v.logger.Debug("google id token verify failed", zap.Error(err))
		return nil, auth.ErrInvalidOAuth
	}
	iss := claims.Issuer
	if iss != googleIssuer1 && iss != googleIssuer2 {
		return nil, auth.ErrInvalidOAuth
	}
	sub := claims.Subject
	if sub == "" {
		return nil, auth.ErrInvalidOAuth
	}
	return &auth.OAuthIdentity{
		Provider:      "google",
		ProviderID:    sub,
		Email:         claims.Email,
		EmailVerified: claims.EmailVerified,
		Name:          claims.Name,
		AvatarURL:     claims.Picture,
	}, nil
}

// ---- GitHub ----

type githubTokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type githubUser struct {
	ID        int64  `json:"id"`
	Login     string `json:"login"`
	Name      string `json:"name"`
	Email     string `json:"email"`
	AvatarURL string `json:"avatar_url"`
}

type githubEmail struct {
	Email    string `json:"email"`
	Primary  bool   `json:"primary"`
	Verified bool   `json:"verified"`
}

func (v *Verifier) VerifyGithub(ctx context.Context, code string) (*auth.OAuthIdentity, error) {
	if v.cfg.Github.ClientID == "" || v.cfg.Github.ClientSecret == "" {
		return nil, auth.ErrOAuthNotConfigured
	}
	token, err := v.exchangeGithubCode(ctx, code)
	if err != nil {
		v.logger.Debug("github code exchange failed", zap.Error(err))
		return nil, auth.ErrInvalidOAuth
	}
	user, err := v.fetchGithubUser(ctx, token)
	if err != nil {
		v.logger.Debug("github fetch user failed", zap.Error(err))
		return nil, auth.ErrInvalidOAuth
	}
	email, verified := v.fetchGithubPrimaryEmail(ctx, token)
	if email == "" {
		email = user.Email
	}
	name := user.Name
	if name == "" {
		name = user.Login
	}
	return &auth.OAuthIdentity{
		Provider:      "github",
		ProviderID:    strconv.FormatInt(user.ID, 10),
		Email:         email,
		EmailVerified: verified,
		Name:          name,
		AvatarURL:     user.AvatarURL,
	}, nil
}

func (v *Verifier) exchangeGithubCode(ctx context.Context, code string) (string, error) {
	body := url.Values{
		"client_id":     {v.cfg.Github.ClientID},
		"client_secret": {v.cfg.Github.ClientSecret},
		"code":          {code},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubTokenURL, strings.NewReader(body.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := v.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return "", fmt.Errorf("github token status %d", resp.StatusCode)
	}
	var out githubTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.Error != "" {
		return "", fmt.Errorf("github token error: %s", out.Error)
	}
	if out.AccessToken == "" {
		return "", errors.New("github token empty")
	}
	return out.AccessToken, nil
}

func (v *Verifier) fetchGithubUser(ctx context.Context, token string) (*githubUser, error) {
	var u githubUser
	if err := v.githubGet(ctx, githubUserURL, token, &u); err != nil {
		return nil, err
	}
	if u.ID == 0 {
		return nil, errors.New("github user missing id")
	}
	return &u, nil
}

func (v *Verifier) fetchGithubPrimaryEmail(ctx context.Context, token string) (string, bool) {
	var emails []githubEmail
	if err := v.githubGet(ctx, githubEmailsURL, token, &emails); err != nil {
		v.logger.Debug("github fetch emails failed", zap.Error(err))
		return "", false
	}
	for _, e := range emails {
		if e.Primary {
			return e.Email, e.Verified
		}
	}
	return "", false
}

func (v *Verifier) githubGet(ctx context.Context, url, token string, out any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	resp, err := v.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return fmt.Errorf("github %s status %d: %s", url, resp.StatusCode, string(b))
	}
	return json.NewDecoder(resp.Body).Decode(out)
}

// ---- Google JWKS cache ----

type jwksKey struct {
	Kid string `json:"kid"`
	Kty string `json:"kty"`
	Alg string `json:"alg"`
	N   string `json:"n"`
	E   string `json:"e"`
}

type jwksResponse struct {
	Keys []jwksKey `json:"keys"`
}

type jwksCache struct {
	url  string
	http *http.Client

	mu        sync.RWMutex
	keys      map[string]*rsa.PublicKey
	expiresAt time.Time
}

func newJWKSCache(client *http.Client, url string) *jwksCache {
	return &jwksCache{url: url, http: client}
}

func (c *jwksCache) key(ctx context.Context, kid string) (*rsa.PublicKey, error) {
	c.mu.RLock()
	if time.Now().Before(c.expiresAt) {
		if k, ok := c.keys[kid]; ok {
			c.mu.RUnlock()
			return k, nil
		}
	}
	c.mu.RUnlock()

	if err := c.refresh(ctx); err != nil {
		return nil, err
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	k, ok := c.keys[kid]
	if !ok {
		return nil, fmt.Errorf("jwks: kid %q not found", kid)
	}
	return k, nil
}

func (c *jwksCache) refresh(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.url, nil)
	if err != nil {
		return err
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("jwks fetch status %d", resp.StatusCode)
	}
	var body jwksResponse
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		return err
	}
	keys := make(map[string]*rsa.PublicKey, len(body.Keys))
	for _, k := range body.Keys {
		if k.Kty != "RSA" || k.N == "" || k.E == "" {
			continue
		}
		pub, err := jwkToRSA(k.N, k.E)
		if err != nil {
			continue
		}
		keys[k.Kid] = pub
	}
	// Cache Google's declared max-age; fall back to 1h if the header is absent.
	ttl := cacheMaxAge(resp.Header.Get("Cache-Control"))
	if ttl <= 0 {
		ttl = time.Hour
	}
	c.mu.Lock()
	c.keys = keys
	c.expiresAt = time.Now().Add(ttl)
	c.mu.Unlock()
	return nil
}

func jwkToRSA(nB64, eB64 string) (*rsa.PublicKey, error) {
	nBytes, err := base64.RawURLEncoding.DecodeString(nB64)
	if err != nil {
		return nil, err
	}
	eBytes, err := base64.RawURLEncoding.DecodeString(eB64)
	if err != nil {
		return nil, err
	}
	e := 0
	for _, b := range eBytes {
		e = e<<8 | int(b)
	}
	return &rsa.PublicKey{N: new(big.Int).SetBytes(nBytes), E: e}, nil
}

func cacheMaxAge(header string) time.Duration {
	for _, part := range strings.Split(header, ",") {
		part = strings.TrimSpace(part)
		if !strings.HasPrefix(part, "max-age=") {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(part, "max-age="))
		if err == nil && n > 0 {
			return time.Duration(n) * time.Second
		}
	}
	return 0
}

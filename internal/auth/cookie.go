package auth

import (
	"net/http"
	"time"
)

type CookieConfig struct {
	Name     string        // cookie name; defaults to "refresh_token"
	Path     string        // cookie path; defaults to "/api/v1/auth"
	Domain   string        // optional cookie domain
	MaxAge   time.Duration // cookie lifetime; defaults to 30 days
	Secure   bool          // require HTTPS — set true in production
	SameSite http.SameSite // defaults to http.SameSiteLaxMode
}

func (c CookieConfig) withDefaults() CookieConfig {
	if c.Name == "" {
		c.Name = "refresh_token"
	}
	if c.Path == "" {
		c.Path = "/api/v1/auth"
	}
	if c.MaxAge <= 0 {
		c.MaxAge = 30 * 24 * time.Hour
	}
	if c.SameSite == 0 {
		c.SameSite = http.SameSiteLaxMode
	}
	return c
}

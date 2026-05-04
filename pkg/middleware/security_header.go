package middleware

import "github.com/gin-gonic/gin"

func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("X-XSS-Protection", "1; mode=block")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		// API-only server — nothing needs to load external resources.
		h.Set("Content-Security-Policy", "default-src 'none'")
		// 1 year HSTS; set Preload when the domain is enrolled.
		h.Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
		// Prevent caching of API responses that carry user data.
		h.Set("Cache-Control", "no-store")
		h.Set("Pragma", "no-cache")
		c.Next()
	}
}

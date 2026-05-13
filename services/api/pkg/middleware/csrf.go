package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

const (
	csrfHeaderName = "X-CSRF-Token"
	csrfCookieName = "csrf_token"
	csrfTokenTTL   = 24 * time.Hour
	csrfKeyPrefix  = "csrf_token:"
)

func CSRFProtect(rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodHead, http.MethodOptions, http.MethodTrace:
			c.Next()
			return
		}

		ctx := c.Request.Context()

		token := c.GetHeader(csrfHeaderName)
		cookie, err := c.Cookie(csrfCookieName)
		if err != nil || token == "" {
			c.AbortWithStatusJSON(403, gin.H{"message": "CSRF token missing"})
			return
		}

		if token != cookie {
			c.AbortWithStatusJSON(403, gin.H{"message": "CSRF token mismatch"})
			return
		}

		exists, err := rdb.Exists(ctx, csrfKeyPrefix+token).Result()
		if err != nil || exists == 0 {
			c.AbortWithStatusJSON(403, gin.H{"message": "Invalid CSRF token"})
			return
		}

		c.Next()
	}
}

func CSRFTokenHandler(rdb *redis.Client, secure bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := generateCSRFToken()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"code":    500,
				"message": "Failed to generate CSRF token",
			})
			return
		}
		rdb.Set(context.Background(), csrfKeyPrefix+token, "1", csrfTokenTTL)
		setCSRFCookie(c, token, secure)
		c.JSON(http.StatusOK, gin.H{"csrf_token": token})
	}
}

func generateCSRFToken() (string, error) {
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(tokenBytes), nil
}

func setCSRFCookie(c *gin.Context, token string, secure bool) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     csrfCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   int(csrfTokenTTL.Seconds()),
		Secure:   secure,
		HttpOnly: false,
		SameSite: http.SameSiteStrictMode,
	})
}

package middleware

import (
	"crypto/rsa"
	"fmt"
	"strings"
	"time"

	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	ContextUserID    = "user_id"
	ContextRoles     = "roles"
	ContextJTI       = "jti"
	ContextSessionID = "sid"
)

type Claims struct {
	jwt.RegisteredClaims
	Email     string   `json:"email"`
	Roles     []string `json:"roles"`
	Provider  string   `json:"provider"`
	SessionID string   `json:"sid,omitempty"`
}

func JWTAuth(publicKey *rsa.PublicKey, rdb *redis.Client) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		token := extractBearerToken(c)
		if token == "" {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		claims, err := parseRS256Token(token, publicKey)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		if rdb != nil {
			blacklisted := fmt.Sprintf("jwt_blacklist:%s", claims.ID)
			exists, err := rdb.Exists(ctx, blacklisted).Result()
			if err != nil || exists > 0 {
				response.Unauthorized(c)
				c.Abort()
				return
			}
		}

		userID, err := uuid.Parse(claims.Subject)
		if err != nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}

		c.Set(ContextUserID, userID)
		c.Set(ContextRoles, claims.Roles)
		c.Set(ContextJTI, claims.ID)
		if claims.SessionID != "" {
			c.Set(ContextSessionID, claims.SessionID)
		}
		c.Next()
	}
}

func RequireRole(role string) gin.HandlerFunc {
	return func(c *gin.Context) {
		roles, _ := c.Get(ContextRoles)
		if roleSlice, ok := roles.([]string); ok {
			for _, r := range roleSlice {
				if r == role {
					c.Next()
					return
				}
			}
		}
		response.Forbidden(c)
		c.Abort()
	}
}

func GetUserID(c *gin.Context) uuid.UUID {
	if id, exists := c.Get(ContextUserID); exists {
		if userID, ok := id.(uuid.UUID); ok {
			return userID
		}
	}
	return uuid.Nil
}

func GetUserRoles(c *gin.Context) []string {
	if roles, exists := c.Get(ContextRoles); exists {
		if r, ok := roles.([]string); ok {
			return r
		}
	}
	return nil
}

func extractBearerToken(c *gin.Context) string {
	authHeader := c.GetHeader("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return ""
	}
	return strings.TrimPrefix(authHeader, "Bearer ")
}

func parseRS256Token(tokenStr string, publicKey *rsa.PublicKey) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("Unexpected signing method: %v", token.Header["alg"])
		}
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, fmt.Errorf("Unexpected signing algorithm: %v", token.Header["alg"])
		}
		return publicKey, nil
	})
	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("Invalid token")
	}
	if claims.ExpiresAt != nil && claims.ExpiresAt.Before(time.Now()) {
		return nil, fmt.Errorf("Token has expired")
	}
	return claims, nil
}

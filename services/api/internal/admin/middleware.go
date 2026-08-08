package admin

import (
	"github.com/Ans1110/trip-app/pkg/middleware"
	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RequireAdminElevated gates a route behind a live Redis-backed elevation flag.
// JWTAuth must have already run; the flag is keyed by user id, minted by
// POST /admin/auth/elevate, and expires after elevationTTL.
func RequireAdminElevated(gate *ElevationGate) gin.HandlerFunc {
	return func(c *gin.Context) {
		if gate == nil || !gate.Configured() {
			response.Forbidden(c)
			c.Abort()
			return
		}
		uid := middleware.GetUserID(c)
		if uid == uuid.Nil {
			response.Unauthorized(c)
			c.Abort()
			return
		}
		if !gate.IsAdminEmail(middleware.GetUserEmail(c)) {
			response.Forbidden(c)
			c.Abort()
			return
		}
		ok, err := gate.IsElevated(c.Request.Context(), uid)
		if err != nil {
			response.InternalError(c, "elevation check failed")
			c.Abort()
			return
		}
		if !ok {
			response.Forbidden(c)
			c.Abort()
			return
		}
		c.Next()
	}
}

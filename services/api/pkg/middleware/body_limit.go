package middleware

import (
	"net/http"
	"strings"

	"github.com/Ans1110/trip-app/pkg/response"
	"github.com/gin-gonic/gin"
)

func BodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			response.BadRequest(c, "request body too large")
			c.Abort()
		}
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		if err := c.Errors.Last(); err != nil {
			if strings.Contains(err.Error(), "http: request body too large") {
				response.BadRequest(c, "request body too large")
				c.Abort()
			}
		}
	}
}

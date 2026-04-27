package requestid

import (
	"crypto/rand"
	"encoding/hex"

	"github.com/gin-gonic/gin"
)

const (
	// HeaderName is the request id header used by QuantSage.
	HeaderName = "X-Request-ID"
	// ContextKey is the gin context key used to store request ids.
	ContextKey = "request_id"
)

// New returns a random request id suitable for logs and headers.
func New() string {
	buf := make([]byte, 12)
	if _, err := rand.Read(buf); err != nil {
		return "request-id-unavailable"
	}

	return hex.EncodeToString(buf)
}

// FromContext returns the current request id from gin context.
func FromContext(c *gin.Context) string {
	if v, ok := c.Get(ContextKey); ok {
		if id, ok := v.(string); ok {
			return id
		}
	}

	return ""
}

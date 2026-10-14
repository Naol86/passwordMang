package httpapi

import (
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	headerRequestID = "X-Request-ID"
	ctxKeyRequestID = "requestID"
	// Item and environment payloads are small ciphertexts; attachments get their own endpoint and limit.
	maxBodyBytes = 1 << 20
)

// requestID gives every request a server-generated ID. Incoming IDs are ignored
// so clients cannot forge entries in logs.
func requestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := uuid.NewString()
		c.Set(ctxKeyRequestID, id)
		c.Header(headerRequestID, id)
		c.Next()
	}
}

func requestIDFrom(c *gin.Context) string {
	return c.GetString(ctxKeyRequestID)
}

func bodyLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Body != nil {
			c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		}
		c.Next()
	}
}

func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		h.Set("Cache-Control", "no-store")
		h.Set("Content-Security-Policy", "default-src 'none'; frame-ancestors 'none'")
		c.Next()
	}
}

// requestLogger never logs bodies, query strings or headers, so ciphertext and tokens stay out of logs (SEC-13).
func requestLogger(log *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		log.InfoContext(c.Request.Context(), "http request",
			"request_id", requestIDFrom(c),
			"method", c.Request.Method,
			"route", routeLabel(c),
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}

func recovery(log *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered any) {
		writeError(c, log, fmt.Errorf("panic: %v", recovered))
	})
}

// routeLabel uses the route template, never the raw path, so IDs in URLs stay out of logs and metrics.
func routeLabel(c *gin.Context) string {
	if route := c.FullPath(); route != "" {
		return route
	}
	return "unmatched"
}

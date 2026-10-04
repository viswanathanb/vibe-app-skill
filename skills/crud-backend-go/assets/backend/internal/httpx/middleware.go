package httpx

import (
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// CSRFGuard requires X-Requested-With on state-changing requests. Browsers cannot send this
// header cross-site without a CORS preflight (which we never allow), so cookie auth is safe.
func CSRFGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		switch c.Request.Method {
		case http.MethodGet, http.MethodHead, http.MethodOptions:
		default:
			if c.GetHeader("X-Requested-With") != "XMLHttpRequest" {
				Fail(c, Forbidden("missing X-Requested-With header"))
				return
			}
		}
		c.Next()
	}
}

func SecurityHeaders(prod bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.Writer.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		if prod {
			h.Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
			h.Set("Content-Security-Policy", "default-src 'self'; img-src 'self' data:; "+
				"style-src 'self' 'unsafe-inline'; script-src 'self'; connect-src 'self'; "+
				"frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
		}
		c.Next()
	}
}

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		slog.InfoContext(c.Request.Context(), "http",
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", time.Since(start).Milliseconds(),
		)
	}
}

// Limiter is a fixed-window in-memory rate limiter. It is per-process: fine for a single
// Render instance; use Postgres/Redis-backed limiting if you scale out.
type Limiter struct {
	mu     sync.Mutex
	max    int
	window time.Duration
	hits   map[string]*windowState
}

type windowState struct {
	count int
	reset time.Time
}

func NewLimiter(max int, window time.Duration) *Limiter {
	return &Limiter{max: max, window: window, hits: map[string]*windowState{}}
}

func (l *Limiter) Allow(key string) bool {
	now := time.Now()
	l.mu.Lock()
	defer l.mu.Unlock()
	if len(l.hits) > 10_000 {
		for k, w := range l.hits {
			if now.After(w.reset) {
				delete(l.hits, k)
			}
		}
	}
	w, ok := l.hits[key]
	if !ok || now.After(w.reset) {
		l.hits[key] = &windowState{count: 1, reset: now.Add(l.window)}
		return true
	}
	if w.count >= l.max {
		return false
	}
	w.count++
	return true
}

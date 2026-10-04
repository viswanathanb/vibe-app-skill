package auth

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"example.com/app/internal/authz"
	"example.com/app/internal/httpx"
	"example.com/app/internal/user"
)

// CookieName holds the session JWT (HttpOnly, SameSite=Lax, Secure in production).
const CookieName = "session"

const userKey = "auth.user"

// RequireUser authenticates the request from the session cookie (browser) or an
// `Authorization: Bearer <jwt>` header (scripts), loads the user, and sets the authz.Principal.
func RequireUser(tokens *TokenIssuer, users *user.Repository) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := bearerToken(c.Request)
		if raw == "" {
			raw, _ = c.Cookie(CookieName)
		}
		if raw == "" {
			httpx.Fail(c, httpx.Unauthorized("authentication required"))
			return
		}
		id, err := tokens.Parse(raw)
		if err != nil {
			httpx.Fail(c, httpx.Unauthorized("invalid or expired session"))
			return
		}
		u, err := users.FindByID(c.Request.Context(), id)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			httpx.Fail(c, httpx.Unauthorized("invalid or expired session"))
			return
		}
		if err != nil {
			httpx.Fail(c, err)
			return
		}
		if u.Disabled {
			httpx.Fail(c, httpx.Unauthorized("account disabled"))
			return
		}
		c.Set(userKey, u)
		authz.SetPrincipal(c, u.Principal())
		c.Next()
	}
}

// CurrentUser returns the authenticated user (nil outside RequireUser).
func CurrentUser(c *gin.Context) *user.User {
	v, _ := c.Get(userKey)
	u, _ := v.(*user.User)
	return u
}

func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if token, ok := strings.CutPrefix(h, "Bearer "); ok {
		return strings.TrimSpace(token)
	}
	return ""
}

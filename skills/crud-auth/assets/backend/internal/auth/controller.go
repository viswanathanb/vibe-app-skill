package auth

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"example.com/app/internal/authz"
	"example.com/app/internal/config"
	"example.com/app/internal/httpx"
	"example.com/app/internal/user"
)

type Controller struct {
	svc *Service
	cfg config.Config
}

func NewController(svc *Service, cfg config.Config) *Controller {
	return &Controller{svc: svc, cfg: cfg}
}

// Register mounts public routes on `public` and session routes on `authed`.
func (ctl *Controller) Register(public, authed *gin.RouterGroup) {
	g := public.Group("/auth")
	g.GET("/config", ctl.config)
	g.POST("/signup", ctl.signup)
	g.POST("/login", ctl.login)
	g.POST("/logout", ctl.logout)
	authed.GET("/auth/me", ctl.me)
}

type meResponse struct {
	User    *user.User `json:"user"`
	Actions []string   `json:"actions"`
}

func (ctl *Controller) config(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"signupEnabled": ctl.cfg.AllowSignup})
}

func (ctl *Controller) signup(c *gin.Context) {
	var in SignupInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	u, token, err := ctl.svc.Signup(c.Request.Context(), in)
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	ctl.setSession(c, token)
	c.JSON(http.StatusCreated, meResponse{User: u, Actions: authz.ActionsFor(u.Role)})
}

func (ctl *Controller) login(c *gin.Context) {
	var in LoginInput
	if !httpx.BindJSON(c, &in) {
		return
	}
	u, token, err := ctl.svc.Login(c.Request.Context(), in, c.ClientIP())
	if err != nil {
		httpx.Fail(c, err)
		return
	}
	ctl.setSession(c, token)
	c.JSON(http.StatusOK, meResponse{User: u, Actions: authz.ActionsFor(u.Role)})
}

func (ctl *Controller) logout(c *gin.Context) {
	http.SetCookie(c.Writer, ctl.cookie("", -1))
	c.Status(http.StatusNoContent)
}

func (ctl *Controller) me(c *gin.Context) {
	u := CurrentUser(c)
	c.JSON(http.StatusOK, meResponse{User: u, Actions: authz.ActionsFor(u.Role)})
}

func (ctl *Controller) setSession(c *gin.Context, token string) {
	http.SetCookie(c.Writer, ctl.cookie(token, int(ctl.svc.tokens.TTL().Seconds())))
}

func (ctl *Controller) cookie(value string, maxAge int) *http.Cookie {
	return newCookie(CookieName, "/", value, maxAge, ctl.cfg.IsProd())
}

// newCookie builds every auth cookie: HttpOnly, SameSite=Lax, Secure in production.
func newCookie(name, path, value string, maxAge int, secure bool) *http.Cookie {
	return &http.Cookie{ //nolint:gosec // G124: Secure is on in production; local dev is plain http://localhost
		Name:     name,
		Value:    value,
		Path:     path,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	}
}

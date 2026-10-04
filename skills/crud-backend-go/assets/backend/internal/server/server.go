package server

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"example.com/app/internal/auth"
	"example.com/app/internal/authz"
	"example.com/app/internal/config"
	"example.com/app/internal/httpx"
	"example.com/app/internal/team"
	"example.com/app/internal/user"
)

// Models lists every GORM model for AutoMigrate (tables are created/updated on startup).
func Models() []any {
	return []any{
		&user.User{},
		&authz.Tuple{},
		&team.Team{},
		// crud:models (add new resource models above this line)
	}
}

// App is the wired HTTP application.
type App struct {
	Handler http.Handler
	auth    *auth.Service
}

// Bootstrap runs one-off startup tasks that need the database (e.g. creating the first admin).
func (a *App) Bootstrap(ctx context.Context) error {
	return a.auth.EnsureBootstrapAdmin(ctx)
}

func New(cfg config.Config, db *gorm.DB) (*App, error) {
	if cfg.IsProd() {
		gin.SetMode(gin.ReleaseMode)
	}
	httpx.UseJSONFieldNames()

	r := gin.New()
	if err := r.SetTrustedProxies(nil); err != nil {
		return nil, err
	}
	r.Use(gin.Recovery(), httpx.RequestLogger(), httpx.SecurityHeaders(cfg.IsProd()))
	r.GET("/healthz", health(db))

	// Shared dependencies.
	az, err := authz.NewEngine(authz.AppSchema, authz.NewGormStore(db))
	if err != nil {
		return nil, err
	}
	users := user.NewRepository(db)
	tokens := auth.NewTokenIssuer(cfg.JWTSecret, cfg.JWTTTL)
	authSvc := auth.NewService(users, tokens, cfg)

	api := r.Group("/api", httpx.CSRFGuard())
	authed := api.Group("", auth.RequireUser(tokens, users))

	auth.NewController(authSvc, cfg).Register(api, authed)
	user.NewController(user.NewService(users)).Register(authed)
	authz.NewController(az, users).Register(authed)
	team.NewController(team.NewService(db, team.NewRepository(db), az)).Register(authed)
	// crud:routes (register new resource controllers on `authed` above this line)

	registerSPA(r, cfg.StaticDir)
	return &App{Handler: r, auth: authSvc}, nil
}

func health(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()
		sqlDB, err := db.DB()
		if err == nil {
			err = sqlDB.PingContext(ctx)
		}
		if err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}

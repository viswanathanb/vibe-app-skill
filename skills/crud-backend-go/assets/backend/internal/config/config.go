package config

import (
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is loaded once from environment variables at startup.
type Config struct {
	Env         string // "development" or "production"
	Port        string
	DatabaseURL string
	StaticDir   string // built SPA directory; empty in local dev (Vite serves the UI)

	JWTSecret   string
	JWTTTL      time.Duration
	AllowSignup bool

	BootstrapAdminEmail    string
	BootstrapAdminPassword string
}

func Load() (Config, error) {
	cfg := Config{
		Env:                    getenv("APP_ENV", "development"),
		Port:                   getenv("PORT", "8080"),
		DatabaseURL:            os.Getenv("DATABASE_URL"),
		StaticDir:              os.Getenv("STATIC_DIR"),
		JWTSecret:              os.Getenv("JWT_SECRET"),
		BootstrapAdminEmail:    strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN_EMAIL")),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}

	ttl, err := time.ParseDuration(getenv("JWT_TTL", "24h"))
	if err != nil {
		return cfg, fmt.Errorf("JWT_TTL: %w", err)
	}
	cfg.JWTTTL = ttl

	// Open signup is convenient locally; production defaults to invite-only (admins create users).
	cfg.AllowSignup, err = strconv.ParseBool(getenv("ALLOW_SIGNUP", strconv.FormatBool(!cfg.IsProd())))
	if err != nil {
		return cfg, fmt.Errorf("ALLOW_SIGNUP: %w", err)
	}

	if cfg.DatabaseURL == "" {
		return cfg, errors.New("DATABASE_URL is required")
	}
	if len(cfg.JWTSecret) < 32 {
		if cfg.IsProd() {
			return cfg, errors.New("JWT_SECRET must be at least 32 characters in production")
		}
		slog.Warn("JWT_SECRET missing or short; using an insecure development secret")
		cfg.JWTSecret = "dev-only-insecure-secret-change-me-0123456789"
	}
	if (cfg.BootstrapAdminEmail == "") != (cfg.BootstrapAdminPassword == "") {
		return cfg, errors.New("set both BOOTSTRAP_ADMIN_EMAIL and BOOTSTRAP_ADMIN_PASSWORD, or neither")
	}
	return cfg, nil
}

func (c Config) IsProd() bool { return c.Env == "production" }

func getenv(key, fallback string) string {
	if v, ok := os.LookupEnv(key); ok && v != "" {
		return v
	}
	return fallback
}

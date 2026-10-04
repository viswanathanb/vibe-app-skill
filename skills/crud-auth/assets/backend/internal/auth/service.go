package auth

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"gorm.io/gorm"

	"example.com/app/internal/authz"
	"example.com/app/internal/config"
	"example.com/app/internal/httpx"
	"example.com/app/internal/user"
)

type Service struct {
	users   *user.Repository
	tokens  *TokenIssuer
	cfg     config.Config
	limiter *httpx.Limiter
}

func NewService(users *user.Repository, tokens *TokenIssuer, cfg config.Config) *Service {
	return &Service{users: users, tokens: tokens, cfg: cfg, limiter: httpx.NewLimiter(10, 15*time.Minute)}
}

type SignupInput struct {
	Email    string `json:"email" binding:"required,email,max=320"`
	Password string `json:"password" binding:"required,min=8,max=72"`
	Name     string `json:"name" binding:"max=200"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email,max=320"`
	Password string `json:"password" binding:"required,max=72"`
}

// Signup creates a member account. In development the very first user becomes admin;
// in production the first admin comes from BOOTSTRAP_ADMIN_EMAIL/PASSWORD instead.
func (s *Service) Signup(ctx context.Context, in SignupInput) (*user.User, string, error) {
	if !s.cfg.AllowSignup {
		return nil, "", httpx.Forbidden("signup is disabled; ask an admin for an account")
	}
	hash, err := user.HashPassword(in.Password)
	if err != nil {
		return nil, "", err
	}
	role := authz.RoleMember
	if !s.cfg.IsProd() {
		n, err := s.users.Count(ctx)
		if err != nil {
			return nil, "", err
		}
		if n == 0 {
			role = authz.RoleAdmin
		}
	}
	u := &user.User{Email: in.Email, Name: strings.TrimSpace(in.Name), PasswordHash: hash, Role: role}
	if err := s.users.Create(ctx, u); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, "", httpx.Conflict("an account with this email already exists")
		}
		return nil, "", err
	}
	token, err := s.tokens.Issue(u.ID)
	return u, token, err
}

// Login verifies credentials. clientKey (e.g. client IP) scopes the brute-force limiter.
func (s *Service) Login(ctx context.Context, in LoginInput, clientKey string) (*user.User, string, error) {
	email := user.NormalizeEmail(in.Email)
	if !s.limiter.Allow(clientKey + "|" + email) {
		return nil, "", httpx.TooManyRequests("too many login attempts; try again later")
	}
	invalid := httpx.Unauthorized("invalid email or password")
	u, err := s.users.FindByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		user.SimulatePasswordCheck(in.Password)
		return nil, "", invalid
	}
	if err != nil {
		return nil, "", err
	}
	if !user.CheckPassword(u.PasswordHash, in.Password) {
		return nil, "", invalid
	}
	if u.Disabled {
		return nil, "", httpx.Forbidden("account disabled")
	}
	token, err := s.tokens.Issue(u.ID)
	return u, token, err
}

// EnsureBootstrapAdmin creates the BOOTSTRAP_ADMIN_EMAIL admin on startup if it does not exist.
func (s *Service) EnsureBootstrapAdmin(ctx context.Context) error {
	if s.cfg.BootstrapAdminEmail == "" {
		return nil
	}
	_, err := s.users.FindByEmail(ctx, s.cfg.BootstrapAdminEmail)
	if err == nil || !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	if len(s.cfg.BootstrapAdminPassword) < 8 || len(s.cfg.BootstrapAdminPassword) > 72 {
		return errors.New("BOOTSTRAP_ADMIN_PASSWORD must be 8-72 characters")
	}
	hash, err := user.HashPassword(s.cfg.BootstrapAdminPassword)
	if err != nil {
		return err
	}
	u := &user.User{Email: s.cfg.BootstrapAdminEmail, Name: "Admin", PasswordHash: hash, Role: authz.RoleAdmin}
	if err := s.users.Create(ctx, u); err != nil && !errors.Is(err, gorm.ErrDuplicatedKey) {
		return err
	}
	slog.Info("bootstrap admin created", "email", u.Email)
	return nil
}

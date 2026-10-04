package auth

// OPTIONAL ENHANCEMENT: "Sign in with <provider>" via OpenID Connect (Google, Microsoft Entra ID,
// Okta, Auth0, Keycloak, GitHub via a broker...). Copy to backend/internal/auth/oidc.go and follow
// crud-auth/references/oidc.md. Needs: go get github.com/coreos/go-oidc/v3/oidc golang.org/x/oauth2

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/coreos/go-oidc/v3/oidc"
	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"example.com/app/internal/authz"
	"example.com/app/internal/user"
)

const oidcFlowCookie = "oidc_flow"

// OIDCConfig comes from OIDC_ISSUER, OIDC_CLIENT_ID, OIDC_CLIENT_SECRET, OIDC_REDIRECT_URL, OIDC_NAME.
type OIDCConfig struct {
	Name         string // button label, e.g. "Google"
	Issuer       string // e.g. https://accounts.google.com
	ClientID     string
	ClientSecret string
	RedirectURL  string // https://<host>/api/auth/oidc/callback
	// TrustEmail accepts tokens without email_verified=true. Only for a single-tenant company IdP
	// (e.g. one Entra ID tenant) where admins control every address; never for multi-tenant/public IdPs.
	TrustEmail bool
}

// UserIdentity links an external account (issuer + subject) to a local user. Add it to server.Models().
type UserIdentity struct {
	ID        uint   `gorm:"primaryKey"`
	UserID    uint   `gorm:"not null;index"`
	Issuer    string `gorm:"size:300;not null;uniqueIndex:ux_identity,priority:1"`
	Subject   string `gorm:"size:300;not null;uniqueIndex:ux_identity,priority:2"`
	CreatedAt time.Time
}

type OIDC struct {
	name       string
	trustEmail bool
	provider   *oidc.Provider
	verifier   *oidc.IDTokenVerifier
	oauth      oauth2.Config
	db         *gorm.DB
	users      *user.Repository
	svc        *Service
	secure     bool
}

func NewOIDC(ctx context.Context, cfg OIDCConfig, db *gorm.DB, users *user.Repository, svc *Service, secure bool) (*OIDC, error) {
	provider, err := oidc.NewProvider(ctx, cfg.Issuer)
	if err != nil {
		return nil, fmt.Errorf("oidc discovery: %w", err)
	}
	return &OIDC{
		name:       cfg.Name,
		trustEmail: cfg.TrustEmail,
		provider:   provider,
		verifier:   provider.Verifier(&oidc.Config{ClientID: cfg.ClientID}),
		oauth: oauth2.Config{
			ClientID:     cfg.ClientID,
			ClientSecret: cfg.ClientSecret,
			RedirectURL:  cfg.RedirectURL,
			Endpoint:     provider.Endpoint(),
			Scopes:       []string{oidc.ScopeOpenID, "email", "profile"},
		},
		db: db, users: users, svc: svc, secure: secure,
	}, nil
}

// Register mounts GET /api/auth/oidc/login and GET /api/auth/oidc/callback (top-level browser navigations).
func (o *OIDC) Register(public *gin.RouterGroup) {
	public.GET("/auth/oidc/login", o.login)
	public.GET("/auth/oidc/callback", o.callback)
}

// Name is shown on the login button (expose it from GET /api/auth/config as "oidc").
func (o *OIDC) Name() string { return o.name }

func (o *OIDC) login(c *gin.Context) {
	state, nonce, verifier := randomToken(), randomToken(), oauth2.GenerateVerifier()
	// state|nonce|verifier live in a short-lived HttpOnly cookie scoped to the callback path.
	flow := state + "|" + nonce + "|" + verifier
	http.SetCookie(c.Writer, newCookie(oidcFlowCookie, "/api/auth/oidc", flow, 600, o.secure))
	url := o.oauth.AuthCodeURL(state, oidc.Nonce(nonce), oauth2.S256ChallengeOption(verifier))
	c.Redirect(http.StatusFound, url)
}

func (o *OIDC) callback(c *gin.Context) {
	ctx := c.Request.Context()
	fail := func(msg string, err error) {
		slog.WarnContext(ctx, "oidc login failed", "reason", msg, "err", err)
		c.Redirect(http.StatusFound, "/login?error=sso")
	}

	raw, err := c.Cookie(oidcFlowCookie)
	http.SetCookie(c.Writer, newCookie(oidcFlowCookie, "/api/auth/oidc", "", -1, o.secure))
	parts := strings.Split(raw, "|")
	if err != nil || len(parts) != 3 || c.Query("state") != parts[0] {
		fail("state mismatch", err)
		return
	}
	token, err := o.oauth.Exchange(ctx, c.Query("code"), oauth2.VerifierOption(parts[2]))
	if err != nil {
		fail("code exchange", err)
		return
	}
	rawID, ok := token.Extra("id_token").(string)
	if !ok {
		fail("no id_token", nil)
		return
	}
	idToken, err := o.verifier.Verify(ctx, rawID)
	if err != nil || idToken.Nonce != parts[1] {
		fail("id_token verification", err)
		return
	}
	var claims struct {
		Email         string `json:"email"`
		EmailVerified *bool  `json:"email_verified"`
		Name          string `json:"name"`
	}
	if err := idToken.Claims(&claims); err != nil || claims.Email == "" {
		fail("missing email claim", err)
		return
	}
	// Accounts are linked by email, so the provider must vouch for it (prevents account takeover).
	verified := claims.EmailVerified != nil && *claims.EmailVerified
	if !verified && !o.trustEmail {
		fail("email not verified", nil)
		return
	}

	u, err := o.findOrCreate(ctx, idToken.Issuer, idToken.Subject, claims.Email, claims.Name)
	if err != nil {
		fail("user provisioning", err)
		return
	}
	if u.Disabled {
		fail("account disabled", nil)
		return
	}
	session, err := o.svc.tokens.Issue(u.ID)
	if err != nil {
		fail("issue session", err)
		return
	}
	http.SetCookie(c.Writer, newCookie(CookieName, "/", session, int(o.svc.tokens.TTL().Seconds()), o.secure))
	c.Redirect(http.StatusFound, "/")
}

// findOrCreate resolves the local user: existing identity link -> same email -> new member account.
func (o *OIDC) findOrCreate(ctx context.Context, issuer, subject, email, name string) (*user.User, error) {
	var ident UserIdentity
	err := o.db.WithContext(ctx).Where("issuer = ? AND subject = ?", issuer, subject).First(&ident).Error
	if err == nil {
		return o.users.FindByID(ctx, ident.UserID)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}

	u, err := o.users.FindByEmail(ctx, email)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		role := authz.RoleMember
		if o.svc.cfg.BootstrapAdminEmail != "" && user.NormalizeEmail(email) == user.NormalizeEmail(o.svc.cfg.BootstrapAdminEmail) {
			role = authz.RoleAdmin
		}
		u = &user.User{Email: email, Name: name, Role: role}
		err = o.users.Create(ctx, u)
	}
	if err != nil {
		return nil, err
	}
	link := UserIdentity{UserID: u.ID, Issuer: issuer, Subject: subject}
	if err := o.db.WithContext(ctx).Create(&link).Error; err != nil {
		return nil, err
	}
	return u, nil
}

func randomToken() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

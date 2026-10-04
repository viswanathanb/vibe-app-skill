# Enhancement: Sign in with OpenID Connect (OIDC)

Adds a "Sign in with <Provider>" button alongside email/password. Works with any OIDC provider:
Google, Microsoft Entra ID, Okta, Auth0, Keycloak, Authentik, GitLab, etc.
(GitHub is OAuth-only; put it behind a broker such as Auth0/Keycloak/Dex.)

Flow: browser → `GET /api/auth/oidc/login` (sets a short-lived state/nonce/PKCE cookie, redirects to the IdP)
→ IdP → `GET /api/auth/oidc/callback` (verifies state, exchanges code with PKCE, verifies the ID token and nonce)
→ finds or creates the local user → sets the normal `session` cookie → redirects to `/`.
After that, everything (RBAC, ReBAC, admin pages) works exactly as with password users.

## 1. Register the app with the provider

- Redirect URI: `http://localhost:5173/api/auth/oidc/callback` (dev, through the Vite proxy) and
  `https://<your-service>.onrender.com/api/auth/oidc/callback` (prod).
- Scopes: `openid email profile`. Note the issuer URL, client ID, client secret.
  - Google issuer: `https://accounts.google.com`
  - Entra ID (single tenant): `https://login.microsoftonline.com/<tenant-id>/v2.0`

## 2. Backend

```bash
cd backend
go get github.com/coreos/go-oidc/v3/oidc golang.org/x/oauth2
cp <skills-dir>/crud-auth/references/oidc/oidc.go internal/auth/oidc.go
# replace example.com/app with the module path in the copied file
```

`internal/config/config.go` — add and load:

```go
OIDCName, OIDCIssuer, OIDCClientID, OIDCClientSecret, OIDCRedirectURL string
OIDCTrustEmail bool // only for a single-tenant company IdP that doesn't send email_verified
```

`internal/server/server.go`:

```go
func Models() []any { return []any{ /* ... */ &auth.UserIdentity{}, /* crud:models */ } }

// in New(), after authSvc is created:
var oidcName string
if cfg.OIDCIssuer != "" {
	sso, err := auth.NewOIDC(context.Background(), auth.OIDCConfig{
		Name: cfg.OIDCName, Issuer: cfg.OIDCIssuer, ClientID: cfg.OIDCClientID,
		ClientSecret: cfg.OIDCClientSecret, RedirectURL: cfg.OIDCRedirectURL, TrustEmail: cfg.OIDCTrustEmail,
	}, db, users, authSvc, cfg.IsProd())
	if err != nil {
		return nil, err
	}
	sso.Register(api)
	oidcName = sso.Name()
}
```

Pass `oidcName` into the auth controller and return it from `GET /api/auth/config`:
`{"signupEnabled": ..., "oidc": "Google"}` (empty string when disabled).

The OIDC routes are GET requests (top-level navigations), so the CSRF header guard doesn't block them;
state + nonce + PKCE protect the flow.

## 3. Frontend

- `features/auth/api.ts`: extend `useAuthConfig` type with `oidc: string`.
- `LoginPage.tsx`: when `config.oidc` is set, render
  `<Button variant="outline" asChild><a href="/api/auth/oidc/login">Sign in with {config.oidc}</a></Button>`
  and show an error when the URL has `?error=sso`.

## 4. Config and deploy

`.env.example` and `render.yaml` (`sync: false` for the secret):

```
OIDC_NAME=Google
OIDC_ISSUER=https://accounts.google.com
OIDC_CLIENT_ID=...
OIDC_CLIENT_SECRET=...
OIDC_REDIRECT_URL=http://localhost:5173/api/auth/oidc/callback
```

## Account linking rules (security)

- Identities are stored in `user_identities (issuer, subject)` — the stable key for later logins.
- First login links to an existing user **only by verified email** (`email_verified: true`).
  Without that claim the login is refused unless `OIDC_TRUST_EMAIL=true`, which is only safe for a
  single-tenant company IdP where admins control every address (multi-tenant Entra ID apps must not use it).
- New SSO users get role `member` (or `admin` if their email equals `BOOTSTRAP_ADMIN_EMAIL`) and an empty
  password hash, so password login is impossible for them.
- Disabled users are rejected at the callback and on every API request.

## Optional: SSO only

Set `ALLOW_SIGNUP=false`, hide the password form when `config.oidc` is set, and stop seeding passwords.

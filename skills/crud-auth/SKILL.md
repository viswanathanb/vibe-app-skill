---
name: crud-auth
description: 'Authentication for crud-* apps: email + password signup/login with bcrypt, JWT session in an HttpOnly SameSite cookie, CSRF header guard, login rate limiting, bootstrap admin, admin user management (create users, change roles, disable), React login/signup pages and useMe/useCan hooks. Optional enhancement: "Sign in with Google/Microsoft/Okta" via OpenID Connect (OIDC). Use when working on login, logout, sessions, passwords, users admin, current user, SSO or OIDC.'
---

# Authentication (email/password + JWT cookie; OIDC optional)

## What ships in the skeleton

Backend (`backend/internal/`):

| File | Purpose |
| --- | --- |
| `user/model.go` | `User{ID, Email (unique, lower-case), Name, PasswordHash, Role, Disabled}` |
| `user/password.go` | bcrypt (cost 12), constant-time compare, timing equaliser for unknown emails |
| `user/repository.go` | CRUD + `LookupUserID`/`UserLabels` (used by the sharing API) |
| `user/service.go`, `controller.go` | admin-only `GET/POST /api/users`, `PATCH /api/users/:id` (role, name, disabled); last active admin cannot be demoted/disabled |
| `auth/token.go` | HS256 JWT with only `sub` = user id, `iss`, `iat`, `exp` |
| `auth/middleware.go` | `RequireUser`: reads `session` cookie or `Authorization: Bearer`, loads the user **from the DB on every request** (role changes and disabling apply immediately), sets `authz.Principal` |
| `auth/service.go` | Signup, Login (rate limited 10/15 min per client+email), `EnsureBootstrapAdmin` |
| `auth/controller.go` | `GET /api/auth/config`, `POST /api/auth/signup|login|logout`, `GET /api/auth/me` |

Frontend (`frontend/src/features/`): `auth/api.ts` (`useMe`, `useCan`, `useLogin`, `useSignup`, `useLogout`, `useAuthConfig`), `LoginPage`, `SignupPage`, `admin/UsersPage` + `CreateUserDialog`. `components/RequireAuth.tsx` guards routes.

## Rules and behaviour

- **First admin**: locally, the first account to sign up becomes `admin`. In production, set
  `BOOTSTRAP_ADMIN_EMAIL` + `BOOTSTRAP_ADMIN_PASSWORD`; the admin is created on startup if missing.
  Signup never grants admin in production (anyone could register that email first).
- **Signup** is on by default locally and **off in production** (`ALLOW_SIGNUP`). With it off, admins create users on the Users page.
- **Session cookie**: `session`, HttpOnly, SameSite=Lax, Secure in production, Path `/`, lifetime `JWT_TTL`.
  Logout clears it. Tokens are stateless: to cut access immediately, disable the user (checked on every request).
- **CSRF**: all POST/PUT/PATCH/DELETE under `/api` need `X-Requested-With: XMLHttpRequest`. The frontend
  `api` client adds it. Scripts (curl) must add it too, or use `Authorization: Bearer <token>`.
- **Passwords**: 8–72 bytes (bcrypt limit). Errors never reveal whether an email exists on login.
- `/api/auth/me` returns `{user, actions}`; `actions` are the role's RBAC actions (`"*"` for admin). The UI uses
  `useCan("team:create")` to hide buttons; the backend enforces the same rule.

## Common tasks

### Add a field to users (e.g. `department`)

1. `user/model.go`: add the column; add it to `CreateInput`/`UpdateInput` with `binding` tags.
2. `user/service.go`: copy the field in `Create`/`Update`.
3. `frontend/src/features/auth/api.ts`: add to `User`; show/edit in `admin/UsersPage.tsx` / `CreateUserDialog.tsx`.

### Let users change their own name/password

Add `PATCH /api/auth/me` in `auth/controller.go` (authed group) → `auth.Service.UpdateProfile(ctx, userID, input)`.
For a password change require the current password (`user.CheckPassword`) and re-hash with `user.HashPassword`.

### Call the API from a script

```bash
TOKEN=$(curl -s -c - -X POST localhost:8080/api/auth/login -H 'X-Requested-With: XMLHttpRequest' \
  -H 'Content-Type: application/json' -d '{"email":"a@b.c","password":"..."}' | awk '/session/{print $7}')
curl -H "Authorization: Bearer $TOKEN" localhost:8080/api/teams
```

### Enhancement: SSO with OpenID Connect

Only when the user asks for "Sign in with Google/Microsoft/Okta/Keycloak". Follow
[references/oidc.md](./references/oidc.md); it adds [oidc.go](./references/oidc/oidc.go) next to the password login
(both keep working) and links accounts by **verified** email.

## Don'ts

- Don't store the JWT in `localStorage` or return it in JSON for the browser.
- Don't put role or permissions in the JWT; they're read from the DB per request.
- Don't add CORS. The SPA is served by the same origin (Vite proxy locally, Go in production).
- Don't log passwords, tokens, or full request bodies of auth endpoints.

## Verify

```bash
task backend:test   # includes token round-trip/expiry tests
```

Manual: sign up two users, sign in/out, disable one as admin → their next request returns 401.

---
name: crud-backend-go
description: 'Go backend conventions for crud-* apps: Gin + GORM + PostgreSQL with controller/service/repository layers, AutoMigrate, config from env, JSON error envelope, validation via binding tags, pagination/sorting/search, SPA serving, health check, graceful shutdown. Use when creating the backend skeleton, editing backend/internal/server wiring, adding middleware, config variables, or fixing Go API structure in an app built from these skills.'
---

# Go Backend (Gin + GORM + Postgres)

## When to use

- Creating `backend/` for a new app (normally via `crud-app-scaffold`).
- Changing server wiring, config, middleware, error handling, or pagination.
- For a new resource use `crud-resource`; for auth use `crud-auth`; for permissions use `crud-authz`.

## Create the backend

```bash
<skills-dir>/crud-backend-go/scripts/create-backend.sh <app-dir> <go-module-path>
```

Copies [assets/backend](./assets/backend/) plus the auth/authz/teams packages from sibling skills,
rewrites the placeholder module `example.com/app` to your module, resolves dependencies
(gin, gorm, postgres driver, golang-jwt v5, x/crypto), pins `go`/`toolchain` in `go.mod` to the installed Go version,
and runs gofmt/vet/build/test.

## Layout

```
backend/
  cmd/server/main.go         config -> logger -> db -> AutoMigrate(server.Models()) -> Bootstrap -> HTTP server
  internal/config            Config struct from env (see table below)
  internal/database          Open(): GORM + pgx, pool limits, ping, TranslateError
  internal/httpx             Error type + Fail(), BindJSON(), ParseID(), Page/List/OrderBy/LikePattern,
                             CSRFGuard, SecurityHeaders, RequestLogger, Limiter
  internal/server            New(): middleware + dependency wiring + routes; Models(); SPA fallback
  internal/<feature>         model.go, repository.go, service.go, controller.go (one package per resource)
  .air.toml                  hot reload config (task backend:dev)
```

## Layer rules

| Layer | Does | Never |
| --- | --- | --- |
| controller | parse path/query/body (`httpx.BindJSON`, `httpx.ParseID`, `httpx.ParsePage`), call service, write JSON/status | touch `*gorm.DB`, make permission decisions |
| service | business rules, trimming/normalising, **authorization first**, transactions | write HTTP responses, build SQL strings |
| repository | GORM queries; `WithTx(tx)`; allow-listed sorting | check permissions, know about HTTP |

Services receive `authz.Principal` (from `authz.PrincipalFrom(c)`) as the 2nd argument after `ctx`.

## Wiring a package

`internal/server/server.go` is the only place that constructs dependencies:

```go
func Models() []any {
	return []any{
		&user.User{}, &authz.Tuple{}, &team.Team{},
		&widget.Widget{},
		// crud:models
	}
}
// in New():
widget.NewController(widget.NewService(db, widget.NewRepository(db), az)).Register(authed)
// crud:routes
```

`authed` routes require a session; `api` routes are public (only auth endpoints use it).

## API conventions

- Base path `/api`. JSON field names are **camelCase** (`json:"createdAt"`).
- Status codes: 200 read/update, 201 create (return the object), 204 delete/no body, 400 validation,
  401 no/invalid session, 403 not allowed, 404 missing, 409 conflict/duplicate.
- Errors are always `{"error":{"code":"not_found","message":"...","details":{...}}}`. Return
  `httpx.BadRequest/Forbidden/NotFound/Conflict(...)` from services; return GORM errors as-is —
  `httpx.Fail` maps `gorm.ErrRecordNotFound` → 404, `ErrDuplicatedKey` → 409, anything else → logged 500.
- Validation: `binding:"required,max=200"` tags on input structs. Failures return 400 with
  `details: {"<jsonField>": "<tag>[=param]"}` which the frontend maps onto form fields.
- PATCH inputs use pointer fields (`*string`) so omitted fields are untouched.
- Lists: `GET /api/things?limit=20&offset=0&sort=-updatedAt&q=text` →
  `{"items":[...],"total":n,"limit":20,"offset":0}`. Max limit 100.
- Sorting goes through `httpx.OrderBy(sort, sortColumns, fallback)` with an explicit allow-list map.
  Search uses `ILIKE ?` with `httpx.LikePattern(q)`. Never concatenate user input into SQL.
- Reuse a filtered query for Count + Find with `q = q.Session(&gorm.Session{})`.

## Models and migrations

- GORM AutoMigrate runs on every start: add the model to `Models()`; new columns appear automatically.
  AutoMigrate never drops or renames columns — for renames/drops, write a one-off SQL step in `main.go`
  before AutoMigrate and tell the user.
- Use `uint` IDs, `CreatedAt/UpdatedAt time.Time`, explicit `size:` on strings, `not null;default:` where sensible,
  `index` on foreign keys and filter columns. Use `uniqueIndex` for natural keys (e.g. email).
- Foreign keys: `ProjectID uint gorm:"not null;index"` + optional `Project *Project json:"-"`.
  Enforce existence/permission in the service, not via cascading deletes.

## Configuration (env)

| Var | Default | Notes |
| --- | --- | --- |
| `DATABASE_URL` | — (required) | `postgres://user:pass@host:5432/db?sslmode=disable` locally |
| `APP_ENV` | `development` | `production` enables secure cookies, HSTS/CSP, JSON logs |
| `PORT` | `8080` | Render injects `10000` via the Dockerfile |
| `JWT_SECRET` | dev fallback | ≥32 chars, required in production |
| `JWT_TTL` | `24h` | session length |
| `ALLOW_SIGNUP` | true dev / false prod | open self-registration |
| `BOOTSTRAP_ADMIN_EMAIL` / `_PASSWORD` | — | creates the first admin on startup |
| `STATIC_DIR` | empty | built SPA dir; set to `/app/public` in the Docker image |

Add new settings to `internal/config/config.go` (with validation) **and** `.env.example` **and** `render.yaml`.

## Security baseline (keep it)

- `httpx.CSRFGuard()` on `/api`: unsafe methods need `X-Requested-With: XMLHttpRequest`.
- `SecurityHeaders` (nosniff, frame deny, HSTS + CSP in production). No CORS: frontend is same-origin.
- `http.Server` timeouts, graceful shutdown on SIGTERM (Render sends it on deploy).
- Unknown errors are logged server-side and returned as a generic 500.

## Verify

```bash
cd backend && gofmt -l . && go vet ./... && go build ./... && go test ./... && golangci-lint run ./...
```

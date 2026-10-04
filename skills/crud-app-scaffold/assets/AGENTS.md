# {{APP_NAME}} — agent rules

CRUD web app generated from the `crud-*` skills. Read this before changing code.

## Architecture

- **Backend**: Go + Gin (`backend/`, port 8080), GORM + PostgreSQL, AutoMigrate on startup (no migration files).
- **Frontend**: React + TypeScript + Vite (`frontend/`, port 5173), shadcn/ui (Radix) + Tailwind v4, TanStack Query, React Router.
- **Auth**: email + password, bcrypt, JWT in an HttpOnly `session` cookie. `X-Requested-With: XMLHttpRequest` required on POST/PUT/PATCH/DELETE (CSRF guard).
- **Authorization**: RBAC (global role `admin` / `member` / `viewer` → actions in `backend/internal/authz/rbac.go`) + ReBAC (relation tuples in Postgres, schema in `backend/internal/authz/app_schema.go`).
- **Deploy**: one Docker image (Go serves the built SPA) on Render + Render Postgres, defined in `render.yaml`.
- **Workflow**: everything runs through `Taskfile.yml` (`task --list`).

## Layout

```
backend/
  cmd/server/main.go          entrypoint: config -> db -> AutoMigrate -> router
  internal/config             env vars
  internal/database           GORM/Postgres connection
  internal/httpx              errors, validation, pagination, middleware
  internal/server             wiring (Models(), routes), SPA serving
  internal/auth               login/signup/logout/me, JWT, RequireUser
  internal/user               users + admin user management
  internal/authz              RBAC actions, ReBAC schema/engine/store, sharing API
  internal/team               teams (groups for sharing)
  internal/<resource>         one package per resource: model, repository, service, controller
frontend/src/
  lib/                        api client, query client, form helpers
  components/                 layout, guards, shared UI; components/ui = shadcn (generated)
  features/<resource>/        api.ts (hooks) + pages + dialogs per resource
  App.tsx                     routes
```

## Conventions

- **Layers**: controller (HTTP only) → service (rules + authorization) → repository (GORM only). Controllers never touch GORM; repositories never check permissions.
- **Every service method takes `authz.Principal` and authorizes first**: `p.Can("<type>:create")` for creation, `az.Authorize(ctx, p, "view|edit|delete", obj)` for objects, `az.LookupResources` to filter lists.
- **Create = row + owner tuple in one transaction**; delete = row + `az.DeleteObject` in one transaction.
- **API shape**: JSON camelCase. Lists return `{items, total, limit, offset}` and accept `limit, offset, sort (-field for desc), q`. Errors return `{"error": {"code", "message", "details"}}`.
- **Validation** lives in `binding:"..."` tags on input structs and is mirrored in the frontend zod schema.
- **Sorting** only through an allow-list map (`sortColumns`), never raw user input in SQL.
- **Extension points** are marked: `// crud:models`, `// crud:routes`, `// crud:schema`, `/* crud:actions */`, `// crud:nav`, `{/* crud:home */}`.
- **Frontend data**: one `api.ts` per feature with query keys + hooks; mutations invalidate the feature's key; UI hides actions using `useCan()` (RBAC) and `permissions` from detail endpoints (ReBAC). The backend is the source of truth.
- Use shadcn components from `@/components/ui`; add new ones with `bunx --bun shadcn@latest add <name>`.

## Commands

| Task | What |
| --- | --- |
| `task setup` | create `.env`, install deps |
| `task dev` | Postgres + API (air) + Vite |
| `task lint` / `task test` / `task build` | quality gates |
| `task check` | lint + test + build + osv-scanner + dependency review |
| `task db:psql` / `task db:reset` | local database |
| `task docker:build` / `task docker:run` | production image locally |
| `task deploy` | trigger a Render deploy |

## After every change

1. `task lint` (gofmt, go vet, golangci-lint, prettier, eslint zero warnings, tsc).
2. `task test` and `task build`.
3. Dependency review: `task deps:review`; if the change needed no new dependencies, manifests and lockfiles must be unchanged. No downgrades.
4. `task osv`. Report new findings; don't add ignores without asking.
5. Fix failures your change introduced; call out pre-existing ones explicitly.

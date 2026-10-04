---
name: crud-app-scaffold
description: 'Create a new full-stack CRUD web app from scratch: Go (Gin, GORM, Postgres) backend, React (Vite, shadcn/ui, TanStack Query) frontend, email/password auth, RBAC + ReBAC sharing, Taskfile workflow, Docker, Render deploy. Use when the user asks to start/scaffold/bootstrap a new app, admin tool, internal tool, dashboard, or "app like X" with users, roles, and CRUD screens. Entry point that orchestrates the other crud-* skills.'
argument-hint: 'App name, Go module path, and a short description of the resources'
---

# CRUD App Scaffold

Builds a working, deployable app skeleton in minutes, then grows it resource by resource.
This is the **entry point**; it delegates to the other `crud-*` skills.

| Skill | Owns |
| --- | --- |
| `crud-app-scaffold` (this) | spec, project layout, root files, orchestration |
| `crud-backend-go` | Go/Gin/GORM conventions, server wiring, errors, pagination |
| `crud-auth` | login/signup, JWT cookie, users admin; OIDC as an enhancement |
| `crud-authz` | RBAC roles/actions, ReBAC schema + tuples, sharing API, teams |
| `crud-resource` | adding one CRUD resource end-to-end (golden example: `project`) |
| `crud-frontend-react` | React app shell, API client, routing, UI conventions |
| `crud-deploy-render` | Dockerfile, `render.yaml`, deploying with Task |
| `crud-e2e-playwright` | Playwright browser tests (`task e2e`) |
| `crud-quality-gates` | lint, tests, dependency review, osv-scanner |

## When to use

- "Create a new app for managing X", "scaffold a CRUD app", "bootstrap the project".
- The target directory is empty or has only docs (README, AGENTS.md, APP_SPEC.md).

Do **not** use for changing an existing app built from these skills — use `crud-resource` (new resource) or the specific skill instead.

## Prerequisites

Check with `go version && bun --version && task --version && docker info >/dev/null && air -v`:

- Go 1.26+, bun 1.2+, Docker, [Task](https://taskfile.dev) (`brew install go-task`), [air](https://github.com/air-verse/air) (`go install github.com/air-verse/air@latest`)
- For quality gates: `golangci-lint` v2, `osv-scanner` v2. For deploys: Render CLI (`brew install render`).

If something is missing, tell the user the exact install command and stop.

## Procedure

### 1. Capture the spec (talk to the user first)

1. Ask for: app name, Go module path (e.g. `github.com/acme/talos-manager`), one-line purpose, the main resources, and who may create what.
2. Write `APP_SPEC.md` from [the template](./assets/APP_SPEC.md). For each resource pick an **access pattern**:
   - `owned` — creator is owner and can share with users or teams (default).
   - `child of <Parent>` — access inherited from the parent (e.g. tasks in a project).
   - `admin-managed catalog` — admins edit, all signed-in users read.
3. Confirm the spec with the user before generating code. Keep v1 small: 1–3 resources.

### 2. Generate the skeleton (one command)

```bash
<skills-dir>/crud-app-scaffold/scripts/new-app.sh <app-dir> <go-module> "<App Name>"
```

`<skills-dir>` is the folder that contains this skill's folder (the parent of the directory holding this SKILL.md):
e.g. `.claude/skills`, `.github/skills`, `.agents/skills`, or the `skills/` folder of an installed plugin.
The script copies the root files, runs `crud-backend-go/scripts/create-backend.sh` and
`crud-frontend-react/scripts/create-frontend.sh`, and verifies build, lint and tests.
It refuses to overwrite an existing `backend/`, `frontend/`, `Taskfile.yml` or `render.yaml`.

If the script fails, read the error, fix the cause (usually a missing tool), delete only the
partially created `backend/` or `frontend/` it just made, and re-run. Do not hand-write the skeleton.

Result:

```
<app>/
  AGENTS.md  CLAUDE.md  APP_SPEC.md  Taskfile.yml  docker-compose.yml  Dockerfile  render.yaml  .env.example  .gitignore
  backend/   Go API: auth, users, RBAC + ReBAC (authz), teams
  frontend/  React SPA: login/signup, teams + members, admin users
```

### 3. Run it

```bash
cd <app-dir>
task dev   # Postgres in Docker, API on :8080 (air), UI on http://localhost:5173
```

The first account created locally becomes `admin`. Verify: sign up, create a team, add a member.
If port 5432 is taken, change `DB_PORT` **and** the port in `DATABASE_URL` in `.env`.

### 4. Add resources from the spec

For each resource in `APP_SPEC.md`, in dependency order (parents first), follow the
**`crud-resource`** skill. Run `task lint test` after each resource.

### 5. Brand and tidy

- App name: `APP_NAME` in `frontend/src/components/AppLayout.tsx`, `<title>` in `frontend/index.html`.
- Home cards: `frontend/src/pages/HomePage.tsx`.
- Remove the Teams feature only if the user explicitly does not want groups (see `crud-authz`).

### 6. Quality gates and deploy

- Follow `crud-quality-gates` (`task check`).
- Follow `crud-deploy-render` to put it online.

## Definition of done

- [ ] `task check` passes (or pre-existing failures are reported).
- [ ] Every resource in `APP_SPEC.md` has list/create/detail/edit/delete screens and enforces its access pattern.
- [ ] A non-owner cannot see or change someone else's data unless shared (verify with two accounts).
- [ ] `AGENTS.md` in the app still matches reality (update the Layout section if you added packages).

## Ground rules for the agent

- Copy and adapt the provided assets; don't invent new structure, libraries or patterns.
- Keep the extension markers (`// crud:models`, `// crud:routes`, `// crud:schema`, `/* crud:actions */`, `// crud:nav`, `{/* crud:home */}`) so later changes stay mechanical.
- Never commit `.env` or secrets. Never weaken the CSRF guard, cookie flags, or authorization checks to "make something work".

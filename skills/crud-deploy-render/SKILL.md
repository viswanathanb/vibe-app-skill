---
name: crud-deploy-render
description: 'Deploy a crud-* app to Render: one multi-stage Docker image (bun builds the React SPA, Go serves API + SPA on distroless), render.yaml Blueprint with a Docker web service + Render Postgres, generated JWT secret, bootstrap admin, health check, auto-deploy on push, and Taskfile tasks (docker:build, docker:run, render:validate, deploy). Use when the user wants to deploy, host, publish, go live, set up Render, a Dockerfile, render.yaml, or production environment variables.'
---

# Deploy to Render

## What gets deployed

```
GitHub repo ──push──> Render Blueprint (render.yaml)
                       ├─ web service (Docker): Dockerfile -> Go binary + built SPA, port 10000, /healthz
                       └─ Postgres: DATABASE_URL injected (internal connection string)
```

Files (copied by `crud-app-scaffold`; originals in [assets](./assets/)): `Dockerfile`, `.dockerignore`, `render.yaml`.
Tables are created by GORM AutoMigrate when the service starts — no migration step.

## Procedure

### 1. Local production check

```bash
task docker:build     # builds exactly what Render builds
task docker:run       # runs it on http://localhost:8080 against local Postgres (APP_ENV=production)
```

Open http://localhost:8080, sign up (ALLOW_SIGNUP=true in docker:run), click through. Fix anything before deploying.

### 2. Review `render.yaml`

- `name` of the service and database (unique in the Render workspace; the scaffold uses the app slug).
- `region` — same for service and database (private networking).
- `plan` — `free` for trying out. Free web services sleep when idle (slow first request); free Postgres expires after a
  limited period (check Render's current free-tier terms). Use paid plans for anything real.
- Env vars: `JWT_SECRET` is generated once by Render; `BOOTSTRAP_ADMIN_EMAIL`/`_PASSWORD` are `sync: false`
  (Render prompts for them at creation); `ALLOW_SIGNUP` is `"false"`.
- Add every new config variable from `backend/internal/config` here. Secrets → `sync: false`, never literal values.

Validate: `task render:validate` (needs `brew install render` + `render login`).

### 3. First deploy (one-time, in the browser)

1. Push the repo to GitHub/GitLab/Bitbucket (render.yaml at the root).
2. Render Dashboard → **New → Blueprint** → pick the repo → it shows the service + database from `render.yaml`.
3. Enter `BOOTSTRAP_ADMIN_EMAIL` and a strong `BOOTSTRAP_ADMIN_PASSWORD` when prompted → **Apply**.
4. Wait for the deploy, open `https://<name>.onrender.com`, sign in as the bootstrap admin, create users on the Users page.

Tell the user to do these steps; the agent cannot click through the dashboard or handle their credentials.

### 4. Subsequent deploys

- Automatic on every push to the linked branch (`autoDeployTrigger: commit`).
- Manual: put the service ID (`srv-…`, from the dashboard URL or `render services -o json`) in `.env` as
  `RENDER_SERVICE_ID`, then `task deploy` (uses `render deploys create <id> --wait`; needs `render login` or
  `RENDER_API_KEY`).
- Changes to `render.yaml` are applied by Render on push (Blueprint sync). Values marked `sync: false` are only asked
  for at creation; add new secrets in the dashboard afterwards.

### 5. Operate

- Logs: Render dashboard → service → Logs (JSON lines from `slog`).
- DB shell: `render psql <database-id>`.
- Rollback: dashboard → Deploys → pick a previous deploy → Rollback.
- Custom domain: add under `domains:` in `render.yaml` or in the dashboard; HTTPS is automatic.

## Image details

- Stage 1 `oven/bun:1`: `bun install --frozen-lockfile` + `bun run build` → `frontend/dist`.
- Stage 2 `golang:<minor>-alpine`: static build (`CGO_ENABLED=0`). `new-app.sh` sets `<minor>` from `backend/go.mod`;
  **keep them in sync** when upgrading Go (`go mod edit -go=1.N -toolchain=go1.N.P`).
- Stage 3 `gcr.io/distroless/static-debian12:nonroot`: binary + `public/`, runs as non-root, `STATIC_DIR=/app/public`,
  `PORT=10000`, `APP_ENV=production`.
- Render sends SIGTERM on deploys; the server shuts down gracefully within 20s.

## Troubleshooting

| Symptom | Cause / fix |
| --- | --- |
| Deploy fails at `bun install --frozen-lockfile` | `bun.lock` not committed or out of date → run `bun install` locally and commit |
| `DATABASE_URL is required` | the `fromDatabase.name` doesn't match the database `name` |
| `JWT_SECRET must be at least 32 characters` | service created without the Blueprint → add a random 32+ char value in the dashboard |
| Health check failing | app crashed on start (see logs) or DB unreachable (regions differ) |
| Login works locally but not on Render | cookies are `Secure` in production → must use the https URL |
| 403 `missing X-Requested-With header` from a script | add `-H 'X-Requested-With: XMLHttpRequest'` |

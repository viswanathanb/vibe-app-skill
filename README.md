# vibe-app-skill — CRUD app skills for AI coding agents

A set of [Agent Skills](https://code.visualstudio.com/docs/copilot/customization/agent-skills) that let anyone with
basic knowledge and an LLM coding agent (GitHub Copilot, Claude Code, Codex, …) build a **production-shaped CRUD web
app** — and then keep customising it.

What you get after one prompt:

- **Backend**: Go + Gin, GORM + PostgreSQL (AutoMigrate), controller → service → repository.
- **Frontend**: React + TypeScript + Vite, Tailwind v4 + shadcn/ui, TanStack Query, React Router.
- **Auth**: email + password (bcrypt), JWT in an HttpOnly cookie, CSRF guard, admin user management.
  Optional "Sign in with Google/Microsoft/…" (OIDC).
- **Authorization**: RBAC (admin / member / viewer) **plus** ReBAC (Zanzibar-style tuples in Postgres): owners,
  editors, viewers, sharing with users or teams, inheritance from parent objects.
- **Workflow**: everything through [Task](https://taskfile.dev) — `task dev`, `task check`, `task deploy`.
- **Deploy**: one Docker image on [Render](https://render.com) + Render Postgres via `render.yaml`.

## Skills

| Skill | Use it for |
| --- | --- |
| [`crud-app-scaffold`](skills/crud-app-scaffold/SKILL.md) | **Start here.** Spec → working app skeleton in one command → resources → deploy |
| [`crud-resource`](skills/crud-resource/SKILL.md) | Add an entity end-to-end (model, API, permissions, pages) — scripted |
| [`crud-backend-go`](skills/crud-backend-go/SKILL.md) | Go/Gin/GORM conventions, wiring, errors, pagination, config |
| [`crud-frontend-react`](skills/crud-frontend-react/SKILL.md) | React app shell, data hooks, forms, UI conventions |
| [`crud-auth`](skills/crud-auth/SKILL.md) | Login/signup, sessions, users admin, OIDC enhancement |
| [`crud-authz`](skills/crud-authz/SKILL.md) | RBAC roles/actions, ReBAC schema, sharing, teams, access patterns |
| [`crud-deploy-render`](skills/crud-deploy-render/SKILL.md) | Dockerfile, `render.yaml`, first deploy, operations |
| [`crud-e2e-playwright`](skills/crud-e2e-playwright/SKILL.md) | Browser tests (Playwright): auth, permissions, one spec per resource |
| [`crud-quality-gates`](skills/crud-quality-gates/SKILL.md) | Lint, tests, builds, dependency review, osv-scanner |

Skills ship **tested assets** (real Go/TS files) and **scripts** (`new-app.sh`, `add-resource.sh`) so the agent copies
and adapts working code instead of improvising.

## Install

Prerequisites for the generated apps: Go 1.26+, bun, Docker, Task (`brew install go-task`), air
(`go install github.com/air-verse/air@latest`); for checks `golangci-lint` v2 and `osv-scanner` v2; for deploys the
Render CLI (`brew install render`).

Pick one way. Skills copied into a project (options 1–2) are committed with it, so every collaborator and the
Copilot cloud agent get them; plugins (options 3–5) are per user.

**1. Install script** (curl + tar only; run inside the project):

```bash
curl -fsSL https://raw.githubusercontent.com/viswanathanb/vibe-app-skill/main/install.sh | bash
# options: --tools claude,github,agents   --global   --dir <path>   --ref <tag>
```

**2. skills CLI** ([skills.sh](https://skills.sh)):

```bash
npx skills add viswanathanb/vibe-app-skill --skill '*'           # interactive agent selection
npx skills add viswanathanb/vibe-app-skill --skill '*' -a claude-code -a github-copilot -y
```

**3. Claude Code plugin**:

```bash
claude plugin marketplace add viswanathanb/vibe-app-skill
claude plugin install vibe-app-skill@vibe-app-skill
```

Skills are then named `/vibe-app-skill:crud-app-scaffold` etc.

**4. VS Code (GitHub Copilot)**: Command Palette → **Chat: Install Plugin From Source** →
`https://github.com/viswanathanb/vibe-app-skill`. Or add `"chat.plugins.marketplaces": ["viswanathanb/vibe-app-skill"]`
to settings and install it from the Extensions view (`@agentPlugins`).

**5. GitHub Copilot CLI**: `copilot plugin install viswanathanb/vibe-app-skill` (VS Code picks up CLI-installed
plugins too).

**From a local checkout** (for maintainers):

```bash
task install TARGET=../my-app                    # .claude/skills, .github/skills, .agents/skills
task install TARGET=../my-app TOOLS=claude       # only Claude Code
task install:user                                # for all your projects (~/.claude, ~/.copilot, ~/.agents)
```

| Agent | Reads skills from |
| --- | --- |
| Claude Code | `.claude/skills/` |
| GitHub Copilot (VS Code, CLI, coding agent) | `.github/skills/`, `.claude/skills/`, `.agents/skills/` |
| Codex and other Agent-Skills tools | `.agents/skills/` |

Copilot reads all three folders; install only `--tools github` if you see duplicates.

## Use it

Agents can pick skills automatically from their descriptions, but naming the skill is more reliable. Start every
request with the skill (plugin installs use the prefixed name, e.g. `/vibe-app-skill:crud-app-scaffold`):

> /crud-app-scaffold Create "Talos Manager" (Go module `github.com/acme/talos-manager`).
> Users manage Clusters (name, endpoint URL, Kubernetes version, status) and Machines that belong to a cluster
> (hostname, IP, role control-plane|worker). Members can create clusters and share them with teams.
> Start with the spec and confirm it with me before generating code.

The agent will write `APP_SPEC.md`, confirm it with you, run `new-app.sh`, add each resource with `add-resource.sh`,
adjust fields, run the quality gates and tell you how to deploy. The generated `AGENTS.md` (imported by `CLAUDE.md`)
tells future sessions which skill to use for which task. Then:

```bash
cd ../my-app
task dev        # http://localhost:5173 — first account you create locally is admin
task check      # lint, tests, builds, osv-scanner, dependency review
task deploy     # after the one-time Render Blueprint setup (see crud-deploy-render)
```

Customise with explicit skill prompts:

| Want | Prompt |
| --- | --- |
| New entity | `/crud-resource Add a MachineConfig resource: name, YAML content (text), version (int); child of Cluster` |
| Change a field | `/crud-resource Add a priority field (low/medium/high, default medium) to Ticket` |
| Permissions | `/crud-authz Editors may delete clusters; viewers may create tickets` |
| SSO | `/crud-auth Add Sign in with Google (OIDC) next to password login` |
| UI change | `/crud-frontend-react Show machine count per cluster on the clusters list` |
| Deploy | `/crud-deploy-render Deploy this app to Render` |
| Browser tests | `/crud-e2e-playwright Add a test that a viewer cannot archive a cluster` |
| Before merging | `/crud-quality-gates Run all checks and report` |

If the transcript doesn't show the skill being loaded, interrupt and say "use the <skill> skill".

## Maintaining the skills

```bash
task check:skills       # frontmatter, names, links, script syntax
task validate:backend   # assemble all Go assets (+ project example, OIDC) and build/vet/test/golangci-lint
task smoke              # real Postgres in Docker; 36 API checks across auth, RBAC, ReBAC, sharing, teams
task e2e                # new-app.sh + 2 resources + lint/test/build/osv/docker build
task check              # all of the above
```

See [AGENTS.md](AGENTS.md) for the rules when changing this repo.

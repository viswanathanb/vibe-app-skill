# Agent rules for this repo (vibe-app-skill)

This repo contains **skills**, not an app. Generated apps live elsewhere (e.g. `../talos-manager`).

## Structure

```
skills/<name>/SKILL.md      frontmatter `name` must equal the folder; description = trigger keywords (≤1024 chars)
skills/<name>/assets/       real files copied into generated apps (paths mirror the app: backend/..., frontend/...)
skills/<name>/references/   docs/code loaded on demand
skills/<name>/scripts/      executable helpers used by agents (new-app.sh, create-*.sh, add-resource.sh)
scripts/                    repo validation (check-skills, validate-backend, smoke-backend, e2e)
Taskfile.yml                install + validation tasks
install.sh                  curl|bash installer (also used by `task install`)
plugin.json                 Agent Plugins 1.0 manifest (VS Code / Copilot CLI)
.claude-plugin/             Claude Code plugin + marketplace manifests
```

## Rules

- Assets must stay **working code**. Go assets use the placeholder module `example.com/app`.
- Keep the extension markers stable: `// crud:models`, `// crud:routes`, `// crud:schema`, `/* crud:actions */`,
  `// crud:nav`, `{/* crud:home */}`. `add-resource.sh` and `smoke-backend.sh` depend on them.
- The `project` example (crud-resource) is the template `add-resource.sh` clones: identifiers containing
  `project`/`Project`/`projects` are rewritten, so don't introduce unrelated words containing them there.
- Keep SKILL.md under ~300 lines; move details to `references/`.
- After creating Go files, make sure each has exactly one `package` line (`task check:skills` checks it).
- Pin nothing in the docs that the scripts don't verify; prefer "run the script" over long manual steps.
- Releasing: bump `version` in `plugin.json`, `.claude-plugin/plugin.json` and `.claude-plugin/marketplace.json`
  together (plugin clients only update on a version change), then tag `vX.Y.Z`.

## Before committing

```bash
task check   # check:skills + validate:backend + smoke + e2e
```

Report osv-scanner findings from `task e2e` (they come from upstream packages; don't add ignores silently).

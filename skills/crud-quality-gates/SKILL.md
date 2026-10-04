---
name: crud-quality-gates
description: 'Mandatory checks after every code change in a crud-* app: gofmt, go vet, golangci-lint v2, Go tests, Prettier, ESLint with zero warnings, tsc, Vitest, production builds, dependency/lockfile review against origin/main, and osv-scanner vulnerability scan. Use before reporting any task as done, when CI fails, when lint/test/build errors appear, or when dependencies change.'
---

# Quality gates

Run these after **every** task that changes code, before saying it's done. Fix failures your change introduced;
report pre-existing failures explicitly instead of ignoring them.

## One command

```bash
task check     # lint -> test -> build -> osv -> deps:review
```

Or step by step:

| Step | Command | Must be |
| --- | --- | --- |
| Go format | `cd backend && gofmt -l .` | empty output (only format files you changed: `gofmt -w <file>`) |
| Go vet + lint | `go vet ./...` and `golangci-lint run ./...` | 0 issues ([config](./assets/backend/.golangci.yml)) |
| Go tests | `go test ./...` | pass |
| Frontend format | `cd frontend && bun run format:check` | pass (`bun run format` fixes; `components/ui` is ignored) |
| ESLint | `bun run lint` | **zero warnings** |
| Types | `bun run typecheck` | pass |
| Frontend tests | `bun run test` | pass |
| Builds | `task build` | both succeed |
| Vulnerabilities | `task osv` | no new findings |
| Dependencies | `task deps:review` | only intended changes |

## Dependency review

```bash
git diff origin/main -- '**/package.json' '**/bun.lock' '**/go.mod' '**/go.sum'
```

- If the task didn't need new dependencies, manifests and lockfiles must be **identical** to base — restore them
  (`git checkout origin/main -- <file>`) if tooling touched them.
- Watch for downgrades, lockfile churn, duplicate keys in `package.json`, and packages you didn't mean to add.
- Verify frozen installs: `cd frontend && bun install --frozen-lockfile`; `cd backend && go mod tidy && git diff --exit-code go.mod go.sum`.
- New/upgraded packages must have no known advisories (GitHub's dependency-review action will fail otherwise).

## osv-scanner

- Run from the repo root: `task osv` = `osv-scanner scan source --no-call-analysis=go -r .` (scans `backend/go.mod`
  and `frontend/bun.lock`). Go call analysis is off because it crashes on newer Go toolchains; findings are a superset.
- The Go stdlib version comes from the `toolchain` line in `go.mod`. Keep it at the Go version you actually build with,
  otherwise every stdlib CVE since `.0` is reported.
- Report new findings with the package, version, advisory ID and the fixed version. Prefer upgrading to the fixed
  version (`go get pkg@vX.Y.Z && go mod tidy`, `bun add pkg@X.Y.Z`). Findings without a fix: report, don't hide.
- Respect an existing `osv-scanner.toml`; **don't add ignores without asking the user**.

## Lint fixes cheat sheet

| Finding | Fix |
| --- | --- |
| `errcheck` | handle or explicitly `_ =` with a reason |
| `gosec G124` cookie flags | use the shared `newCookie` helper in `internal/auth` |
| `errorlint` | `errors.Is/As` instead of `==` / type assertions |
| `noctx` | pass `ctx` (`http.NewRequestWithContext`, `db.WithContext(ctx)`) |
| `react-hooks/exhaustive-deps` | add the dependency or restructure; don't disable the rule |
| `react-refresh/only-export-components` | move hooks/constants out of component files (into `api.ts` or `lib/`) |
| `@typescript-eslint/no-explicit-any` | type it, or `unknown` + narrowing |

Never silence a rule globally to get green. A targeted `//nolint:<linter> // reason` or
`// eslint-disable-next-line <rule> -- reason` is acceptable only with a concrete reason.

## Report format

End the task with a short list: each gate → pass / fixed / pre-existing failure (with details).

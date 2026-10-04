#!/usr/bin/env bash
# Adds Playwright browser tests to an app built with the crud-* skills (new or existing):
# @playwright/test + Chromium, playwright.config.ts, e2e/ (auth, admin, teams + one spec per resource),
# Taskfile tasks (e2e, e2e:ui, e2e:report, e2e:install), ignore rules. Safe to re-run.
#
# Usage: add-e2e.sh <app-dir>          SKIP_BROWSER_INSTALL=1 to skip the Chromium download
set -euo pipefail

APP_DIR="$(cd "${1:?usage: add-e2e.sh <app-dir>}" && pwd)"
SKILLS="$(cd "$(dirname "$0")/../.." && pwd)"
A="$SKILLS/crud-e2e-playwright/assets"
FE="$APP_DIR/frontend"
[[ -f "$FE/package.json" && -f "$APP_DIR/Taskfile.yml" ]] || { echo "error: $APP_DIR is not a crud-* app" >&2; exit 1; }

cd "$FE"
echo "==> @playwright/test"
grep -q '"@playwright/test"' package.json || bun add -d @playwright/test >/dev/null
if [[ "${SKIP_BROWSER_INSTALL:-}" != 1 ]]; then
  echo "==> Chromium for Playwright"
  bunx playwright install chromium >/dev/null
fi

echo "==> config + base specs"
cp "$A/frontend/playwright.config.ts" .
mkdir -p e2e
for f in "$A"/frontend/e2e/*; do
  dest="e2e/$(basename "$f")"
  [[ -e "$dest" ]] || cp "$f" "$dest"
done

# tsc -b also type-checks the e2e project.
bun -e '
  const fs = require("fs");
  const cfg = JSON.parse(fs.readFileSync("tsconfig.json", "utf8").replace(/\/\*[\s\S]*?\*\//g, ""));
  cfg.references ??= [];
  if (!cfg.references.some((r) => r.path === "./e2e")) cfg.references.push({ path: "./e2e" });
  fs.writeFileSync("tsconfig.json", JSON.stringify(cfg, null, 2) + "\n");
'

# Apps created before e2e support: keep Vitest, ESLint and Prettier away from Playwright files.
grep -q 'src/\*\*/\*.test' vite.config.ts ||
  perl -0pi -e 's#(environment: "node",)#$1\n    include: ["src/**/*.test.{ts,tsx}"],#' vite.config.ts
grep -q 'playwright-report' eslint.config.js ||
  perl -pi -e 's#globalIgnores\(\["dist"\]\)#globalIgnores(["dist", "playwright-report", "test-results"])#' eslint.config.js
for line in playwright-report test-results e2e/.auth; do
  grep -qxF "$line" .prettierignore || echo "$line" >>.prettierignore
done
for line in frontend/playwright-report/ frontend/test-results/ frontend/e2e/.auth/; do
  grep -qxF "$line" "$APP_DIR/.gitignore" || echo "$line" >>"$APP_DIR/.gitignore"
done
if ! grep -q '^  e2e:' "$APP_DIR/Taskfile.yml"; then
  echo "==> Taskfile e2e tasks"
  export SNIPPET="$A/taskfile-e2e.yml"
  perl -0pi -e 'open my $f, "<", $ENV{SNIPPET} or die; local $/; my $s = <$f>; s/(  # ---------- quality gates)/$s$1/' "$APP_DIR/Taskfile.yml"
  perl -0pi -e 's/(\n      - task: build\n)(      - task: osv)/$1      - task: e2e\n$2/' "$APP_DIR/Taskfile.yml"
fi

echo "==> specs for existing resources"
for dir in src/features/*/; do
  name="$(basename "$dir")"
  case "$name" in auth | admin | sharing | teams) continue ;; esac
  detail="$(ls "$dir"*DetailPage.tsx 2>/dev/null | head -1 || true)"
  list="$(ls "$dir"*Page.tsx 2>/dev/null | grep -v DetailPage | head -1 || true)"
  [[ -n "$detail" && -n "$list" ]] || continue
  pascal="$(basename "$detail" DetailPage.tsx)"
  plural="$(basename "$list" Page.tsx)"
  E2E_ONLY=1 "$SKILLS/crud-resource/scripts/add-resource.sh" "$APP_DIR" "$pascal" "$plural"
done

echo "==> formatting + verifying"
bunx prettier --write e2e playwright.config.ts tsconfig.json >/dev/null
bun run typecheck
bun run lint
echo "OK: run 'task e2e' (needs Docker for Postgres)"

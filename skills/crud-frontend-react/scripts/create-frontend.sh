#!/usr/bin/env bash
# Creates <app>/frontend: Vite + React + TypeScript + Tailwind v4 + shadcn/ui (Radix) + TanStack Query
# + React Router, then copies the skeleton UI from the crud-* skills and verifies it builds.
#
# Usage: create-frontend.sh <app-dir> [App Name]
# Needs: bun. Run from anywhere; sibling skills (crud-auth, crud-authz) must be installed next to this one.
set -euo pipefail

APP_DIR="$(cd "${1:?usage: create-frontend.sh <app-dir> [App Name]}" && pwd)"
APP_NAME="${2:-App}"
SKILLS="$(cd "$(dirname "$0")/../.." && pwd)"
FE="$APP_DIR/frontend"

if [[ -e "$FE" ]]; then
  echo "error: $FE already exists" >&2
  exit 1
fi

echo "==> scaffolding Vite react-ts app"
(cd "$APP_DIR" && bun create vite@latest frontend --template react-ts --no-interactive >/dev/null)
cd "$FE"
bun install >/dev/null
bun remove oxlint >/dev/null 2>&1 || true
rm -rf src/App.css src/assets

echo "==> installing libraries"
bun add @tanstack/react-query react-router react-hook-form @hookform/resolvers zod tailwindcss @tailwindcss/vite >/dev/null
bun add -d eslint @eslint/js typescript-eslint eslint-plugin-react-hooks eslint-plugin-react-refresh globals \
  prettier vitest >/dev/null

echo "==> configuring @/ alias, Tailwind, scripts"
bun -e '
  const fs = require("fs");
  for (const f of ["tsconfig.json", "tsconfig.app.json"]) {
    const cfg = JSON.parse(fs.readFileSync(f, "utf8").replace(/\/\*[\s\S]*?\*\//g, ""));
    cfg.compilerOptions = { ...cfg.compilerOptions, paths: { "@/*": ["./src/*"] } };
    fs.writeFileSync(f, JSON.stringify(cfg, null, 2) + "\n");
  }
  const pkg = JSON.parse(fs.readFileSync("package.json", "utf8"));
  pkg.scripts = {
    dev: "vite",
    build: "tsc -b && vite build",
    preview: "vite preview",
    typecheck: "tsc -b",
    lint: "eslint . --max-warnings 0",
    format: "prettier --write .",
    "format:check": "prettier --check .",
    test: "vitest run",
  };
  fs.writeFileSync("package.json", JSON.stringify(pkg, null, 2) + "\n");
'
echo '@import "tailwindcss";' >src/index.css
cp "$SKILLS/crud-frontend-react/assets/frontend/vite.config.ts" .

echo "==> shadcn/ui (Radix, Nova)"
bunx --bun shadcn@latest init -t vite --base radix --preset nova -y --no-monorepo --silent
bunx --bun shadcn@latest add -y --silent \
  input label textarea card table dialog alert-dialog dropdown-menu badge select sonner skeleton separator

echo "==> copying skeleton UI"
for skill in crud-frontend-react crud-auth crud-authz; do
  src="$SKILLS/$skill/assets/frontend"
  [[ -d "$src" ]] && cp -R "$src/." "$FE/"
done
sed -i.bak "s|<title>.*</title>|<title>$APP_NAME</title>|" index.html && rm index.html.bak
sed -i.bak "s|const APP_NAME = \"App\";|const APP_NAME = \"$APP_NAME\";|" src/components/AppLayout.tsx && rm src/components/AppLayout.tsx.bak

bunx prettier --write . >/dev/null

echo "==> verifying (format, lint, test, build)"
bun run format:check >/dev/null
bun run lint
bun run test >/dev/null
bun run build >/dev/null
echo "OK: $FE"

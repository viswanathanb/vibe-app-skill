#!/usr/bin/env bash
# Creates a complete, working CRUD app skeleton in <app-dir>:
#   backend/ (Go, Gin, GORM, Postgres, JWT auth, RBAC + ReBAC, teams)
#   frontend/ (React, Vite, shadcn/ui, TanStack Query)
#   Taskfile.yml, docker-compose.yml, Dockerfile, render.yaml, AGENTS.md, APP_SPEC.md, .env.example
#
# Usage: new-app.sh <app-dir> <go-module-path> "<App Name>"
#   e.g. new-app.sh ~/code/talos-manager github.com/acme/talos-manager "Talos Manager"
# Needs: go, bun, docker (for local Postgres), task. Installed skills must sit side by side.
set -euo pipefail

APP_DIR="${1:?usage: new-app.sh <app-dir> <go-module-path> \"<App Name>\"}"
MODULE="${2:?go module path required, e.g. github.com/acme/my-app}"
APP_NAME="${3:?app display name required}"
SKILLS="$(cd "$(dirname "$0")/../.." && pwd)"
SLUG="$(echo "$APP_NAME" | tr '[:upper:]' '[:lower:]' | sed -E 's/[^a-z0-9]+/-/g; s/^-|-$//g')"

mkdir -p "$APP_DIR"
APP_DIR="$(cd "$APP_DIR" && pwd)"
for f in backend frontend Taskfile.yml render.yaml; do
  if [[ -e "$APP_DIR/$f" ]]; then
    echo "error: $APP_DIR/$f already exists; this script only creates new apps" >&2
    exit 1
  fi
done

echo "==> root files"
A="$SKILLS/crud-app-scaffold/assets"
D="$SKILLS/crud-deploy-render/assets"
cp "$A/Taskfile.yml" "$A/docker-compose.yml" "$APP_DIR/"
cp "$A/env.example" "$APP_DIR/.env.example"
[[ -e "$APP_DIR/.gitignore" ]] || cp "$A/gitignore" "$APP_DIR/.gitignore"
[[ -e "$APP_DIR/AGENTS.md" ]] || sed "s/{{APP_NAME}}/$APP_NAME/g" "$A/AGENTS.md" >"$APP_DIR/AGENTS.md"
[[ -e "$APP_DIR/APP_SPEC.md" ]] || sed "s/{{APP_NAME}}/$APP_NAME/g" "$A/APP_SPEC.md" >"$APP_DIR/APP_SPEC.md"
cp "$D/Dockerfile" "$APP_DIR/"
cp "$D/dockerignore" "$APP_DIR/.dockerignore"
sed -e "s/name: app #/name: $SLUG #/" -e "s/name: app-db/name: $SLUG-db/g" "$D/render.yaml" >"$APP_DIR/render.yaml"
sed -i.bak "s/^  APP: app #/  APP: $SLUG #/" "$APP_DIR/Taskfile.yml" && rm "$APP_DIR/Taskfile.yml.bak"

"$SKILLS/crud-backend-go/scripts/create-backend.sh" "$APP_DIR" "$MODULE"
GO_MINOR="$(cd "$APP_DIR/backend" && go list -m -f '{{.GoVersion}}')"
sed -i.bak -E "s#golang:[0-9.]+-alpine#golang:$GO_MINOR-alpine#" "$APP_DIR/Dockerfile" && rm "$APP_DIR/Dockerfile.bak"
"$SKILLS/crud-frontend-react/scripts/create-frontend.sh" "$APP_DIR" "$APP_NAME"

[[ -e "$APP_DIR/.env" ]] || cp "$APP_DIR/.env.example" "$APP_DIR/.env"
if [[ ! -d "$APP_DIR/.git" ]]; then
  git -C "$APP_DIR" init -q -b main
fi

cat <<EOF

Done: $APP_DIR
Next:
  cd $APP_DIR
  task dev          # Postgres + API on :8080 + UI on http://localhost:5173
  # first account you sign up locally becomes admin
EOF

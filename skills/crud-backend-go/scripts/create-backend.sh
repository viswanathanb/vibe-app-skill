#!/usr/bin/env bash
# Creates <app>/backend: Go + Gin + GORM + Postgres skeleton with auth (JWT cookie), RBAC + ReBAC
# and teams, copied from the crud-* skills, then verifies it builds and its tests pass.
#
# Usage: create-backend.sh <app-dir> <go-module-path>     e.g. create-backend.sh . github.com/acme/talos
# Needs: go. Sibling skills (crud-auth, crud-authz, crud-quality-gates) must be installed next to this one.
set -euo pipefail

APP_DIR="$(cd "${1:?usage: create-backend.sh <app-dir> <go-module-path>}" && pwd)"
MODULE="${2:?usage: create-backend.sh <app-dir> <go-module-path>}"
SKILLS="$(cd "$(dirname "$0")/../.." && pwd)"
BE="$APP_DIR/backend"

if [[ -e "$BE" ]]; then
  echo "error: $BE already exists" >&2
  exit 1
fi
mkdir -p "$BE"

echo "==> copying skeleton"
for skill in crud-backend-go crud-auth crud-authz crud-quality-gates; do
  src="$SKILLS/$skill/assets/backend"
  [[ -d "$src" ]] && cp -R "$src/." "$BE/"
done

cd "$BE"
echo "==> setting module path to $MODULE"
find . -name '*.go' -exec perl -pi -e "s#\"example\\.com/app/#\"$MODULE/#g" {} +
go mod edit -module "$MODULE"

echo "==> resolving dependencies"
go get github.com/gin-gonic/gin gorm.io/gorm gorm.io/driver/postgres \
  github.com/golang-jwt/jwt/v5 golang.org/x/crypto >/dev/null 2>&1
go mod tidy
# Fresh projects start on the latest patch release of every required module (security fixes).
awk '/^require \(/{r=1;next} r&&/^\)/{r=0} r{print $1, $2} /^require [^(]/{print $2, $3}' go.mod |
  while read -r path ver; do
    [[ "$ver" == *-* ]] && continue # pseudo-versions / pre-releases
    base="${ver%.*}"
    best="$(go list -m -versions "$path" 2>/dev/null | tr ' ' '\n' | grep -E "^${base//./\\.}\.[0-9]+$" |
      sort -t. -k3,3n | tail -1)"
    if [[ -n "$best" && "$best" != "$ver" ]]; then
      go get "$path@$best" >/dev/null 2>&1 || true
    fi
  done
go mod tidy

# Pin the exact toolchain so builds are reproducible and osv-scanner checks the real stdlib version
# (a bare `go 1.26.0` line makes it report every stdlib CVE fixed since .0).
GOV="$(go env GOVERSION | cut -d' ' -f1)"
MINOR="$(echo "$GOV" | sed -E 's/^go([0-9]+\.[0-9]+).*/\1/')"
go mod edit -go="$MINOR" -toolchain="$GOV"
echo "==> go $MINOR, toolchain $GOV (Dockerfile must use golang:$MINOR-alpine)"

echo "==> verifying (gofmt, vet, build, test)"
test -z "$(gofmt -l .)" || { gofmt -l .; exit 1; }
go vet ./...
go build ./...
go test ./... >/dev/null
echo "OK: $BE"

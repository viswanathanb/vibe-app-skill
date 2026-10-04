#!/usr/bin/env bash
# Assembles the backend skeleton from all skills into a temp Go module and runs the quality gates.
# Usage: scripts/validate-backend.sh [workdir]   (workdir defaults to a fresh temp dir)
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${1:-$(mktemp -d -t vibe-backend-XXXXXX)}"
OUT="$WORK/backend"

rm -rf "$OUT"
mkdir -p "$OUT"
for skill in crud-backend-go crud-auth crud-authz crud-resource crud-quality-gates; do
  src="$ROOT/skills/$skill/assets/backend"
  [[ -d "$src" ]] && cp -R "$src/." "$OUT/"
done
# Optional OIDC enhancement must also compile against the skeleton.
cp "$ROOT/skills/crud-auth/references/oidc/oidc.go" "$OUT/internal/auth/oidc.go"

cd "$OUT"
echo "==> assembled in $OUT"
for f in $(find . -name '*.go'); do
  if [[ "$(grep -c '^package ' "$f")" -ne 1 ]]; then
    echo "duplicate or missing package clause: $f"
    exit 1
  fi
done
go get github.com/gin-gonic/gin gorm.io/gorm gorm.io/driver/postgres \
  github.com/golang-jwt/jwt/v5 golang.org/x/crypto github.com/coreos/go-oidc/v3/oidc golang.org/x/oauth2 >/dev/null
go mod tidy

echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [[ -n "$unformatted" ]]; then
  echo "$unformatted"
  exit 1
fi
echo "==> go vet"
go vet ./...
echo "==> go build"
go build ./...
echo "==> go test"
go test ./...
echo "==> golangci-lint"
golangci-lint run ./...
echo "OK: backend skeleton builds, tests and lints cleanly ($OUT)"

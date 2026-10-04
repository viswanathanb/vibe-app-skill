#!/usr/bin/env bash
# Full end-to-end check of the skills exactly as an agent would use them:
# new-app.sh -> add-resource.sh (single + multi-word) -> lint, tests, builds, osv-scanner -> Docker image build.
# Usage: scripts/e2e.sh [workdir]
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${1:-$(mktemp -d -t vibe-e2e-XXXXXX)}"
APP="$WORK/e2e-app"
S="$ROOT/skills"
rm -rf "$APP"

echo "==> new-app.sh"
"$S/crud-app-scaffold/scripts/new-app.sh" "$APP" github.com/example/e2e-app "E2E App" >"$WORK/new-app.log" 2>&1 ||
  { tail -40 "$WORK/new-app.log"; exit 1; }

echo "==> add-resource.sh Cluster, MachineConfig"
"$S/crud-resource/scripts/add-resource.sh" "$APP" Cluster >"$WORK/add1.log" 2>&1 || { tail -40 "$WORK/add1.log"; exit 1; }
"$S/crud-resource/scripts/add-resource.sh" "$APP" MachineConfig >"$WORK/add2.log" 2>&1 || { tail -40 "$WORK/add2.log"; exit 1; }

cd "$APP"
echo "==> task lint"
task lint >"$WORK/lint.log" 2>&1 || { tail -40 "$WORK/lint.log"; exit 1; }
echo "==> task test"
task test >"$WORK/test.log" 2>&1 || { tail -40 "$WORK/test.log"; exit 1; }
echo "==> task build"
task build >"$WORK/build.log" 2>&1 || { tail -40 "$WORK/build.log"; exit 1; }
echo "==> osv-scanner"
if ! osv-scanner scan source --no-call-analysis=go -r . >"$WORK/osv.log" 2>&1; then
  echo "WARN: osv-scanner reported findings (not fatal here; review them):"
  grep -E '^\| https' "$WORK/osv.log" || tail -20 "$WORK/osv.log"
fi
if command -v docker >/dev/null && docker info >/dev/null 2>&1; then
  echo "==> docker build"
  docker build -q -t vibe-e2e:local . >"$WORK/docker.log" 2>&1 || { tail -40 "$WORK/docker.log"; exit 1; }
fi
echo "PASS: $APP"

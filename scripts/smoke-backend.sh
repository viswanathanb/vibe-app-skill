#!/usr/bin/env bash
# End-to-end smoke test: assembles the backend (skeleton + the crud-resource `project` example wired in
# exactly as the crud-resource skill describes), starts Postgres in Docker, runs the API and exercises
# auth, RBAC and ReBAC flows with curl. Requires: go, docker, curl, jq.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${1:-$(mktemp -d -t vibe-smoke-XXXXXX)}"
"$ROOT/scripts/validate-backend.sh" "$WORK" >"$WORK/validate.log" 2>&1 || { tail -30 "$WORK/validate.log"; exit 1; }
B="$WORK/backend"

# --- apply the crud-resource wiring steps for `project` ---
perl -0pi -e 's|(\t\t// crud:models)|\t\t&project.Project{},\n$1|' "$B/internal/server/server.go"
perl -0pi -e 's|(\t// crud:routes)|\tproject.NewController(project.NewService(db, project.NewRepository(db), az)).Register(authed)\n$1|' "$B/internal/server/server.go"
perl -0pi -e 's|("example.com/app/internal/httpx"\n)|$1\t"example.com/app/internal/project"\n|' "$B/internal/server/server.go"
perl -0pi -e 's|/\* crud:actions \*/|, "project:create" /* crud:actions */|' "$B/internal/authz/rbac.go"
perl -0pi -e 's|(\t// crud:schema)|\t"project": {\n\t\tRelations: map[string][]string{"owner": {"user"}, "editor": {"user", "team#member"}, "viewer": {"user", "team#member"}},\n\t\tPermissions: map[string][]string{"view": {"viewer", "edit"}, "edit": {"editor", "manage"}, "delete": {"owner"}, "manage": {"owner"}},\n\t},\n$1|' "$B/internal/authz/app_schema.go"
(cd "$B" && gofmt -w internal && go build -o "$WORK/server" ./cmd/server)

# --- Postgres + server ---
PGPORT=$((20000 + RANDOM % 10000))
APIPORT=$((30000 + RANDOM % 10000))
CID=$(docker run -d --rm -e POSTGRES_USER=app -e POSTGRES_PASSWORD=app -e POSTGRES_DB=app -p "$PGPORT:5432" postgres:18-alpine 2>/dev/null)
cleanup() { [[ -n "${SPID:-}" ]] && kill "$SPID" 2>/dev/null || true; docker stop "$CID" >/dev/null; }
trap cleanup EXIT
for _ in $(seq 1 30); do docker exec "$CID" pg_isready -U app -d app >/dev/null 2>&1 && break; sleep 1; done
sleep 1

DATABASE_URL="postgres://app:app@localhost:$PGPORT/app?sslmode=disable" PORT="$APIPORT" APP_ENV=development \
  "$WORK/server" >"$WORK/server.log" 2>&1 &
SPID=$!
BASE="http://localhost:$APIPORT"
for _ in $(seq 1 30); do curl -fs "$BASE/healthz" >/dev/null 2>&1 && break; sleep 1; done

pass=0
fail() { echo "FAIL: $*"; echo "--- server log ---"; tail -20 "$WORK/server.log"; exit 1; }
expect() { # expect <desc> <want-status> <got-status>
  [[ "$2" == "$3" ]] || fail "$1: want HTTP $2, got $3 ($(cat "$WORK/body"))"
  pass=$((pass + 1)); echo "ok  $1"
}
# req <user> <method> <path> [json]  -> sets $code, body in $WORK/body
req() {
  local jar="$WORK/$1.jar"
  local args=(-s -o "$WORK/body" -w '%{http_code}' -X "$2" -b "$jar" -c "$jar" -H 'X-Requested-With: XMLHttpRequest')
  [[ $# -ge 4 ]] && args+=(-H 'Content-Type: application/json' -d "$4")
  code=$(curl "${args[@]}" "$BASE$3")
}
body() { jq -r "$1" "$WORK/body"; }

req anon GET /healthz; expect "healthz" 200 "$code"
code=$(curl -s -o "$WORK/body" -w '%{http_code}' -X POST "$BASE/api/auth/login" -d '{}'); expect "CSRF guard rejects missing header" 403 "$code"
req anon GET /api/nope; expect "unknown api route is JSON 404" 404 "$code"
req anon GET /api/projects; expect "anonymous is 401" 401 "$code"

for u in alice bob carol; do
  req $u POST /api/auth/signup "{\"email\":\"$u@example.com\",\"password\":\"password123\",\"name\":\"$u\"}"
  expect "signup $u" 201 "$code"
done
req alice GET /api/auth/me; expect "me" 200 "$code"
[[ "$(body .user.role)" == admin ]] || fail "first dev user should be admin"
req bob GET /api/auth/me; [[ "$(body .user.role)" == member ]] || fail "bob should be member"
req bob POST /api/auth/signup '{"email":"bob@example.com","password":"password123"}'; expect "duplicate email" 409 "$code"
req bob POST /api/projects '{"name":""}'; expect "validation error" 400 "$code"
[[ "$(body .error.details.name)" == required ]] || fail "validation details should use json field names"

req bob POST /api/projects '{"name":"Apollo","description":"moon"}'; expect "bob creates project" 201 "$code"
P=$(body .id)
req carol GET /api/projects; expect "carol lists" 200 "$code"
[[ "$(body .total)" == 0 ]] || fail "carol should see no projects"
req carol GET "/api/projects/$P"; expect "carol cannot view" 403 "$code"

req bob POST /api/teams '{"name":"Crew"}'; expect "bob creates team" 201 "$code"
T=$(body .id)
req bob POST "/api/authz/team/$T/tuples" '{"relation":"member","email":"carol@example.com"}'; expect "add carol to team" 204 "$code"
req carol POST "/api/authz/team/$T/tuples" '{"relation":"member","email":"alice@example.com"}'; expect "carol cannot manage team" 403 "$code"
req bob POST "/api/authz/project/$P/tuples" "{\"relation\":\"viewer\",\"subject\":\"team:$T#member\"}"; expect "share project with team" 204 "$code"
req bob POST "/api/authz/project/$P/tuples" '{"relation":"owner","subject":"team:1#member"}'; expect "owner cannot be a team" 400 "$code"

req carol GET /api/projects; [[ "$(body .total)" == 1 ]] || fail "carol should now see 1 project: $(cat "$WORK/body")"
pass=$((pass + 1)); echo "ok  carol sees shared project via team"
req carol GET "/api/projects/$P"; expect "carol views" 200 "$code"
[[ "$(body '.permissions|join(",")')" == view ]] || fail "carol perms: $(body .permissions)"
req carol PATCH "/api/projects/$P" '{"name":"Hacked"}'; expect "carol cannot edit" 403 "$code"
req bob PATCH "/api/projects/$P" '{"status":"archived"}'; expect "bob edits" 200 "$code"
req bob GET "/api/authz/project/$P/tuples"; expect "bob lists access" 200 "$code"
req bob DELETE "/api/authz/project/$P/tuples?relation=owner&subject=user:2"; expect "cannot remove last owner" 409 "$code"
req carol GET "/api/authz/project/$P/permissions"; expect "permissions endpoint" 200 "$code"

req bob GET /api/users; expect "member cannot list users" 403 "$code"
req alice GET /api/users; expect "admin lists users" 200 "$code"
[[ "$(body .total)" == 3 ]] || fail "expected 3 users"
req alice PATCH /api/users/1 '{"role":"member"}'; expect "cannot demote last admin" 409 "$code"
req alice PATCH /api/users/3 '{"role":"viewer"}'; expect "admin sets carol viewer" 200 "$code"
req carol POST /api/teams '{"name":"Nope"}'; expect "viewer cannot create teams" 403 "$code"

req bob DELETE "/api/teams/$T"; expect "bob deletes team" 204 "$code"
req carol GET /api/projects; [[ "$(body .total)" == 0 ]] || fail "team deletion must revoke shared access"
pass=$((pass + 1)); echo "ok  deleting team revokes its shares"
req bob DELETE "/api/projects/$P"; expect "bob deletes project" 204 "$code"
req alice GET "/api/projects/$P"; expect "deleted project is 404 for admin" 404 "$code"
req bob POST /api/auth/logout; expect "logout" 204 "$code"
req bob GET /api/auth/me; expect "after logout" 401 "$code"

echo "PASS: $pass checks"

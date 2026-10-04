#!/usr/bin/env bash
# Adds a new "owned + shareable" CRUD resource to an app built with the crud-* skills by cloning the
# golden `project` example (backend package + frontend feature) and wiring it at the crud:* markers.
# Afterwards, edit the fields (see SKILL.md step 3).
#
# Usage: add-resource.sh <app-dir> <PascalName> [PascalPlural]
#   e.g. add-resource.sh . Cluster
#        add-resource.sh . MachineConfig MachineConfigs
set -euo pipefail

APP_DIR="$(cd "${1:?usage: add-resource.sh <app-dir> <PascalName> [PascalPlural]}" && pwd)"
PASCAL="${2:?PascalName required, e.g. Cluster}"
PASCAL_PLURAL="${3:-${PASCAL}s}"
SKILL="$(cd "$(dirname "$0")/.." && pwd)"
BE="$APP_DIR/backend"
FE="$APP_DIR/frontend"

[[ "$PASCAL" =~ ^[A-Z][A-Za-z0-9]*$ && "$PASCAL_PLURAL" =~ ^[A-Z][A-Za-z0-9]*$ ]] ||
  { echo "error: names must be PascalCase letters/digits (e.g. MachineConfig)" >&2; exit 1; }
[[ "$PASCAL" != "Project" ]] || { echo "error: 'Project' is the template name; copy assets manually instead" >&2; exit 1; }

words() { echo "$1" | perl -pe 's/(?<=[a-z0-9])([A-Z])/ $1/g'; } # MachineConfig -> Machine Config
lower() { echo "$1" | tr '[:upper:]' '[:lower:]'; }
PKG="$(lower "$PASCAL")"                                       # machineconfig  (Go package)
SNAKE="$(lower "$(words "$PASCAL" | tr ' ' '_')")"             # machine_config (authz type)
KEBAB="$(lower "$(words "$PASCAL" | tr ' ' '-')")"             # machine-config
KEBAB_PLURAL="$(lower "$(words "$PASCAL_PLURAL" | tr ' ' '-')")" # machine-configs (URLs, folder)
CAMEL="$(echo "${PASCAL:0:1}" | tr '[:upper:]' '[:lower:]')${PASCAL:1}"   # machineConfig
HUMAN_PLURAL="$(words "$PASCAL_PLURAL")"                       # Machine Configs
HUMAN_PLURAL_LOWER="$(lower "$HUMAN_PLURAL")"                  # machine configs
HUMAN_LOWER="$(lower "$(words "$PASCAL")")"                   # machine config
ucfirst() { echo "$(echo "${1:0:1}" | tr '[:lower:]' '[:upper:]')${1:1}"; }
HUMAN_LABEL="$(ucfirst "$HUMAN_LOWER")"                        # Machine config
HUMAN_PLURAL_LABEL="$(ucfirst "$HUMAN_PLURAL_LOWER")"          # Machine configs
MODULE="$(cd "$BE" && go list -m)"
export PASCAL PASCAL_PLURAL PKG SNAKE KEBAB KEBAB_PLURAL CAMEL HUMAN_PLURAL_LOWER MODULE HUMAN_LOWER HUMAN_LABEL HUMAN_PLURAL_LABEL

# Rewrites the `project` template identifiers/labels in a frontend file (TS/TSX, e2e spec).
render_fe() {
  perl -pi -e '
    s#"project:create"#"$ENV{SNAKE}:create"#g;
    s#"project"#"$ENV{SNAKE}"#g;
    s#/projects#/$ENV{KEBAB_PLURAL}#g;
    s#"Projects"#"$ENV{HUMAN_PLURAL_LABEL}"#g;
    s#Projects you own#$ENV{HUMAN_PLURAL_LABEL} you own#g;
    s#Project (created|updated|deleted)#$ENV{HUMAN_LABEL} $1#g;
    s#(Edit|New) project#$1 $ENV{HUMAN_LOWER}#g;
    s#Projects#$ENV{PASCAL_PLURAL}#g;
    s#Project#$ENV{PASCAL}#g;
    s#projects#$ENV{HUMAN_PLURAL_LOWER}#g;
    s#project-#$ENV{KEBAB}-#g;
    s#backend/internal/project/#backend/internal/$ENV{PKG}/#g;
    s#project#$ENV{CAMEL}#g;
  ' "$1"
}

# Browser test for the resource, only when the app has Playwright (crud-e2e-playwright).
write_e2e_spec() {
  [[ -d "$FE/e2e" ]] || return 0
  local dest="$FE/e2e/$KEBAB_PLURAL.spec.ts"
  if [[ -e "$dest" ]]; then echo "skip: $dest exists"; return 0; fi
  cp "$SKILL/assets/frontend/e2e/projects.spec.ts" "$dest"
  render_fe "$dest"
  echo "==> frontend/e2e/$KEBAB_PLURAL.spec.ts"
}

if [[ "${E2E_ONLY:-}" == 1 ]]; then
  write_e2e_spec
  exit 0
fi

[[ ! -e "$BE/internal/$PKG" ]] || { echo "error: backend/internal/$PKG already exists" >&2; exit 1; }
[[ ! -e "$FE/src/features/$KEBAB_PLURAL" ]] || { echo "error: frontend/src/features/$KEBAB_PLURAL exists" >&2; exit 1; }

echo "==> backend/internal/$PKG"
cp -R "$SKILL/assets/backend/internal/project" "$BE/internal/$PKG"
for f in "$BE/internal/$PKG"/*.go; do
  perl -pi -e '
    s#"example\.com/app/#"$ENV{MODULE}/#g;
    s#ObjectType = "project"#ObjectType = "$ENV{SNAKE}"#g;
    s#"/projects"#"/$ENV{KEBAB_PLURAL}"#g;
    s#Projects#$ENV{PASCAL_PLURAL}#g;
    s#Project#$ENV{PASCAL}#g;
    s#projects#$ENV{HUMAN_PLURAL_LOWER}#g;
    s#project#$ENV{PKG}#g;
  ' "$f"
done

echo "==> frontend/src/features/$KEBAB_PLURAL"
mkdir -p "$FE/src/features/$KEBAB_PLURAL"
src="$SKILL/assets/frontend/src/features/projects"
cp "$src/api.ts" "$FE/src/features/$KEBAB_PLURAL/api.ts"
cp "$src/ProjectsPage.tsx" "$FE/src/features/$KEBAB_PLURAL/${PASCAL_PLURAL}Page.tsx"
cp "$src/ProjectDetailPage.tsx" "$FE/src/features/$KEBAB_PLURAL/${PASCAL}DetailPage.tsx"
cp "$src/ProjectFormDialog.tsx" "$FE/src/features/$KEBAB_PLURAL/${PASCAL}FormDialog.tsx"
for f in "$FE/src/features/$KEBAB_PLURAL"/*; do
  render_fe "$f"
done
write_e2e_spec

echo "==> wiring markers"
perl -0pi -e '
  s#(\t\t// crud:models)#\t\t&$ENV{PKG}.$ENV{PASCAL}\{\},\n$1#;
  s#(\t// crud:routes)#\t$ENV{PKG}.NewController($ENV{PKG}.NewService(db, $ENV{PKG}.NewRepository(db), az)).Register(authed)\n$1#;
  s#(\t"\Q$ENV{MODULE}\E/internal/httpx"\n)#$1\t"$ENV{MODULE}/internal/$ENV{PKG}"\n#;
' "$BE/internal/server/server.go"
perl -0pi -e 's#/\* crud:actions \*/#, "$ENV{SNAKE}:create" /* crud:actions */#' "$BE/internal/authz/rbac.go"
perl -0pi -e '
  s|(\t// crud:schema)|\t"$ENV{SNAKE}": {\n\t\tRelations: map[string][]string{\n\t\t\t"owner":  {"user"},\n\t\t\t"editor": {"user", "team#member"},\n\t\t\t"viewer": {"user", "team#member"},\n\t\t},\n\t\tPermissions: map[string][]string{\n\t\t\t"view":   {"viewer", "edit"},\n\t\t\t"edit":   {"editor", "manage"},\n\t\t\t"delete": {"owner"},\n\t\t\t"manage": {"owner"},\n\t\t},\n\t},\n$1|;
' "$BE/internal/authz/app_schema.go"

perl -0pi -e '
  s#(import \{ HomePage \})#import { $ENV{PASCAL}DetailPage } from "\@/features/$ENV{KEBAB_PLURAL}/$ENV{PASCAL}DetailPage";\nimport { $ENV{PASCAL_PLURAL}Page } from "\@/features/$ENV{KEBAB_PLURAL}/$ENV{PASCAL_PLURAL}Page";\n$1#;
  s#(\s*// crud:routes)#\n          { path: "$ENV{KEBAB_PLURAL}", element: <$ENV{PASCAL_PLURAL}Page /> },\n          { path: "$ENV{KEBAB_PLURAL}/:id", element: <$ENV{PASCAL}DetailPage /> },$1#;
' "$FE/src/App.tsx"
perl -0pi -e 's#(\s*// crud:nav)#\n  { to: "/$ENV{KEBAB_PLURAL}", label: "$ENV{HUMAN_PLURAL_LABEL}" },$1#' "$FE/src/components/AppLayout.tsx"
perl -0pi -e 's#(\s*\{/\* crud:home)#\n        <Link to="/$ENV{KEBAB_PLURAL}">\n          <Card className="transition-colors hover:bg-muted/50">\n            <CardHeader>\n              <CardTitle>$ENV{HUMAN_PLURAL_LABEL}</CardTitle>\n              <CardDescription>Browse and manage $ENV{HUMAN_PLURAL_LOWER}.</CardDescription>\n            </CardHeader>\n          </Card>\n        </Link>$1#' "$FE/src/pages/HomePage.tsx"

echo "==> formatting + verifying"
(cd "$BE" && gofmt -w internal && go build ./... && go vet ./...)
(cd "$FE" && bunx prettier --write src $([[ -d e2e ]] && echo e2e) >/dev/null && bun run typecheck && bun run lint)

cat <<EOF

Added $PASCAL ($SNAKE). Now customise:
  backend/internal/$PKG/model.go        fields, CreateInput/UpdateInput binding tags
  backend/internal/$PKG/repository.go   sortColumns, search columns, filters
  backend/internal/$PKG/service.go      copy new fields in Create/Update; business rules
  frontend/src/features/$KEBAB_PLURAL/  types in api.ts, zod schema + inputs, table columns, detail fields, labels
  frontend/e2e/$KEBAB_PLURAL.spec.ts     fill the new required fields in the create/edit steps (if present)
Access pattern: owned + shareable. For child/catalog see crud-authz/references/patterns.md.
EOF

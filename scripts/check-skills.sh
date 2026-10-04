#!/usr/bin/env bash
# Static checks for the skills: frontmatter, name/folder match, description length, file size,
# executable scripts, Go files with exactly one package clause.
set -euo pipefail
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
fail=0
err() { echo "FAIL: $*"; fail=1; }

for dir in "$ROOT"/skills/*/; do
  name="$(basename "$dir")"
  f="$dir/SKILL.md"
  [[ -f "$f" ]] || { err "$name: missing SKILL.md"; continue; }
  [[ "$(head -1 "$f")" == "---" ]] || err "$name: SKILL.md must start with YAML frontmatter"
  fm="$(awk 'NR==1{next} /^---$/{exit} {print}' "$f")"
  fm_name="$(echo "$fm" | sed -n 's/^name: *//p')"
  [[ "$fm_name" == "$name" ]] || err "$name: frontmatter name '$fm_name' != folder"
  [[ "$name" =~ ^[a-z0-9-]{1,64}$ ]] || err "$name: invalid skill name"
  desc="$(echo "$fm" | sed -n 's/^description: *//p')"
  [[ -n "$desc" ]] || err "$name: missing description"
  (( ${#desc} <= 1024 )) || err "$name: description is ${#desc} chars (max 1024)"
  lines="$(wc -l <"$f")"
  (( lines <= 500 )) || err "$name: SKILL.md has $lines lines (keep under 500)"
  for s in "$dir"scripts/*.sh; do
    [[ -e "$s" ]] || continue
    [[ -x "$s" ]] || err "$name: $(basename "$s") is not executable"
    bash -n "$s" || err "$name: $(basename "$s") has syntax errors"
  done
  # Relative links in SKILL.md must exist.
  while IFS= read -r link; do
    [[ -e "$dir/$link" ]] || err "$name: broken link $link"
  done < <(grep -oE '\]\(\./[^)#]+' "$f" | sed 's/](\.\///')
done

while IFS= read -r go; do
  [[ "$(grep -c '^package ' "$go")" -eq 1 ]] || err "$go: must have exactly one package clause"
done < <(find "$ROOT/skills" -name '*.go')

# Plugin manifests (VS Code/Copilot root plugin.json, Claude .claude-plugin/) must agree on name and version.
for f in plugin.json .claude-plugin/plugin.json .claude-plugin/marketplace.json; do
  jq empty "$ROOT/$f" 2>/dev/null || err "$f: invalid JSON"
done
names="$(jq -r .name "$ROOT/plugin.json" "$ROOT/.claude-plugin/plugin.json"; jq -r '.plugins[0].name' "$ROOT/.claude-plugin/marketplace.json")"
[[ "$(echo "$names" | sort -u | wc -l)" -eq 1 ]] || err "plugin names differ: $(echo $names)"
versions="$(jq -r .version "$ROOT/plugin.json" "$ROOT/.claude-plugin/plugin.json"; jq -r '.plugins[0].version' "$ROOT/.claude-plugin/marketplace.json")"
[[ "$(echo "$versions" | sort -u | wc -l)" -eq 1 ]] || err "plugin versions differ: $(echo $versions)"
if command -v claude >/dev/null; then
  (cd "$ROOT" && claude plugin validate . >/dev/null) || err "claude plugin validate failed"
fi
bash -n "$ROOT/install.sh" || err "install.sh has syntax errors"

if (( fail )); then exit 1; fi
echo "OK: skills look well-formed"

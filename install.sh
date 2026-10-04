#!/usr/bin/env bash
# Installs the crud-* skills from GitHub (or a local checkout) into a project or your user profile.
#
#   curl -fsSL https://raw.githubusercontent.com/viswanathanb/vibe-app-skill/main/install.sh | bash
#   curl -fsSL https://raw.githubusercontent.com/viswanathanb/vibe-app-skill/main/install.sh | bash -s -- --tools claude
#
# Options:
#   --dir <path>      project to install into (default: current directory)
#   --global          install into your user profile instead (~/.claude, ~/.copilot, ~/.agents)
#   --tools <list>    comma-separated: claude, github, agents (default: claude,github,agents)
#   --ref <git ref>   branch or tag to install (default: main)
#   --from <path>     install from a local checkout instead of downloading
#   --repo <o/r>      GitHub repository (default: viswanathanb/vibe-app-skill)
set -euo pipefail

REPO="viswanathanb/vibe-app-skill"
REF="main"
DIR="$PWD"
TOOLS="claude,github,agents"
GLOBAL=0
FROM=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --dir) DIR="$2"; shift 2 ;;
    --global) GLOBAL=1; shift ;;
    --tools) TOOLS="$2"; shift 2 ;;
    --ref) REF="$2"; shift 2 ;;
    --from) FROM="$2"; shift 2 ;;
    --repo) REPO="$2"; shift 2 ;;
    -h | --help) sed -n '2,15p' "$0" 2>/dev/null || echo "see https://github.com/$REPO#install"; exit 0 ;;
    *) echo "unknown option: $1" >&2; exit 1 ;;
  esac
done

if [[ -z "$FROM" ]]; then
  command -v curl >/dev/null && command -v tar >/dev/null || { echo "error: curl and tar are required" >&2; exit 1; }
  TMP="$(mktemp -d)"
  trap 'rm -rf "$TMP"' EXIT
  echo "==> downloading $REPO@$REF"
  curl -fsSL "https://codeload.github.com/$REPO/tar.gz/$REF" | tar -xz -C "$TMP"
  FROM="$(find "$TMP" -mindepth 1 -maxdepth 1 -type d | head -1)"
fi
SRC="$FROM/skills"
[[ -d "$SRC" ]] || { echo "error: no skills/ directory in $FROM" >&2; exit 1; }

dest_for() {
  local base
  if (( GLOBAL )); then base="$HOME"; else base="$DIR"; fi
  case "$1" in
    claude) echo "$base/.claude/skills" ;;
    agents) echo "$base/.agents/skills" ;;
    github) if (( GLOBAL )); then echo "$HOME/.copilot/skills"; else echo "$DIR/.github/skills"; fi ;;
    *) echo "unknown tool '$1' (use claude, github, agents)" >&2; return 1 ;;
  esac
}

IFS=',' read -r -a tools <<<"$TOOLS"
for tool in "${tools[@]}"; do
  dest="$(dest_for "$tool")"
  mkdir -p "$dest"
  count=0
  for skill in "$SRC"/*/; do
    name="$(basename "$skill")"
    rm -rf "${dest:?}/$name"
    cp -R "$skill" "$dest/$name"
    count=$((count + 1))
  done
  chmod +x "$dest"/*/scripts/*.sh 2>/dev/null || true
  echo "installed $count skills into $dest"
done

cat <<EOF

Next: open the project with your agent and ask, e.g.
  "Use the crud-app-scaffold skill to create <App Name> (Go module github.com/<you>/<app>) for managing ..."
EOF

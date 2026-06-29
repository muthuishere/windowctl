#!/usr/bin/env sh

set -eu

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
SKILL_NAME="window-ctl-skill"

link_skill() {
  root="$1"
  link="$root/$SKILL_NAME"
  mkdir -p "$root"
  rm -rf -- "$link"
  ln -s "$SCRIPT_DIR" "$link"
}

CLAUDE_ROOT=${CLAUDE_SKILLS_DIR:-"$HOME/.claude/skills"}
link_skill "$CLAUDE_ROOT"

if [ -n "${AGENTS_SKILLS_DIR:-}" ] || command -v codex >/dev/null 2>&1; then
  AGENTS_ROOT=${AGENTS_SKILLS_DIR:-"$HOME/.agents/skills"}
  link_skill "$AGENTS_ROOT"
fi

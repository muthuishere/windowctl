#!/usr/bin/env sh

set -eu

SKILL_NAME="window-ctl-skill"

remove_skill() {
  root="$1"
  link="$root/$SKILL_NAME"
  rm -rf -- "$link"
}

CLAUDE_ROOT=${CLAUDE_SKILLS_DIR:-"$HOME/.claude/skills"}
remove_skill "$CLAUDE_ROOT"

if [ -n "${AGENTS_SKILLS_DIR:-}" ] || command -v codex >/dev/null 2>&1; then
  AGENTS_ROOT=${AGENTS_SKILLS_DIR:-"$HOME/.agents/skills"}
  remove_skill "$AGENTS_ROOT"
fi

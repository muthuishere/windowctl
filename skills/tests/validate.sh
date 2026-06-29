#!/usr/bin/env bash
#
# Recipe format + cross-reference validator for window-ctl-skill.
#
# Asserts:
#   - SKILL.md has frontmatter with name + description
#   - SKILL.md's "Families at a glance" table cites every references/*.md
#   - every references/*.md has the `name: window-ctl-skill` frontmatter
#   - every `### <ID>:` recipe block has When-to-use, Command/Sequence,
#     Expected response/Synthesis, Common errors
#   - no duplicate recipe IDs across families
#   - composite recipes in recipes.md only call windowctl subcommands
#     that exist (windows, monitors, move, focus, permissions)
#
# Exit code = count of failures. Zero = all green.

set -u -o pipefail

SCRIPT_DIR=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd -P)
SKILL_DIR=$(CDPATH= cd -- "$SCRIPT_DIR/.." && pwd -P)
REF_DIR="$SKILL_DIR/references"

green()  { printf "\033[32m%s\033[0m\n" "$*"; }
red()    { printf "\033[31m%s\033[0m\n" "$*"; }
hdr()    { printf "\n\033[1m-- %s --\033[0m\n" "$*"; }

fails=0
passes=0
pass() { green "  OK  $*"; passes=$((passes+1)); }
fail() { red   "  FAIL $*"; fails=$((fails+1)); }

REF_FILES=(windows.md monitors.md move.md resize.md batch.md focus.md permissions.md zones.md recipes.md)

# 1. SKILL.md frontmatter
hdr "SKILL.md frontmatter"
if [[ ! -f "$SKILL_DIR/SKILL.md" ]]; then
  fail "SKILL.md missing"
else
  if grep -q '^name: window-ctl-skill$' "$SKILL_DIR/SKILL.md"; then
    pass "name: window-ctl-skill"
  else
    fail "name: window-ctl-skill not found in frontmatter"
  fi
  if grep -q '^description: ' "$SKILL_DIR/SKILL.md"; then
    pass "description present"
  else
    fail "description missing"
  fi
fi

# 2. SKILL.md cites every reference file
hdr "SKILL.md families table cross-refs"
for f in "${REF_FILES[@]}"; do
  if grep -q "references/$f" "$SKILL_DIR/SKILL.md"; then
    pass "SKILL.md cites references/$f"
  else
    fail "SKILL.md does not cite references/$f"
  fi
done

# 3. Every references/*.md has frontmatter
hdr "references/ frontmatter"
for f in "${REF_FILES[@]}"; do
  path="$REF_DIR/$f"
  if [[ ! -f "$path" ]]; then
    fail "$f: missing"
    continue
  fi
  if grep -q '^name: window-ctl-skill$' "$path"; then
    pass "$f: frontmatter ok"
  else
    fail "$f: missing 'name: window-ctl-skill' frontmatter"
  fi
done

# 4. Every recipe heading has the four required labels.
# Recipes use ##  XXX-N[-N]:  (h2). Match an uppercase prefix + dash-separated tail.
hdr "recipe blocks"
RECIPE_FILES=(windows.md monitors.md move.md resize.md batch.md focus.md permissions.md recipes.md)
RECIPE_RE='^## [A-Z][A-Z0-9-]+:'
for f in "${RECIPE_FILES[@]}"; do
  path="$REF_DIR/$f"
  [[ -f "$path" ]] || continue
  ids=$(grep -E "$RECIPE_RE" "$path" | sed -E 's/^## ([A-Z0-9-]+):.*/\1/' || true)
  for id in $ids; do
    block=$(awk "/^## $id:/{flag=1;next} /^## /{flag=0} flag" "$path")
    miss=()
    echo "$block" | grep -qE '\*\*(When to use|User intent)' || miss+=("When-to-use")
    echo "$block" | grep -qE '\*\*(Command|Call sequence|Sequence)' || miss+=("Command/Sequence")
    echo "$block" | grep -qE '\*\*(Expected response|Synthesis)' || miss+=("Expected/Synthesis")
    echo "$block" | grep -qE '\*\*Common errors' || miss+=("Common-errors")
    if [[ ${#miss[@]} -eq 0 ]]; then
      pass "$f $id complete"
    else
      fail "$f $id missing: ${miss[*]}"
    fi
  done
done

# 5. No duplicate recipe IDs across families
hdr "recipe ID uniqueness"
all_ids=$(for f in "${RECIPE_FILES[@]}"; do
  [[ -f "$REF_DIR/$f" ]] || continue
  grep -E "$RECIPE_RE" "$REF_DIR/$f" | sed -E 's/^## ([A-Z0-9-]+):.*/\1/'
done)
dups=$(echo "$all_ids" | sort | uniq -d)
if [[ -z "$dups" ]]; then
  pass "no duplicate recipe IDs"
else
  while IFS= read -r d; do fail "duplicate id: $d"; done <<< "$dups"
fi

# 6. recipes.md only invokes known subcommands
hdr "recipes.md subcommand whitelist"
KNOWN_RE='^(windows|monitors|move|focus|permissions|resize|batch)$'
bad=$(grep -E '^[[:space:]]*windowctl ' "$REF_DIR/recipes.md" 2>/dev/null \
  | sed -E 's/.*windowctl[[:space:]]+([a-z]+).*/\1/' \
  | sort -u \
  | grep -vE "$KNOWN_RE" || true)
if [[ -z "$bad" ]]; then
  pass "recipes.md uses only known subcommands"
else
  while IFS= read -r b; do fail "recipes.md uses unknown subcommand: $b"; done <<< "$bad"
fi

hdr "summary"
printf "  pass=%d fail=%d\n" "$passes" "$fails"
exit "$fails"

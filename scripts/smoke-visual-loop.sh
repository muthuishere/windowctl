#!/usr/bin/env bash
# smoke-visual-loop.sh — proves the complete visual loop on a SAFE local
# app (TextEdit) with NO hardcoded pixel coordinates:
#
#   seed a doc → find --text the seeded word (OCR) → click its coords →
#   type an edit → find --text the edit (OCR) to VERIFY the state changed.
#
# Every click point comes from `windowctl find --text`, never a literal.
# Emits one JSON receipt line per run to $WCTL_RECEIPTS (default stderr)
# so a fleet can audit the flow. macOS-only (native Vision OCR); skips
# with exit 0 elsewhere so CI on linux/windows stays green.
set -euo pipefail

if [[ "$(uname)" != "Darwin" ]]; then
  echo "smoke-visual-loop: not macOS, skipping"; exit 0
fi

BIN="${WCTL_BIN:-$(cd "$(dirname "$0")/.." && pwd)/bin/windowctl}"
[[ -x "$BIN" ]] || { echo "smoke-visual-loop: build first (task build) — $BIN missing"; exit 1; }

SEED="ClickTargetAlpha"
EDIT="OrganEdited$$"
TMP="$(mktemp -d)"
DOC="$TMP/wctl-visual-loop.txt"
printf '%s line one\nsecond line\n' "$SEED" > "$DOC"

receipt() {  # ok/why → one JSON line
  printf '{"flow":"visual-loop","app":"TextEdit","status":"%s","detail":"%s","ts":"%s"}\n' \
    "$1" "$2" "$(date -u +%FT%TZ)" >> "${WCTL_RECEIPTS:-/dev/stderr}"
}
cleanup() { osascript -e 'tell application "TextEdit" to quit saving no' >/dev/null 2>&1 || true; rm -rf "$TMP"; }
trap cleanup EXIT

# 1. Open the seeded doc and wait for its window.
open -a TextEdit "$DOC"
"$BIN" wait --app TextEdit --timeout 8000 >/dev/null || { receipt fail "window never appeared"; exit 1; }
sleep 1

# 2. OCR the TextEdit window for the seeded word → click coordinates.
COORDS="$("$BIN" find --text "$SEED" --app TextEdit --first)" \
  || { receipt fail "OCR did not find seed word $SEED"; exit 1; }
read -r CX CY <<<"$COORDS"
echo "smoke-visual-loop: OCR located '$SEED' at global point ($CX,$CY) — NO hardcoded pixels"

# 3. Click the found point (cursor into the doc), then select-all + type the edit.
"$BIN" mouse click --x "$CX" --y "$CY"
sleep 0.3
"$BIN" key --combo "cmd+a" --app TextEdit      # focus-verified select-all
"$BIN" type --text "$EDIT" --app TextEdit       # focus-verified type
sleep 0.6

# 4. VERIFY: OCR again; the edit text must now be on screen and the seed gone.
if "$BIN" find --text "$EDIT" --app TextEdit --first >/dev/null 2>&1; then
  echo "smoke-visual-loop: PASS — OCR confirms the typed edit '$EDIT' is on screen"
  receipt ok "found=$SEED@$CX,$CY typed=$EDIT verified=yes"
  exit 0
fi
receipt fail "edit text $EDIT not found after typing"
echo "smoke-visual-loop: FAIL — edit not visible after typing"; exit 1

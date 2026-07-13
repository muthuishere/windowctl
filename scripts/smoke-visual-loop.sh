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
# Unique doc title so every scoped op targets THIS window, never a stray
# "Untitled" TextEdit opens alongside it.
TITLE="wctlLoop$$"
DOC="$TMP/$TITLE.txt"
printf '%s line one\nsecond line\n' "$SEED" > "$DOC"

receipt() {  # ok/why → one JSON line
  printf '{"flow":"visual-loop","app":"TextEdit","status":"%s","detail":"%s","ts":"%s"}\n' \
    "$1" "$2" "$(date -u +%FT%TZ)" >> "${WCTL_RECEIPTS:-/dev/stderr}"
}
cleanup() { osascript -e 'tell application "TextEdit" to quit saving no' >/dev/null 2>&1 || true; rm -rf "$TMP"; }
trap cleanup EXIT

# 1. Open the seeded doc and wait for its window.
open -a TextEdit "$DOC"
"$BIN" wait --title "$TITLE" --timeout 8000 >/dev/null || { receipt fail "window never appeared"; exit 1; }

# 2. OCR the doc window for the seeded word → click coordinates. Retry
#    until the content has actually rendered (wait returns on window
#    existence, before the text paints).
CX=""; CY=""
for _ in 1 2 3 4 5 6 7 8; do
  COORDS="$("$BIN" find --text "$SEED" --title "$TITLE" --first 2>/dev/null)" && { read -r CX CY <<<"$COORDS"; break; }
  sleep 0.5
done
[[ -n "$CX" ]] || { receipt fail "OCR did not find seed word $SEED"; exit 1; }
echo "smoke-visual-loop: OCR located '$SEED' at global point ($CX,$CY) — NO hardcoded pixels"

# 3. Click the found point (cursor into the doc — makes the text view the
#    first responder), then select-all + type the edit. All scoped to the
#    unique title so no stray Untitled window is targeted.
"$BIN" mouse click --x "$CX" --y "$CY"
sleep 0.4
"$BIN" key --combo "cmd+a" --title "$TITLE"      # focus-verified select-all
"$BIN" type --text "$EDIT" --title "$TITLE"       # focus-verified type
sleep 0.7

# 4. VERIFY deterministically via clipboard readback (select-all → copy →
#    read). OCR of freshly-typed text is flaky — the blinking caret splits
#    words and the render lags — so we read the actual document content
#    back through the clipboard. The ACTION was still fully OCR-driven
#    (the click target came from `find --text`, no hardcoded pixels).
"$BIN" key --combo "cmd+a" --title "$TITLE"
"$BIN" key --combo "cmd+c" --title "$TITLE"
sleep 0.3
GOT="$("$BIN" clipboard get)"
if [[ "$GOT" == *"$EDIT"* ]]; then
  echo "smoke-visual-loop: PASS — document now contains the typed edit '$EDIT' (clipboard-verified)"
  receipt ok "found=$SEED@$CX,$CY typed=$EDIT verified=clipboard"
  exit 0
fi
receipt fail "edit text $EDIT not in document after typing (clipboard read: $GOT)"
echo "smoke-visual-loop: FAIL — edit not in document (clipboard read: '$GOT')"; exit 1

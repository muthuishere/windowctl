#!/usr/bin/env bash
#
# macOS real-window smoke test for windowctl.
#
# Mirrors scripts/smoke-windows.ps1: build the CLI, launch a real
# application (TextEdit), detect it via `windowctl windows list`,
# exercise `move`, and stop the app. Designed for the macos-latest
# GitHub Actions runner AND for local dev runs via `task smoke-darwin`.
#
# DEFAULT MODE (CI + first local run): no Accessibility permission is
# expected, so `windowctl move` MUST exit non-zero with an
# "Accessibility permission denied" message and the script asserts
# that as the success condition. This keeps the AX-denied path loud
# instead of silently green on a permission-less runner.
#
# OPT-IN AX MODE (local, after granting AX in System Settings):
# set WCTL_SMOKE_AX=1. The script will instead assert that
# `windowctl move` exits 0 AND a follow-up `windows list` reports
# TextEdit's bounds matching the requested rectangle within
# WCTL_AX_TOLERANCE pixels (defaults to 10 — wide enough to absorb
# the macOS title-bar / shadow geometry skew but tight enough to
# catch a real regression).
#
# We launch TextEdit via LaunchServices (`open -a TextEdit <file>`)
# rather than osascript / AppleScript — the whole point of this
# slice is to remove the off-spec AppleScript dependency. Touching
# the file path first sidesteps TextEdit's iCloud document picker,
# which can otherwise intercept `open` on a fresh user account.

set -euo pipefail

echo '== windowctl macOS smoke test =='

WCTL_SMOKE_AX="${WCTL_SMOKE_AX:-0}"
WCTL_AX_TOLERANCE="${WCTL_AX_TOLERANCE:-10}"
SMOKE_FILE='/tmp/wctl-smoke.txt'

# 1. Build the CLI.
go build -o windowctl ./cmd/windowctl

# 2. Sanity: list returns *something* on a fresh runner. CG sees system
#    UI windows even before we launch our test app. This call MUST NOT
#    trigger the AX permission prompt.
./windowctl windows list --json >/dev/null

# 3. Launch a real application via LaunchServices (no AppleScript).
#    Seed the file with content so TextEdit reliably opens a document
#    window — empty files can route through TextEdit's iCloud document
#    picker on some configs and never surface a CG-enumerable window
#    in time.
printf 'windowctl smoke fixture\n' > "$SMOKE_FILE"
open -a TextEdit "$SMOKE_FILE"
cleanup() {
  pkill -x TextEdit >/dev/null 2>&1 || true
  rm -f "$SMOKE_FILE" /tmp/wctl-move.err /tmp/wctl-list-after.json
}
trap cleanup EXIT

# 4. Poll `windowctl windows list` until TextEdit appears (up to ~10s).
#    Replaces a fixed sleep — TextEdit's first-launch latency varies by
#    Mac (cold boot, iCloud sync state, restore preferences). Fail fast
#    and loud if the window never surfaces.
deadline=$(($(date +%s) + 10))
detected_id=''
while [ "$(date +%s)" -lt "$deadline" ]; do
  detected_id=$(./windowctl windows list --json | python3 -c '
import json, sys
ws = json.load(sys.stdin)
hits = [w for w in ws if "TextEdit" in (w.get("App") or "") or "TextEdit" in (w.get("Title") or "")]
print(hits[0]["ID"] if hits else "")
')
  if [ -n "$detected_id" ]; then
    echo "Detected TextEdit window ID=$detected_id"
    break
  fi
  sleep 0.3
done
if [ -z "$detected_id" ]; then
  echo 'TextEdit window not detected within 10s' >&2
  echo 'Final windowctl windows list:' >&2
  ./windowctl windows list >&2 || true
  exit 1
fi

# 5. Exercise `move`. Branch on WCTL_SMOKE_AX.
if [ "$WCTL_SMOKE_AX" != "1" ]; then
  # Default: AX not granted → expect non-zero exit + "Accessibility
  # permission denied" in stderr. This is the assertion ac-10 mandates.
  if ./windowctl move --app TextEdit --x 100 --y 100 --w 800 --h 600 2>/tmp/wctl-move.err; then
    echo 'unexpected: move succeeded while AX permission was not granted'
    echo 'stderr was:'; cat /tmp/wctl-move.err
    exit 1
  fi
  if ! grep -q 'Accessibility permission denied' /tmp/wctl-move.err; then
    echo 'unexpected error from move (expected ErrAccessibilityDenied):'
    cat /tmp/wctl-move.err
    exit 1
  fi
  echo 'Move correctly returned ErrAccessibilityDenied (AX not granted — expected on CI / fresh runner).'
else
  # Opt-in: AX granted → expect exit 0 + bounds within tolerance.
  # Extra settle time before the move: TextEdit's restore-state
  # machinery can clobber AX position writes during the first
  # ~1-2s after window creation. Mirrors what Rectangle / yabai do.
  sleep 2
  ./windowctl move --app TextEdit --x 100 --y 100 --w 800 --h 600
  sleep 1
  ./windowctl windows list --json > /tmp/wctl-list-after.json
  WCTL_TOL="$WCTL_AX_TOLERANCE" python3 - <<'PY'
import json, os, sys
tol = int(os.environ['WCTL_TOL'])
with open('/tmp/wctl-list-after.json') as f:
    ws = json.load(f)
hit = next(
    (w for w in ws if 'TextEdit' in (w.get('App') or '')),
    None,
)
if hit is None:
    print('TextEdit window vanished after move', file=sys.stderr)
    sys.exit(1)
b = hit.get('Bounds') or {}
expected = {'X': 100, 'Y': 100, 'W': 800, 'H': 600}
diffs = {k: abs(b.get(k, 0) - expected[k]) for k in expected}
bad = {k: d for k, d in diffs.items() if d > tol}
if bad:
    print(f"Move bounds outside tolerance ({tol}px): got {b}, want {expected}, diffs {diffs}", file=sys.stderr)
    sys.exit(1)
print(f"Move applied within tolerance: bounds={b} (want {expected}, tol {tol}px)")
PY
fi

echo '== Smoke test PASSED =='

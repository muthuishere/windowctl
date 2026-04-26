#!/usr/bin/env bash
#
# macOS real-window smoke test for windowctl.
#
# Mirrors scripts/smoke-windows.ps1: build the CLI, launch a real
# application (TextEdit), detect it via `windowctl windows list`, run
# a `move`, and stop the app. Designed for the macos-latest GitHub
# Actions runner.
#
# Use as:
#   - if: matrix.os == 'macos-latest'
#     run: bash ./scripts/smoke-darwin.sh
#
# ListWindows / ListMonitors run against real CoreGraphics. Move and
# Focus are still stubbed (Accessibility API wiring is the next macOS
# slice), so the move step asserts ErrNotImplemented for now.

set -euo pipefail

echo '== windowctl macOS smoke test =='

# 1. Build the CLI.
go build -o windowctl ./cmd/windowctl

# 2. Sanity: list returns *something* on a fresh runner. CG sees system
#    UI windows even before we launch our test app.
./windowctl windows list --json >/dev/null

# 3. Launch a real application.
osascript -e 'tell application "TextEdit" to activate'
cleanup() {
  osascript -e 'tell application "TextEdit" to quit' >/dev/null 2>&1 || true
}
trap cleanup EXIT

sleep 2

# 4. Detect TextEdit via `windowctl windows list --json`.
raw=$(./windowctl windows list --json)
echo "$raw" | python3 - <<'PY'
import json, sys
ws = json.load(sys.stdin)
hit = next(
    (w for w in ws if 'TextEdit' in (w.get('App') or '') or 'TextEdit' in (w.get('Title') or '')),
    None,
)
if hit is None:
    print('TextEdit window not detected', file=sys.stderr)
    print(json.dumps(ws, indent=2), file=sys.stderr)
    sys.exit(1)
print(f"Detected TextEdit: ID={hit['ID']}, App='{hit.get('App','')}'", flush=True)
PY

# 5. Move is still stubbed on macOS (Accessibility API not wired). Assert
#    that we see the expected ErrNotImplemented rather than a silent pass
#    or some surprise behaviour. Once Move is real, flip this block to
#    require exit 0.
if ./windowctl move --title TextEdit --x 100 --y 100 --w 800 --h 600 2>/tmp/wctl-move.err; then
  echo 'unexpected: move succeeded while macOS adapter Move is stubbed'
  exit 1
fi
if ! grep -qi 'not implemented' /tmp/wctl-move.err; then
  echo 'unexpected error from move:'
  cat /tmp/wctl-move.err
  exit 1
fi
echo 'Move correctly returned ErrNotImplemented (macOS Accessibility wiring is the next slice).'

echo '== Smoke test PASSED =='

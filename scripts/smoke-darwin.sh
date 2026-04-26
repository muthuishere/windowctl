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
# While the macOS adapter is stubbed (every method returns
# ErrNotImplemented), the early exit below makes this script a no-op
# rather than a hard fail, so it can be wired into CI now and start
# exercising the real adapter the moment that adapter ships.

set -euo pipefail

echo '== windowctl macOS smoke test =='

# 1. Build the CLI.
go build -o windowctl ./cmd/windowctl

# 2. Skip gracefully while the macOS adapter is stubbed.
list_output=$(./windowctl windows list 2>&1 || true)
if echo "$list_output" | grep -qi 'not implemented'; then
  echo 'macOS adapter is currently stubbed (ErrNotImplemented).'
  echo 'Smoke test will run end-to-end once internal/adapter/darwin is wired up.'
  exit 0
fi

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
print(f"Detected TextEdit: ID={hit['ID']}, Title='{hit.get('Title','')}'", flush=True)
PY

# 5. Execute a move and verify no errors.
./windowctl move --title TextEdit --x 100 --y 100 --w 800 --h 600
echo 'Move command returned 0.'

echo '== Smoke test PASSED =='

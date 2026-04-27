#!/usr/bin/env bash
#
# Symlink this repo's bin/windowctl into /usr/local/bin/windowctl so
# `windowctl ...` on PATH runs the locally-built dev binary. Each
# `task build` then immediately reflects in the next `windowctl` call —
# no copying, no global npm install.
#
# Pairs with scripts/local-uninstall.sh which only removes the symlink
# when it points at this repo (so it can't accidentally clobber a real
# npm install).

set -euo pipefail
cd "$(dirname "$0")/.."

TARGET=/usr/local/bin/windowctl
SOURCE="$(pwd)/bin/windowctl"

if [ ! -x "$SOURCE" ]; then
  echo "✗ $SOURCE not built — run 'task build' first" >&2
  exit 1
fi

if [ -L "$TARGET" ]; then
  current=$(readlink "$TARGET")
  if [ "$current" = "$SOURCE" ]; then
    echo "✓ already symlinked: $TARGET → $SOURCE"
    exit 0
  fi
  echo "→ $TARGET currently points at: $current"
  echo "  replacing with → $SOURCE"
  rm "$TARGET"
elif [ -e "$TARGET" ]; then
  # Refuse to overwrite a regular file — that's not something we put
  # there. Likely a stale install from a tool we don't recognise.
  echo "✗ $TARGET exists and is NOT a symlink. Refusing to overwrite." >&2
  echo "  Investigate manually, then re-run." >&2
  exit 1
fi

ln -s "$SOURCE" "$TARGET"
echo "✓ symlinked: $TARGET → $SOURCE"
echo "  next 'task build' will be immediately live; remove with 'task local-uninstall'"

#!/usr/bin/env bash
#
# Remove the dev symlink that scripts/local-install.sh created.
# Refuses to remove anything that doesn't point at this repo's
# bin/windowctl — so an existing `npm install -g @muthuishere/windowctl`
# (or any other install) is left untouched.

set -euo pipefail
cd "$(dirname "$0")/.."

TARGET=/usr/local/bin/windowctl
SOURCE="$(pwd)/bin/windowctl"

if [ ! -e "$TARGET" ] && [ ! -L "$TARGET" ]; then
  echo "✓ $TARGET does not exist — nothing to uninstall"
  exit 0
fi

if [ ! -L "$TARGET" ]; then
  echo "✗ $TARGET is not a symlink. Refusing to remove a regular file." >&2
  exit 1
fi

current=$(readlink "$TARGET")
if [ "$current" != "$SOURCE" ]; then
  echo "✗ $TARGET points at: $current"
  echo "  Not this repo's bin/windowctl — leaving it alone." >&2
  echo "  (Looks like an npm install — use 'npm uninstall -g --prefix=/usr/local @muthuishere/windowctl' instead.)" >&2
  exit 1
fi

rm "$TARGET"
echo "✓ removed: $TARGET"

#!/usr/bin/env bash
# publish-npm-local.sh — publish all 6 @muthuishere/windowctl* packages to npm
# from the local machine. Non-interactive: assumes a publish-capable npm
# token is configured (no OTP prompt). For accounts that require 2FA at
# publish time, run `npm login` first or set NPM_TOKEN to an automation
# token that bypasses 2FA.
#
# Usage:
#   bash scripts/publish-npm-local.sh
#
# Prerequisites:
#   1. npm whoami prints your username.
#   2. Versions are bumped: `node scripts/bump-npm-version.js 0.2.0`.
#   3. Binaries are staged for all 5 platforms (run goreleaser first):
#      goreleaser release --snapshot --clean --skip=publish
#      Each npm/platforms/<platform>/bin/windowctl must be a real binary
#      (not the .gitkeep placeholder).

set -e
cd "$(dirname "$0")/.."

WHOAMI=$(npm whoami 2>/dev/null || true)
if [ -z "$WHOAMI" ]; then
  echo "✗ not logged in to npm. Run: npm login" >&2
  exit 1
fi
echo "→ logged in as: $WHOAMI"

for p in darwin-arm64 darwin-x64 linux-arm64 linux-x64 windows-x64; do
  bin="npm/platforms/$p/bin/windowctl"
  [ "$p" = "windows-x64" ] && bin="npm/platforms/$p/bin/windowctl.exe"
  if [ ! -s "$bin" ]; then
    echo "✗ missing binary: $bin" >&2
    echo "  Run: goreleaser release --snapshot --clean --skip=publish" >&2
    exit 1
  fi
done
echo "→ all 5 binaries staged"

# Packages — platform sub-packages FIRST (main's optionalDependencies
# resolve against them), main package LAST.
PACKAGES=(
  "npm/platforms/darwin-arm64"
  "npm/platforms/darwin-x64"
  "npm/platforms/linux-arm64"
  "npm/platforms/linux-x64"
  "npm/platforms/windows-x64"
  "npm"
)

for dir in "${PACKAGES[@]}"; do
  name=$(basename "$dir")
  [ "$dir" = "npm" ] && name="@muthuishere/windowctl (main)"
  echo ""
  echo "=== publishing $name ==="
  (cd "$dir" && npm publish --access public)
done

echo ""
echo "✓ all 6 packages published. Verify:"
echo "    npm view @muthuishere/windowctl"
echo "    npm install -g @muthuishere/windowctl"

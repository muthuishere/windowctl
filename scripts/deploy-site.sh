#!/usr/bin/env bash
#
# Publish site/dist to Cloudflare Pages (winctl.deemwar.com) — ADR 0012.
#
# Independent of the binary release pipeline: this can ship a copy fix without
# a version bump, and `task release` never needs the site to build.
#
# The API token is read into a variable and handed to wrangler through the
# environment. It is never echoed, logged, or written to a file.

set -euo pipefail

REPO=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
PROJECT=winctl-deemwar
DIST="$REPO/site/dist"

[ -f "$DIST/index.html" ] || {
  echo "no build at $DIST — run 'task site:build' first" >&2
  exit 1
}

# Prefer whatever is already exported; otherwise pull from the login shell
# without printing it.
CLOUDFLARE_API_TOKEN=${CLOUDFLARE_API_TOKEN:-$(zsh -ic 'echo $CLOUDFLARE_ALLPURPOSE_TOKEN' 2>/dev/null)}
CLOUDFLARE_ACCOUNT_ID=${CLOUDFLARE_ACCOUNT_ID:-$(zsh -ic 'echo $CLOUDFLARE_ACCOUNT_ID' 2>/dev/null)}
export CLOUDFLARE_API_TOKEN CLOUDFLARE_ACCOUNT_ID

[ -n "$CLOUDFLARE_API_TOKEN" ] || { echo "CLOUDFLARE_ALLPURPOSE_TOKEN not set" >&2; exit 1; }
[ -n "$CLOUDFLARE_ACCOUNT_ID" ] || { echo "CLOUDFLARE_ACCOUNT_ID not set" >&2; exit 1; }

wrangler pages deploy "$DIST" \
  --project-name="$PROJECT" \
  --branch=main \
  --commit-dirty=true

echo
echo "verifying https://winctl.deemwar.com/ …"
for _ in 1 2 3 4 5 6; do
  code=$(curl -sS -o /dev/null -w '%{http_code}' https://winctl.deemwar.com/ || true)
  [ "$code" = "200" ] && { echo "live (HTTP 200)"; exit 0; }
  echo "  HTTP $code — retrying"
  sleep 10
done
echo "site did not return 200 — check the Pages dashboard" >&2
exit 1

#!/usr/bin/env bash
#
# Spike S9 — winctl.deemwar.com deploy path (ADR 0012).
#
# De-risks the site BEFORE any live resource is created. Read-only against
# Cloudflare; the only local side effect is an Astro build.
#
#   1. the all-purpose Cloudflare token is active
#   2. the deemwar.com zone resolves
#   3. winctl.deemwar.com is unclaimed (we are not about to stomp a record)
#   4. wrangler is installed and can talk to the account
#   5. the Astro/Starlight site builds, and emits an index + sitemap
#
# Secrets: the token is read into a shell var and only ever expanded inside a
# curl header — never echoed, never written to a file.
#
# Run: bash spikes/s9_site_deploy.sh

set -u -o pipefail

REPO=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd -P)
SITE="$REPO/site"
DOMAIN="winctl.deemwar.com"
ZONE="deemwar.com"

green() { printf "\033[32m%s\033[0m\n" "$*"; }
red()   { printf "\033[31m%s\033[0m\n" "$*"; }
hdr()   { printf "\n\033[1m-- %s --\033[0m\n" "$*"; }

fails=0; passes=0
pass() { green "  OK   $*"; passes=$((passes+1)); }
fail() { red   "  FAIL $*"; fails=$((fails+1)); }

TOKEN=$(zsh -ic 'echo $CLOUDFLARE_ALLPURPOSE_TOKEN' 2>/dev/null)
ACCOUNT=$(zsh -ic 'echo $CLOUDFLARE_ACCOUNT_ID' 2>/dev/null)
API=https://api.cloudflare.com/client/v4

hdr "1. token"
if [ -z "$TOKEN" ]; then
  fail "CLOUDFLARE_ALLPURPOSE_TOKEN not found in the login shell"
else
  status=$(curl -sS "$API/user/tokens/verify" -H "Authorization: Bearer $TOKEN" |
    python3 -c 'import json,sys; print(json.load(sys.stdin).get("result",{}).get("status","?"))')
  [ "$status" = "active" ] && pass "token active" || fail "token status=$status"
fi

hdr "2. zone"
ZID=$(curl -sS "$API/zones?name=$ZONE" -H "Authorization: Bearer $TOKEN" |
  python3 -c 'import json,sys; r=json.load(sys.stdin).get("result") or []; print(r[0]["id"] if r else "")')
[ -n "$ZID" ] && pass "$ZONE resolved" || fail "$ZONE zone not found"

hdr "3. $DOMAIN is unclaimed"
if [ -n "$ZID" ]; then
  n=$(curl -sS "$API/zones/$ZID/dns_records?name=$DOMAIN" -H "Authorization: Bearer $TOKEN" |
    python3 -c 'import json,sys; print(len(json.load(sys.stdin).get("result") or []))')
  [ "$n" = "0" ] && pass "no existing record — safe to create" || fail "$n record(s) already exist at $DOMAIN"
fi

hdr "4. wrangler + account"
if command -v wrangler >/dev/null 2>&1; then
  pass "wrangler $(wrangler --version 2>/dev/null | tail -1)"
else
  fail "wrangler not on PATH"
fi
if [ -n "$ACCOUNT" ]; then
  ok=$(curl -sS "$API/accounts/$ACCOUNT/pages/projects" -H "Authorization: Bearer $TOKEN" |
    python3 -c 'import json,sys; print(json.load(sys.stdin).get("success"))')
  [ "$ok" = "True" ] && pass "Pages API reachable for the account" || fail "Pages API returned success=$ok"
else
  fail "CLOUDFLARE_ACCOUNT_ID not found"
fi

hdr "5. site builds"
if [ -d "$SITE/node_modules" ]; then
  if (cd "$SITE" && npm run build >/tmp/wctl-site-build.log 2>&1); then
    pass "astro build succeeded"
    [ -f "$SITE/dist/index.html" ] && pass "dist/index.html emitted" || fail "no dist/index.html"
    [ -f "$SITE/dist/sitemap-0.xml" ] && pass "sitemap emitted" || fail "no sitemap"
    grep -qi 'href="/quickstart' "$SITE/dist/index.html" && pass "internal links are root-relative (base '/')" ||
      fail "hero links are not root-relative — check astro.config base"
  else
    fail "astro build failed — see /tmp/wctl-site-build.log"
  fi
else
  fail "site/node_modules missing — run npm install in site/ first"
fi

hdr "summary"
printf "  pass=%d fail=%d\n" "$passes" "$fails"
exit "$fails"

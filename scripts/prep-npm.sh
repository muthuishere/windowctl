#!/usr/bin/env bash
# Fallback / idempotent helper: scans goreleaser's dist/ output and copies each
# built binary into the matching npm/platforms/<os>-<arch>/bin/ directory.
#
# The goreleaser post-build hook (stage-npm-binary.sh) handles this during
# `goreleaser release`, but this script can be run manually if the hook path
# needs bypassing.
set -euo pipefail

repo_root="$(cd "$(dirname "$0")/.." && pwd)"
dist_dir="${1:-$repo_root/dist}"

if [ ! -d "$dist_dir" ]; then
  echo "prep-npm: dist directory not found: $dist_dir" >&2
  exit 1
fi

declare -a MAP=(
  "darwin_amd64_v1:darwin-x64:windowctl"
  "darwin_arm64:darwin-arm64:windowctl"
  "linux_amd64_v1:linux-x64:windowctl"
  "linux_arm64:linux-arm64:windowctl"
  "windows_amd64_v1:windows-x64:windowctl.exe"
)

for entry in "${MAP[@]}"; do
  IFS=':' read -r key npm_platform bin_name <<<"$entry"

  # goreleaser build dirs follow either `windowctl-<id>_<target>` (when the
  # build id is set) or bare `<target>`; check both layouts.
  candidate_dirs=(
    "$dist_dir/windowctl-darwin_${key}"
    "$dist_dir/windowctl-nondarwin_${key}"
    "$dist_dir/${key}"
  )
  src=""
  for d in "${candidate_dirs[@]}"; do
    if [ -f "$d/$bin_name" ]; then
      src="$d/$bin_name"
      break
    fi
  done

  if [ -z "$src" ]; then
    echo "prep-npm: WARNING no binary found for $key (looked in ${candidate_dirs[*]})" >&2
    continue
  fi

  dest_dir="$repo_root/npm/platforms/${npm_platform}/bin"
  mkdir -p "$dest_dir"
  cp "$src" "$dest_dir/$bin_name"
  chmod +x "$dest_dir/$bin_name" || true
  echo "prep-npm: $src -> $dest_dir/$bin_name"
done

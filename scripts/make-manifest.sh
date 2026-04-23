#!/usr/bin/env bash
#
# rocq-platform-starter
# Reproducible and version-pinned Rocq environment bootstrapper.
#
# Copyright (c) 2026 Sylvain Borgogno
# Licensed under the MIT License.
#
# https://github.com/justme0606/rocq-platform-starter
#

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

OWNER="rocq-prover"
REPO="platform"

TAG=""
OUT="$REPO_ROOT/manifest/latest.json"
CHANNEL="stable"
COMPUTE_SHA256=0

usage() {
  cat <<EOF
make-manifest.sh — Generate manifest/latest.json from a Rocq Platform GitHub release

Usage:
  $0 --tag <TAG> [options]

Required:
  --tag <TAG>            GitHub release tag (e.g. 2025.08.1)

Options:
  --out <path>           Output manifest file (default: manifest/latest.json)
  --channel <name>       Channel name (default: stable)
  --compute-sha256       Download assets and compute sha256 hashes
  -h, --help             Show this help message

Examples:

  # Generate manifest for release 2025.08.1
  $0 --tag 2025.08.1

  # Generate manifest and compute sha256
  $0 --tag 2025.08.1 --compute-sha256

  # Custom output file
  $0 --tag 2025.08.1 --out manifest/2025.08.1.json

Environment:
  GITHUB_TOKEN           Optional GitHub token (avoids API rate limits)

Dependencies:
  - curl
  - jq
  - shasum or sha256sum (if --compute-sha256)

EOF
}


need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }; }

# Compute sha256 in a portable-ish way (macOS: shasum, Linux: sha256sum)
sha256_file() {
  local f="$1"
  if command -v shasum >/dev/null 2>&1; then
    shasum -a 256 "$f" | awk '{print $1}'
  elif command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$f" | awk '{print $1}'
  else
    echo ""
  fi
}

# Infer target from filename (best-effort)
infer_target() {
  local name="$1"
  local os="" arch=""

  if [[ "$name" == *.exe ]]; then
    os="windows"
    arch="x86_64"
  elif [[ "$name" == *.dmg ]]; then
    os="macos"
    # Heuristics: prefer explicit intel/x86_64 markers, otherwise default to arm64 (common for signed dmg in recent releases)
    if [[ "$name" =~ (intel|x86_64|amd64) ]]; then
      arch="x86_64"
    else
      arch="arm64"
    fi
  else
    os="unknown"
    arch="unknown"
  fi

  printf "%s %s" "$os" "$arch"
}

# Extract Rocq version from asset name if possible (best-effort)
# e.g. "...version.9.0.2025.08-..." -> 9.0.0
infer_rocq_version() {
  local assets_text="$1"
  if [[ "$assets_text" =~ version\.([0-9]+)\.([0-9]+)\.[0-9]{4}\.[0-9]{2} ]]; then
    echo "${BASH_REMATCH[1]}.${BASH_REMATCH[2]}.0"
  else
    # fallback
    echo ""
  fi
}

# --- args ---
while [[ $# -gt 0 ]]; do
  case "$1" in
    --tag) TAG="${2:-}"; shift 2 ;;
    --out) OUT="${2:-}"; shift 2 ;;
    --channel) CHANNEL="${2:-}"; shift 2 ;;
    --compute-sha256) COMPUTE_SHA256=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "Unknown argument: $1" >&2; usage; exit 1 ;;
  esac
done


[[ -n "$TAG" ]] || { echo "Missing --tag" >&2; usage; exit 1; }

need curl
need jq

mkdir -p "$(dirname "$OUT")"

API_URL="https://api.github.com/repos/${OWNER}/${REPO}/releases/tags/${TAG}"

# Note: For public repos you can call without token; add token via env if you hit rate limits:
#   export GITHUB_TOKEN=...
AUTH_HEADER=()
if [[ -n "${GITHUB_TOKEN:-}" ]]; then
  AUTH_HEADER=(-H "Authorization: Bearer $GITHUB_TOKEN")
fi

echo "Fetching release JSON for tag: $TAG" >&2
release_json="$(curl -fsSL -H "Accept: application/vnd.github+json" "${AUTH_HEADER[@]}" "$API_URL")"

# Keep only signed dmg/exe assets
assets="$(echo "$release_json" | jq -c '[.assets[] | select(.name | startswith("signed_")) | select(.name | endswith(".dmg") or endswith(".exe")) | {name, url: .browser_download_url}]')"

if [[ "$(echo "$assets" | jq 'length')" -eq 0 ]]; then
  echo "No signed .dmg/.exe assets found for tag=$TAG" >&2
  exit 1
fi

rocq_version="$(infer_rocq_version "$(echo "$assets" | jq -r '.[].name' | tr '\n' ' ' )")"
# If not detected, leave empty and let your installer decide default behavior
# (or you can hardcode / pass separately)
platform_release="$TAG"

tmpdir="$(mktemp -d)"
declare -A SHA_BY_URL
if [[ "$COMPUTE_SHA256" -eq 1 ]]; then
  echo "Computing sha256 for assets (downloading)..." >&2
  while IFS=$'\t' read -r name url; do
    f="$tmpdir/$name"
    echo "  - $name" >&2
    curl -fL --retry 3 --retry-delay 1 -o "$f" "$url"
    SHA_BY_URL["$url"]="$(sha256_file "$f")"
  done < <(echo "$assets" | jq -r '.[] | [.name, .url] | @tsv')
fi

# Build manifest skeleton with linux/opam default
manifest="$(jq -n \
  --arg channel "$CHANNEL" \
  --arg platform_release "$platform_release" \
  --arg rocq_version "$rocq_version" \
  '{
    channel: $channel,
    platform_release: $platform_release,
    rocq_version: ($rocq_version // ""),
    assets: {
      macos: {},
      windows: {},
      linux: {
        x86_64: {
          type: "opam",
          opam: {
            ocaml_compiler: "ocaml-base-compiler.4.14.2",
            switch_prefix: "CP",
            repo_name: "rocq-released",
            repo_url: "https://rocq-prover.org/opam/released",
            packages: [
              { name: "rocq-runtime",           version: $rocq_version },
              { name: "rocq-core",              version: $rocq_version },
              { name: "rocq-stdlib",            version: $rocq_version },
              { name: "rocq-prover",            version: $rocq_version },
              { name: "vsrocq-language-server", version: "2.3.4", optional: "skip_vscode" },
              { name: "rocqide",                version: $rocq_version, optional: "with_rocqide" }
            ]
          }
        }
      }
    }
  }'
)"

# Fill in assets
while IFS=$'\t' read -r name url; do
  read -r os arch <<<"$(infer_target "$name")"
  [[ "$os" != "unknown" ]] || continue

  sha=""
  if [[ "$COMPUTE_SHA256" -eq 1 ]]; then
    sha="${SHA_BY_URL[$url]:-}"
  fi

  type="dmg"
  [[ "$name" == *.exe ]] && type="exe"

  manifest="$(echo "$manifest" | jq \
    --arg os "$os" \
    --arg arch "$arch" \
    --arg type "$type" \
    --arg url "$url" \
    --arg sha "$sha" \
    '.assets[$os][$arch] = {type: $type, url: $url, sha256: $sha}'
  )"
done < <(echo "$assets" | jq -r '.[] | [.name, .url] | @tsv')

# Write out
echo "$manifest" | jq '.' > "$OUT"
echo "Wrote manifest: $OUT" >&2

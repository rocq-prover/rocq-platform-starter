#!/usr/bin/env bash
#
# rocq-platform-starter
# Reproducible and version-pinned Rocq environment bootstrapper.
#
# Copyright (c) 2026 Sylvain Borgogno
# Licensed under the MIT License.
#
# https://github.com/rocq-prover/rocq-platform-starter
#

set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd -- "$SCRIPT_DIR/.." && pwd)"

MANIFEST="$REPO_ROOT/manifest/latest.json"
OUT_DIR="$REPO_ROOT/.devcontainer"
OUT="$OUT_DIR/devcontainer.json"

usage() {
  cat <<EOF
generate-devcontainer.sh — Generate .devcontainer/devcontainer.json from the manifest

Usage:
  $0 [variant]

Arguments:
  variant    Docker image variant: ide, extended, or full
             (default: read from manifest docker.default_variant)

Options:
  -h, --help    Show this help message

Examples:

  # Generate devcontainer with default variant (ide)
  $0

  # Generate devcontainer with full variant
  $0 full

Dependencies:
  - jq

EOF
}

need() { command -v "$1" >/dev/null 2>&1 || { echo "Missing dependency: $1" >&2; exit 1; }; }

# --- args ---
VARIANT=""
while [[ $# -gt 0 ]]; do
  case "$1" in
    -h|--help) usage; exit 0 ;;
    -*) echo "Unknown option: $1" >&2; usage; exit 1 ;;
    *) VARIANT="$1"; shift ;;
  esac
done

need jq

if [[ ! -f "$MANIFEST" ]]; then
  echo "Manifest not found: $MANIFEST" >&2
  exit 1
fi

# Validate that the manifest has a docker section
if ! jq -e '.docker' "$MANIFEST" >/dev/null 2>&1; then
  echo "Manifest does not contain a 'docker' section" >&2
  exit 1
fi

# Read default variant from manifest if not specified
if [[ -z "$VARIANT" ]]; then
  VARIANT="$(jq -r '.docker.default_variant' "$MANIFEST")"
fi

# Validate variant exists
if ! jq -e ".docker.variants.\"$VARIANT\"" "$MANIFEST" >/dev/null 2>&1; then
  echo "Unknown variant: $VARIANT" >&2
  echo "Available variants: $(jq -r '.docker.variants | keys | join(", ")' "$MANIFEST")" >&2
  exit 1
fi

# Read values from manifest
registry="$(jq -r '.docker.registry' "$MANIFEST")"
tag="$(jq -r '.docker.tag' "$MANIFEST")"
user="$(jq -r '.docker.user' "$MANIFEST")"
vsrocqtop_path="$(jq -r '.docker.vsrocqtop_path' "$MANIFEST")"
image_name="$(jq -r ".docker.variants.\"$VARIANT\".image" "$MANIFEST")"

full_image="${registry}/${image_name}:${tag}"

mkdir -p "$OUT_DIR"

# Generate devcontainer.json
jq -n \
  --arg name "Rocq Platform" \
  --arg image "$full_image" \
  --arg vsrocqtop "$vsrocqtop_path" \
  --arg user "$user" \
  --arg workspace "/home/${user}/workspace" \
  --arg mount "source=\${localWorkspaceFolder},target=/home/${user}/workspace,type=bind,consistency=cached" \
  '{
    name: $name,
    image: $image,
    customizations: {
      vscode: {
        extensions: [
          "rocq-prover.vsrocq"
        ],
        settings: {
          "vsrocq.path": $vsrocqtop
        }
      }
    },
    remoteUser: $user,
    updateRemoteUserUID: false,
    workspaceFolder: $workspace,
    workspaceMount: $mount,
    postCreateCommand: "rocq --version"
  }' > "$OUT"

echo "Generated: $OUT (variant: $VARIANT, image: $full_image)" >&2

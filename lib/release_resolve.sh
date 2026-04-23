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

DEFAULT_MANIFEST_URL="https://raw.githubusercontent.com/<ORG>/<REPO>/main/manifest/latest.json"

ROCQ_VERSION=""
ASSET_URL=""
ASSET_SHA256=""
ASSET_TYPE=""
PLATFORM_RELEASE=""
OPAM_SNAPSHOT=""
OPAM_COMPILER=""
OPAM_SWITCH_PREFIX=""
OPAM_REPO_NAME=""
OPAM_REPO_URL=""
OPAM_PACKAGES_JSON=""

json_get() {
  local expr="$1" file="$2"
  ensure_jq
  jq -r "$expr" "$file"
}

resolve_release() {
  local manifest_file=""
  local local_manifest

  local_manifest="$SCRIPT_DIR/manifest/latest.json"

  if [[ "${TEST_ONLY:-0}" -eq 1 && ! -f "$local_manifest" ]]; then
    die "TEST_ONLY requires local manifest at $local_manifest (run scripts/make-manifest.sh first)"
  fi

  if [[ -f "$local_manifest" ]]; then
    manifest_file="$local_manifest"
    log "Using local manifest: $manifest_file"
  else
    local url="${MANIFEST_URL:-$DEFAULT_MANIFEST_URL}"
    log "Local manifest not found. Using remote manifest: $url"
    local tmp
    tmp="$(mktemp -d)"
    manifest_file="$tmp/latest.json"
    download "$url" "$manifest_file"
  fi

  # Read manifest metadata (globals for other modules)
  PLATFORM_RELEASE="$(json_get '.platform_release' "$manifest_file")"
  OPAM_SNAPSHOT="$(json_get '.opam_snapshot // ""' "$manifest_file")"

  # Rocq version: from manifest unless overridden by --rocq-version
  ROCQ_VERSION="$(json_get '.rocq_version' "$manifest_file")"
  if [[ "$ROCQ_VERSION_ARG" != "latest" ]]; then
    ROCQ_VERSION="$ROCQ_VERSION_ARG"
  fi

  log "Manifest: platform_release=$PLATFORM_RELEASE rocq_version=$ROCQ_VERSION opam_snapshot=$OPAM_SNAPSHOT"

  # Linux = opam (no binary download)
  if [[ "$OS_NAME" == "linux" ]]; then
    ASSET_TYPE="opam"
    ASSET_URL=""
    ASSET_SHA256=""

    local opam_base=".assets.linux.${ARCH}.opam"
    OPAM_COMPILER="$(json_get "${opam_base}.ocaml_compiler" "$manifest_file")"
    OPAM_SWITCH_PREFIX="$(json_get "${opam_base}.switch_prefix" "$manifest_file")"
    OPAM_REPO_NAME="$(json_get "${opam_base}.repo_name" "$manifest_file")"
    OPAM_REPO_URL="$(json_get "${opam_base}.repo_url" "$manifest_file")"
    OPAM_PACKAGES_JSON="$(json_get "${opam_base}.packages" "$manifest_file")"

    [[ -n "$OPAM_COMPILER" && "$OPAM_COMPILER" != "null" ]] || die "No ocaml_compiler in manifest for linux/${ARCH}"
    [[ -n "$OPAM_REPO_URL" && "$OPAM_REPO_URL" != "null" ]] || die "No repo_url in manifest for linux/${ARCH}"
    [[ -n "$OPAM_PACKAGES_JSON" && "$OPAM_PACKAGES_JSON" != "null" ]] || die "No packages in manifest for linux/${ARCH}"

    log "Opam config: compiler=$OPAM_COMPILER repo=$OPAM_REPO_NAME($OPAM_REPO_URL) switch_prefix=$OPAM_SWITCH_PREFIX"
    return 0
  fi

  # macOS binaries
  local os_key="$OS_NAME" # expected "macos"
  ASSET_TYPE="$(json_get ".assets.${os_key}.${ARCH}.type" "$manifest_file")"
  ASSET_URL="$(json_get ".assets.${os_key}.${ARCH}.url" "$manifest_file")"
  ASSET_SHA256="$(json_get ".assets.${os_key}.${ARCH}.sha256" "$manifest_file")"

  [[ -n "$ASSET_TYPE" && "$ASSET_TYPE" != "null" ]] || die "No asset type for $OS_NAME/$ARCH in manifest"
  [[ -n "$ASSET_URL"  && "$ASSET_URL"  != "null" ]] || die "No asset url for $OS_NAME/$ARCH in manifest"

  [[ "$ASSET_URL" == *"/signed_"* ]] || die "Resolved URL is not a signed installer: $ASSET_URL"

  log "Resolved asset: type=$ASSET_TYPE url=$ASSET_URL"
}

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

ensure_opam_deps() {
  # command -> package name mapping (command:package)
  local deps="unzip:unzip bwrap:bubblewrap make:make cc:gcc bzip2:bzip2"
  local missing_pkgs=()

  for entry in $deps; do
    local cmd="${entry%%:*}"
    local pkg="${entry##*:}"
    command -v "$cmd" >/dev/null 2>&1 || missing_pkgs+=("$pkg")
  done

  [[ ${#missing_pkgs[@]} -eq 0 ]] && return 0

  log "Installing opam dependencies: ${missing_pkgs[*]}"

  local SUDO=""
  if [[ "$(id -u)" -ne 0 ]]; then
    command -v sudo >/dev/null 2>&1 || die "Cannot install opam dependencies (${missing_pkgs[*]}): not root and sudo not available. Please install them manually."
    SUDO="sudo"
  fi

  if command -v apt-get >/dev/null 2>&1; then
    $SUDO apt-get update -qq >&2 && $SUDO apt-get install -y -qq "${missing_pkgs[@]}" >&2
  elif command -v dnf >/dev/null 2>&1; then
    $SUDO dnf install -y "${missing_pkgs[@]}" >&2
  elif command -v yum >/dev/null 2>&1; then
    $SUDO yum install -y "${missing_pkgs[@]}" >&2
  elif command -v pacman >/dev/null 2>&1; then
    $SUDO pacman -S --noconfirm "${missing_pkgs[@]}" >&2
  elif command -v zypper >/dev/null 2>&1; then
    $SUDO zypper install -y "${missing_pkgs[@]}" >&2
  else
    die "Cannot install opam dependencies (${missing_pkgs[*]}): no supported package manager found. Please install them manually."
  fi
}

ensure_opam() {
  if command -v opam >/dev/null 2>&1; then
    return 0
  fi

  log "opam not found — installing via official installer..."
  need_cmd curl

  ensure_opam_deps

  local tmp
  tmp="$(mktemp)"
  curl -fL --retry 3 --retry-delay 1 -o "$tmp" https://opam.ocaml.org/install.sh
  chmod +x "$tmp"
  echo /usr/local/bin | sh "$tmp" --no-backup >&2
  rm -f "$tmp"

  # The installer places opam in /usr/local/bin or ~/.opam/bin; verify it worked
  if ! command -v opam >/dev/null 2>&1; then
    # Try common install location
    export PATH="/usr/local/bin:$HOME/.opam/bin:$PATH"
    command -v opam >/dev/null 2>&1 || die "opam installation failed — please install opam manually: https://opam.ocaml.org/doc/Install.html"
  fi

  log "opam installed successfully: $(command -v opam)"
}

install_rocq_linux_opam() {
  ensure_opam

  # Auto-confirm system dependency installation (replaces interactive prompts)
  export OPAMCONFIRMLEVEL=unsafe-yes

  local opam_ver
  opam_ver="$(opam --version | tr -d '\r')"
  log "opam version: $opam_ver"
  [[ "$opam_ver" == 2.* ]] || die "opam >= 2.1.0 required (found $opam_ver)"

  if [[ ! -d "$HOME/.opam" ]]; then
    log "Initializing opam..."
    opam init -y --bare --disable-sandboxing
  fi

  # Switch name style Rocq Platform
  local rocq_mm="${ROCQ_VERSION%.*}"  # 9.0 from 9.0.0
  local prefix="${OPAM_SWITCH_PREFIX:-CP}"
  local switch="${prefix}.${PLATFORM_RELEASE}~${rocq_mm}"
  if [[ -n "${OPAM_SNAPSHOT:-}" ]]; then
    switch="${switch}~${OPAM_SNAPSHOT}"
  fi
  
   OPAM_SWITCH_NAME="$switch"

  # Recreate switch if requested
  if opam switch list --short | grep -qx "$switch"; then
    if [[ "${RECREATE_SWITCH:-0}" -eq 1 ]]; then
      log "Recreating opam switch (requested): $switch"
      opam switch remove "$switch" -y
    else
      log "Opam switch already exists: $switch"
      log "Tip: re-run with --recreate-switch to start from a clean switch"
    fi
  fi

  # Create switch if missing
  if ! opam switch list --short | grep -qx "$switch"; then
    log "Creating opam switch: $switch ($OPAM_COMPILER)"
    opam switch create "$switch" "$OPAM_COMPILER" -y
  fi

  local repo_name="${OPAM_REPO_NAME:-rocq-released}"
  local repo_url="${OPAM_REPO_URL:-https://rocq-prover.org/opam/released}"

  # Ensure opam repo is present and correctly configured IN THIS SWITCH
  log "Ensuring opam repo $repo_name -> $repo_url (switch=$switch)"
  if ! opam repo add --switch="$switch" "$repo_name" "$repo_url" -y; then
    log "$repo_name already exists; forcing set-url to $repo_url"
    opam repo set-url --switch="$switch" "$repo_name" "$repo_url" -y
  fi

  # Force repo selection for THIS switch (avoid surprises).
  # Note: the "archive" repo may not exist on some opam installs (e.g. opam 2.1 on ubuntu-latest).
  local repos=("$repo_name" "default")

  if opam repo list --switch="$switch" --short | grep -qx "archive"; then
    repos+=("archive")
  fi

  log "Setting opam repositories for switch $switch: ${repos[*]}"
  opam repo set-repos --switch="$switch" "${repos[@]}"

  # Make repo top priority (opam 2.5 syntax)
  opam repo priority --switch="$switch" "$repo_name" 1
  opam update --switch="$switch"

  # Build package list from manifest
  local required_pkgs=()
  local count
  count="$(echo "$OPAM_PACKAGES_JSON" | jq 'length')"

  for (( i=0; i<count; i++ )); do
    local pkg_name pkg_version pkg_optional
    pkg_name="$(echo "$OPAM_PACKAGES_JSON" | jq -r ".[$i].name")"
    pkg_version="$(echo "$OPAM_PACKAGES_JSON" | jq -r ".[$i].version")"
    pkg_optional="$(echo "$OPAM_PACKAGES_JSON" | jq -r ".[$i].optional // empty")"

    # Handle optional packages based on flags
    if [[ "$pkg_optional" == "skip_vscode" ]]; then
      if [[ "${SKIP_VSCODE:-0}" -eq 1 ]]; then
        log "Skipping $pkg_name (SKIP_VSCODE=1)"
        continue
      fi
    elif [[ "$pkg_optional" == "with_rocqide" ]]; then
      if [[ "${WITH_ROCQIDE:-no}" != "yes" ]]; then
        log "Skipping $pkg_name (WITH_ROCQIDE=${WITH_ROCQIDE:-no})"
        continue
      fi
    fi

    required_pkgs+=("${pkg_name}=${pkg_version}")
  done

  log "Installing Rocq packages in switch $switch: ${required_pkgs[*]}"
  opam install --switch="$switch" -y "${required_pkgs[@]}"

  local bin
  bin="$(opam var --switch="$switch" bin)"

  VSROCQTOP_PATH="$bin/vsrocqtop"
  ROCQ_PATH="$bin/rocq"

  # vsrocqtop is only required if we configure VSCode
  if [[ ! -x "$VSROCQTOP_PATH" ]]; then
    if [[ "${SKIP_VSCODE:-0}" -eq 1 ]]; then
      log "SKIP_VSCODE=1: vsrocqtop not found (ok)"
      VSROCQTOP_PATH=""
    else
      die "vsrocqtop not found in switch bin: $VSROCQTOP_PATH (install vsrocq-language-server)"
    fi
  fi

  [[ -x "$ROCQ_PATH" ]] || die "rocq not found in switch bin: $ROCQ_PATH"

  log "vsrocqtop: ${VSROCQTOP_PATH:-<none>}"
  log "rocq: $ROCQ_PATH"

  # Verify version matches requested (major.minor)
  local got want_mm
  got="$("$ROCQ_PATH" --print-version 2>/dev/null || true)"
  if [[ -z "$got" ]]; then
    got="$("$ROCQ_PATH" --version 2>&1 | head -n 1 || true)"
  fi
  log "rocq version after install: $got"

  want_mm="${ROCQ_VERSION%.*}" # 9.0
  echo "$got" | grep -q "$want_mm" || die "Installed rocq does not match requested Rocq $ROCQ_VERSION (got: $got)"

}
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

LOG_DIR="${LOG_DIR:-$HOME/.rocq-setup/logs}"
LOG_FILE=""
VERBOSE=0
NON_INTERACTIVE=1
SKIP_VSCODE=0
WORKSPACE_DIR="${WORKSPACE_DIR:-$HOME/rocq-workspace}"
ROCQ_VERSION_ARG="latest"
WITH_ROCQIDE="no"
FORCE=0
TEST_ONLY=0
RECREATE_SWITCH=0
DOCTOR=0

log() { echo "[$(date +'%F %T')] $*" | tee -a "$LOG_FILE" >&2; }
die() { log "ERROR: $*"; exit 1; }

init_logging() {
  mkdir -p "$LOG_DIR"
  LOG_FILE="$LOG_DIR/rocq-setup-$(date +'%Y%m%d-%H%M%S').log"
  : > "$LOG_FILE"
  log "Log file: $LOG_FILE"
}

parse_args() {
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --rocq-version) ROCQ_VERSION_ARG="${2:-}"; shift 2 ;;
      --workspace) WORKSPACE_DIR="${2:-}"; shift 2 ;;
      --with-rocqide) WITH_ROCQIDE="${2:-}"; shift 2 ;;
      --skip-vscode) SKIP_VSCODE=1; shift ;;
      --interactive) NON_INTERACTIVE=0; shift ;;
      --verbose) VERBOSE=1; shift ;;
      --force) FORCE=1; shift ;;
      --doctor) DOCTOR=1; shift ;;
      --test-only) TEST_ONLY=1; shift ;;
      --recreate-switch) RECREATE_SWITCH=1; shift ;;
      -h|--help)
        cat <<EOF
Usage: ./install.sh [options]
--rocq-version <x.y.z|latest>  (default: latest)
--workspace <path>             (default: ~/rocq-workspace)
--with-rocqide <yes|no>        (default: no)
--skip-vscode                  (default: false)
--interactive                  (default: non-interactive)
--doctor                      Run diagnostics only (no installation)
--verbose
--force
--test-only                   Run checks/tests only (no installation, no downloads)
--recreate-switch             Remove and recreate the opam switch if it already exists (Linux/opam)

EOF
        exit 0
        ;;
      *) die "Unknown argument: $1" ;;
    esac
  done
}

need_cmd() { command -v "$1" >/dev/null 2>&1 || die "Missing required command: $1"; }

ensure_jq() {
  if command -v jq >/dev/null 2>&1; then
    return 0
  fi

  log "jq not found — attempting automatic installation..."

  local SUDO=""
  if [[ "$(id -u)" -ne 0 ]]; then
    command -v sudo >/dev/null 2>&1 || die "jq is required but cannot install: not root and sudo not available. Please install jq manually: https://jqlang.github.io/jq/download/"
    SUDO="sudo"
  fi

  if command -v apt-get >/dev/null 2>&1; then
    $SUDO apt-get update -qq && $SUDO apt-get install -y -qq jq >&2
  elif command -v dnf >/dev/null 2>&1; then
    $SUDO dnf install -y jq >&2
  elif command -v yum >/dev/null 2>&1; then
    $SUDO yum install -y jq >&2
  elif command -v pacman >/dev/null 2>&1; then
    $SUDO pacman -S --noconfirm jq >&2
  elif command -v zypper >/dev/null 2>&1; then
    $SUDO zypper install -y jq >&2
  elif command -v brew >/dev/null 2>&1; then
    brew install jq >&2
  else
    die "jq is required but could not be installed automatically. Please install it manually: https://jqlang.github.io/jq/download/"
  fi

  command -v jq >/dev/null 2>&1 || die "jq installation failed — please install it manually: https://jqlang.github.io/jq/download/"
  log "jq installed successfully: $(command -v jq)"
}

download() {
  local url="$1" out="$2"
  need_cmd curl
  log "Downloading: $url"
  curl -fL --retry 3 --retry-delay 1 -o "$out" "$url"
}

sha256_check() {
  local file="$1" expected="$2"
  [[ -z "$expected" ]] && return 0
  need_cmd shasum
  local got
  got="$(shasum -a 256 "$file" | awk '{print $1}')"
  [[ "$got" == "$expected" ]] || die "SHA256 mismatch for $file (got=$got expected=$expected)"
}

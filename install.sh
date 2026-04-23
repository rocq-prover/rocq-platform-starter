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

source "$SCRIPT_DIR/lib/common.sh"
source "$SCRIPT_DIR/lib/os_detect.sh"
source "$SCRIPT_DIR/lib/release_resolve.sh"
source "$SCRIPT_DIR/lib/workspace.sh"
source "$SCRIPT_DIR/lib/vscode.sh"
source "$SCRIPT_DIR/lib/test_cli.sh"
source "$SCRIPT_DIR/lib/find_vsrocqtop.sh"
source "$SCRIPT_DIR/lib/install_macos.sh"
source "$SCRIPT_DIR/lib/install_linux_opam.sh"
source "$SCRIPT_DIR/lib/doctor.sh"

main() {
  parse_args "$@"
  init_logging

  if [[ "${DOCTOR:-0}" -eq 1 ]]; then
    log "DOCTOR=1: running diagnostics (no installation)"
    run_doctor
    log "✅ Doctor completed"
    exit 0
  fi

  detect_os_arch
  log "Detected: OS=$OS_NAME ARCH=$ARCH"

  resolve_release
  log "Target Rocq version: $ROCQ_VERSION"
  log "Resolved asset: type=$ASSET_TYPE url=$ASSET_URL"

  if [[ "$TEST_ONLY" -eq 1 ]]; then
    log "TEST_ONLY=1: skipping Rocq Platform installation"
    if [[ "$OS_NAME" == "macos" ]]; then
      find_vsrocqtop_macos || die "vsrocqtop not found (install Rocq Platform first or run without --test-only)"
    elif [[ "$OS_NAME" == "linux" ]]; then
      if [[ "${SKIP_VSCODE:-0}" -eq 1 ]]; then
        log "SKIP_VSCODE=1: not requiring vsrocqtop in --test-only"
      else
        VSROCQTOP_PATH="$(command -v vsrocqtop 2>/dev/null || true)"
        [[ -n "${VSROCQTOP_PATH:-}" ]] || die "vsrocqtop not found in PATH (install first or run without --test-only)"
      fi
    else
      die "Unsupported OS for test-only: $OS_NAME"
    fi
    log "vsrocqtop: ${VSROCQTOP_PATH:-<none>}"
  else
    if [[ "$OS_NAME" == "macos" ]]; then
      install_rocq_macos
    elif [[ "$OS_NAME" == "linux" ]]; then
      install_rocq_linux_opam
    else
      die "Unsupported OS for install.sh: $OS_NAME"
    fi
  fi

  # VSROCQTOP_PATH is only required if we configure VSCode
  if [[ "${SKIP_VSCODE:-0}" -eq 0 ]]; then
    [[ -n "${VSROCQTOP_PATH:-}" ]] || die "VSROCQTOP_PATH is empty after installation"
  fi

  ensure_workspace

  if [[ "${SKIP_VSCODE:-0}" -eq 0 ]]; then
    ensure_vscode_if_needed
    ensure_vsrocq_extension
    configure_vsrocq_settings
  else
    log "SKIP_VSCODE=1: skipping VSCode setup"
  fi

  run_cli_test

  summary_success

  if [[ "$OS_NAME" == "linux" && -n "${OPAM_SWITCH_NAME:-}" ]]; then
    log "To enable rocq in your current terminal, run:"
    log "  source \"$WORKSPACE_DIR/activate.sh\""
    log ""
    log "Or open a new activated shell:"
    log "  $WORKSPACE_DIR/activate-shell.sh"
  fi
}

main "$@"

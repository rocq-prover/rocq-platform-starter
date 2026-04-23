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

run_cli_test() {
  log "Running CLI test..."

  local testfile="$WORKSPACE_DIR/test.v"
  [[ -f "$testfile" ]] || die "Missing test file: $testfile"

  local cmd=""
  if [[ -n "${ROCQ_PATH:-}" && -x "$ROCQ_PATH" ]]; then
    cmd="$ROCQ_PATH"
  elif command -v rocq >/dev/null 2>&1; then
    cmd="$(command -v rocq)"
  else
    die "No rocq binary found to run test"
  fi

  log "Using rocq: $cmd"
  (cd "$WORKSPACE_DIR" && "$cmd" compile "$(basename "$testfile")") || die "CLI test failed"

  # sanity: ensure artifact exists (optional but nice)
  if [[ ! -f "$WORKSPACE_DIR/test.vo" ]]; then
    die "CLI test failed: expected test.vo not found"
  fi

  log "CLI test OK (test.vo generated)"
}

summary_success() {
  log "✅ Installation complete"
  log "Workspace: $WORKSPACE_DIR"
  if [[ "${SKIP_VSCODE:-0}" -eq 0 ]]; then
    log "Open VSCode: $CODE_BIN \"$WORKSPACE_DIR\""
    "$CODE_BIN" --disable-workspace-trust "$WORKSPACE_DIR" >/dev/null 2>&1 || true
  fi
}

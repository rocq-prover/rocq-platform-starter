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

CODE_BIN=""
VSROCQ_EXTENSION_ID="${VSROCQ_EXTENSION_ID:-rocq-prover.vsrocq}"


ensure_vscode_if_needed() {
  if [[ "$SKIP_VSCODE" -eq 1 ]]; then
    log "Skipping VSCode checks (SKIP_VSCODE=1)"
    return 0
  fi

  # Trouver code
  CODE_BIN="$(command -v code || true)"

  if [[ -z "$CODE_BIN" && "$OS_NAME" == "macos" ]]; then
    local fallback="/Applications/Visual Studio Code.app/Contents/Resources/app/bin/code"
    [[ -x "$fallback" ]] && CODE_BIN="$fallback"
  fi

  if [[ -z "$CODE_BIN" ]]; then
    log "WARNING: VSCode non détecté — la configuration de l'éditeur sera ignorée."
    log "Pour utiliser Rocq avec VSCode, installez-le depuis https://code.visualstudio.com puis activez la commande 'code' dans le PATH."
    SKIP_VSCODE=1
    return 0
  fi

  log "VSCode CLI: $CODE_BIN"
}

VSROCQ_EXTENSION_ID="${VSROCQ_EXTENSION_ID:-rocq-prover.vsrocq}"

ensure_vsrocq_extension() {
  [[ "$SKIP_VSCODE" -eq 1 ]] && return 0

  if "$CODE_BIN" --list-extensions | grep -qi "^${VSROCQ_EXTENSION_ID//./\\.}$"; then
    log "vsrocq extension already installed: $VSROCQ_EXTENSION_ID"
    return 0
  fi

  log "Installing VSCode extension: $VSROCQ_EXTENSION_ID"
  "$CODE_BIN" --install-extension "$VSROCQ_EXTENSION_ID" || die "Failed Installing Extensions: $VSROCQ_EXTENSION_ID"
}


configure_vsrocq_settings() {
  [[ "$SKIP_VSCODE" -eq 1 ]] && return 0

  local out="$WORKSPACE_DIR/.vscode/settings.json"

  [[ -n "${VSROCQTOP_PATH:-}" ]] || die "VSROCQTOP_PATH is empty (cannot configure VSCode)"

  local settings_key="vsrocq.path"
  if [[ "${VSROCQTOP_PATH}" == *vscoqtop* ]]; then
    settings_key="vscoq.path"
  fi

  printf '{\n  "%s": "%s"\n}\n' "$settings_key" "$VSROCQTOP_PATH" > "$out"
  log "Wrote VSCode workspace settings: $out"
}

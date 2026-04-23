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

# doctor.sh: post-install diagnostics (no installation)

ok()   { log "✔ $*"; }
warn() { log "⚠ $*"; }
fail() { die "✘ $*"; }

# small helper: print package version in a given opam switch
opam_pkg_version() {
  local sw="$1" pkg="$2"
  opam show --switch="$sw" -f version "$pkg" 2>/dev/null | head -n 1 | tr -d '\r'
}

doctor_environment() {
  ok "Doctor mode: starting checks"
  ok "Detected OS=$OS_NAME ARCH=$ARCH"
  ok "Target Rocq version (from manifest/args): $ROCQ_VERSION"
  ok "Platform release: ${PLATFORM_RELEASE:-<unknown>}"
}

doctor_workspace() {
  [[ -n "${WORKSPACE_DIR:-}" ]] || fail "WORKSPACE_DIR is empty"
  if [[ ! -d "$WORKSPACE_DIR" ]]; then
    fail "Workspace directory does not exist: $WORKSPACE_DIR"
  fi
  ok "Workspace directory: $WORKSPACE_DIR"

  # basic templates
  [[ -f "$WORKSPACE_DIR/test.v" ]] || warn "Missing $WORKSPACE_DIR/test.v (CLI test may fail)"
  [[ -f "$WORKSPACE_DIR/_RocqProject" ]] || warn "Missing $WORKSPACE_DIR/_RocqProject"
}

doctor_vscode() {
  if [[ "${SKIP_VSCODE:-0}" -eq 1 ]]; then
    ok "VSCode checks skipped (SKIP_VSCODE=1)"
    return 0
  fi

  if [[ -z "${CODE_BIN:-}" ]]; then
    warn "VSCode CLI not detected (CODE_BIN empty). Trying to detect..."
    if command -v code >/dev/null 2>&1; then
      CODE_BIN="$(command -v code)"
    fi
  fi

  if [[ -z "${CODE_BIN:-}" || ! -x "${CODE_BIN:-/nonexistent}" ]]; then
    warn "VSCode CLI not available; skipping VSCode checks"
    return 0
  fi

  ok "VSCode CLI: $CODE_BIN"

  # extension check (you already use rocq-prover.vsrocq)
  if "$CODE_BIN" --list-extensions 2>/dev/null | grep -qx "rocq-prover.vsrocq"; then
    ok "VSCode extension installed: rocq-prover.vsrocq"
  else
    warn "VSCode extension NOT installed: rocq-prover.vsrocq"
  fi

  # workspace settings check
  local settings="$WORKSPACE_DIR/.vscode/settings.json"
  if [[ -f "$settings" ]]; then
    ok "Workspace settings found: $settings"
    if [[ -n "${VSROCQTOP_PATH:-}" ]]; then
      if grep -Fq "$VSROCQTOP_PATH" "$settings"; then
        ok "settings.json references vsrocqtop path"
      else
        warn "settings.json does not reference VSROCQTOP_PATH=$VSROCQTOP_PATH"
      fi
    else
      warn "VSROCQTOP_PATH is empty; cannot validate settings vsrocqtop path"
    fi
  else
    warn "Workspace settings missing: $settings"
  fi
}

doctor_linux_opam() {
  need_cmd opam
  local opam_ver
  opam_ver="$(opam --version | tr -d '\r')"
  ok "opam version: $opam_ver"

  local rocq_mm="${ROCQ_VERSION%.*}"
  local prefix="${OPAM_SWITCH_PREFIX:-CP}"
  local sw="${prefix}.${PLATFORM_RELEASE}~${rocq_mm}"
  if [[ -n "${OPAM_SNAPSHOT:-}" ]]; then
    sw="${sw}~${OPAM_SNAPSHOT}"
  fi

  if ! opam switch list --short 2>/dev/null | grep -qx "$sw"; then
    fail "opam switch not found: $sw (run install.sh first)"
  fi
  ok "opam switch exists: $sw"

  # repos: ensure rocq-released selected
  local repos
  repos="$(opam repo list --switch="$sw" 2>/dev/null | awk 'NR>2 {print $2}' | tr -d '\r' || true)"
  if opam repo list --switch="$sw" --short 2>/dev/null | grep -qx "rocq-released"; then
    ok "opam repo selected in switch: rocq-released"
  else
    warn "opam repo NOT selected in switch: rocq-released"
  fi

  # package versions must match ROCQ_VERSION
  local pkgs=(rocq-runtime rocq-core rocq-stdlib rocq-prover)
  if [[ "${WITH_ROCQIDE:-no}" == "yes" ]]; then
    pkgs+=(rocqide)
  fi

  local bad=0
  for p in "${pkgs[@]}"; do
    local v
    v="$(opam_pkg_version "$sw" "$p" || true)"
    if [[ -z "$v" ]]; then
      warn "Package not installed in switch: $p"
      bad=1
      continue
    fi
    if [[ "$v" == "$ROCQ_VERSION" ]]; then
      ok "$p version: $v"
    else
      warn "$p version mismatch: got $v, expected $ROCQ_VERSION"
      bad=1
    fi
  done

  # vsrocqtop presence only if VSCode not skipped
  local bin
  bin="$(opam var --switch="$sw" bin 2>/dev/null | tr -d '\r')"
  ROCQ_PATH="$bin/rocq"
  VSROCQTOP_PATH="$bin/vsrocqtop"

  [[ -x "$ROCQ_PATH" ]] || fail "rocq not found in switch bin: $ROCQ_PATH"
  ok "rocq path: $ROCQ_PATH"

  if [[ "${SKIP_VSCODE:-0}" -eq 0 ]]; then
    [[ -x "$VSROCQTOP_PATH" ]] || warn "vsrocqtop not found in switch bin: $VSROCQTOP_PATH"
    [[ -x "$VSROCQTOP_PATH" ]] && ok "vsrocqtop path: $VSROCQTOP_PATH"
  else
    ok "vsrocqtop not required (SKIP_VSCODE=1)"
  fi

  # version check via rocq
  local got
  got="$("$ROCQ_PATH" --print-version 2>/dev/null || true)"
  [[ -n "$got" ]] || got="$("$ROCQ_PATH" --version 2>&1 | head -n 1 || true)"
  ok "rocq reports: $got"
  echo "$got" | grep -q "${ROCQ_VERSION%.*}" || warn "rocq version string does not contain ${ROCQ_VERSION%.*} (string: $got)"

  # final CLI compilation test
  run_cli_test
  ok "Doctor checks completed"

    log "Hint: to use rocq in your current shell:"
    log "  eval \"\$(opam env --switch=$sw)\""
    log "Or inside the workspace:"
    log "  $WORKSPACE_DIR/activate.sh"
}

doctor_macos_app() {
  # We rely on install_macos.sh / find_vsrocqtop.sh to set ROCQ_PATH/VSROCQTOP_PATH when installed.
  [[ -n "${ROCQ_PATH:-}" && -x "${ROCQ_PATH:-/nonexistent}" ]] || warn "rocq binary not detected (ROCQ_PATH empty or non-executable)"
  [[ -n "${VSROCQTOP_PATH:-}" && -x "${VSROCQTOP_PATH:-/nonexistent}" ]] || warn "vsrocqtop not detected (VSROCQTOP_PATH empty or non-executable)"

  if [[ -n "${ROCQ_PATH:-}" && -x "$ROCQ_PATH" ]]; then
    ok "rocq path: $ROCQ_PATH"
    local got
    got="$("$ROCQ_PATH" --print-version 2>/dev/null || true)"
    [[ -n "$got" ]] || got="$("$ROCQ_PATH" --version 2>&1 | head -n 1 || true)"
    ok "rocq reports: $got"
  fi

  if [[ -n "${VSROCQTOP_PATH:-}" && -x "$VSROCQTOP_PATH" ]]; then
    ok "vsrocqtop path: $VSROCQTOP_PATH"
  fi

  run_cli_test
  ok "Doctor checks completed"
}

run_doctor() {
  detect_os_arch
  resolve_release

  # Workspace checks first (works for all OS)
  ensure_workspace

  doctor_environment
  doctor_workspace

  # VSCode checks need VSROCQTOP_PATH; on Linux we can compute from switch, on macOS from app install
  if [[ "$OS_NAME" == "linux" ]]; then
    doctor_linux_opam
    doctor_vscode
  elif [[ "$OS_NAME" == "macos" ]]; then
    # On macOS, we can still run checks if Rocq is already installed and find_vsrocqtop.sh can locate it.
    # If ROCQ_PATH/VSROCQTOP_PATH are not set, doctor will warn (not fail) and still attempt CLI test.
    doctor_macos_app
    doctor_vscode
  else
    fail "Unsupported OS for doctor: $OS_NAME"
  fi
}

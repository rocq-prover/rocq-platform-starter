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

install_rocq_macos() {
  [[ "$ASSET_URL" != "__TODO_DMG_URL__" ]] || die "macOS DMG URL not configured yet"

  need_cmd hdiutil
  need_cmd rsync

  local tmp dmg mount_point app_src app_name app_dst
  tmp="$(mktemp -d)"
  dmg="$tmp/rocq-platform.dmg"

  download "$ASSET_URL" "$dmg"
  sha256_check "$dmg" "$ASSET_SHA256"

  log "Mounting DMG..."
  mount_point="$(hdiutil attach "$dmg" -nobrowse | awk '/Volumes\// {print $3; exit}')"
  [[ -n "$mount_point" ]] || die "Failed to mount DMG"

  # Trouver la .app (on prend la première)
  app_src="$(find "$mount_point" -maxdepth 2 -name "*.app" -print -quit)"
  [[ -n "$app_src" ]] || die "No .app found in DMG"

  app_name="$(basename "$app_src")"

  # Destination: /Applications si possible, sinon ~/Applications
  if [[ -w "/Applications" ]]; then
    app_dst="/Applications/$app_name"
  else
    mkdir -p "$HOME/Applications"
    app_dst="$HOME/Applications/$app_name"
  fi

  log "Installing app: $app_name -> $app_dst"
  # Idempotent: on remplace si --force, sinon on skip si déjà là
  if [[ -e "$app_dst" && "$FORCE" -eq 0 ]]; then
    log "App already installed (use --force to replace): $app_dst"
  else
    rm -rf "$app_dst" || true
    rsync -a --delete "$app_src/" "$app_dst/"
  fi

  log "Detaching DMG..."
  hdiutil detach "$mount_point" >/dev/null 2>&1 || true

  # Résoudre vsrocqtop
  find_vsrocqtop_macos
  [[ -n "${VSROCQTOP_PATH:-}" ]] || die "vsrocqtop not found after macOS install"
  log "vsrocqtop: $VSROCQTOP_PATH"
}

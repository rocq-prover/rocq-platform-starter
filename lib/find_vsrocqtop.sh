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

VSROCQTOP_PATH=""

find_vsrocqtop_macos() {
  # 1) PATH
  VSROCQTOP_PATH="$(command -v vsrocqtop 2>/dev/null || true)"
  [[ -n "$VSROCQTOP_PATH" ]] && return 0

  # 2) chemins connus
  for p in /usr/local/bin/vsrocqtop /opt/homebrew/bin/vsrocqtop; do
    if [[ -x "$p" ]]; then
      VSROCQTOP_PATH="$p"
      return 0
    fi
  done

  # 3) Scan limité dans les .app
  local candidates=(
    "/Applications"
    "$HOME/Applications"
  )

  for base in "${candidates[@]}"; do
    [[ -d "$base" ]] || continue
    # on scanne seulement dans les bundles qui match "Rocq" / "Coq" pour rester rapide
    local app
    while IFS= read -r -d '' app; do
      local found
      found="$(find "$app/Contents" -maxdepth 6 -type f -name "vsrocqtop" -perm -111 -print -quit 2>/dev/null || true)"
      if [[ -n "$found" ]]; then
        VSROCQTOP_PATH="$found"
        return 0
      fi
    done < <(find "$base" -maxdepth 1 -type d \( -iname "*rocq*.app" -o -iname "*coq*.app" \) -print0 2>/dev/null)
  done

  return 1
}

find_vsrocqtop_linux_opam() {
  # après eval opam env
  local bin
  bin="$(opam var bin 2>/dev/null || true)"
  if [[ -n "$bin" && -x "$bin/vsrocqtop" ]]; then
    VSROCQTOP_PATH="$bin/vsrocqtop"
    return 0
  fi
  VSROCQTOP_PATH="$(command -v vsrocqtop 2>/dev/null || true)"
  [[ -n "$VSROCQTOP_PATH" ]]
}

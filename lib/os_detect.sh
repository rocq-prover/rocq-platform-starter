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

OS_NAME=""
ARCH=""

detect_os_arch() {
  local uname_s uname_m
  uname_s="$(uname -s)"
  uname_m="$(uname -m)"

  case "$uname_s" in
    Darwin) OS_NAME="macos" ;;
    Linux)  OS_NAME="linux" ;;
    *)      OS_NAME="unknown" ;;
  esac

  case "$uname_m" in
    x86_64|amd64) ARCH="x86_64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) ARCH="$uname_m" ;;
  esac
}
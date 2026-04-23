#
# rocq-platform-starter
# Reproducible and version-pinned Rocq environment bootstrapper.
#
# Copyright (c) 2026 Sylvain Borgogno
# Licensed under the MIT License.
#
# https://github.com/justme0606/rocq-platform-starter
#

# activate.sh
# Usage:
#   source ./activate.sh
# or:
#   . ./activate.sh

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
  echo "This script must be sourced:"
  echo "  source ./activate.sh"
  exit 1
fi

set -euo pipefail

if ! command -v opam >/dev/null 2>&1; then
  echo "opam not found in PATH"
  return 1
fi

eval "$(opam env --switch=${OPAM_SWITCH_NAME})"

echo "Activated opam switch: ${OPAM_SWITCH_NAME}"
echo "rocq version:"
rocq --print-version 2>/dev/null || rocq --version

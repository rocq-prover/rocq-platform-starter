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

ensure_workspace() {
  mkdir -p -- "$WORKSPACE_DIR/.vscode"

  # main.v
  if [[ -f "$SCRIPT_DIR/templates/main.v" ]]; then
    cp -n -- "$SCRIPT_DIR/templates/main.v" "$WORKSPACE_DIR/main.v" || true
  else
    [[ -f "$WORKSPACE_DIR/main.v" ]] || cat > "$WORKSPACE_DIR/main.v" <<'EOF'
Theorem hello : 0 = 0.
Proof. reflexivity. Qed.
EOF
  fi

  # test.v
  if [[ -f "$SCRIPT_DIR/templates/test.v" ]]; then
    cp -n -- "$SCRIPT_DIR/templates/test.v" "$WORKSPACE_DIR/test.v" || true
  else
    [[ -f "$WORKSPACE_DIR/test.v" ]] || cat > "$WORKSPACE_DIR/test.v" <<'EOF'
Theorem t : 0 = 0.
Proof. reflexivity. Qed.
EOF
  fi

  # _RocqProject (preferred)
  if [[ -f "$SCRIPT_DIR/templates/_RocqProject" ]]; then
    cp -n -- "$SCRIPT_DIR/templates/_RocqProject" "$WORKSPACE_DIR/_RocqProject" || true
  else
    [[ -f "$WORKSPACE_DIR/_RocqProject" ]] || cat > "$WORKSPACE_DIR/_RocqProject" <<'EOF'
# Minimal Rocq project file
-Q . ""
EOF
  fi

  # Activate scripts (Linux only)
if [[ "$OS_NAME" == "linux" && -n "${OPAM_SWITCH_NAME:-}" ]]; then

  # activate.sh (source-friendly)
  cat > "$WORKSPACE_DIR/activate.sh" <<EOF
# activate.sh
# Usage:
#   source ./activate.sh

if [[ "\${BASH_SOURCE[0]}" == "\${0}" ]]; then
  echo "This script must be sourced:"
  echo "  source ./activate.sh"
  exit 1
fi

set -euo pipefail

if ! command -v opam >/dev/null 2>&1; then
  echo "opam not found in PATH"
  return 1
fi

eval "\$(opam env --switch=${OPAM_SWITCH_NAME})"

echo "Activated opam switch: ${OPAM_SWITCH_NAME}"
echo "rocq version:"
rocq --print-version 2>/dev/null || rocq --version
EOF

  chmod +x "$WORKSPACE_DIR/activate.sh"

  # activate-shell.sh
  cat > "$WORKSPACE_DIR/activate-shell.sh" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"

# shellcheck source=activate.sh
source "$SCRIPT_DIR/activate.sh"

exec "${SHELL:-bash}"
EOF

  chmod +x "$WORKSPACE_DIR/activate-shell.sh"

  log "Created activation scripts in workspace"
fi

}

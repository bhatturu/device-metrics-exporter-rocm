#!/usr/bin/env bash
# PreToolUse hook: reject Write/Edit on generated, vendored, or jobd-owned files.
# Reads tool input JSON on stdin; exits 2 with a message on stderr to block.
set -euo pipefail

input=$(cat)
path=$(printf '%s' "$input" | python3 -c '
import json, sys
d = json.load(sys.stdin)
ti = d.get("tool_input", {})
print(ti.get("file_path") or ti.get("notebook_path") or "")
')

[[ -z "$path" ]] && exit 0

root="${CLAUDE_PROJECT_DIR:-$(pwd)}"
rel="${path#${root%/}/}"

deny() {
  printf 'BLOCKED edit of %s\nReason: %s\n' "$rel" "$1" >&2
  exit 2
}

case "$rel" in
  entrypoint.sh)
    deny "Root entrypoint.sh is the jobd CI dev-container helper, NOT the runtime container entrypoint. Edit docker/entrypoint.sh instead (see CLAUDE.md \"Don't touch\")."
    ;;
  *.pb.go)
    deny "Generated protobuf code. Edit the .proto and run 'make gen' instead."
    ;;
  pkg/*/gen/*|pkg/*/gen/**/*)
    deny "Generated code under pkg/*/gen/. Regenerate via 'make gen' instead of hand-editing."
    ;;
  vendor/*)
    deny "Vendored dependency. Update via go.mod + 'go mod vendor', do not hand-edit."
    ;;
  libamdsmi/*|libgimsmi/*)
    deny "Vendored library tree. Use /amdsmi-update or /rocm-update skill to refresh."
    ;;
esac

exit 0

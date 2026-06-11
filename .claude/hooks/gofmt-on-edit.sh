#!/usr/bin/env bash
# PostToolUse hook: run `gofmt -w` on any .go file that was just edited.
# Silent on success; prints to stderr (non-blocking) only if gofmt itself fails.
set -euo pipefail

input=$(cat)
path=$(printf '%s' "$input" | python3 -c '
import json, sys
d = json.load(sys.stdin)
ti = d.get("tool_input", {})
print(ti.get("file_path") or "")
')

[[ -z "$path" ]] && exit 0
[[ "$path" != *.go ]] && exit 0
[[ ! -f "$path" ]] && exit 0

if ! command -v gofmt >/dev/null 2>&1; then
  exit 0  # gofmt not on host (e.g. inside container without Go) — silently skip
fi

if ! gofmt -w "$path" 2>/tmp/gofmt-hook.err; then
  printf 'gofmt failed on %s:\n' "$path" >&2
  cat /tmp/gofmt-hook.err >&2
  exit 1  # non-blocking warning (PostToolUse exit codes don't block the tool — Claude sees stderr)
fi

exit 0

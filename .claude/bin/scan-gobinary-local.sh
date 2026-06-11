#!/usr/bin/env bash
# scan-gobinary-local.sh — local CVE scan of Go binaries using trivy
#
# Usage: scan-gobinary-local.sh [bin-dir]
#        scan-gobinary-local.sh          # defaults to ./bin/
#
# Output: trivy table output to stdout
# Exit code: 1 if any HIGH or CRITICAL findings exist; 0 otherwise

set -euo pipefail

if [ $# -gt 1 ]; then
    echo "Usage: $0 [bin-dir]" >&2
    exit 1
fi

if ! command -v trivy &>/dev/null; then
    echo "Error: 'trivy' not found on PATH. Install from https://aquasecurity.github.io/trivy/latest/getting-started/installation/" >&2
    exit 1
fi

trivy rootfs --scanners vuln \
    --severity HIGH,CRITICAL \
    --format table \
    --exit-code 1 \
    "${1:-bin/}"

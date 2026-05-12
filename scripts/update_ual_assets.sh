#!/bin/bash

set -euo pipefail

# Script to download and process UAL artifacts from remote server
# Usage: ./scripts/update_ual_assets.sh <version>
# Example: ./scripts/update_ual_assets.sh 1.127.0-63

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
ASSETS_DIR="${PROJECT_ROOT}/assets"
TEMP_DIR=$(mktemp -d)

# Remote server details (can be overridden via UAL_REMOTE_SERVER environment variable)
REMOTE_SERVER="${UAL_REMOTE_SERVER:-remote_fqdn}"
REMOTE_BASE_PATH="/vol/builds/hourly"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

cleanup() {
    log_info "Cleaning up temporary directory: ${TEMP_DIR}"
    rm -rf "${TEMP_DIR}"
}

trap cleanup EXIT

# Check if version is provided
if [ $# -eq 0 ]; then
    log_error "Version not provided"
    echo "Usage: $0 <version>"
    echo "Example: $0 1.127.0-63"
    exit 1
fi

VERSION="$1"
log_info "Updating UAL assets to version: ${VERSION}"
log_info "Remote server: ${REMOTE_SERVER}"

# Construct remote paths
REMOTE_GPUAGENT_PATH="${REMOTE_BASE_PATH}/${VERSION}/rudra-bundle/internal-artifacts/nodemgmt/gpuagent_${VERSION}.tar.gz"
REMOTE_GPUCTL_PATH="${REMOTE_BASE_PATH}/${VERSION}/rudra-bundle/internal-artifacts/nodemgmt/gpuctl_${VERSION}.tar.gz"

# Download gpuagent
log_info "Downloading gpuagent from ${REMOTE_SERVER}:${REMOTE_GPUAGENT_PATH}"
if ! scp "${REMOTE_SERVER}:${REMOTE_GPUAGENT_PATH}" "${TEMP_DIR}/gpuagent_${VERSION}.tar.gz"; then
    log_error "Failed to download gpuagent from remote server"
    exit 1
fi

# Download gpuctl
log_info "Downloading gpuctl from ${REMOTE_SERVER}:${REMOTE_GPUCTL_PATH}"
if ! scp "${REMOTE_SERVER}:${REMOTE_GPUCTL_PATH}" "${TEMP_DIR}/gpuctl_${VERSION}.tar.gz"; then
    log_error "Failed to download gpuctl from remote server"
    exit 1
fi

# Extract gpuagent
log_info "Extracting gpuagent archive"
mkdir -p "${TEMP_DIR}/gpuagent_extract"
tar -xzf "${TEMP_DIR}/gpuagent_${VERSION}.tar.gz" -C "${TEMP_DIR}/gpuagent_extract"

# Find the gpuagent binary (it should be in the extracted directory)
GPUAGENT_BIN=$(find "${TEMP_DIR}/gpuagent_extract" -type f -name "gpuagent" | head -1)
if [ -z "${GPUAGENT_BIN}" ]; then
    log_error "gpuagent binary not found in extracted archive"
    exit 1
fi
log_info "Found gpuagent binary: ${GPUAGENT_BIN}"

# Get the directory containing the binary
GPUAGENT_DIR=$(dirname "${GPUAGENT_BIN}")

# Strip and create tar.gz for gpuagent
log_info "Stripping gpuagent binary"
strip "${GPUAGENT_BIN}"

log_info "Creating tar.gz archive for gpuagent"
# Change to the directory containing the binary to avoid path prefixes in the archive
pushd "${GPUAGENT_DIR}" > /dev/null
tar czf "${ASSETS_DIR}/gpuagent_ual.bin.gz" gpuagent
popd > /dev/null

chmod +x "${ASSETS_DIR}/gpuagent_ual.bin.gz"
log_info "Created: ${ASSETS_DIR}/gpuagent_ual.bin.gz"

# Extract gpuctl
log_info "Extracting gpuctl archive"
mkdir -p "${TEMP_DIR}/gpuctl_extract"
tar -xzf "${TEMP_DIR}/gpuctl_${VERSION}.tar.gz" -C "${TEMP_DIR}/gpuctl_extract"

# Find the gpuctl binary
GPUCTL_BIN=$(find "${TEMP_DIR}/gpuctl_extract" -type f -name "gpuctl" | head -1)
if [ -z "${GPUCTL_BIN}" ]; then
    log_error "gpuctl binary not found in extracted archive"
    exit 1
fi
log_info "Found gpuctl binary: ${GPUCTL_BIN}"

# Strip and copy gpuctl
log_info "Stripping and copying gpuctl"
strip "${GPUCTL_BIN}"
cp "${GPUCTL_BIN}" "${ASSETS_DIR}/gpuctl_ual"
chmod +x "${ASSETS_DIR}/gpuctl_ual"
log_info "Created: ${ASSETS_DIR}/gpuctl_ual"

# Update version.yaml
log_info "Updating version.yaml with version ${VERSION}"
VERSION_FILE="${ASSETS_DIR}/version.yaml"

# Update gpuagent_ual.bin.gz version
sed -i "/name.*:.*gpuagent_ual.bin.gz/,/version.*:/ s/version.*/version : ${VERSION}/" "${VERSION_FILE}"

# Update gpuctl_ual version
sed -i "/name.*:.*gpuctl_ual/,/version.*:/ s/version.*/version : ${VERSION}/" "${VERSION_FILE}"

log_info "Successfully updated UAL assets to version ${VERSION}"
log_info "Files updated:"
log_info "  - ${ASSETS_DIR}/gpuagent_ual.bin.gz"
log_info "  - ${ASSETS_DIR}/gpuctl_ual"
log_info "  - ${ASSETS_DIR}/version.yaml"

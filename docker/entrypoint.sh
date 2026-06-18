#!/usr/bin/env bash
set -euo pipefail
#
# Copyright(C) Advanced Micro Devices, Inc. All rights reserved.
#
# You may not use this software and documentation (if any) (collectively,
# the "Materials") except in compliance with the terms and conditions of
# the Software License Agreement included with the Materials or otherwise as
# set forth in writing and signed by you and an authorized signatory of AMD.
# If you do not have a copy of the Software License Agreement, contact your
# AMD representative for a copy.
#
# You agree that you will not reverse engineer or decompile the Materials,
# in whole or in part, except as allowed by applicable law.
#
# THE MATERIALS ARE DISTRIBUTED ON AN "AS IS" BASIS, WITHOUT WARRANTIES OR
# REPRESENTATIONS OF ANY KIND, EITHER EXPRESS OR IMPLIED.
#
#
# entry point script run on creating a node management container
# default is gpu monitoring enabled
MONITOR_GPU="true"

# IFOE field-exporter default. Overridable via ENABLE_IFOE env (true|false)
# or via explicit -monitor-ifoe= CLI arg (CLI wins). The Go-side capability
# gate in pkg/amdgpu/gpuagent/gpuagent_ifoe.go gracefully no-ops on hosts
# without IFOE-capable devices, so existing GPU-only customers see no regression.
DEFAULT_ENABLE_IFOE="${ENABLE_IFOE:-true}"
case "$DEFAULT_ENABLE_IFOE" in
  true|false) ;;
  *)
    echo "WARN: invalid ENABLE_IFOE='$DEFAULT_ENABLE_IFOE' (expected true|false); defaulting to true" >&2
    DEFAULT_ENABLE_IFOE="true"
    ;;
esac

# Loop through all arguments — capture monitor-gpu state and detect whether
# the user explicitly supplied -monitor-ifoe= (their flag wins over the env).
USER_SET_MONITOR_IFOE=0
for arg in "$@"; do
  case $arg in
    -monitor-gpu=*)
      MONITOR_GPU="${arg#*=}"   # Extract value after '='
      ;;
    -monitor-ifoe=*)
      USER_SET_MONITOR_IFOE=1
      ;;
  esac
done

# Build the final exporter argument list — pass through user args verbatim,
# then append our env-derived -monitor-ifoe only if the user didn't set it.
# Note: ARGS=("$@") + later access via "${ARGS[@]+"${ARGS[@]}"}" is the
# bash-3/4 safe form when the array may be empty under `set -u`.
ARGS=()
if [ "$#" -gt 0 ]; then
  ARGS=("$@")
fi
if [ "$USER_SET_MONITOR_IFOE" -eq 0 ]; then
  ARGS+=("-monitor-ifoe=${DEFAULT_ENABLE_IFOE}")
fi

if [ "$MONITOR_GPU" == "true" ]; then
  # version-agnostic: /home/amd/lib/libamd_smi.so is an unversioned symlink to the
  # shipped libamd_smi.so.<ver>, so an amdsmi version bump needs no change here.
  LD_PRELOAD=/home/amd/lib/libamd_smi.so /home/amd/bin/gpuagent -s /var/run/gpuagent.sock &

  # sleep before starting exporter
  sleep 10
fi

# start exporter
# Run the underlying binary with all arguments passed to the script
exec /home/amd/bin/server "${ARGS[@]}"

---
topology: single-node
timeout: 600
pass_criteria: "scripts/update_ual_assets.sh fetches from non-mock REMOTE_GPUAGENT_PATH; assets/gpuagent_ual.bin.gz contains the real binary"
stability: stable
retries: 0
validation_groups: [post-test]
---

# update_ual_assets.sh fetches non-mock gpuagent

## Purpose

Verify Sub-task S1 (drop `mock/` from `REMOTE_GPUAGENT_PATH` line 56) actually pulls the real binary, and the same `assets/gpuagent_ual.bin.gz` file is consumed by deb, rpm, and container builds.

## Category

negative

## Prerequisites

- Build host with scp access to the UAL build server (`UAL_REMOTE_SERVER`)
- A known UAL version available on the build server (e.g., `1.127.0-116`)

## Steps

1. **[host]** `bash -x scripts/update_ual_assets.sh 1.127.0-116 2>&1 | tee /tmp/asset-fetch.log`
   Expected: log shows `REMOTE_GPUAGENT_PATH=...nodemgmt/gpuagent_1.127.0-116.tar.gz` (no `/mock/` segment)
2. **[host]** `ls -la assets/gpuagent_ual.bin.gz`
   Expected: file present, recent mtime
3. **[host]** `gzip -dc assets/gpuagent_ual.bin.gz | strings | grep -iE 'mock|stub' | head`
   Expected: no obvious mock/stub strings (or, mock strings count clearly lower than the prior mock binary)
4. **[host]** `cat assets/version.yaml | grep -A1 gpuagent_ual`
   Expected: version updated to `1.127.0-116`

## Expected Result

The script's REMOTE_GPUAGENT_PATH no longer contains `mock/`; the fetched binary is the real UAL gpuagent.

## Failure Indicators

- scp 404 (path not found on server) — indicates the non-mock path doesn't exist for that version
- `/mock/` still appearing in the log
- Binary still has mock strings in same density as before

## Cleanup

- `rm /tmp/asset-fetch.log`

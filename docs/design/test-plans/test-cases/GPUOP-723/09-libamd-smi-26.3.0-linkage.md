---
topology: single-node
timeout: 180
pass_criteria: "gpuagent process starts under LD_PRELOAD of libamd_smi.so.26 → libamd_smi.so.26.3.0 with no undefined-symbol or shared-object load errors"
stability: stable
retries: 0
validation_groups: [post-test]
---

# libamd_smi 26.3.0 ↔ gpuagent runtime parity (LD_PRELOAD)

## Purpose

This is the **only available parity check** between the bundled
libamd_smi 26.3.0 and the UAL gpuagent shipped in the same image.
The gpuagent does not expose its built-against amd-smi version anywhere
consultable (no tarball manifest, no `--version` flag, no useful ELF
SONAME mapping) — confirmed during planning. The only practical
verification is runtime symbol resolution: LD_PRELOAD the bundled
libamd_smi and ensure gpuagent loads + runs without undefined-symbol
errors.

This test case is the production gate that closes Sub-task S7 from the
Story design doc (the standalone "spike + parity script" plan was
dropped — TC13 stub explains).

## Category

integration

## Prerequisites

- `${TEST_HOST_NON_IFOE}` + `${TEST_USER}` set at invocation
- Image `device-metrics-exporter:${IMAGE_TAG}` available
- Container `ame-noifoe-${USER}` already running from TC03 (used for step 2 only)

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `TEST_HOST_NON_IFOE` | (required) | non-IFOE test host |
| `TEST_USER` | `${USER}` | SSH login |
| `IMAGE_TAG` | `${USER}-GPUOP-723` | user-specific image tag |

## Steps

1. **[host]** Standalone gpuagent startup with LD_DEBUG to surface any linker issues:
   `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker run --rm --entrypoint /bin/bash device-metrics-exporter:${IMAGE_TAG} -c 'LD_DEBUG=libs LD_PRELOAD=/home/amd/lib/libamd_smi.so.26 /home/amd/bin/gpuagent --help 2>&1 | tail -100'"`
   Expected: gpuagent prints help; no `undefined symbol` or `cannot open shared object` errors anywhere in the LD_DEBUG output
2. **[host]** Confirm the running gpuagent (from TC03) actually loaded the new lib:
   `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-noifoe-\${USER} sh -c 'cat /proc/\$(pgrep gpuagent)/maps | grep libamd_smi'"`
   Expected: `libamd_smi.so.26.3.0` mapped into gpuagent's address space (NOT the old `26.2.1`)
3. **[host]** Scan exporter logs for amd-smi-related errors:
   `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-noifoe-\${USER} cat /var/log/exporter.log | grep -iE 'amdsmi|amd_smi' | grep -iE 'error|fail|undefined|missing'"`
   Expected: no output (no symbol-resolution failures at runtime)
4. **[host]** Confirm gpuagent stayed alive across the test (no late SIGSEGV from a missing symbol resolved lazily):
   `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-noifoe-\${USER} sh -c 'pgrep gpuagent && echo OK'"`
   Expected: PID and `OK` printed

## Expected Result

gpuagent loads and operates against libamd_smi 26.3.0 via the
`libamd_smi.so.26 → libamd_smi.so.26.3.0` symlink with zero linker
errors at startup or runtime. This passing case is the parity guarantee
between gpuagent and the bundled lib (no further parity check
infrastructure is needed in `update_ual_assets.sh`).

## Failure Indicators

- `undefined symbol` errors in LD_DEBUG output
- `cannot open shared object` errors
- gpuagent crash on startup or shortly after
- libamd_smi.so.26.2.1 (old version) still mapped into the gpuagent
  process (build context wasn't updated; S2 broken)
- amd-smi-related error / fail / missing log lines in exporter output
- gpuagent process gone between step 2 and step 4 (lazy symbol
  resolution failure)

## Cleanup

(none — read-only inspections against the running container from TC03)

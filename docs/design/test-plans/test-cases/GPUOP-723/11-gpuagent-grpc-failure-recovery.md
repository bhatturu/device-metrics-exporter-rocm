---
topology: single-node
timeout: 300
pass_criteria: "When gpuagent socket disappears, exporter retries with backoff; no panic in IFOE codepath; on socket return, IFOE recovers OR cleanly stays disabled"
stability: stable
retries: 0
validation_groups: [post-test]
---

# gpuagent gRPC failure recovery

## Purpose

Validate the IFOE Go code's error-recovery behavior when the underlying gpuagent socket is unreachable mid-run.

## Category

error-recovery

- `${TEST_HOST_NON_IFOE}` + `${TEST_USER}` set at invocation
- Container `ame-noifoe-${USER}` already running from TC03 with `ENABLE_IFOE=true`

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `TEST_HOST_NON_IFOE` | (required) | non-IFOE test host |
| `TEST_USER` | `${USER}` | SSH login |
| `IMAGE_TAG` | `${USER}-GPUOP-723` | user-specific image tag |
| `EXPORTER_PORT` | (inherited from TC03) | host port from TC03 Step 0; pattern in `.claude/skills/test-port-allocation.md` |

## Steps

1. **[host]** Start container as in TC03 (`ame-noifoe-${USER}`); let it stabilize for 60s
2. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-noifoe-\${USER} sh -c 'kill -9 \$(pgrep gpuagent)'"`
   Expected: gpuagent process gone; socket may disappear
3. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "sleep 30 && docker exec ame-noifoe-\${USER} cat /var/log/exporter.log | tail -50"`
   Expected: log shows gRPC connection errors with retry/backoff; no panic; no fatal exit
4. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-noifoe-\${USER} ps -ef | grep amd-metrics-exporter"`
   Expected: process present
5. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "curl -s http://localhost:${EXPORTER_PORT}/metrics | grep -c '^gpu_'"`
   Expected: count > 0 (GPU metrics may degrade but exporter still serves)

## Expected Result

Exporter handles gRPC failure gracefully — no panic, no exit; IFOE codepath logs error and either retries or stays disabled.

## Failure Indicators

- Exporter panic / fatal exit
- Container restart loop
- Hung process / no log output for > 60s

## Cleanup

- `docker restart ame-noifoe-${USER}` to restore baseline (or stop/rm)

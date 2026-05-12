---
topology: single-node
timeout: 180
pass_criteria: "ENABLE_IFOE=false → exporter starts; -monitor-ifoe=false in process args; no IFOE codepath invoked; no IFOE metrics; GPU metrics unaffected"
stability: stable
retries: 0
validation_groups: [post-test]
---

# ENABLE_IFOE=false → IFOE codepath never invoked

## Purpose

Verify the env-var override works: explicitly setting `ENABLE_IFOE=false` disables IFOE entirely (the entrypoint passes `-monitor-ifoe=false` and the Go IFOE init never runs).

## Category

positive

## Prerequisites

- `${TEST_HOST_NON_IFOE}` and `${TEST_USER}` set at invocation time (see Parameters); SSH auth via key
- Image `device-metrics-exporter:${USER}-GPUOP-723` available on the host

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `TEST_HOST_NON_IFOE` | (required) | non-IFOE test host |
| `TEST_USER` | `${USER}` | SSH login |
| `IMAGE_TAG` | `${USER}-GPUOP-723` | user-specific image tag |
| `EXPORTER_PORT` | (auto, see Step 0) | host port; pattern in `.claude/skills/test-port-allocation.md` |

## Steps

0. **[host]** Port discovery (per `.claude/skills/test-port-allocation.md`):
   `EXPORTER_PORT=$(comm -23 <(seq 5000 5100 | sort) <(ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "ss -tnl" | awk 'NR>1{print $4}' | sed 's/.*://' | sort -u) | head -1); [ -n "${EXPORTER_PORT}" ] || exit 1`
   Expected: integer in [5000,5100]
1. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker run -d --name ame-off-${USER} -e ENABLE_IFOE=false --device=/dev/kfd --device=/dev/dri -v /sys:/sys:ro -p \${EXPORTER_PORT}:5000 device-metrics-exporter:${IMAGE_TAG}"`
   Expected: container starts
2. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-off-${USER} ps -ef | grep amd-metrics-exporter"`
   Expected: process args contain `-monitor-ifoe=false`
3. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-off-${USER} cat /var/log/exporter.log | grep -iE 'ifoe (init|enabled|disabled)'"`
   Expected: no IFOE init/enabled/disabled log lines (codepath never reached)
4. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "curl -s http://localhost:${EXPORTER_PORT}/metrics | grep -cE '^amd_ifoe_'"`
   Expected: `0`
5. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "curl -s http://localhost:${EXPORTER_PORT}/metrics | grep -c '^gpu_'"`
   Expected: > 0 (GPU metrics flow)

## Expected Result

`-monitor-ifoe=false` reaches the binary, IFOE Go code never initializes, no IFOE metrics published, GPU monitoring works.

## Failure Indicators

- IFOE init log lines despite ENABLE_IFOE=false
- Any IFOE metric series
- GPU metric series missing

## Cleanup

- `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker stop ame-off-${USER} && docker rm ame-off-${USER}"`

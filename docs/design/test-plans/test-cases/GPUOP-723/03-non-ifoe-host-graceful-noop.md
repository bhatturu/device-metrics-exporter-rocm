---
topology: single-node
timeout: 300
pass_criteria: "Container starts cleanly; logs show 'IFOE disabled' exactly once; /metrics returns zero IFOE series; no panics; GPU metrics flow normally"
stability: stable
retries: 0
validation_groups: [post-test]
---

# Non-IFOE host with ENABLE_IFOE=true → graceful no-op

## Purpose

Validate the Go-side capability gate: on a host with no IFOE-capable devices, the container must default to `ENABLE_IFOE=true` (the new default) but the Go code in `pkg/amdgpu/gpuagent/gpuagent_ifoe.go` must detect zero IFOE devices and gracefully disable the IFOE collector. This guards every existing customer running the exporter on plain GPU-only hardware.

## Category

negative

## Prerequisites

- Non-IFOE test host (1-GPU AMD, no Pensando NIC) reachable via SSH; details supplied at invocation time via `${TEST_HOST_NON_IFOE}` and `${TEST_USER}` (see Parameters). SSH auth via key or `${TEST_USER}` ambient credential — never stored in this file.
- New image loaded on the host as `device-metrics-exporter:${USER}-GPUOP-723` (transfer `docker/device-metrics-exporter-latest.tar.gz` from build host, `docker load`, then `docker tag device-metrics-exporter:latest device-metrics-exporter:${USER}-GPUOP-723`)
- AMDGPU driver loaded; existing GPU monitoring works on the host

## Parameters

| Parameter | Default | Range | Description |
|-----------|---------|-------|-------------|
| `TEST_HOST_NON_IFOE` | (none — required) | hostname or IP | Test host with AMD GPU but no Pensando IFOE NIC |
| `TEST_USER` | `${USER}` | any user with SSH access | SSH login on the test host |
| `IMAGE_TAG` | `${USER}-GPUOP-723` | any docker tag | User-specific tag from TC01 retag |
| `EXPORTER_PORT` | (auto, see Step 0) | 5000-5100 | host port forwarded to container 5000; auto-picked to avoid hourly-build collision (pattern: `.claude/skills/test-port-allocation.md`) |

## Steps

0. **[host]** Discover a free port on the test host (pattern from `.claude/skills/test-port-allocation.md`):
   `EXPORTER_PORT=$(comm -23 <(seq 5000 5100 | sort) <(ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "ss -tnl" | awk 'NR>1{print $4}' | sed 's/.*://' | sort -u) | head -1); echo "EXPORTER_PORT=${EXPORTER_PORT}"; [ -n "${EXPORTER_PORT}" ] || exit 1`
   Expected: integer in [5000,5100] echoed; non-empty (export this for downstream tests TC04/TC05/TC09/TC11 that share the container)
1. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "lspci -d 1dd8:"`
   Expected: empty output (no Pensando IFOE NIC)
2. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker run -d --name ame-noifoe-\${USER} --device=/dev/kfd --device=/dev/dri -v /sys:/sys:ro -p \${EXPORTER_PORT}:5000 device-metrics-exporter:${USER}-GPUOP-723"`
   Expected: container starts, no immediate exit (deployment matches `docs/installation/docker.md`)
3. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "sleep 30 && docker exec ame-noifoe-\${USER} cat /var/log/exporter.log | grep -c 'IFOE disabled'"`
   Expected: exactly `1` (single occurrence; the gate must not log repeatedly)
4. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-noifoe-\${USER} cat /var/log/exporter.log | grep -E 'panic|fatal'"`
   Expected: no output (zero panics / fatals)
5. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "curl -s http://localhost:${EXPORTER_PORT}/metrics | grep -cE '^amd_ifoe_'"`
   Expected: `0` (no IFOE series published)
6. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "curl -s http://localhost:${EXPORTER_PORT}/metrics | grep -cE '^gpu_(temperature|utilization|power)'"`
   Expected: count > 0 (existing GPU metrics still flow normally — series names per `docs/configuration/metricslist.md`)

## Expected Result

The container with `ENABLE_IFOE=true` (the new default) on a non-IFOE host: starts cleanly, logs the "IFOE disabled" reason exactly once, publishes zero IFOE metric series, but continues to publish all existing GPU metrics. Existing customers see no regression.

## Failure Indicators

- "IFOE disabled" log line appearing more than once (gate log loop bug)
- Any panic or fatal log line
- Exporter restart loop
- IFOE metric series present despite no IFOE hardware
- Existing GPU metric series count drops vs pre-change baseline

## Cleanup

- `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker stop ame-noifoe-\${USER} && docker rm ame-noifoe-\${USER}"` (user-suffixed name avoids collision with hourly build's `device-metrics-exporter` container)

---
topology: single-node
timeout: 1200
pass_criteria: "Container deployed; /validate-ifoe-exporter end-to-end harness reports overall PASS — counts, UUIDs/labels, and per-entity field values match between Prometheus /metrics and gpuctl on the live system"
stability: stable
retries: 0
validation_groups: [post-test]
---

# IFOE metrics scraped + validated end-to-end on real IFOE-capable hardware

## Purpose

Validate the end-to-end happy path: container with `ENABLE_IFOE=true`
deployed on a real IFOE-capable host exposes IFOE port/station/device
Prometheus metrics that **correctly reflect** the live UAL state as
reported by `gpuctl show ual`. The repo ships a dedicated validation
harness for this — `.claude/commands/validate-ifoe-exporter.md` (slash
command `/validate-ifoe-exporter`) — which performs counts validation,
UUID/label validation, and per-entity field value cross-checks across
~30 metric ↔ gpuctl field mappings. This test case wraps that harness
with deployment setup/teardown.

## Category

positive

## Prerequisites

- IFOE-capable host (Pensando NIC + AMD GPU) reachable via SSH; supplied at invocation via `${TEST_HOST_IFOE}` and `${TEST_USER}` (see Parameters). SSH auth via key.
- Image loaded on the host as `device-metrics-exporter:${IMAGE_TAG}` (load via `docker load -i device-metrics-exporter-latest.tar.gz` then retag, or pull from a registry)
- AMDGPU kernel driver loaded
- `gpuctl` available on the host (shipped with the gpuagent / IFOE deb/rpm; the validation harness uses it for cross-check)
- `/etc/metrics/config.json` exists on the host with `ServerPort` set (the harness reads the port from here — never assumes 5000)

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `TEST_HOST_IFOE` | (required) | IFOE-capable test host (Pensando NIC + AMD GPU) |
| `TEST_USER` | `${USER}` | SSH login on the test host |
| `IMAGE_TAG` | `${USER}-GPUOP-723` | user-specific image tag from TC01 retag |
| `EXPORTER_PORT` | (auto-discovered, see Step 0) | host TCP port forwarded to container 5000; written to `/etc/metrics/config.json:ServerPort` so the validation harness picks it up. Pattern: `.claude/skills/test-port-allocation.md` |

## Steps

0. **[host]** Discover a free port (per `.claude/skills/test-port-allocation.md`):
   `EXPORTER_PORT=$(comm -23 <(seq 5000 5100 | sort) <(ssh "${TEST_USER}@${TEST_HOST_IFOE}" "ss -tnl" | awk 'NR>1{print $4}' | sed 's/.*://' | sort -u) | head -1); [ -n "${EXPORTER_PORT}" ] || exit 1`
   Expected: integer in [5000,5100]
1. **[host]** Confirm hardware presence: `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "lspci -d 1dd8: && lspci -d 1002:"`
   Expected: at least one Pensando IFOE NIC AND one AMD GPU listed
2. **[host]** Stage `/etc/metrics/config.json` with the discovered port so the validation harness picks the right port:
   `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "sudo mkdir -p /etc/metrics && echo '{\"ServerPort\": '${EXPORTER_PORT}'}' | sudo tee /etc/metrics/config.json"`
   Expected: file written; readable by the SSH user
3. **[host]** Deploy the container with the discovered port + config mount (matches `docs/configuration/docker.md`):
   `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "docker run -d --name ame-ifoe-${USER} --device=/dev/kfd --device=/dev/dri -v /sys:/sys:ro -v /etc/metrics:/etc/metrics -p \${EXPORTER_PORT}:5000 device-metrics-exporter:${IMAGE_TAG}"`
   Expected: container starts; `docker ps` shows it Up
4. **[host]** Verify the IFOE codepath actually engaged (no graceful no-op):
   `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "sleep 30 && docker exec ame-ifoe-${USER} cat /var/log/exporter.log | grep -i 'IFOE disabled'"`
   Expected: no output (because IFOE hardware IS present, the gate must NOT disable IFOE)
5. **[host]** Sanity-scrape: at least one IFOE metric series present:
   `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "curl -s http://localhost:${EXPORTER_PORT}/metrics | grep -cE '^amd_ifoe_'"`
   Expected: count > 0
6. **[host]** Run the dedicated end-to-end validation harness — this is the authoritative pass/fail gate for this test case:
   `/validate-ifoe-exporter ${TEST_USER}@${TEST_HOST_IFOE}`
   Expected: harness produces `IFOE_EXPORTER_<HOSTNAME>_SUMMARY.md` in cwd with **Overall: PASS**. Counts (devices, stations, ports) match between Prometheus and `gpuctl show ual`; UUID labels match decoded `gpuctl` byte arrays; per-entity field values match within the harness's 5%/monotonic tolerance for counters.
7. **[host]** Final no-error scan: `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "docker exec ame-ifoe-${USER} cat /var/log/exporter.log | grep -E 'panic|fatal|error.*ifoe' | head"`
   Expected: no panic / fatal / IFOE-error log lines

## Expected Result

Container deploys successfully on real IFOE hardware. The
`/validate-ifoe-exporter` harness reports overall **PASS** — meaning
device/station/port counts in Prometheus match `gpuctl`, UUID labels
correctly identify each entity, and ~30 per-entity field values
(linkstate, speed, link counters, FEC codeword bins, RX/TX bytes/packets,
station request/response counters, stream remaps) match between
Prometheus and the live UAL state. No IFOE-related errors in container
logs.

## Failure Indicators

- "IFOE disabled" in logs despite real IFOE hardware present (capability gate misfiring)
- Validation harness reports **Overall: FAIL** with count mismatch (devices/stations/ports diverge)
- Harness reports UUID label mismatch (Prometheus labels don't decode to the gpuctl byte arrays)
- Harness reports per-entity field FAIL (a counter is non-monotonic or differs by > 5%)
- Harness flags **UNEXPECTED METRIC** (a new `amd_ifoe_*` metric not in the known UAL proto field list)
- gpuagent gRPC connection errors in container logs
- Container restart loop

## Cleanup

- `ssh "${TEST_USER}@${TEST_HOST_IFOE}" "docker stop ame-ifoe-${USER} && docker rm ame-ifoe-${USER}"`
- Optionally restore the host's prior `/etc/metrics/config.json` if you backed it up before step 2
- Archive the harness report `IFOE_EXPORTER_<HOSTNAME>_SUMMARY.md` for the test record

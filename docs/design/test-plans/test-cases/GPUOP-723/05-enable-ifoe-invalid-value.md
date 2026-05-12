---
topology: single-node
timeout: 180
pass_criteria: "Invalid ENABLE_IFOE value → entrypoint warns and defaults to true; container still starts"
stability: stable
retries: 0
validation_groups: [post-test]
---

# ENABLE_IFOE=garbage → defaults to true with warn

## Purpose

Boundary check on entrypoint env-var parsing. Invalid values must not crash the container.

## Category

boundary

## Prerequisites

- `${TEST_HOST_NON_IFOE}` and `${TEST_USER}` set at invocation time
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
1. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker run -d --name ame-bad-${USER} -e ENABLE_IFOE=garbage --device=/dev/kfd --device=/dev/dri -v /sys:/sys:ro -p \${EXPORTER_PORT}:5000 device-metrics-exporter:${IMAGE_TAG}"`
   Expected: container starts (no exit)
2. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker logs ame-bad-${USER} 2>&1 | grep -iE 'enable_ifoe.*invalid|warn'"`
   Expected: warn log noting invalid value, defaulting to true
3. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-bad-${USER} ps -ef | grep amd-metrics-exporter"`
   Expected: process args contain `-monitor-ifoe=true`
4. **[host]** `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker exec ame-bad-${USER} grep 'IFOE disabled' /var/log/exporter.log"`
   Expected: present (since this is a non-IFOE host and we defaulted to true, Go gate kicks in)

## Expected Result

Invalid value treated as "true" with a warn log; container behaves identically to the default case on the non-IFOE host.

## Failure Indicators

- Container exits / restarts
- No warn log about invalid value
- `-monitor-ifoe=garbage` literally passed (parse bug)

## Cleanup

- `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}" "docker stop ame-bad-${USER} && docker rm ame-bad-${USER}"`

---
topology: single-node
timeout: 60
pass_criteria: "ldconfig -p inside the container lists libmnl, libnl-3, libnl-genl-3"
stability: stable
retries: 0
---

# IFOE runtime libs installed in container

## Purpose

Verify Sub-task S3 (add `libmnl libnl3 libnl3-cli` to microdnf install) succeeded — the IFOE-required RHEL packages are present in the built image.

## Category

integration

- Built image present locally as `device-metrics-exporter:${IMAGE_TAG}` (TC01 retag)

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `IMAGE_TAG` | `${USER}-GPUOP-723` | user-specific image tag |

## Steps

1. **[host]** `docker run --rm --entrypoint /bin/bash device-metrics-exporter:${IMAGE_TAG} -c "ldconfig -p | grep -E 'libmnl|libnl-3|libnl-genl-3'"`
   Expected: 3 or more matching lines covering libmnl, libnl-3, libnl-genl-3
2. **[host]** `docker run --rm --entrypoint /bin/bash device-metrics-exporter:${IMAGE_TAG} -c "rpm -qa | grep -E 'libmnl|libnl3'"`
   Expected: at least 3 packages (libmnl, libnl3, libnl3-cli)

## Expected Result

All three IFOE runtime libraries installed and discoverable by ld.so.

## Failure Indicators

- ldconfig output empty
- rpm -qa shows missing packages

## Cleanup

(none)

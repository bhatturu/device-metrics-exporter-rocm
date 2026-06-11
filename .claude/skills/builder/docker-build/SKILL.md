---
name: Docker Build
description: Build Docker container images for deployment in multiple variants (standard, SR-IOV, AINIC, Ubuntu 22.04, mock)
version: 2.0.0
---

# Docker Image Build Workflow

Builds Docker container images for deploying the AMD Device Metrics Exporter.

## Overview

**Prerequisites**: Completed gpuagent-build AND exporter-build
**Build Environment**: Host-based (no container)

## Build Variants

| Variant | Command | Use Case | Dev Variant Suffix |
|---------|---------|----------|--------------------|
| Standard | `make docker` | Production (RHEL-based) | `rhel9` |
| SR-IOV | `make docker-sriov` | SR-IOV enabled GPUs (RHEL9 only for dev) | `rhel9` |
| AINIC | `make docker-ainic` | AINIC driver integration | `ainic` |
| Ubuntu 22.04 SR-IOV | `make docker-sriov-ub22` | Ubuntu-based SR-IOV deployment | `ub22` |
| Mock | `make docker-mock` | Testing/development | `mock` |

## Local Dev Image Tagging (REQUIRED for local builds)

Local dev images **MUST** follow this naming convention so they are distinguishable from CI and production images and don't collide across users:

```
<DOCKER_REGISTRY>/<username>/<product>:dev<N>-<variant>
```

- **`DOCKER_REGISTRY`** — taken from `dev.env`. Do not hardcode; resolve from `dev.env` (`$(grep ^DOCKER_REGISTRY dev.env)`).
- **`username`** — local developer handle (e.g. your unix username).
- **`product`** — base product name (e.g. `device-metrics-exporter`, `device-metrics-exporter-sriov`, `device-metrics-exporter-ainic`).
- **`dev<N>`** — incremental local build number (`dev1`, `dev2`, …). Bump on every new local build so HW test runs can be matched back to a specific build.
- **`variant`** — OS/feature distinguisher from the table above (`rhel9`, `ub22`, `ainic`, `mock`).

### Example shape

```
<DOCKER_REGISTRY>/<username>/device-metrics-exporter-sriov:dev1-rhel9
<DOCKER_REGISTRY>/<username>/device-metrics-exporter:dev3-rhel9
<DOCKER_REGISTRY>/<username>/device-metrics-exporter-ainic:dev2-ainic
<DOCKER_REGISTRY>/<username>/device-metrics-exporter-sriov:dev1-ub22
<DOCKER_REGISTRY>/<username>/device-metrics-exporter:dev5-mock
```

### How to apply the tag at build time

Each variant has a Makefile image variable (`EXPORTER_IMAGE`, `EXPORTER_SRIOV_IMAGE`, `EXPORTER_AINIC_IMAGE`, etc.) that resolves to `$(DOCKER_REGISTRY)/$(EXPORTER_*_IMAGE_NAME):$(EXPORTER_IMAGE_TAG)`. Override the full ref on the make command line — this wins over `dev.env`:

```bash
# SR-IOV (RHEL9 — the only dev variant for SR-IOV)
make EXPORTER_SRIOV_IMAGE=<DOCKER_REGISTRY>/<username>/device-metrics-exporter-sriov:dev1-rhel9 \
     EXPORTER_SRIOV_BASE_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.6 \
     docker-sriov

# Standard
make EXPORTER_IMAGE=<DOCKER_REGISTRY>/<username>/device-metrics-exporter:dev1-rhel9 docker

# AINIC
make EXPORTER_AINIC_IMAGE=<DOCKER_REGISTRY>/<username>/device-metrics-exporter-ainic:dev1-ainic docker-ainic

# Ubuntu 22.04 SR-IOV
make EXPORTER_SRIOV_IMAGE=<DOCKER_REGISTRY>/<username>/device-metrics-exporter-sriov:dev1-ub22 \
     BUILD_BASE_IMAGE=ubuntu:22.04 \
     docker-sriov-ub22
```

### Picking `dev<N>`

Before tagging, check the highest existing dev tag for that product+variant and bump by 1:

```bash
docker images --format '{{.Repository}}:{{.Tag}}' \
  | grep "<username>/device-metrics-exporter-sriov:dev.*-rhel9" \
  | sed 's/.*:dev\([0-9]*\)-rhel9/\1/' | sort -n | tail -1
```

If nothing returns, start at `dev1`. Otherwise use `dev<N+1>`.

### Base-image overrides (network)

`dev.env` may rewrite base images through an internal registry that is HTTP-only and unreachable from a dev box. Always pair the dev tag override with a public base image:

| Variant | Base override |
|---------|---------------|
| SR-IOV RHEL9 | `EXPORTER_SRIOV_BASE_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.6` |
| SR-IOV ub22  | `BUILD_BASE_IMAGE=ubuntu:22.04` |
| Standard / AINIC / Mock | check Makefile for the relevant `*_BASE_IMAGE` var and override to the public image |

## Build Process

### 1. Validate Prerequisites
```bash
# Check assets
ls -lh assets/
# Must contain: 4 files (3 .bin.gz + gpuctl.gobin)

# Check binary
ls -lh bin/amd-metrics-exporter
# Must exist and be > 10 MB
```

### 2. Build Docker Image
```bash
# Standard variant
make -C docker TOP_DIR=$(PWD)

# OR specific variant
make docker-sriov
make docker-ainic
make docker-sriov-ub22
make docker-mock
```

### 3. Verify Image
```bash
docker images | grep exporter
# Should show image with recent timestamp
```

### 4. Test Image (Optional)
```bash
docker run --rm <image>:<tag> --help
```

## Common Issues

**Binary not found** (`bin/amd-metrics-exporter`): Run `exporter-build` first
**Assets missing**: Run `gpuagent-build` first
**Base image pull failure**: Check network/registry credentials
**Stale layers**: Use `--no-cache` flag to force rebuild

## Success Criteria

- ✅ Image appears in `docker images`
- ✅ Recent creation timestamp
- ✅ Size: 500 MB - 1 GB (varies by variant)
- ✅ Correct tag applied

## Build Time

1-2 minutes (with Docker layer caching)

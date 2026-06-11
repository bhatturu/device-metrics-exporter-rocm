---
name: rocm-update
description: This skill should be used when updating the Device Metrics Exporter to a new ROCm/therock version. Orchestrates the full update workflow: create branches, update amdsmi libraries, rebuild gpuagent, update DME assets, build docker image.
version: 1.0.0
---

# ROCm Version Update Workflow

Updates the Device Metrics Exporter stack to a new ROCm/therock release. Coordinates changes across gpu-agent and DME repositories.

## Overview

A ROCm version update requires changes in two repos in this order:

1. **gpu-agent** (`~/src/gpu-agent`) — update amdsmi library + header, rebuild binaries
2. **DME** (`~/src/device-metrics-exporter`) — update tarball URL, asset copy, rebuild docker image

## Inputs Required

Before starting, gather:
- **New ROCm version** (e.g. `7.13`)
- **Therock tarball URL** — found in the BKC PDF in `docs/` or from `https://therock-nightly-tarball.s3.amazonaws.com/`
  - Format: `https://therock-nightly-tarball.s3.amazonaws.com/therock-dist-linux-gfx950-dcgpu-<VERSION>a<DATE>.tar.gz`
- **Base DME branch** — typically `collab-<prev_version>` (e.g. `collab-7.12`)
- **Base gpu-agent commit** — latest main excluding any unwanted commits (check with user)
- **Source branch for amdsmi** — check `https://github.com/ROCm/rocm-systems/branches` for `release/therock-<version>`; if not available, extract from tarball

## Step 1: Verify Tarball

```bash
# Confirm tarball exists and find libamd_smi.so version
curl -s --head <TARBALL_URL> | grep -i "200\|content-length"

# Find .so version and amdsmi.h path
curl -s <TARBALL_URL> | tar -tz 2>/dev/null | grep -E "libamd_smi\.so\.|amdsmi\.h"
```

## Step 2: Update gpu-agent

### 2a. Pull latest and create branch

```bash
cd ~/src/gpu-agent
git fetch origin
git checkout <BASE_COMMIT>
git checkout -b feature/update-amdsmi-therock-<VERSION>
```

### 2b. Check if amdsmi source branch exists in rocm-systems

The gpu-agent tree contains a sparse partial clone of `ROCm/rocm-systems` at
`sw/nic/third-party/amdsmi-src/rocm-systems/`, currently tracking `release/therock-<PREV_VERSION>`.

```bash
# Check if release/therock-<VERSION> exists (no auth token needed for public repo)
curl -s "https://api.github.com/repos/ROCm/rocm-systems/branches?per_page=100" | \
  python3 -c "import json,sys; [print(b['name']) for b in json.load(sys.stdin) if '<VERSION>' in b['name']]"
# Also check page 2 if nothing found:
curl -s "https://api.github.com/repos/ROCm/rocm-systems/branches?per_page=100&page=2" | \
  python3 -c "import json,sys; [print(b['name']) for b in json.load(sys.stdin) if '<VERSION>' in b['name']]"
```

**If branch exists** — update the in-tree sparse clone:

```bash
cd sw/nic/third-party/amdsmi-src/rocm-systems
git remote set-branches origin release/therock-<VERSION>
git fetch origin release/therock-<VERSION>
git checkout -b release/therock-<VERSION> origin/release/therock-<VERSION>
# Then copy amdsmi.h from projects/amdsmi/include/ into the vendor tree
```

**If branch does not exist** (common for new versions) — extract from tarball (see 2c).

### 2c. Extract amdsmi from tarball

```bash
mkdir -p /tmp/therock-<VERSION>
curl -s <TARBALL_URL> | tar -xz -C /tmp/therock-<VERSION> \
  "./include/amd_smi/amdsmi.h" \
  "./lib/libamd_smi.so.<SO_VERSION>"

# Copy into gpu-agent vendor tree
cp /tmp/therock-<VERSION>/include/amd_smi/amdsmi.h \
   sw/nic/third-party/rocm/amd_smi_lib/include/amd_smi/amdsmi.h

cp /tmp/therock-<VERSION>/lib/libamd_smi.so.<SO_VERSION> \
   sw/nic/third-party/rocm/amd_smi_lib/x86_64/lib/libamd_smi.so.<SO_VERSION>

ln -sf libamd_smi.so.<SO_VERSION> \
   sw/nic/third-party/rocm/amd_smi_lib/x86_64/lib/libamd_smi.so.26

# Update version.txt
echo "therock-<VERSION>/rocm-systems/amdsmi-<SO_VERSION>" > \
  sw/nic/third-party/rocm/amd_smi_lib/version.txt
```

### 2d. Commit and push

```bash
git add sw/nic/third-party/rocm/amd_smi_lib/
git commit -m "feat: update amdsmi to therock-<VERSION>"
git push origin feature/update-amdsmi-therock-<VERSION>
```

## Step 3: Rebuild gpuagent binaries

Use the **builder skill** (`/builder gpuagent`) — this is the recommended approach. The builder skill handles the correct Docker container invocation and build ordering automatically.

Verify outputs:
```bash
ls -lh ~/src/gpu-agent/sw/nic/build/x86_64/sim/bin/gpuagent{,_gim,_mock}
~/src/gpu-agent/sw/nic/build/x86_64/sim/bin/gpuctl version
```

## Step 4: Update DME

### 4a. Pull latest base branch and create branch

```bash
cd ~/src/device-metrics-exporter
git fetch pensando <BASE_BRANCH>
git checkout <BASE_BRANCH>
git reset --hard pensando/<BASE_BRANCH>
git checkout -b feature/rocm-<VERSION>-support
```

### 4b. Update version references

| File | Change |
|---|---|
| `docker/Dockerfile.exporter-release` | `ARG ROCM_VERSION` and `ARG AMDGPU_VERSION` default values |
| `docker/Dockerfile.exporter-release` | `ADD ./libamd_smi.so.<SO_VERSION>` line and `ln -sf` symlink line (if .so version changed) |
| `dev.env` | `ROCM_TARBALL_URL` value (and the comment line above it with the date) |
| `Makefile` | `AMDSMI_BRANCH` (e.g. `release/therock-<VERSION>`) and `ROCM_VERSION` (format: `.yum_<X.Y.Z>`) |
| `assets/version.yaml` | `amd_smi_lib`, `rocprofiler`, `profiler_lib` branch/version fields |

```bash
# Verify no stray old version references remain
grep -rn "<OLD_VERSION>" docker/Dockerfile.exporter-release Makefile assets/version.yaml dev.env
```

### 4c. Copy assets and libamd_smi to DME

```bash
make gpuagent-asset-copy GPUAGENT_SRC_DIR=~/src/gpu-agent

# Copy new .so to docker/ for image build
cp ~/src/gpu-agent/sw/nic/third-party/rocm/amd_smi_lib/x86_64/lib/libamd_smi.so.<SO_VERSION> \
   docker/libamd_smi.so.<SO_VERSION>
```

### 4d. Update gpuagent submodule pointer

```bash
git -C gpuagent checkout <GPU_AGENT_COMMIT>

# If .so version bumped, remove the old .so file from docker/
# git rm docker/libamd_smi.so.<OLD_SO_VERSION>

git add gpuagent assets/ docker/libamd_smi.so.<SO_VERSION> \
  docker/Dockerfile.exporter-release Makefile assets/version.yaml dev.env
git commit -m "feat(docker): update ROCm version references to <VERSION>"
git push origin feature/rocm-<VERSION>-support
```

## Step 5: Build and validate

Use the **builder skill** (`/builder exporter docker`) — this is the recommended approach. The builder skill handles containerized builds and docker image creation automatically.

For the docker image, the Makefile target is:
```bash
make -C docker docker \
  TOP_DIR=$(pwd) \
  ROCM_VERSION=<VERSION> \
  AMDGPU_VERSION=<VERSION> \
  ROCM_TARBALL_URL=<TARBALL_URL> \
  EXPORTER_IMAGE=<REGISTRY>/device-metrics-exporter:collab-<VERSION>-1
```

### Quick smoke test (mock — no hardware needed)

Use `/builder mock` to build the mock docker image, then curl the endpoint:
```bash
curl -s localhost:5001/metrics | grep vram_max_bandwidth
# Expected: gpu_vram_max_bandwidth 3.2768e+06  PASS
```

### K8s e2e (real hardware — manual)

Run the DME standalone e2e suite:
```bash
docker run --rm \
  -v /tmp/kubeconfig.yaml:/kubeconfig:ro \
  -v /tmp/helm-charts:/helm-charts:ro \
  dme-k8s-e2e:latest \
  -kubeconfig /kubeconfig \
  -helmchart /helm-charts \
  -registry <REGISTRY>/device-metrics-exporter \
  -imagetag collab-<VERSION>-1 \
  -namespace dme-standalone-test \
  -platform k8s -test.timeout 60m -v
```

## Key Rules

- **Rebuild gpuagent** whenever amdsmi.h or libamd_smi.so content changes (even if .so version number is the same)
- **DCM changes are separate** — do not update DCM as part of this workflow
- **Check if .so version bumped** — if it does, update the `ADD ./libamd_smi.so.<X>` line in Dockerfile and the symlink
- **Sequential builds** — gpuagent → gim → mock → asset-copy → DME binary → Docker

## Reused Skills

- `/builder gpuagent` — builds gpuagent/gim/mock binaries
- `/builder exporter` — builds DME binary
- `/builder docker` — builds docker image

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `libamdsmi: undefined symbol` | Header/binary mismatch | Rebuild gpuagent against new amdsmi.h |
| `libamd_smi.so.26: not found` | Wrong .so name in Dockerfile | Check `ADD` line matches actual .so filename |
| Tarball URL 404 | Wrong date or version | Re-check BKC PDF for correct URL |
| `release/therock-<X>` branch missing | Branch not cut yet | Extract from tarball (Step 2c) |
| `GLIBC_2.38` symbols in .so | Ubuntu-built .so | Use RHEL9-built .so (build from source) |

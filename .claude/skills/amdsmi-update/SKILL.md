---
name: amdsmi-update
description: This skill should be used when updating amdsmi (libamd_smi.so) to a new version or source branch in the collab DME stack. Orchestrates: build amdsmi from source, update gpu-agent vendor tree, rebuild gpuagent binaries, update DME assets, rebuild docker images.
version: 1.0.0
---

# AMDSMI Update Workflow

Updates `libamd_smi.so` to a new version or source branch across the collab DME stack (gpu-agent + DME). This is distinct from the ROCm version update — it targets just the amdsmi library, typically tracking an internal `amd-npi` or `release/therock-X.Y` branch.

## When to Use This Skill

- New `libamd_smi.so` SO version (e.g. `26.3.0` → `26.4.0`)
- New commit on an existing branch (e.g. `amd-npi` advances)
- Switching amdsmi source repo (public `ROCm/rocm-systems` ↔ private `AMD-ROCm-Internal/rocm-systems`)
- New amdsmi API changes require header + binary update in gpu-agent

## Key Facts from Past Runs

- **SSH key** — only needed for private `AMD-ROCm-Internal/rocm-systems`. Public `ROCm/rocm-systems` needs no credentials.
- **Builder image**: use the appropriate platform builder (`amdsmi-builder:rhel9`, `:ub22`, `:ub24`) — build for each platform and update all three asset dirs
- **`DOCKER_API_VERSION=1.43`** required on this host (Docker client 1.52 > daemon max 1.43)
- **New runtime deps** introduced by amd-npi amdsmi 26.4.0: `libnl3`, `libnl-genl-3`, `libmnl` — must be in mock Dockerfile (Ubuntu 22.04 base); already present in release Dockerfile (UBI9 via RPM deps)

## Inputs Required

Ask the user for these before starting:

- **`DME_DIR`** — path to device-metrics-exporter checkout (e.g. `~/src/device-metrics-exporter`)
- **`GPUAGENT_DIR`** — path to gpu-agent checkout (e.g. `~/src/gpu-agent`)
- **Source repo** — `https://github.com/ROCm/rocm-systems.git` (public) or `git@github.com:AMD-ROCm-Internal/rocm-systems.git` (private, needs SSH)
- **Branch** — e.g. `amd-npi`, `release/therock-7.12`
- **Commit hash** — pin to a specific commit for reproducibility
  ```bash
  git ls-remote <REPO_URL> refs/heads/<BRANCH>
  ```
- **SO version** — determined after building (e.g. `26.4.0`); check `${DME_DIR}/libamdsmi/build/exporterout/libamd_smi.so.*`
- **Base DME branch** — e.g. `collab-2.0.0`, `main`
- **Base gpu-agent branch/commit** — check with user

## Architecture: Two-Repo Workflow

```
AMD-ROCm-Internal/rocm-systems  (or ROCm/rocm-systems)
        ↓  make amdsmi-compile-rhel (Step 1: build script clones + builds)
${DME_DIR}
  libamdsmi/build/exporterout/           ← .so + amdsmi.h (intermediate)
  assets/amd_smi_lib/x86_64/RHEL9/lib/  ← .so + amdsmi.h (committed)
        ↓  copy header + .so to gpu-agent vendor tree (Step 2)
${GPUAGENT_DIR}
  sw/nic/third-party/rocm/amd_smi_lib/  ← updated .so + .h
        ↓  rebuild gpuagent (Step 3)
  sw/nic/build/x86_64/sim/bin/          ← new gpuagent{,_gim,_mock}, gpuctl
        ↓  make gpuagent-asset-copy (Step 4)
${DME_DIR}
  assets/gpuagent_static.bin.gz          ← updated tarballs
        ↓  build_prep_docker.sh copies .so from assets/ automatically (Step 5)
  docker build → release + mock images
```

---

## Step 0: Create Branches

```bash
cd ${GPUAGENT_DIR}
git fetch origin
git checkout <BASE_GPU_AGENT_COMMIT_OR_BRANCH>
git checkout -b feature/amdsmi-<BRANCH_NAME>

cd ${DME_DIR}
git fetch pensando <BASE_DME_BRANCH>
git checkout pensando/<BASE_DME_BRANCH>
git checkout -b feature/amdsmi-<BRANCH_NAME>
```

---

## Step 1: Build amdsmi from Source

### 1a. Pin the commit
```bash
git ls-remote <REPO_URL> refs/heads/<BRANCH>
# → <COMMIT_HASH>
```

### 1b. Build builder image + compile amdsmi (make targets handle both)

For private repos requiring SSH, set `GIT_SSH_COMMAND` before running make:
```bash
export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new -i ~/.ssh/id_rsa_new"
```

**RHEL9 only (recommended — output is `GLIBC ≤ 2.34`, compatible with UBI9 release containers):**
```bash
cd ${DME_DIR}
DOCKER_API_VERSION=1.43 make amdsmi-compile-rhel \
  AMDSMI_BRANCH=<BRANCH> AMDSMI_COMMIT=<COMMIT_HASH>
```

`amdsmi-compile-rhel` builds the `amdsmi-builder:rhel9` image then runs the container and
**automatically copies** artifacts into `build/assets/RHEL9/exporterout/` and
`assets/amd_smi_lib/x86_64/RHEL9/lib/`.

> **Skip image rebuild** (if `amdsmi-builder:rhel9` already exists and unchanged):
> ```bash
> DOCKER_API_VERSION=1.43 OS=RHEL9 make amdsmi-compile \
>   AMDSMI_BRANCH=<BRANCH> AMDSMI_COMMIT=<COMMIT_HASH>
> ```

**All platforms (RHEL9 + Ubuntu 22 + Ubuntu 24):**
```bash
cd ${DME_DIR}
DOCKER_API_VERSION=1.43 make amdsmi-compile-all \
  AMDSMI_BRANCH=<BRANCH> AMDSMI_COMMIT=<COMMIT_HASH>
```

**Known Dockerfile.rhel9 requirements for amd-npi amdsmi** (may not be needed for public ROCm/rocm-systems):
- CentOS Stream `BaseOS` repo — for `libnl3` packages
- `libnl3`, `libnl3-devel`, `libmnl-devel` — new netlink deps in amd-npi CMakeLists
- `--exclude=openssl-fips-provider` on `dnf update` — avoids FIPS package conflict between CentOS Stream and UBI9

### 1c. Verify GLIBC cap
```bash
strings ${DME_DIR}/libamdsmi/build/exporterout/libamd_smi.so.<SO_VERSION> \
  | grep "GLIBC_2\." | sort -V | tail -3
# Must be ≤ GLIBC_2.34 for UBI9 compatibility
```

---

## Step 2: Update gpu-agent Vendor Tree

```bash
cd ${GPUAGENT_DIR}
SO_VERSION=<SO_VERSION>   # e.g. 26.4.0
# Source from assets/ (already populated by make amdsmi-compile-rhel in Step 1)
DME_ASSETS=${DME_DIR}/assets/amd_smi_lib/x86_64/RHEL9/lib

# Copy new .so
cp ${DME_ASSETS}/libamd_smi.so.${SO_VERSION} \
   sw/nic/third-party/rocm/amd_smi_lib/x86_64/lib/libamd_smi.so.${SO_VERSION}

# Update symlinks (must be in the lib dir for ln -sf)
(cd sw/nic/third-party/rocm/amd_smi_lib/x86_64/lib && \
  ln -sf libamd_smi.so.${SO_VERSION} libamd_smi.so.26 && \
  ln -sf libamd_smi.so.26 libamd_smi.so)

# Copy updated header
cp ${DME_ASSETS}/amdsmi.h \
   ${GPUAGENT_DIR}/sw/nic/third-party/rocm/amd_smi_lib/include/amd_smi/amdsmi.h

# Remove stale old .so if SO version changed
# git rm sw/nic/third-party/rocm/amd_smi_lib/x86_64/lib/libamd_smi.so.<OLD_SO_VERSION>

# Update version.txt
echo "<BRANCH>" > sw/nic/third-party/rocm/amd_smi_lib/version.txt

# Commit (from ${GPUAGENT_DIR})
git add sw/nic/third-party/rocm/amd_smi_lib/
git commit -m "feat: update amdsmi to <BRANCH> SO <SO_VERSION>"
git push origin feature/amdsmi-<BRANCH_NAME>
```

---

## Step 3: Rebuild gpuagent Binaries

Use the **builder skill** — it handles the correct RHEL9 container invocation and build ordering:

```
/builder gpuagent
```

Verify outputs:
```bash
ls -lh ${GPUAGENT_DIR}/sw/nic/build/x86_64/sim/bin/gpuagent{,_gim,_mock} \
       ${GPUAGENT_DIR}/sw/nic/build/x86_64/sim/bin/gpuctl
# Expected: ~60MB each (gpuagent*), ~14MB (gpuctl)
```

---

## Step 4: Update DME Assets

```bash
cd ${DME_DIR}

# Copy gpuagent tarballs
make gpuagent-asset-copy GPUAGENT_SRC_DIR=${GPUAGENT_DIR}

# amdsmi .so is already in assets/amd_smi_lib/ from Step 1.
# build_prep_docker.sh stages it into docker/ automatically at build time — no manual copy needed.

# Update gpuagent submodule pointer
git -C gpuagent checkout <GPU_AGENT_COMMIT>
```

Update the `amd_smi_lib` entry in `assets/version.yaml`:
```yaml
  - name    : amd_smi_lib
    version : <COMMIT_HASH>   # e.g. f09c4481
    branch  : <BRANCH>        # e.g. amd-npi
```

If the SO version changed, update `ADD ./libamd_smi.so.<NEW_SO>` and the `ln -sf` line in
`docker/Dockerfile.exporter-release`. For `Dockerfile.exporter-mock-release`, no `.so` ADD needed —
verify runtime deps (`libnl-3-200`, `libnl-genl-3-200`, `libmnl0`) if new netlink deps were added.

---

## Step 5: Build DME Binary and Docker Images

Use the **builder skill** for both — it handles containerized builds and staging automatically:

```
/builder exporter
/builder docker
```

`docker/build_prep_docker.sh` copies `libamd_smi.so.*` from `assets/amd_smi_lib/x86_64/$OS/lib/`
into `docker/` automatically — ensure the assets tree is updated (Step 4) before running.

---

## Step 6: Validate

### Mock smoke test (no hardware)
```bash
DOCKER_API_VERSION=1.43 docker run -d --name dme-amdsmi-test \
  -p 5099:5000 dme-amdsmi-<BRANCH_NAME>:mock

sleep 15
curl -s localhost:5099/metrics | grep vram_max_bandwidth
# Expected: gpu_vram_max_bandwidth{...} 3.2768e+06

DOCKER_API_VERSION=1.43 docker rm -f dme-amdsmi-test
```

### Real hardware test

Run on a machine with AMD GPU hardware available. Stop the system gpuagent first to avoid port
conflict, then run the release image with `/dev/kfd` and `/dev/dri/renderD*` devices passed in.

---

## Commit and Push DME

```bash
cd ${DME_DIR}
# docker/libamd_smi.so.* are NOT committed — build_prep_docker.sh stages them
# from assets/ at build time and build_post_docker.sh removes them after.
git add gpuagent assets/ docker/Dockerfile.exporter-release
git commit -m "feat: update amdsmi to <BRANCH> SO <SO_VERSION>"
git push origin feature/amdsmi-<BRANCH_NAME>
```

---

## Key Rules

- **Always rebuild gpuagent** when amdsmi.h or libamd_smi.so content changes, even if SO version number is unchanged
- **All platforms** — update assets for RHEL9, UBUNTU22, and UBUNTU24; use the matching builder image for each
- **Check runtime deps** — if new amdsmi adds netlink/other libs, add them to `Dockerfile.exporter-mock-release` (Ubuntu base); release Dockerfile (UBI9) usually gets them via `amd-smi-lib` RPM
- **SO version bump** — update `ADD ./libamd_smi.so.<NEW_SO>` and `ln -sf` in `docker/Dockerfile.exporter-release`; `docker/libamd_smi.so.*` files are untracked (staged by `build_prep_docker.sh` at build time, not committed)
- **Private repo SSH** — `AMD-ROCm-Internal/rocm-systems` requires an AMD engineering GitHub SSH key; test with `ssh -T git@github.com`
- **Commit hash pinning** — always pin to a specific commit in Makefile/version.yaml for reproducibility

## Troubleshooting

| Symptom | Cause | Fix |
|---|---|---|
| `libamdsmi: undefined symbol` | Header/binary mismatch | Rebuild gpuagent against new amdsmi.h |
| `GLIBC_2.38` in .so | Ubuntu-built binary incompatible with UBI9 | Switch to RHEL9 builder (`amdsmi-builder:rhel9`) |
| `libnl-3.so.200: not found` in mock container | New netlink dep not in Dockerfile | Add `libnl-3-200 libnl-genl-3-200 libmnl0` to mock Dockerfile |
| SSH `Permission denied` in builder container (private repo only) | Wrong SSH key or key not mounted | `export GIT_SSH_COMMAND="ssh -o StrictHostKeyChecking=accept-new -i ~/.ssh/id_rsa_new"` |
| `openssl-fips-provider` conflict on dnf update | CentOS Stream vs UBI9 FIPS conflict | Add `--exclude=openssl-fips-provider` to `dnf update` in Dockerfile.rhel9 |
| `unrecognized image format` on `k3s ctr import` | Docker API version mismatch | Use `DOCKER_API_VERSION=1.43 docker save ...` |

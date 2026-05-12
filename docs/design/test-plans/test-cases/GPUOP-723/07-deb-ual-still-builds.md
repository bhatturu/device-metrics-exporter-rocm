---
topology: single-node
timeout: 1800
pass_criteria: "make libcopy-assets-UBUNTU22 debpkg-ual produces amdgpuifoe-exporter_22.04_amd64.deb (matches .job.yml target build-ifoe-debian-package-ub22.04)"
stability: stable
retries: 0
validation_groups: [post-test]
---

# Deb UAL package still builds after asset switch (Ubuntu 22.04 + 24.04)

## Purpose

Regression check — the universal switch from mock to real gpuagent (S1) must not break the existing IFOE deb builds. Mirror the CI targets `build-ifoe-debian-package-ub22.04` and `build-ifoe-debian-package-ub24.04` from `.job.yml:24-40` so the test catches the same failures CI would.

## Category

regression

## Prerequisites

- Build host with Docker access (libcopy-assets runs in a container)
- `assets/gpuagent_ual.bin.gz` already updated by S1 (real gpuagent path)
- Per project memory `project_debpkg_build_gotchas.md`: `debpkg-ual` is host-only but gated on `libcopy-assets-UBUNTU22`/`24` output; libcopy targets need a TTY when run via `docker run`. Failed runs may leave 3 transient sed scaffolding files dirty in `debian/` — those are NOT real edits.

## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `UBUNTU_VARIANTS` | `22 24` | which Ubuntu variants to build (space-separated) |

## Steps

1. **[host]** **Ubuntu 22.04 build (mirrors .job.yml target `build-ifoe-debian-package-ub22.04`):**
   `make libcopy-assets-UBUNTU22 debpkg-ual 2>&1 | tee /tmp/GPUOP-723-debpkg-ual-22.log`
   Expected: exit code 0; `bin/amdgpuifoe-exporter_22.04_amd64.deb` produced
2. **[host]** `ls -la bin/amdgpuifoe-exporter_22.04_amd64.deb`
   Expected: file present, recent mtime, non-zero size (matches `.job.yml:31` artifact)
3. **[host]** `dpkg-deb --contents bin/amdgpuifoe-exporter_22.04_amd64.deb | grep -E '(gpuagent|amd-metrics-exporter)'`
   Expected: both binaries listed in the package
4. **[host]** `dpkg-deb --info bin/amdgpuifoe-exporter_22.04_amd64.deb | grep '^Package:'`
   Expected: `Package: amdgpuifoe-exporter` (matches `Makefile.package:268` rename)
5. **[host]** **Ubuntu 24.04 build (mirrors .job.yml target `build-ifoe-debian-package-ub24.04`):**
   `make UBUNTU_VERSION=noble libcopy-assets-UBUNTU24 debpkg-ual 2>&1 | tee /tmp/GPUOP-723-debpkg-ual-24.log`
   Expected: exit code 0; `bin/amdgpuifoe-exporter_24.04_amd64.deb` produced
6. **[host]** `ls -la bin/amdgpuifoe-exporter_24.04_amd64.deb`
   Expected: file present, matches `.job.yml:40` artifact
7. **[host]** `git status --short debian/`
   Expected: clean OR only the 3 known-transient sed scaffolding files (per memory `project_debpkg_build_gotchas.md` — those are NOT real edits and can be discarded)

## Expected Result

Both Ubuntu 22.04 and 24.04 IFOE deb packages build cleanly with the `libcopy-assets-*` prerequisite, contain the renamed `amdgpuifoe-exporter` package + both binaries, matching the artifact contracts in `.job.yml`.

## Failure Indicators

- libcopy-assets-UBUNTU22 / 24 fails (TTY issue per memory — escalate to manual `docker run -t`)
- `make debpkg-ual` build error related to missing/empty `assets/gpuagent_ual.bin.gz`
- Package missing gpuagent binary
- Package name still `amdgpu-exporter` (rename at Makefile.package:268 broken)
- `debian/` dirty with files NOT in the known-transient list

## Cleanup

- `rm -f bin/amdgpuifoe-exporter_22.04_amd64.deb bin/amdgpuifoe-exporter_24.04_amd64.deb` (only if rebuilding)
- `git checkout -- debian/` if known-transient sed files dirty

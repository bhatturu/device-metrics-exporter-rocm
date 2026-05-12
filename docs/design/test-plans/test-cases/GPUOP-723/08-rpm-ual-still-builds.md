---
topology: single-node
timeout: 1800
pass_criteria: "make libcopy-assets-RHEL9 rpmpkg-ual produces amdgpuifoe-exporter-rhel9.x86_64.rpm (matches .job.yml target build-rpm-package-rhel9)"
stability: stable
retries: 0
validation_groups: [post-test]
---

# RPM UAL package still builds after asset switch (RHEL9)

## Purpose

Regression check matching TC07 but for the RPM build path. Mirror `.job.yml:49-58` target `build-rpm-package-rhel9` (which builds `rpmpkg`, `rpmpkg-sriov`, AND `rpmpkg-ual` together with shared `libcopy-assets-RHEL9` prerequisite). For this Story we focus on the IFOE RPM artifact.

## Category

regression

## Prerequisites

- Build host with Docker access (`libcopy-assets-RHEL9` runs in a container)
- `assets/gpuagent_ual.bin.gz` updated by S1
- Per project memory `project_rpmpkg_ual_build_gotchas.md`: `libcopy-assets-RHEL9` has the same TTY trap as the deb path — manual `docker run -t` workaround documented; PATH+GOPATH must be exported for `make rpmpkg-ual` inside the build container; profilerlibs must be staged into `build/assets/RHEL9/profilerlibs/` first (use the project's `build-ual-rpm` skill end-to-end if running from scratch).

## Parameters

(no test-specific parameters; `.job.yml` target uses fixed RHEL9 paths)

## Steps

1. **[host]** **Mirror .job.yml target `build-rpm-package-rhel9` (just the IFOE component):**
   `make libcopy-assets-RHEL9 rpmpkg-ual 2>&1 | tee /tmp/GPUOP-723-rpmpkg-ual.log`
   Expected: exit code 0; RPM file path printed in log
2. **[host]** `find bin/ -name 'amdgpuifoe-exporter*.rpm' -newer /tmp/GPUOP-723-rpmpkg-ual.log`
   Expected: at least one RPM file (matches `.job.yml:58` artifact: `bin/amdgpuifoe-exporter-rhel9.x86_64.rpm`)
3. **[host]** `rpm -qpl bin/amdgpuifoe-exporter-rhel9.x86_64.rpm | grep -E '(gpuagent|amd-metrics-exporter)'`
   Expected: both binaries listed
4. **[host]** `rpm -qpi bin/amdgpuifoe-exporter-rhel9.x86_64.rpm | grep -E '^Name'`
   Expected: `Name        : amdgpuifoe-exporter`
5. **[host]** Optional: re-run the full CI target form to confirm sibling RPMs still build (`make libcopy-assets-RHEL9 rpmpkg rpmpkg-sriov rpmpkg-ual`); checks that the asset switch doesn't break adjacent (non-IFOE) RPM artifacts
   Expected: all three artifacts produced (`amdgpu-exporter-rhel9.x86_64.rpm`, `amdgpu-exporter-sriov-rhel9.x86_64.rpm`, `amdgpuifoe-exporter-rhel9.x86_64.rpm`)

## Expected Result

IFOE RPM builds cleanly with the `libcopy-assets-RHEL9` prerequisite. Sibling non-IFOE RPMs (rpmpkg, rpmpkg-sriov) also build, confirming the asset-path switch (S1) doesn't break adjacent CI artifacts.

## Failure Indicators

- libcopy-assets-RHEL9 fails (TTY issue per memory — manual `docker run -t` workaround)
- `make rpmpkg-ual` error about missing/empty asset
- RPM missing gpuagent binary
- RPM package name not `amdgpuifoe-exporter`
- Sibling RPMs (`rpmpkg`, `rpmpkg-sriov`) regress

## Cleanup

- `rm /tmp/GPUOP-723-rpmpkg-ual.log`
- Keep RPM artifacts for manual install validation if needed

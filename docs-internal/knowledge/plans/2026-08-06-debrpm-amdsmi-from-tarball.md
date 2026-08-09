# deb/rpm amd_smi from tarball (GPUOP-1049 follow-on)

**Date:** 2026-08-06
**JIRA:** GPUOP-1050 (Story GPUOP-1043)
**Status:** implemented.

## Problem

With `AMDSMI_FROM_TARBALL=1`, the deb/rpm package targets still copy `libamd_smi.so*`
from the committed `assets/amd_smi_lib/x86_64/<OS>/lib/` tree (`Makefile.package:88` rpm,
`:229` deb), which is stale SO 26.5.0 (7.14-era) — while the docker image ships SO 27 from
the tarball. Packages mismatch the image, and the knob is ignored for deb/rpm libs.

Observed: `make docker` deb phase prints `Copy prebuilt libraries ... assets/amd_smi_lib/.../libamd_smi.so.26.5.0 -> amdgpu-exporter-1.5.2-0/lib/...`.

## Fix (deb/rpm only; committed-asset deletion deferred by user)

1. **Makefile.compile `amdsmi-from-tarball`**: extract from the once-downloaded
   `$(ROCM_TARBALL_PATH)` (build/rocm-tarball/therock.tar.gz) instead of its own
   `curl -fSL "$(ROCM_TARBALL_URL)"` (line 77). Add `$(ROCM_TARBALL_DEP)` as a prereq.
   Reuses the GPUOP-1049 single download.

2. **Makefile.package**: introduce `AMDSMI_LIBS_SRC_RHEL9` / `AMDSMI_LIBS_SRC` that resolve to
   the tarball-staged dir (`$(AMDSMI_TARBALL_STAGE)/lib` + include) when
   `AMDSMI_FROM_TARBALL=1`, else the committed `GPUAGENT_LIBS*`. Repoint the `cp -rvf` at
   `:88` (rpm) and `:229` (deb) to it. Make `rpmpkg`/`debpkg` depend on `amdsmi-from-tarball`
   in tarball mode.
   - Note: sysdeps handling — 10.0 tarball still bundles `librocm_sysdeps_*` (SO 27), so
     copy those too (already extracted by amdsmi-from-tarball's wildcard).
   - Escape hatch `=0`: keep copying from committed assets (unchanged).

3. **Deferred (NOT now):** deleting committed `assets/amd_smi_lib/` binaries. Still consumed
   in tarball mode by the gpuagent producer COPY-staging (`Makefile:360-362`) and
   `build_prep_docker.sh:80,100`. Remove only after those are also repointed to the tarball
   stage (separate follow-up).

## Consumers of assets/amd_smi_lib (for the deferred deletion)

- Makefile.package:88 (rpm copy), :229 (deb copy)  <- fixed in this plan
- Makefile:360-362 (producer stages amdsmi.h/.so into docker/ context)  <- deferred
- docker/build_prep_docker.sh:80,100  <- deferred
- Makefile.compile:34-36 (amdsmi-compile writer, =0 escape hatch)
- Makefile.compile:92,99 (amdsmi-sync-assets writer)

## Verify

- `make debpkg AMDSMI_FROM_TARBALL=1` copies SO 27 from build/amdsmi-from-tarball, not
  assets/. `AMDSMI_FROM_TARBALL=0` still copies committed assets.
- No second 9GB download (reuses build/rocm-tarball).

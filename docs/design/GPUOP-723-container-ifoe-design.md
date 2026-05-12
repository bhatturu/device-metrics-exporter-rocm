---
jira_id: GPUOP-723
story_summary: >-
  Container exposes IFOE field exporter metrics by default on capable hardware
epic_id: GPUOP-722
author: praveenkumar.shanmugam@amd.com
status: draft
created: 2026-05-10
---

# Design: GPUOP-723 — Container exposes IFOE field exporter metrics by default on capable hardware

## Problem Statement

UAL field exporter ships only as deb (`amdgpuifoe-exporter`) and rpm today. Customers running container-based deployments cannot scrape IFOE port/station metrics. The released `amd-device-metrics-exporter` container (built via `make docker` → `docker/Dockerfile.exporter-release`) ships a mock gpuagent and lacks the runtime libraries IFOE needs (libmnl, libnl3). Target branch: `bugfix/exporter-ifoe` off `collab-2.0.0` (the unannounced IFOE-only release line). Parent Epic: GPUOP-722.

## Solution Overview

Single-image strategy: the existing `amd-device-metrics-exporter` container is extended to ship the real (non-mock) UAL gpuagent, the IFOE-required runtime libs, and `ENABLE_IFOE=true` as a default ENV. The entrypoint reads `ENABLE_IFOE` and passes `-monitor-ifoe=$ENABLE_IFOE` to `amd-metrics-exporter`. Capability detection lives in Go (`pkg/amdgpu/gpuagent/gpuagent_ifoe.go`) — when no IFOE-capable devices are present, the IFOE client logs a structured "IFOE disabled" message and publishes no IFOE metrics. The shell never probes hardware.

The asset pipeline (`scripts/update_ual_assets.sh`) switches universally from the `mock/` subpath to the real gpuagent path; deb, rpm, and container all benefit. The bundled `libamd_smi` is bumped to **26.3.0** (sourced from `upstream/collab-7.12` directly — already present on that branch, no dependency on PR #1315) — RHEL9 only, since `Dockerfile.exporter-release` is UBI9-based.

Trade-offs: a single image (vs split SKU) avoids two-image confusion for ops, mirrors the customer expectation of one container, and leverages the Go capability gate to make IFOE invisible on non-IFOE hosts. A two-image split (mirroring the deb `amdgpu-exporter` vs `amdgpuifoe-exporter` naming) was considered and rejected — adds CI/release surface for no functional benefit when the Go gate already gracefully degrades.

## Build Targets

<!-- Discovered via find . -name 'Makefile' / 'go.mod' filtered to repo-relevant entries. -->

| Build Target | Type | Build File |
|-------------|------|------------|
| `docker` | container image | `Makefile:403` (top-level) → `docker/Makefile` (Dockerfile.exporter-release) |
| `amdexporter` | go binary (`amd-metrics-exporter`) | `Makefile:370` → `Makefile.compile` |
| `metricsclient` | go binary | `Makefile:388` |
| `amdgpuhealth` | go binary | `Makefile:392` |
| `update-ual-assets` | bash script | `scripts/update_ual_assets.sh` |
| `gpuagent_ifoe` (Go package) | go package (within `amdexporter`) | `pkg/amdgpu/gpuagent/` (no separate Makefile; built as part of `amdexporter`) |

Sub-task creation (via `/codie:task-create`) maps to the `docker` build target plus the supporting script and Go-package code paths.

## Code Path Design

### Code Path: ual-asset-fetch

**Build target:** `update-ual-assets` (bash script)

**Responsibility:** Fetch UAL gpuagent + gpuctl artifacts from the internal build server and stage them under `assets/` for downstream consumers (deb, rpm, container).

**Interfaces:**
- CLI: `./scripts/update_ual_assets.sh <version>` (e.g. `1.127.0-116`)
- Env: `UAL_REMOTE_SERVER` (defaults to `remote_fqdn`)
- Outputs: `assets/gpuagent_ual.bin.gz`, `assets/gpuctl_ual`, `assets/version.yaml`

**Dependencies:** `scp` to internal build server; `tar`, `strip`, `sed`.

**Implementation Notes:**
- Line 56 currently constructs `REMOTE_GPUAGENT_PATH=${REMOTE_BASE_PATH}/${VERSION}/rudra-bundle/internal-artifacts/nodemgmt/mock/gpuagent_${VERSION}.tar.gz`.
- Change: drop the `mock/` segment so the path becomes `${REMOTE_BASE_PATH}/${VERSION}/rudra-bundle/internal-artifacts/nodemgmt/gpuagent_${VERSION}.tar.gz`.
- All downstream targets (`debpkg-ual` in Makefile.package:264, `rpmpkg-ual` in Makefile.package:178, `docker` in Makefile:403) consume the same `assets/gpuagent_ual.bin.gz` so the switch is universal.

**Error Handling:**
- Existing script has `set -euo pipefail` and explicit `log_error` + `exit 1` on scp failure. No change needed.
- Add a parity check (see Code Path: amdsmi-parity-check) that runs after extraction and warns or fails if the gpuagent's required amd-smi version mismatches the bundled `libamd_smi`.

### Code Path: amdsmi-parity-check

**Build target:** `update-ual-assets` (extension of the same script)

**Responsibility:** After fetching the UAL gpuagent, detect whether the gpuagent's required amd-smi version matches the `libamd_smi.so.*` currently bundled at `assets/amd_smi_lib/x86_64/RHEL9/lib/`. Document the escalation path on mismatch.

**Interfaces:**
- Inputs: extracted gpuagent binary or its tarball manifest; `assets/amd_smi_lib/x86_64/RHEL9/lib/libamd_smi.so.*` symlink.
- Outputs: log line + nonzero exit on hard mismatch (with override flag for known-good drift).

**Dependencies:** None new — bash + standard tools (`readelf`, `strings`, `ls -l`).

**Implementation Notes:**
- **Spike required first**: confirm how the gpuagent tarball exposes its amd-smi build version. Candidate sources, in priority order: a manifest/version file inside the tarball; `gpuagent --version` output; ELF SONAME of the linked `libamd_smi`; a hardcoded mapping table in the script (last resort).
- Once the source is known, encode the parity check as a small bash function called from `update_ual_assets.sh` after gpuagent extraction.
- On mismatch: print which versions diverge and an escalation hint (file an internal ticket, contact UAL release engineering); exit 1 unless `UAL_ALLOW_AMDSMI_DRIFT=1` is set in env.

**Error Handling:**
- Spike failure (no parity source identifiable) → log a warning, document the limitation in `docs/design/GPUOP-723-container-ifoe-design.md`, and emit only an informational log (no fail) until the source is established.

### Code Path: container-runtime-libs

**Build target:** `docker` (Dockerfile.exporter-release)

**Responsibility:** Install the IFOE-required runtime libraries into the UBI9 container image so the UAL gpuagent + `amd-metrics-exporter` IFOE codepath can dlopen them.

**Interfaces:**
- Dockerfile `microdnf install` line at `docker/Dockerfile.exporter-release:29-31`.

**Dependencies:** UBI9 base image microdnf repos (already configured for AMDGPU + ROCm). No new repos.

**Implementation Notes:**
- Add `libmnl`, `libnl3`, and `libnl3-cli` (RHEL equivalents of Debian `libmnl-dev`, `libnl-3-dev`, `libnl-genl-3-dev`).
- Confirm via container shell after build that `ldconfig -p | grep -E 'libmnl|libnl-3|libnl-genl'` returns expected entries.

**Error Handling:**
- Build fails fast if any package is unavailable — caught by CI.

### Code Path: container-libamd-smi-bundle

**Build target:** `docker` (Dockerfile.exporter-release + bundled assets)

**Responsibility:** Bundle libamd_smi **26.3.0** into the container image, replacing the current 26.2.1 bundle. Keep header, drm libs, and version symlinks in sync.

**Interfaces:**
- Files (copied from `upstream/collab-7.12` directly — already present on that branch):
  - `assets/amd_smi_lib/x86_64/RHEL9/lib/libamd_smi.so.26.3.0`
  - `assets/amd_smi_lib/x86_64/RHEL9/lib/libamd_smi.so.26` (symlink → 26.3.0)
  - `assets/amd_smi_lib/x86_64/RHEL9/lib/amdsmi.h`
  - `assets/amd_smi_lib/x86_64/RHEL9/lib/libdrm*.so*`
  - `docker/libamd_smi.so.26.3.0` (build-context file; the Dockerfile ADD source)
  - `docker/libamd_smi.so.26` (symlink → 26.3.0)
- Dockerfile changes at `docker/Dockerfile.exporter-release:42-43`: ADD line and symlink target updated to `26.3.0`.
- `dev.env`: bump ROCM_VERSION only if collab-7.12 differs from current main; verify during impl.

**Dependencies:** None — files exist on `upstream/collab-7.12` already; no PR #1315 dependency.

**Implementation Notes:**
- Pin to the specific commit SHA on `upstream/collab-7.12` in the commit message for traceability.
- RHEL9-only scope. Do not copy UBUNTU22/UBUNTU24 trees in this Story (Dockerfile.exporter-release is UBI9; other variants out of scope per Epic).

**Error Handling:**
- Parity check (Code Path: amdsmi-parity-check) catches any future drift between the bundled lib and gpuagent's required amd-smi version.

### Code Path: container-entrypoint-ifoe

**Build target:** `docker` (Dockerfile.exporter-release entrypoint)

**Responsibility:** Honor `ENABLE_IFOE` env var and pass `-monitor-ifoe=$ENABLE_IFOE` to the `amd-metrics-exporter` binary. Preserve existing `MONITOR_GPU` behavior so customers running plain GPU monitoring see no change.

**Interfaces:**
- Env: `ENABLE_IFOE` (default `true`, override `false`)
- File: `entrypoint.sh` (root, copied to `/home/amd/tools/entrypoint.sh` per Dockerfile.exporter-release:51)
- Downstream: `/home/amd/bin/server` (== `amd-metrics-exporter`)

**Dependencies:** Existing `MONITOR_GPU` arg parsing in entrypoint.sh.

**Implementation Notes:**
- Read `ENABLE_IFOE` from env; default to `true` if unset.
- Append `-monitor-ifoe=$ENABLE_IFOE` to the exec arg list before calling `/home/amd/bin/server`.
- Keep the existing arg-loop for `-monitor-gpu` so an explicit user-supplied flag continues to win over the env-derived default.
- Add `ENV ENABLE_IFOE=true` to `Dockerfile.exporter-release` before the ENTRYPOINT line.

**Error Handling:**
- Invalid `ENABLE_IFOE` value (anything other than `true`/`false`): default to `true` and log a warning. Do not abort startup — the Go-side capability gate is the safety net.

### Code Path: ifoe-go-capability-gate

**Build target:** `amdexporter` (Go package `pkg/amdgpu/gpuagent`)

**Responsibility:** When `-monitor-ifoe=true` is passed but the host has no IFOE-capable devices (gpuagent reports zero IFOE devices over UAL gRPC), the IFOE client must:
1. Start cleanly (no panic, no fatal error).
2. Log a single structured "IFOE disabled" message including the reason (e.g., "no IFOE devices reported by gpuagent").
3. Publish no IFOE metric series.
4. Not retry forever — accept the no-IFOE state for the lifetime of the process.

**Interfaces:**
- Go: `GPUAgentIFOEClient` in `pkg/amdgpu/gpuagent/gpuagent_ifoe.go`
- gRPC: `amdgpu.UALSvcClient` (capability discovery RPC — exact name TBD during impl, likely `GetIFOEDevices` or similar)
- Logger: `pkg/exporter/logger`

**Dependencies:** UAL gRPC connection (`gpuHandler.GetGRPCConnection()`).

**Implementation Notes:**
- Inspect existing `InitClients()` and metric-collection loop to confirm whether the no-capability path is already graceful. If yes, add only a structured log line at startup. If no, add an early-exit branch that disables the IFOE collector for the process lifetime.
- The structured log MUST be emitted exactly once per process — repeated logs spam customer dashboards.

**Error Handling:**
- gRPC connection failure (gpuagent not running) is a separate failure mode handled by existing GPUAgentClient retry logic — no change.
- Genuine capability-discovery RPC error (vs zero-result) → log error, retry with backoff up to N times, then disable IFOE for process lifetime with a distinct error log.

## Cross-Target Interfaces

- **Container ↔ entrypoint ↔ exporter binary:** ENV `ENABLE_IFOE` flows from Dockerfile → entrypoint shell → CLI flag `-monitor-ifoe=` on the Go binary.
- **Exporter ↔ gpuagent:** existing UAL gRPC over Unix socket `/var/run/gpuagent.sock`. No protocol change.
- **Asset script ↔ container build:** `assets/gpuagent_ual.bin.gz` is the contract — script writes, Dockerfile (via `docker/Makefile`) reads.
- **libamd_smi ↔ gpuagent:** runtime LD_PRELOAD at entrypoint startup (`LD_PRELOAD=/home/amd/lib/libamd_smi.so.26 /home/amd/bin/gpuagent ...`). Symlink `libamd_smi.so.26 → libamd_smi.so.26.4.0` is the indirection point.

## Testing Strategy

| Category | What to Test |
|----------|-------------|
| Functional | (a) On real IFOE-capable hardware: container starts with `ENABLE_IFOE=true`, Prometheus scrape returns IFOE port/station metric series defined in `pkg/amdgpu/gpuagent/gpuagent_ifoe_metrics.go`. (b) `make docker` build succeeds end-to-end. |
| Negative | (a) On non-IFOE host (test rig 10.30.60.190, 1-GPU AMD, no Pensando NIC): container starts with `ENABLE_IFOE=true`, logs "IFOE disabled" exactly once, no panics, no IFOE series in `/metrics`. (b) `ENABLE_IFOE=false` → IFOE codepath never invoked, plain GPU monitoring works. (c) Invalid `ENABLE_IFOE=garbage` → defaults to true with warn log, container still starts. |
| Regression | Deb (`amdgpuifoe-exporter`) and rpm IFOE packages still build green after `update_ual_assets.sh` mock-path drop. Existing `amd-device-metrics-exporter` deployments on non-IFOE hosts see no degraded GPU metrics. |
| Build / CI | `make docker` produces a pushable image; image push to test registry succeeds; image size delta reasonable (libmnl+libnl3 are small, libamd_smi 26.4.0 is similar size to 26.2.1). |
| Spike | Identify gpuagent ↔ amd-smi version parity source (manifest? `--version`? SONAME?). Outcome feeds into `amdsmi-parity-check` code-path implementation. |

## Open Questions

- **Spike: amd-smi parity source** — how does the gpuagent tarball expose its required amd-smi version? Resolution: spike Sub-task before parity-check implementation. Fallback: warn-only mode until source is established.
- **gpuagent IFOE-mode args** — does the gpuagent binary need new CLI args when running in IFOE mode, or is the same `gpuagent -s /var/run/gpuagent.sock` invocation sufficient and IFOE behavior is purely controlled by the exporter-side `-monitor-ifoe` flag? Verify during entrypoint Sub-task implementation.
- **gpuagent ↔ 26.3.0 compatibility** — confirm the UAL gpuagent (1.127.0-116) was built against amd-smi 26.3.0 (or backwards-compatible). If the gpuagent in production deb/rpm packages already runs against 26.3.0 successfully, this is implicitly validated.

## Source references

- Epic charter: `docs/design/container-ifoe-epic.md`
- Current Dockerfile: `docker/Dockerfile.exporter-release`
- Current entrypoint: `entrypoint.sh` (root)
- Current asset script: `scripts/update_ual_assets.sh:56`
- IFOE flag wiring: `cmd/exporter/main.go:62`
- Existing IFOE Go client: `pkg/amdgpu/gpuagent/gpuagent_ifoe.go`
- Deb/rpm IFOE precedent: `Makefile.package:181,266`
- Top-level docker target: `Makefile:403`
- libamd_smi 26.3.0 source: `upstream/collab-7.12` branch, paths under `assets/amd_smi_lib/x86_64/RHEL9/lib/` (already committed there)
- Test rig: 10.30.60.190 (1-GPU AMD, no IFOE — for negative regression)

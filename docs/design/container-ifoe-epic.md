# Epic: Container support for IFOE field exporter (UAL)

**JIRA project**: GPUOP (pensando.atlassian.net)
**Branch**: `bugfix/exporter-ifoe` (off `collab-2.0.0`)
**Assignee**: praveenkumar.shanmugam@amd.com

## Problem

UAL field exporter ships only as deb/rpm today. Customers running container-based deployments cannot scrape IFOE port/station metrics. `collab-2.0.0` is the unannounced IFOE-only release line; this work targets it.

## Goal

Extend `docker/Dockerfile.exporter-release` so the released container exposes IFOE metrics by default on capable hardware, gracefully no-ops on non-capable hosts, and uses an amd-smi library version that matches the bundled UAL gpuagent.

## Out of scope

- Other Dockerfile variants (`azure`, `sriov`, `ainic`, `ub22`).
- Deb/rpm packaging changes beyond verifying they don't regress when the asset path switches from `mock/` to non-mock.

## Decisions

| # | Decision | Rationale |
|---|---|---|
| D1 | Single image with `ENABLE_IFOE=true` ENV default (overridable) | Avoids two-image SKU split; mirrors customer expectation of one container |
| D2 | Capability gate lives in Go (`pkg/amdgpu/gpuagent/gpuagent_ifoe.go`), not entrypoint shell | gpuagent is the authoritative source of device capability; bash probes are brittle |
| D3 | `update_ual_assets.sh` switches from `mock/` to non-mock path universally (deb/rpm/container all benefit) | Mock path is legacy; PR #1297 already exposes real IFOE metrics in production |
| D4 | libamd_smi **26.3.0** copied from `upstream/collab-7.12` directly (RHEL9 only) | Already present on collab-7.12; no dependency on PR #1315; RHEL9-only minimizes blast radius since Dockerfile.exporter-release is UBI9 |
| D5 | Acceptance = CI image build + push **and** real-HW smoke showing IFOE port/station metrics scraped via Prometheus | Real-HW is the only proof of end-to-end value |

## Sub-task decomposition

| # | Sub-task | Notes |
|---|---|---|
| S1 | `scripts/update_ual_assets.sh`: drop `mock/` from `REMOTE_GPUAGENT_PATH` (line 56) | universal switch |
| S2 | Copy libamd_smi **26.3.0** RHEL9 assets from `upstream/collab-7.12` into `bugfix/exporter-ifoe`. Files: `assets/amd_smi_lib/x86_64/RHEL9/lib/{libamd_smi.so.26.3.0,libamd_smi.so.26 symlink,amdsmi.h,libdrm*.so*}`, `docker/libamd_smi.so.26.3.0` (+ symlink), `docker/Dockerfile.exporter-release` ADD-line update, `dev.env` ROCM_VERSION bump if needed | no PR #1315 dependency |
| S3 | Add `libmnl`, `libnl3`, `libnl3-cli` to `microdnf install` in `docker/Dockerfile.exporter-release` (line 29-31) | RHEL equivalents of Debian `libmnl-dev/libnl-3-dev/libnl-genl-3-dev` |
| S4 | Update root `entrypoint.sh`: read `ENABLE_IFOE` env var, pass `-monitor-ifoe=$ENABLE_IFOE` to amd-metrics-exporter; preserve existing `-monitor-gpu` flow | gpuagent invocation may need new args for IFOE mode — verify during impl |
| S5 | Set `ENV ENABLE_IFOE=true` in `docker/Dockerfile.exporter-release` | default-on |
| S6 | Verify/harden Go IFOE no-capability path in `pkg/amdgpu/gpuagent/gpuagent_ifoe.go`: graceful no-op + structured warn log when zero IFOE devices found | gates whole feature for non-IFOE hosts |
| S7 | amd-smi 26.4.0 ↔ gpuagent 1.127.0-116 parity verification: detect mismatch (manifest in tarball or `gpuagent --version`), document escalation path | spike may be needed |
| S8 | CI: container image build + push to test registry (reuse existing make targets where possible) | |
| S9 | Smoke test on real IFOE-capable HW: deploy container, validate Prometheus scrape of IFOE port/station metrics | acceptance gate |
| S10 | Negative regression: `ENABLE_IFOE=false` on non-IFOE host → clean start, no panics, no IFOE metrics, structured "IFOE disabled" log | guards existing customers |
| S11 | Docs: README + `docs/configuration/ifoe-metricslist.md` updates for container deployment & `ENABLE_IFOE` env | |

## Open risks

- **S7**: Unknown how gpuagent tarball exposes its required amd-smi version. Spike-style sub-task may be needed before parity logic can be coded.
- **S2**: 26.3.0 is the version already on `upstream/collab-7.12`. Pin to the specific SHA on collab-7.12 in the commit message for traceability.

## Source references

- Current Dockerfile: `docker/Dockerfile.exporter-release`
- Current entrypoint: `entrypoint.sh` (root, used by Dockerfile.exporter-release)
- Current asset script: `scripts/update_ual_assets.sh`
- IFOE flag wiring: `cmd/exporter/main.go:62`
- Existing IFOE Go client: `pkg/amdgpu/gpuagent/gpuagent_ifoe.go`
- Deb/rpm IFOE precedent: `Makefile.package:181,266`
- amd-smi 26.3.0 source: `upstream/collab-7.12` branch, paths under `assets/amd_smi_lib/x86_64/RHEL9/lib/`

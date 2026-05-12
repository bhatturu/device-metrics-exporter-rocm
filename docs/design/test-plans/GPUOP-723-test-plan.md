---
jira_id: GPUOP-723
epic_id: GPUOP-722
design_doc: docs/design/GPUOP-723-container-ifoe-design.md
status: draft
created: 2026-05-11
template: feature-test-plan
---

# Test Plan: GPUOP-723 — Container exposes IFOE field exporter metrics by default on capable hardware

## Source

- **JIRA Story:** [GPUOP-723](https://pensando.atlassian.net/browse/GPUOP-723)
- **Epic:** [GPUOP-722](https://pensando.atlassian.net/browse/GPUOP-722)
- **Build target image:** `device-metrics-exporter` (image name from `Makefile:14`; built via `make docker-cicd` matching `.job.yml` target `build-device-metrics-exporter-docker-ubi9.6`; Dockerfile `docker/Dockerfile.exporter-release`)
- **Build artifact tarball:** `docker/device-metrics-exporter-latest.tar.gz` (matches `.job.yml:72`)
- **Test deployment image tag:** `${IMAGE_TAG}` (default `${USER}-GPUOP-723`) — applied by user-specific retag in TC01 to avoid collision with the `latest` hourly build's container on shared test hosts
- **Test deployment host port:** `${EXPORTER_PORT}` — auto-discovered in [5000,5100] at test invocation time per `.claude/skills/test-port-allocation.md`. The hourly build typically binds 5000 on lab hosts; tests must not assume 5000 is free. Container port (internal) remains 5000. For TC02 (real-HW IFOE validation), the discovered port is also written into `/etc/metrics/config.json` so the `/validate-ifoe-exporter` harness reads the same port automatically.
- **IFOE metrics correctness harness:** `.claude/commands/validate-ifoe-exporter.md` (slash command `/validate-ifoe-exporter <user>@<host>`) — cross-checks `/metrics` against `gpuctl show ual` for count, UUID/label, and per-entity field-value parity. This is the authoritative pass/fail gate for TC02.
- **Testbeds (set via env vars at invocation — never hardcoded in test cases):**
  - **`TEST_HOST_IFOE`**: IFOE-capable host (Pensando NIC + AMD GPU) for TC02 — TBD
  - **`TEST_HOST_NON_IFOE`**: 1-GPU AMD host without Pensando NIC for TC03, TC04, TC05, TC09, TC11 — engineer supplies
  - **`TEST_USER`**: SSH login on the test hosts (defaults to `${USER}`); auth via SSH key
  - **`IMAGE_TAG`**: user-specific tag from TC01 retag (default `${USER}-GPUOP-723`)
  - **Build host:** local dev box for `make docker-cicd`, deb/rpm regression
- **Branch:** `bugfix/exporter-ifoe` (off `collab-2.0.0`)
- **Date:** 2026-05-11

## Feature Summary

Extend `docker/Dockerfile.exporter-release` so the released `device-metrics-exporter` container ships the real (non-mock) UAL gpuagent + IFOE runtime libs (libmnl, libnl3) + libamd_smi 26.3.0, runs the IFOE field exporter by default via `ENABLE_IFOE=true` ENV, and gracefully no-ops on non-IFOE hosts via the Go-side capability gate in `pkg/amdgpu/gpuagent/gpuagent_ifoe.go`. Deployment patterns documented in `docs/installation/docker.md` and `docs/configuration/docker.md` (mount `/sys:/sys:ro`, expose port 5000, optional configmap via `-v ./config:/etc/metrics`).

## Handoff Checklist

- [x] Design doc reviewed (GPUOP-723 comment 515420 + update 515578)
- [x] Build targets identified (docker, amdexporter, update-ual-assets)
- [x] CLI/env interfaces documented (`ENABLE_IFOE`, `-monitor-ifoe`)
- [x] Error conditions enumerated (no IFOE devices, gRPC fail, invalid env value)
- [ ] IFOE-capable testbed identified (blocker for TC02)
- [x] Non-IFOE testbed parameter (`TEST_HOST_NON_IFOE`) — engineer supplies; details kept out of repo
- [x] Test cases enumerated and prioritized

## Test Cases

### Positive

| # | Test Case | Description | Priority |
|---|-----------|-------------|----------|
| 1 | container-build-end-to-end | `make docker` completes successfully with new Dockerfile changes (libmnl/libnl3, libamd_smi 26.3.0, ENV ENABLE_IFOE) | P0 |
| 2 | ifoe-metrics-scraped-real-hw | On IFOE-capable host: container with `ENABLE_IFOE=true` exposes IFOE port/station/device metrics via Prometheus `/metrics`; **validated end-to-end via the repo's `/validate-ifoe-exporter` slash command** (in `.claude/commands/`) which cross-checks metric counts, UUID labels, and ~30 per-entity field values against live `gpuctl show ual` output | P0 |
| 4 | enable-ifoe-false-disables | Container with `ENABLE_IFOE=false` starts cleanly, IFOE codepath never invoked, plain GPU monitoring unaffected | P1 |

### Negative

| # | Test Case | Description | Priority |
|---|-----------|-------------|----------|
| 3 | non-ifoe-host-graceful-noop | On `${TEST_HOST_NON_IFOE}` with `ENABLE_IFOE=true`: exporter starts cleanly, logs "IFOE disabled" exactly once, no panics, no IFOE series in `/metrics` | P0 |
| 6 | mock-path-removed-universal | After `update_ual_assets.sh` change, the script fetches from non-mock path; verify `assets/gpuagent_ual.bin.gz` content reflects real (not mock) gpuagent | P1 |

### Boundary

| # | Test Case | Description | Priority |
|---|-----------|-------------|----------|
| 5 | enable-ifoe-invalid-value | `ENABLE_IFOE=garbage` → entrypoint defaults to true with warn log; container starts | P1 |

### Error Recovery

| # | Test Case | Description | Priority |
|---|-----------|-------------|----------|
| 11 | gpuagent-grpc-failure-recovery | gpuagent socket missing or unreachable → exporter retry/backoff, no IFOE codepath crash, no panics | P1 |

### Integration

| # | Test Case | Description | Priority |
|---|-----------|-------------|----------|
| 9 | libamd-smi-26.3.0-linkage | gpuagent process starts cleanly with LD_PRELOAD of `/home/amd/lib/libamd_smi.so.26 → libamd_smi.so.26.3.0`; no missing symbols | P1 |
| 10 | runtime-libs-installed | Inside built container, `ldconfig -p` lists `libmnl.so`, `libnl-3.so`, `libnl-genl-3.so` | P1 |

### Regression

| # | Test Case | Description | Priority |
|---|-----------|-------------|----------|
| 7 | deb-ual-still-builds | `make debpkg-ual` produces `amdgpuifoe-exporter_*.deb` after `update_ual_assets.sh` mock-path drop | P1 |
| 8 | rpm-ual-still-builds | `make rpmpkg-ual` produces RPM after `update_ual_assets.sh` mock-path drop | P1 |

**Total: 11 dev test cases — 3 P0, 8 P1.** (TC12 long-run handed off to system test team; TC13 spike dropped — gpuagent does not expose its built-against amd-smi version, so runtime LD_PRELOAD verification in TC09 is the only viable parity check and the Story design's S7 sub-task collapses into "TC09 passes".)

**Categories present:** Positive (3), Negative (2), Boundary (1), Error Recovery (1), Integration (2), Regression (2).

**Categories N/A for this Story:** Concurrency (single-process, no shared state), Upgrade/Downgrade (no version-boundary semantics in container deploy), Scale (single container per host), Performance (no latency/throughput target), Loop/Stability (handed off to system test team), Spike (no answerable spike question remains — see TC13 stub).

## Validation Results (template — populated during execution)

| Stage | Validation | Result | Notes |
|-------|-----------|--------|-------|
| Pre-Test | Container builds | TBD | `make docker` |
| Pre-Test | libamd_smi 26.3.0 present in image | TBD | `ls /home/amd/lib/` inside container |
| Pre-Test | libmnl, libnl3, libnl3-cli installed | TBD | `ldconfig -p` inside container |
| Pre-Test | Non-IFOE testbed reachable | TBD | `ssh "${TEST_USER}@${TEST_HOST_NON_IFOE}"` |
| Post-Test | All P0 cases passed | TBD | TC01, TC02, TC03 |
| Post-Test | All regression cases passed | TBD | TC07, TC08 |

## Verdict (filled in during execution)

- **Status:** Pending
- **P0 pass rate:** 0/3
- **P1 pass rate:** 0/8
- **Blockers:** IFOE-capable testbed (`TEST_HOST_IFOE`) needed for TC02
- **Summary:** TBD

## Test case files

All test cases have standalone instruction files under
`docs/design/test-plans/test-cases/GPUOP-723/`:

- `01-container-build-end-to-end.md` (P0)
- `02-ifoe-metrics-scraped-real-hw.md` (P0)
- `03-non-ifoe-host-graceful-noop.md` (P0)
- `04-enable-ifoe-false-disables.md` (P1)
- `05-enable-ifoe-invalid-value.md` (P1)
- `06-mock-path-removed-universal.md` (P1)
- `07-deb-ual-still-builds.md` (P1)
- `08-rpm-ual-still-builds.md` (P1)
- `09-libamd-smi-26.3.0-linkage.md` (P1)
- `10-runtime-libs-installed.md` (P1)
- `11-gpuagent-grpc-failure-recovery.md` (P1)

(TC12 + TC13 stubs remain on disk only because `rm` is restricted in this session — both are marked dropped in their frontmatter and excluded from this listing.)

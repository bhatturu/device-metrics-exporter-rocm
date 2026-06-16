# gpuagent asset refresh + base image bump to RHEL 9.8

- **Date:** 2026-06-15
- **Author:** Bhanu Kiran Atturu
- **Related PR(s):** #TBD
- **Related issue(s) / JIRA:** NO-JIRA (maintenance)

## Context

The vendored gpuagent assets and the container base image had drifted behind
the gpu-agent repo and the RHEL 9.x stream. The gpu-agent HEAD picked up a Go
toolchain CVE fix (1.25.11), a `card_model` market-name fallback, and a
gimamdsmi macro fix that need to ship in the exporter image. Separately, the
ubi9-minimal base was still pinned to 9.6, which is now superseded by 9.8.

## Approach

- Bump the `gpuagent` submodule pointer `9645999 -> 9af0bf5`.
- Rebuild the four gpuagent binaries (`gpuagent`, `gpuagent_gim`,
  `gpuagent_mock`, `gpuctl`) from `9af0bf5` and refresh the committed assets
  (`assets/*.bin.gz`, `assets/gpuctl.gobin`).
- Bump the base image `ubi9-minimal 9.6 -> 9.8` in `Makefile`, `dev.env`, and
  the standard / sriov / ainic exporter Dockerfiles, including the amdgpu
  `rhel/9.8` repo path.

### Alternatives considered

- Pin to a tagged gpu-agent release instead of a commit — rejected; the fixes
  needed are only on the branch HEAD, no tag cut yet.
- Defer the base bump to a separate PR — rejected; the asset rebuild and base
  bump were validated together and share the same image, so splitting adds
  churn without benefit.

## Scope

- **In scope:** gpuagent submodule + assets refresh; ubi9-minimal 9.6->9.8
  across Makefile, dev.env, and exporter Dockerfiles.
- **Out of scope:** any exporter Go source changes; metric schema changes;
  gpuagent source changes (consumed via submodule only).

## Validation

- Unit tests: `make unit-test` (mock asset must enumerate the expected GPU
  count — guards Test017/Test019 regressions).
- Integration / e2e tests: mock e2e against the rebuilt assets.
- Manual / hardware steps: built the standard exporter image
  (`device-metrics-exporter:dev-5077`, ROCm 7.13.0, base ubi-minimal:9.8) and
  ran it on an MI300A host; `/metrics` returns HTTP 200 with `gpu_*` metrics
  and `card_model` populated via the new market-name fallback.

## Risks and rollback

- Known risks: refreshed gpuagent assets can shift where card identity fields
  land (e.g. `card_model` vs `card_series`); base 9.8 changes the amdgpu repo
  path.
- Rollback plan: revert this commit to restore the `9645999` submodule
  pointer, prior assets, and the 9.6 base pins.

# Run Tests — device-metrics-exporter

Resolves the unit-test command for this project. Used by
`/codie:task-test` (Phase 0 step 5) and `/codie:story-test`.

## Setup

Run from the repo root. No additional setup required; vendor
dependencies are already committed under `vendor/`.

## Test command

```bash
make unit-test
```

This expands (per `Makefile:456`) to:

```bash
PATH=$PATH LOGDIR=$(pwd)/ go test -v -cover -mod=vendor ./pkg/...
```

It exercises every Go package under `pkg/` (covers the
`amdexporter`, `metricsclient`, `amdgpuhealth` build targets and
their subordinate Go packages including `pkg/amdgpu/gpuagent` —
where the IFOE capability gate from GPUOP-736 lives).

`.job.yml:137` `device-metrics-sanity` target uses the same
command in CI, so passing locally implies CI parity.

## Pass criteria

- Exit code 0
- Last line of output shows `ok` for every package under `pkg/...`
  (no `FAIL` lines)

## Scope

This is the **unit-test** runner only. For end-to-end / e2e
validation use `make e2e` (which builds the mock docker image and
runs `make e2e-test`); that is the responsibility of
`/codie:story-test`, not `/codie:task-test`.

## Build targets without runnable unit tests

For build targets that are pure IaC, shell scripts, or assets
(e.g., `docker` build target, `update-ual-assets` script) — `make
unit-test` will simply not exercise their changes. That is
expected. Their verification path is the test plan executed by
`/codie:story-test` (e.g., GPUOP-723 TC01, TC06, TC07, TC08
exercise the docker + script changes via real builds).

## Cross-references

- Makefile target: `Makefile:455-456`
- CI usage: `.job.yml:137-138` (`device-metrics-sanity` target)
- E2E test command (separate, heavier): `make e2e` → `Makefile:536-538`

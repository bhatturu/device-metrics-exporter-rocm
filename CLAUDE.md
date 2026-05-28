# CLAUDE.md — AMD Device Metrics Exporter

Workflow rules and non-obvious gotchas for this repo. Architecture, component deep-dives, and troubleshooting walkthroughs live in [`.claude/kb_source/exporter/`](.claude/kb_source/exporter/) — load on demand, not every session.

## Build & test

All builds happen inside the build container. Never `go build` against the host toolchain.

```bash
make docker-shell   # enter build container — required shell for all builds below
make gen            # regenerate protobuf
make all            # build amdexporter binary
make unit-test      # run unit tests
make docker         # build deployment container image
make pkg            # build Debian + RPM packages
```

Custom skills wrap the multi-step builds — prefer them over invoking make manually:
`/builder`, `/prd-dev-workflow`, `/prd-metric-add`, `/prd-metric-implementation`, `/rocm-update`, `/amdsmi-update`, `/bkc-upgrade`.

## Repo etiquette

- **Branch off `main`** for general work. The `collab-*` branches (e.g. `collab-2.0.0`) are integration branches for specific epics — only base on them if your work is explicitly part of that epic.
- **Config schema:** all runtime config goes through [`pkg/exporter/proto/exporterconfig.proto`](pkg/exporter/proto/exporterconfig.proto). Edit the `.proto`, then `make gen`. Never hand-edit generated `*.pb.go`.
- **Runtime config auto-reloads every 3 seconds** — no restart needed when changing `/etc/metrics/config.json`.

## Don't touch

- **Root `entrypoint.sh`** — this is the jobd CI dev-container helper, NOT the runtime container entrypoint. The runtime entrypoint is [`docker/entrypoint.sh`](docker/entrypoint.sh). Editing the root file silently breaks CI.
- **Anything generated** — `*.pb.go`, `pkg/amdgpu/gen/`, vendored amdsmi/gpuagent assets. Regenerate via `make gen` or the relevant `/amdsmi-update` / `/rocm-update` skill.

## Project conventions

- **Metric name prefix is `amd_*`**, derived from `MetricsFieldPrefix` in `config.json`. When grepping `/metrics` output, search `^amd_ifoe_`, never `^ifoe_` or `^gpu_ual_ifoe_` — those won't match and a test asserting on them will silently pass.
- **Test port discovery:** any test that hits `/metrics` on a shared lab host must read `ServerPort` from the target's `/etc/metrics/config.json` first. The hourly build owns port 5000 — never hardcode it.

## Build gotchas

- `make debpkg-ual` and `make rpmpkg-ual` leave three transient dirty files in `debian/` from sed scaffolding (see `Makefile.package:266-275`) — these are normal mid-build, not real edits. Don't commit them.
- All `docker run` targets in `Makefile.package` need a TTY. Running them via a non-TTY harness will hang.
- `make rpmpkg-ual` requires `libcopy-assets-RHEL9` to have populated `build/assets/RHEL9/profilerlibs/` first. If `libcopy` failed silently (TTY trap), `rpmpkg-ual` produces a misleading error.

## Troubleshooting (top hits)

| Symptom | Likely cause | First check |
|---|---|---|
| `ErrZeroGPUs`, all GPUs unhealthy at startup | amdgpu driver loads after gpuagent | `-exit-on-agent-down` flag + K8s `restartPolicy: Always` |
| `GPU_PROF_*` metrics missing | ROCProfiler init failed (3-failure auto-disable kicked in) | Disable in config: `"ProfilerMetrics": {"all": false}` |
| Config changes not taking effect | Invalid JSON | `jq . /etc/metrics/config.json` |

Deeper troubleshooting tree: [`.claude/kb_source/exporter/troubleshooting.md`](.claude/kb_source/exporter/troubleshooting.md).

## Where to look

- **Entry point:** [`cmd/exporter/main.go`](cmd/exporter/main.go)
- **GPU client:** [`pkg/amdgpu/gpuagent/`](pkg/amdgpu/gpuagent/)
- **NIC client:** [`pkg/amdnic/nicagent/`](pkg/amdnic/nicagent/)
- **Architecture / deep dives:** [`.claude/kb_source/exporter/`](.claude/kb_source/exporter/)
- **User-facing docs (Sphinx):** [`docs/`](docs/)
- **PRDs:** [`.claude/prds/`](.claude/prds/)

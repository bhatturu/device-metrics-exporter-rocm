# SR-IOV / GIM Deployment Knowledge

Operational details for the GIM-based SR-IOV variant of the Device Metrics Exporter (`device-metrics-exporter-sriov`), including the lazy-SMI gpuagent mode (`AGA_SMI_LAZY_INIT=1`).

This file complements [deployment.md](deployment.md) and [troubleshooting.md](troubleshooting.md) with SR-IOV-specific gotchas that have caused real wasted-cycle incidents.

---

## 1. Required `docker run` devices

The standard install command in [docs/installation/docker.md](../../docs/installation/docker.md) shows only `--device=/dev/dri`. **That is insufficient for the SR-IOV variant on a GIM host.** The gpuagent daemon needs direct access to the GIM SMI character device and AMD GIM command channels, otherwise it either aborts at SMI init (`Assertion 'ret == 0' failed` in `main.cc`) or hangs before creating its unix socket (`/var/run/gpuagent.sock` never appears).

Working SR-IOV `docker run`:

```bash
docker run -d \
  --device=/dev/dri \
  --device=/dev/kfd \
  --device=/dev/gim-smi0 \
  --device=/dev/amdgv-cmd-handle \
  --device=/dev/amdgpu_rdl \
  -v /sys:/sys:ro \
  -p 5000:5000 \
  --name device-metrics-exporter \
  <DOCKER_REGISTRY>/<namespace>/device-metrics-exporter-sriov:<tag>
```

Device summary (on a typical GIM host):

| Path | Mode | Owner | Why required |
|---|---|---|---|
| `/dev/gim-smi0` | char 600 | `root:root` | GIM SMI device — required by gpuagent for GIM/SR-IOV |
| `/dev/kfd` | char 660 | `root:render` | ROCm KFD |
| `/dev/dri/card<N>` | char 660 | `root:video` | DRM render node (PF) |
| `/dev/amdgv-cmd-handle` | char | `root:root` | AMD GIM command channel |
| `/dev/amdgpu_rdl` | dir of char | `root:root` | AMD GPU RDL per-BDF entries |

The container runs as uid 0 by default, which satisfies the `root:root 600` permission on `/dev/gim-smi0`. If you change the container user, ensure it has matching groups or relax the host-side device mode.

---

## 2. `AGA_SMI_LAZY_INIT=1` — lazy-init SMI lifecycle

### Intent
By default the gpuagent daemon opens `/dev/gim-smi0` once at startup and **holds the fd for the lifetime of the process**. This blocks other consumers (host-side tools, VF passthrough flows) from owning the device.

With `AGA_SMI_LAZY_INIT=1` the daemon defers SMI init until a metrics request arrives, then runs `init → serve → shutdown` per request, releasing the fd between scrapes.

### Where it is set
The env is set in two places, one per deployment surface:

**Docker images** — `ENV AGA_SMI_LAZY_INIT=1` baked into:
- `docker/Dockerfile.sriov.exporter-release`
- `docker/Dockerfile.sriov.ub22.exporter-release`

Both also set `ENV GPUAGENT_EVENTS_DISABLE=1`. The container's `docker/entrypoint-host.sh` still launches the daemon — **do not remove the daemon from the entrypoint**, only its SMI lifecycle changes.

**DEB and RPM packages** — `AGA_SMI_LAZY_INIT=1` line in:
- `debian-sriov/usr/local/etc/metrics/gpuagent.conf` (git-tracked, separate from the non-SR-IOV `debian/.../gpuagent.conf`)

Both `gpuagent-sriov.service` and `amd-metrics-exporter-sriov.service` reference this file via `EnvironmentFile=/usr/local/etc/metrics/gpuagent.conf`. The RPM spec at `rpmbuild/amdgpu-exporter-sriov.spec` installs the same file. Earlier package builds reused the non-SR-IOV conf (`LIB_CONF_PATH`) which is why DEB/RPM lacked the env even after the Dockerfiles got it; `Makefile.package` no longer does that copy.

The non-SR-IOV `gpuagent.conf` at `debian/usr/local/etc/metrics/gpuagent.conf` does NOT set `AGA_SMI_LAZY_INIT`; the lazy-init code path is only meaningful for the GIM/SR-IOV backend.

The semantics live in the gpuagent C++ code (`gpu-agent` repo, `sw/nic/gpuagent/api/smi/gimamdsmi/`). The DME side just sets the env; the binary in `assets/gpuagent_sriov_static.bin.gz` must be a build that honors the env.

### Verifying it is actually taking effect
On the test host, get the in-container gpuagent PID and inspect its fd table:

```bash
GPID=$(docker exec device-metrics-exporter pidof gpuagent)

# Should print "(no gim/smi fd held)" when idle
docker exec device-metrics-exporter sh -c \
  "ls -la /proc/$GPID/fd/ | grep -E 'gim|smi' || echo '(no gim/smi fd held)'"

# Trigger a scrape and re-check; cycle should be too fast to catch
curl -s -o /dev/null http://localhost:5000/metrics
sleep 0.05
docker exec device-metrics-exporter sh -c \
  "ls -la /proc/$GPID/fd/ | grep -E 'gim|smi' || echo '(no gim/smi fd held)'"
```

If `/dev/gim-smi0` is listed in the fd table when the daemon is idle, the lazy-init path is **not** taking effect. Likely causes:
- The gpuagent binary in the asset tarball pre-dates the lazy-init implementation
- The env var name was changed in the source but the Dockerfile wasn't updated
- The lazy-init code path is gated on something else (e.g. only in `-sriov-enable` mode)

---

## 3. Metric prefix gotcha — `gpu_*` not `amd_gpu_*`

The SR-IOV docker image's default config emits metrics **without** the `amd_` prefix that other DME deployments use:

```
gpu_clock{...} 1700
gpu_edge_temperature{...} 50
pcie_bandwidth{...} 0
```

The `amd_` prefix is set via `MetricsFieldPrefix` in `config.json`. The SR-IOV image's default container config does NOT set this field, so the prefix is empty.

**Implication for diagnostics:** Do NOT use `curl /metrics | grep -c '^amd_'` as a health check for the SR-IOV image — it will report 0 even when the exporter is fully healthy and emitting hundreds of `gpu_*` samples. The correct check is:

```bash
curl -s http://localhost:5000/metrics | grep -cE '^(gpu_|pcie_)'
```

**Implication for downstream consumers:** Prometheus rules, Grafana dashboards, and alerts written against `amd_gpu_*` metric names will silently match nothing against an SR-IOV image. Either:
- Bake a per-deployment config into the image with `MetricsFieldPrefix: "amd"`, or
- Maintain SR-IOV-specific dashboards/rules using the bare `gpu_*` names.

(The `.deb` / `.rpm` packaging paths under `Makefile.package` apply a different jq transform to the config than the docker path uses, which is why behavior differs by deployment mode.)

---

## 4. Diagnostic recipe — "exporter HTTP 200 but no metrics"

Symptoms: `/metrics` returns HTTP 200 but contains only `# HELP …` lines plus `go_*` / `promhttp_*` counters. No `gpu_*` data.

Decision tree (run inside the container unless noted):

### Step 1 — Is the gpuagent process alive?
```bash
docker exec device-metrics-exporter ps -ef | grep -v grep | grep gpuagent
```
- **No process:** look at `docker logs <container>` for an assertion abort like `gpuagent: main.cc:NNN: ... Assertion 'ret == 0' failed.` That's a SMI init crash — confirm the 5 GIM devices from Section 1 are passed in.
- **Process present, ppid=1:** continue to step 2.

### Step 2 — Does the unix socket exist?
```bash
docker exec device-metrics-exporter ls -la /var/run/gpuagent.sock
```
- **`No such file`:** gpuagent is running but has not reached the listener-creation step. Likely hung before logging init too (next step confirms).
- **Socket present (`srw-...`):** continue to step 4.

### Step 3 — Are the gpuagent logs being written?
```bash
docker exec device-metrics-exporter ls -la /var/log/gpu-agent*.log
```
The three log files are:
- `/var/log/gpu-agent.log` — main daemon log
- `/var/log/gpu-agent-err.log` — SMI errors
- `/var/log/gpu-agent-api.log` — API trace

If all three are **0 bytes**, the daemon has not progressed past its pre-logging init phase. This is almost always a stuck/deadlocked startup path in the gpuagent binary itself — collect the fd table and process state and hand off to a gpuagent debug cycle:

```bash
docker exec device-metrics-exporter ls -la /proc/$(pidof gpuagent)/fd/
docker exec device-metrics-exporter cat /proc/$(pidof gpuagent)/status   # State, Threads, etc.
docker exec device-metrics-exporter cat /proc/$(pidof gpuagent)/wchan   # what it's sleeping on
```

If `gpu-agent.log` IS being written (API IPC msgs flowing) but `gpu-agent-err.log` shows `Failed to get accelerator partition config ... err 2` and similar — those `err 2` (ENOENT) entries on a single-GPU/no-partition host are expected and not fatal; the daemon will still serve basic metrics.

### Step 4 — Probe gpuagent directly via gpuctl
```bash
timeout 10 docker exec device-metrics-exporter /home/amd/bin/gpuctl show gpu
```
- **Hangs/timeout:** the daemon is listening but not responding — see step 3's process-state checks.
- **Returns GPU info:** gpuagent works. The exporter side is the issue — check `/var/log/exporter.log` for repeated `gpuagent get metrics failed rpc error: ... no such file or directory` (means exporter is looking at the wrong socket path) or for `Platform doesn't support field name: ...` lines that are normal SMI filtering.

### Step 5 — Confirm metrics with the right prefix
See Section 3. Use `grep -cE '^(gpu_|pcie_)'` not `^amd_`.

---

## 5. Refreshing only the SR-IOV asset (`gpuagent-asset-copy`)

The `gpuagent-asset-copy` Makefile target in `Makefile.compile` always processes **all four** binaries (`gpuagent`, `gpuagent_gim`, `gpuagent_mock`, `gpuctl`) from `$(GPUAGENT_SRC_DIR)/sw/nic/build/x86_64/sim/bin/` and refuses to run if any is missing.

### Name mapping (source binary → asset tarball)

| Source binary | Asset file | Notes |
|---|---|---|
| `gpuagent` | `assets/gpuagent_static.bin.gz` | non-SR-IOV (default) |
| `gpuagent_gim` | `assets/gpuagent_sriov_static.bin.gz` | **GIM build = SR-IOV image** |
| `gpuagent_mock` | `assets/gpuagent_mock.bin.gz` | mock |
| `gpuctl` | `assets/gpuctl.gobin` | plain copy + strip, no tar |

Inside each tarball the binary is renamed to literally `gpuagent` so the Dockerfile's `ADD ./gpuagent /home/amd/bin/gpuagent` works regardless of which variant was packaged.

### Surgical recipe (just refresh `gpuagent_sriov_static.bin.gz`)

When you only want to update the SR-IOV asset (e.g. iterating on the GIM gpuagent code without rebuilding the other 3 binaries), run the equivalent transform manually:

```bash
cp -vf ${GPUAGENT_SRC_DIR}/sw/nic/build/x86_64/sim/bin/gpuagent_gim assets/gpuagent
strip assets/gpuagent
( cd assets && tar czf gpuagent_sriov_static.bin.gz gpuagent && chmod +x gpuagent_sriov_static.bin.gz )
rm -f assets/gpuagent
```

`git status assets/` should then show only `gpuagent_sriov_static.bin.gz` as modified.

---

## Related files

- [deployment.md](deployment.md) — broader deployment context (K8s, debian/rpm, SR-IOV summary)
- [troubleshooting.md](troubleshooting.md) — generic failure modes (driver-load timing, ROCProfiler, etc.)
- [build-system.md](build-system.md) — overall build workflow
- `docker/Dockerfile.sriov.exporter-release`, `docker/Dockerfile.sriov.ub22.exporter-release` — SR-IOV image recipes
- `docker/entrypoint-host.sh` — entrypoint that launches gpuagent + exporter (daemon stays)
- `Makefile.compile` — `gpuagent-asset-copy` target source

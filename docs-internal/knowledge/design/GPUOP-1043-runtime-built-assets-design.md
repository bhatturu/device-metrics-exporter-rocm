# GPUOP-1043 — Runtime-built amdsmi assets + single gpuagent build for non-SR-IOV (and SR-IOV) DME artifacts

**JIRA:** GPUOP-1043 (Story)
**Status:** Design
**Author:** Praveen kumar Shanmugam
**Date:** 2026-08-06
**Branch base:** `main`

---

## Problem

DME build targets pick up `libamd_smi.so` / `amdsmi.h` from the repo's pre-populated
`assets/amd_smi_lib/` tree, and — except for the release docker image — pick up gpuagent
from committed prebuilt blobs (`assets/gpuagent_*.bin.gz`) that have **no in-tree producer
target** (updated out-of-band via `/amdsmi-update`).

Consequences:
- The release docker image builds gpuagent **from source** in-Dockerfile, while
  docker-mock / deb / rpm (and the SR-IOV variants) consume **different, separately-built
  prebuilt blobs** — two divergent gpuagent provenance paths.
- Sourcing amdsmi from the therock tarball is opt-in (`AMDSMI_FROM_TARBALL=1`) and only
  wired for the release docker path.
- Binary stripping is applied inconsistently (deb strips a subset, **rpm strips nothing**,
  docker strips only the C++ binaries).

## Goal

1. Make **tarball-sourced amdsmi** and **from-source gpuagent** the standard (default) path
   for all non-SR-IOV targets (`docker`, `docker-mock`, `debpkg`, `rpmpkg`) **and** the
   SR-IOV targets (`docker-sriov`, `debpkg-sriov`, `rpmpkg-sriov`), driven by `dev.env`
   toggles. Keep the prebuilt `assets/` path as a default-disabled escape hatch.
2. Build gpuagent (and siblings `gpuctl`, `gpuagent_mock`, `gpuagent_gim`) **once per
   `make` invocation** via a single shared producer target that all image/package targets
   consume.
3. Apply a **per-target-type binary strip policy**.
4. Bundle the **ROCm nightly bump** (`release/therock-7.14` → `release/therock-10.0`),
   accounting for the amdsmi SO version of the chosen build.
5. Clean up dead/legacy build infra touched by this work.
6. **Consolidate the docker build targets and CI jobs** (GPUOP-1048): a single
   `make docker` that produces both exporter images plus all deb/rpm packages, and a
   collapsed set of jobd build jobs.

## Scope

- **In scope:** `docker`, `docker-mock`, `docker-sriov`, `debpkg` (Ubuntu 22/24),
  `rpmpkg` (RHEL9), `debpkg-sriov`, `rpmpkg-sriov`; the shared single gpuagent producer;
  strip policy; ROCm nightly bump; cleanup of orphaned build infra.
- **Out of scope:** AINIC targets (`docker-ainic`, `nic-debpkg`) — no gpuagent dependency,
  remain separate targets/jobs (the GPUOP-1048 CI cleanup only drops the redundant
  `make all` prefix from the ainic job; it does not fold ainic into `make docker`).
  Sourcing the **GIM SMI** lib from a tarball — SR-IOV keeps `gim_smi_lib` on its existing
  `assets/gimsmi-compile` path (the therock tarball ships `amd_smi`, not `gim_smi`).

> **Note on JIRA vs. current code.** The JIRA description contains three inaccuracies
> confirmed against the tree; this design follows the JIRA *intent*, corrected to reality:
> 1. **rpm strips nothing today** (JIRA says deb/rpm already strip gpuagent/rocpctl/exporter).
>    Only the deb path strips; `rpmpkg`/`rpmpkg-sriov` have zero `strip` commands. The rpm
>    strip step is added from scratch.
> 2. **The prebuilt `gz` blobs have no producer target** (retired per `Makefile.compile`).
>    "Build once, share" requires a *new* producer, not just dependency wiring.
> 3. **CI is jobd (`.job.yml`), not GitHub Actions.** `.github/workflows/` only runs
>    doc-audit + plan-check. The "GHA artifact sharing" option is moot; the delivered
>    mechanism is Make-level.

---

## Architecture: the two-knob source model

Each target gets **two independent source knobs**, defined once in `dev.env` and threaded
through `Makefile` → `docker/Makefile` / `Makefile.package`:

| Knob | `=1` (default) | `=0` (escape hatch) |
|---|---|---|
| `AMDSMI_FROM_TARBALL` | extract `amdsmi.h` + `libamd_smi.so` (+ deps) from the therock tarball at build time | use pre-staged `assets/amd_smi_lib/x86_64/<OS>/lib` |
| `GPUAGENT_FROM_SOURCE` *(new)* | build gpuagent/gpuctl/mock/gim from source against the (tarball) amdsmi | use committed `assets/gpuagent_*.bin.gz` prebuilt blob |

**Default flips (`dev.env`):** `AMDSMI_FROM_TARBALL ?= 0` → **`?= 1`**; add
`GPUAGENT_FROM_SOURCE ?= 1`.

- On `main`, the Story mandates **tar + source** as the standard path.
- Setting either knob to `0` reverts a single target to prebuilt behavior without disturbing
  the others. This mirrors what `collab-2.0.0` already does (picking prebuilt gpuagent in
  some cases) and keeps the committed `assets/` blobs as a supported fallback.
- SMI source split: non-SR-IOV → `amd_smi` from tarball; SR-IOV → `gim_smi_lib` from
  `assets/` (the tarball knob does not source the GIM SMI lib).

---

## Shared gpuagent producer (Make-level, single target)

The from-source gpuagent build is **irreducibly containerized** (ubi9 toolchain, Go
1.25.11, exact source path `/usr/src/github.com/ROCm/gpu-agent`, recursive submodules,
repo patches, `ranlib` fixes) — this is why the old host-side gpuagent targets were
retired. The producer therefore runs the existing `gpuagent-build` Dockerfile **stage**
standalone and extracts the binaries to a shared host directory.

**One consolidated target** builds all shared binaries in a single pass:

```makefile
# builds gpuagent + gpuctl + gpuagent_mock + gpuagent_gim ONCE against tarball amdsmi,
# extracts to build/gpuagent/. The stamp is the "once per make invocation" guarantee.
build/gpuagent/.stamp:
	docker build --target gpuagent-build \
	    --build-arg AMDSMI_FROM_TARBALL=$(AMDSMI_FROM_TARBALL) \
	    --build-arg ROCM_TARBALL_URL=$(ROCM_TARBALL_URL) \
	    --build-arg GPUAGENT_COMMIT=$(GPUAGENT_COMMIT) \
	    -t gpuagent-staged -f docker/Dockerfile.exporter-release .
	# docker create + docker cp -> build/gpuagent/{gpuagent,gpuctl,gpuagent_mock,gpuagent_gim}
	touch $@

gpuagent-build: build/gpuagent/.stamp
```

**Consumer map** (when `GPUAGENT_FROM_SOURCE=1`, the default):

| Target | gpuagent source | Today |
|---|---|---|
| `docker` (release) | `build/gpuagent/gpuagent` | builds in-stage |
| `docker-mock` | `build/gpuagent/gpuagent_mock` | `assets/gpuagent_mock.bin.gz` |
| `docker-sriov` | `build/gpuagent/gpuagent_gim` | `assets/gpuagent_sriov_static.bin.gz` |
| `debpkg` | `build/gpuagent/gpuagent` | `assets/gpuagent_static.bin.gz` |
| `rpmpkg` | `build/gpuagent/gpuagent` | `assets/gpuagent_static.bin.gz` |
| `debpkg-sriov` / `rpmpkg-sriov` | `build/gpuagent/gpuagent_gim` | `assets/gpuagent_sriov_static.bin.gz` |

- **Once guarantee:** `make docker` (which now builds both images + all deb/rpm — see the
  GPUOP-1048 consolidation section) builds gpuagent exactly once — all consumers depend on
  the same `build/gpuagent/.stamp`.
- **Escape hatch (`GPUAGENT_FROM_SOURCE=0`):** each consumer falls back to its
  `assets/*.bin.gz` blob. The committed blobs stay in the tree as the `=0` source.
- The release Dockerfile's stage-1 `gpuagent-build` is **reused** as the shared producer
  (not duplicated); the runtime stage `COPY`s from `build/gpuagent/` when the producer
  path is active.

**Limitation (documented, accepted):** `.job.yml` runs each target as a separate jobd job
with its own checkout and `make`. Make-level sharing collapses redundancy *within* one
`make` invocation, **not across jobd jobs** — each CI job still builds gpuagent once in its
own container. True cross-job artifact sharing (jobd `build-dependencies`) is a possible
future follow-up, deliberately out of scope here.

---

## Binary strip policy

Legend: ✅ strip · ❌ keep symbols · — n/a.

| Binary | docker (release) | docker-mock | deb | rpm | sriov deb/rpm |
|---|---|---|---|---|---|
| `gpuagent` / `gpuagent_gim` | ✅ (stage 1) | as-is | ✅ | ✅ **add** | ✅ **add** |
| `gpuctl` | ✅ (stage 1) | as-is | ✅ **add** | ✅ **add** | ✅ **add** |
| `metricsclient` | ✅ **add** | as-is | ✅ **add** | ✅ **add** | ✅ **add** |
| `amdgpuhealth` | ✅ **add** | as-is | ✅ **add** | ✅ **add** | ✅ **add** |
| `rocpctl` | — | as-is | ✅ | ✅ **add** | ✅ **add** |
| `amd-metrics-exporter` (server) | ❌ **keep** | as-is | ✅ | ✅ **add** | ✅ **add** |
| `amd-test-runner` | ❌ **keep** | as-is | ✅ **add** | ✅ **add** | ✅ **add** |

**Rationale.** Docker/docker-mock images keep debug symbols on `amd-metrics-exporter` and
`amd-test-runner` for correct in-container backtraces during field triage. deb/rpm strip
**everything** — installs prioritize package size over in-place symbols.

**Concrete edits:**
- **docker (release):** strip `metricsclient` + `amdgpuhealth` in the runtime stage after
  they are `ADD`ed. gpuagent/gpuctl already stripped in stage 1. Server + test-runner stay
  unstripped.
- **docker-mock:** **no change** — lean local unit-test image; size irrelevant.
- **deb (`Makefile.package`):** add strip for `gpuctl`, `metricsclient`, `amdgpuhealth`,
  `amd-test-runner`; collapse the redundant double-strip of gpuagent to one.
- **rpm (`rpmpkg`):** add a **full strip step from scratch** (rpm strips nothing today)
  covering every shipped binary.
- **sriov deb/rpm:** adopt strip-all (previously unstripped in rpm; partial in deb).

---

## ROCm nightly bump + amdsmi SO 26→27 shift

**Version bump (`release/therock-10.0` branch):**
- Bump `ROCM_VERSION` / `ROCM_TARBALL_URL` in `dev.env` **and** the duplicated defaults in
  `Makefile` (~`:129-130`) — keep both in sync. Also update `AMDSMI_BRANCH`
  (`release/therock-7.14` → `release/therock-10.0`).
- **Target the `release/therock-10.0` branch** (the current nightly line for this Story) —
  the 10.0-based build (bucket shows `10.0.0` on 2026-07-29/30). Deliberately **not** the
  10.1 line. Pick the specific date-stamped `10.0.0` tarball from that branch and verify it
  is live (HTTP 200) at implementation time. `main` only; no `collab-*` changes.
- Refresh stale provenance (`assets/version.yaml` says 7.13; `assets/amd_smi_lib/version.txt`)
  to match.

**amdsmi SO version — verify first (do NOT assume SO 27):**
- The SO 26→27 drop of `librocm_sysdeps_*` was observed on the **develop / 10.1** line
  (`collab-2.0.0` commit `221643289`, [KUBE-16], #1511). A **10.0 build from 07-29/30 may
  still ship SO 26 + `librocm_sysdeps_*`** — so the SO-27 handling below is **conditional**,
  gated on the actual SO version of the chosen 10.0 tarball.
- **At implementation time, inspect the chosen tarball's `libamd_smi.so.*`** to determine
  the real SO major.
- **If SO 26 (likely for 10.0):** keep the existing `librocm_sysdeps_*` COPY/extract steps
  as-is; no symlink/sysdeps changes needed.
- **If SO 27:** **conditionalize/drop** the `librocm_sysdeps_*.so*` COPY/extract steps in
  `Dockerfile.exporter-release` (currently ~`:55,60,63`), and ensure `libnl-3` / `libmnl`
  are dnf-installed in the runtime stage.
- **SO version is already derived dynamically** in the from-source stage (computes
  `so_ver`/`so_maj` from the actual file), so building gpuagent from source against the
  tarball amdsmi handles 26→27 automatically — no hardcoded 26.
- **`.so.26 → real .so` compat symlink is NOT needed on the default path.** With
  `GPUAGENT_FROM_SOURCE=1`, gpuagent links against SO-27 → its `DT_NEEDED` is `.so.27`. The
  compat shim (and the `/home/amd/lib` `LD_LIBRARY_PATH` entry it needs) matters **only** for
  the `GPUAGENT_FROM_SOURCE=0` prebuilt-blob escape hatch when that blob was built against
  SO-26.

---

## Cleanup (folded into this work)

Scoped to build-infra touched by this restructure (not unrelated repo-wide dead code):
- Remove dead `GPUAGENT_BUILDER_IMAGE` var (`Makefile`, exported but consumed nowhere).
- Remove retired-target comment cruft in `Makefile.compile` (`gpuagent-compile` /
  `-build-only` / `-asset-copy` / `-inject`).
- Collapse redundant double-`strip` calls in `Makefile.package`.
- Consolidate to the **single** gpuagent producer target — no per-binary producer targets.

---

## Docker target + CI consolidation (GPUOP-1048)

Folded into this Story as a follow-up. Collapses the fragmented docker/package targets
and jobd jobs.

**Make targets** (`Makefile`, `docker/Makefile`), composed via make **prerequisites** so no
build step is written twice and shared prereqs (`gen`, `amdexporter`, gpuagent `.stamp`)
build once per run:

| Target | Produces |
|---|---|
| `docker` | both exporter images (non-SR-IOV + SR-IOV) **and** all deb (Ubuntu 22/24) + rpm (RHEL9), non-SR-IOV + SR-IOV |
| `docker-sriov` | SR-IOV image + SR-IOV deb + SR-IOV rpm (a subset of `docker`) |
| helpers | `docker-image`, `docker-image-sriov`, `docker-pkgs`, `docker-pkgs-sriov` |

- **`docker-cicd` removed** (root + `docker/Makefile`); its `HOURLY_TAG` label folded into
  the image helper (test-runner's own `docker-cicd` is unrelated, untouched).
- **Ubuntu 22 vs 24:** `UBUNTU_VERSION` is process-global (selects `UBUNTU_LIBDIR` once per
  make), so ub24 is a recursive `UBUNTU_VERSION=noble` sub-make. The `debpkg` /
  `debpkg-sriov` clean globs were narrowed to the current `$(UBUNTU_VERSION_NUMBER)` so
  building ub22 then ub24 in one `make docker` does not wipe the ub22 `.deb`.
- **amdexporter dedup:** package targets guard `${MAKE} amdexporter` behind
  `EXPORTER_PREBUILT`; `docker-pkgs` passes `EXPORTER_PREBUILT=1` (binary already built as a
  prereq — CGO-static amd64, identical across ub22/ub24/rhel9). Standalone `make debpkg` /
  `rpmpkg` still rebuild fresh.
- **AINIC** (`docker-ainic`), **azure** (`docker-azure`, no longer supported), and
  **mock** stay separate targets.

**jobd CI** (`.job.yml`, `asset-build/.job.yml`):
- The separate `build-debian-package-ub22.04` / `-ub24.04` / `build-rpm-package-rhel9` and
  `build-device-metrics-exporter-docker-sriov-ubi9.6` jobs are folded into one job,
  **renamed** `build-device-metrics-exporter-docker-ubi9.6` → **`build-device-metrics-exporter-gpu`**
  (now emits both images + all six gpu packages via `make docker`).
- Redundant `make all &&` prefix dropped from the gpu and ainic jobs (`make docker` /
  `docker-ainic` build `gen`+`amdexporter` themselves; `metricutil`/`amdtestrunner` aren't
  shipped by them).
- `build-dependencies` in both `.job.yml` and `asset-build/.job.yml` updated to the
  consolidated job. **NIC deb** (`build-nic-debian-package`) stays separate (not produced by
  `make docker`); `exporter-asset-push.sh` needs no change (references artifact paths, which
  are unchanged).

---

## Build Targets

| Build target | File | Type | Change |
|---|---|---|---|
| dev.env config | `dev.env` | make-config | Flip `AMDSMI_FROM_TARBALL ?= 1`; add `GPUAGENT_FROM_SOURCE ?= 1`; ROCm bump |
| root Makefile | `Makefile` | make | New single `gpuagent-build` producer (`build/gpuagent/.stamp`); ROCm dup sync; remove dead `GPUAGENT_BUILDER_IMAGE` |
| docker Makefile | `docker/Makefile` | make | Wire producer + knobs; `docker` builds both images; remove `docker-cicd` |
| package Makefile | `Makefile.package` | make | deb/rpm + sriov consume `build/gpuagent/*`; strip-all; collapse double-strips; `EXPORTER_PREBUILT` guard; narrow deb clean glob |
| release Dockerfile | `docker/Dockerfile.exporter-release` | dockerfile | Reuse stage-1 as shared producer; strip metricsclient+amdgpuhealth in runtime; SO-27 sysdeps handling |
| compile Makefile | `Makefile.compile` | make | Remove retired-target cruft; keep `gimsmi-compile` |
| root Makefile (docker/pkgs) | `Makefile` | make | Dependency-composed `docker` / `docker-sriov` (+ `docker-image*` / `docker-pkgs*` helpers) building images + all deb/rpm |
| CI jobs | `.job.yml`, `asset-build/.job.yml` | jobd | Collapse package + sriov docker jobs into `build-device-metrics-exporter-gpu`; drop redundant `make all`; update `build-dependencies` |

### Code Path: amdsmi source-selection knob
**Build target:** dev.env config, docker Makefile, package Makefile.
Thread `AMDSMI_FROM_TARBALL` into every non-SR-IOV target; default `1`.

### Code Path: gpuagent source-selection + shared producer
**Build target:** root Makefile, docker Makefile, package Makefile, release Dockerfile.
New single producer target + `GPUAGENT_FROM_SOURCE` knob; consumers depend on the stamp.

### Code Path: strip policy
**Build target:** release Dockerfile (docker), package Makefile (deb/rpm + sriov).
Per the strip matrix above.

### Code Path: ROCm bump + SO-27 handling
**Build target:** dev.env config, root Makefile, release Dockerfile, provenance files.
Re-derive live nightly; conditionalize `librocm_sysdeps_*`; dnf `libnl-3`/`libmnl`.

### Code Path: cleanup
**Build target:** root Makefile, Makefile.compile, package Makefile.
Remove dead vars/comments; collapse double-strips.

### Code Path: docker target + CI consolidation (GPUOP-1048)
**Build target:** root Makefile (docker/pkgs), docker Makefile, package Makefile, CI jobs.
Dependency-composed `docker` (both images + all deb/rpm) and `docker-sriov` (sriov subset);
remove `docker-cicd`; `EXPORTER_PREBUILT` dedup; narrow deb clean glob; collapse jobd package
jobs into `build-device-metrics-exporter-gpu` and drop redundant `make all`.

---

## Testing strategy

Build-infra validation = "does it build + produce correct artifacts":
- **Per-target build smoke:** `make docker`, `docker-mock`, `docker-sriov`, `debpkg`,
  `rpmpkg`, sriov variants each build clean with defaults (tar + source).
- **Knob matrix:** each `=0` escape hatch reverts that target to assets/blob and still builds.
- **Once-guarantee:** `make docker` (builds both images + all deb/rpm) builds gpuagent
  exactly once (assert single `build/gpuagent/.stamp`, single stage build); and
  `amdexporter` builds once (EXPORTER_PREBUILT dedup across the package sub-makes).
- **Strip assertion:** `file` / `nm` on shipped binaries — stripped where policy says
  stripped; symbols present on server + test-runner in docker.
- **`gpuagent_gim` equivalence check** vs. today's `gpuagent_sriov_static.bin.gz` before
  wiring SR-IOV to consume the producer output.
- **ROCm URL liveness:** HTTP 200 on the re-derived nightly.
- **SO-27 runtime smoke:** container starts, gpuagent loads amdsmi (no missing DT_NEEDED),
  `/metrics` serves `amd_*` metrics (read `ServerPort` from the target's
  `/etc/metrics/config.json` — never hardcode 5000).

## Error handling / risks

- **Tarball download failure** (~10 GB nightly): build fails hard. Top fragility; `=0`
  fallback is the mitigation.
- **SR-IOV parity risk:** if `gpuagent_gim` ≠ old sriov blob, SR-IOV regresses — gated by
  the equivalence check.
- **SO-27 sysdeps regression:** dropping `librocm_sysdeps_*` without dnf `libnl-3`/`libmnl`
  breaks amdsmi load — covered by the runtime smoke test.
- **main-wide default flip:** every `main` build now pulls the tarball — heavier CI;
  intended.
- **Nightly staleness:** re-verify-live avoids shipping a 404 URL.
- **Process:** per-PR plan-file gate + `/pr-create` format required.

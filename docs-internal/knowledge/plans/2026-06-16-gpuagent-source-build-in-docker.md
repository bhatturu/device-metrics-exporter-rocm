# Plan: Remove gpuagent submodule, build gpuagent from source inside the exporter Docker build

**Date:** 2026-06-16
**Goal:** Stop vendoring gpuagent as a git submodule. Instead, clone `ROCm/gpu-agent` at a pinned
commit during a multi-stage exporter Docker build, inject the therock-tarball amdsmi
(`libamd_smi.so` + `amdsmi.h`) and required source patches, compile gpuagent in-image, and COPY the
resulting binaries into the runtime stage. `make docker` produces everything end-to-end with no
submodule and no host-side gpuagent build step.

## Why this pivot

The submodule + host-side build chain has been fragile (stale builder images, libzmq/abseil link
breaks, root-owned build artifacts, manual amdsmi injection). Building gpuagent from a fresh clone
inside the Docker build makes the build hermetic and reproducible, and removes the submodule
maintenance burden. The amdsmi the gpuagent links against and the amdsmi shipped at runtime both
come from the same therock tarball.

## Decisions (resolved with user)

1. **gpuagent source** = clone `ROCm/gpu-agent` at `GPUAGENT_COMMIT` (keep the pin var for
   reproducibility; bump it to advance), then apply repo-local patches.
2. **Build location** = multi-stage `docker/Dockerfile.exporter-release`: a gpuagent *builder*
   stage clones + compiles; the runtime stage `COPY --from=` the binaries. No separate make target
   runs the gpuagent compile anymore.
3. **Patches applied to the clone before compiling**:
   - **amdsmi swap** — replace `sw/nic/third-party/rocm/amd_smi_lib/{include/amd_smi/amdsmi.h,
     x86_64/lib/libamd_smi.so*}` with the tarball-extracted 26.4.0 files + fix symlink chain.
   - **abseil link fix** — make gpuagent link on the RHEL9 toolchain: `ranlib` the abseil static
     archives (they ship with an empty armap) and/or wrap the grpc+absl archive list in
     `-Wl,--start-group ... -Wl,--end-group` in `sw/nic/gpuagent/Makefile`. Carried as a committed
     `.patch` in `patch/gpuagent/`.
4. **Clone auth + depth** = HTTPS public clone (`https://github.com/ROCm/gpu-agent.git`). The repo
   is public; no creds in the build. Some third-party submodules use `git@github.com:` URLs
   (libzmq, abseil-cpp, boost) — add a global
   `git config --global url."https://github.com/".insteadOf "git@github.com:"` in the builder stage
   so recursive clone works over HTTPS.
   - **Shallow (`--depth 1`) for clone AND submodules** — we don't need history for a build, so
     minimize sync time:
     `git clone --depth 1 --recurse-submodules --shallow-submodules <repo>`.
   - **Pinned-commit caveat**: a depth-1 clone only fetches the branch *tip*. To honor
     `GPUAGENT_COMMIT` when it is NOT the current tip, fetch just that commit shallowly instead of
     `clone --branch`:
     `git init && git remote add origin <repo> && git fetch --depth 1 origin <GPUAGENT_COMMIT> &&
      git checkout FETCH_HEAD && git submodule update --init --recursive --depth 1`
     (GitHub allows fetching a specific SHA shallowly via allowReachableSHA1InWant). When
     `GPUAGENT_COMMIT` == branch tip, the simpler `clone --depth 1 --branch ${GPUAGENT_BRANCH}` is
     equivalent.
5. **Cleanup** = remove the submodule (`git rm gpuagent`, drop `.gitmodules` entry,
   `git submodule deinit`) and retire the now-obsolete host-side targets: `gpuagent-compile`,
   `gpuagent-build-only`, `gpuagent-build`, `gpuagent-asset-copy`, `amdsmi-inject-gpuagent`,
   `docker-tarball-amdsmi` (and the `gpuagent-shell` helper). Keep `amdsmi-from-tarball` /
   `amdsmi-sync-assets` only if still useful for the toggle (see #6).
6. **amdsmi into the builder stage = TOGGLEABLE (key requirement)** — the tarball is ~10.5 GB and
   changes only occasionally, so we must NOT re-extract it on every build.
   - Build arg `AMDSMI_FROM_TARBALL` (default `0`).
   - **Default (`0`)**: the builder stage uses a **pre-staged** amdsmi committed in-repo
     (`assets/amd_smi_lib/x86_64/RHEL9/lib/libamd_smi.so*` + `amdsmi.h`), COPY'd into the stage.
     No download. This is the common path.
   - **Opt-in (`AMDSMI_FROM_TARBALL=1`)**: the builder stage runs the selective
     `curl -s $ROCM_TARBALL_URL | tar -xz --wildcards 'amdsmi.h' 'libamd_smi.so*'` to refresh from
     the tarball (use when the tarball/ROCm version actually changed), and the refreshed files are
     what gets injected + synced back into `assets/` for future default builds.

## Pinned coordinates (current)

- `GPUAGENT_BRANCH = main`, `GPUAGENT_COMMIT = 9af0bf5` (= upstream origin/main tip, verified via
  `git ls-remote https://github.com/ROCm/gpu-agent.git HEAD`).
- amdsmi SO `26.4.0`, glibc cap 2.27 (UBI9-safe), already staged at
  `build/amdsmi-from-tarball/` and `assets/amd_smi_lib/x86_64/RHEL9/lib/`.
- gpuagent builder toolchain: ubi9.8, glibc 2.34, gcc 11.5, go 1.25.11.

## Implementation

### Step 1 — Remove the submodule
- `git submodule deinit -f gpuagent`
- `git rm -f gpuagent` (removes the gitlink + `.gitmodules` entry; verify `.gitmodules` no longer
  references gpuagent — the `libgimsmi` entry stays).
- `rm -rf .git/modules/gpuagent`.
- Keep `GPUAGENT_REPO`/`GPUAGENT_BRANCH`/`GPUAGENT_COMMIT` vars in `Makefile` (now used by the
  Dockerfile clone, not a submodule). Add `GPUAGENT_REPO ?= https://github.com/ROCm/gpu-agent.git`.

### Step 2 — Capture patches under `patch/gpuagent/`
- `0001-abseil-link-fix.patch` — the `ranlib` step (or `--start-group` wrap) needed for the RHEL9
  link. Authored from the working-tree fix already validated against the ld error.
- amdsmi swap is NOT a patch (binary file replacement) — handled by Step 3 logic in-Dockerfile.

### Step 3 — Multi-stage `docker/Dockerfile.exporter-release`
Add a builder stage before the runtime stage:
```
ARG GPUAGENT_BUILDER_BASE_IMAGE=registry.access.redhat.com/ubi9/ubi:9.8
ARG GPUAGENT_REPO=https://github.com/ROCm/gpu-agent.git
ARG GPUAGENT_COMMIT=9af0bf5
ARG AMDSMI_FROM_TARBALL=0
ARG ROCM_TARBALL_URL=

FROM ${GPUAGENT_BUILDER_BASE_IMAGE} AS gpuagent-build
# toolchain: dnf install git gcc gcc-c++ make automake libtool cmake autoconf \
#   glibc-static libstdc++-static wget yum sudo + go 1.25.11
RUN git config --global url."https://github.com/".insteadOf "git@github.com:"
# shallow fetch of the pinned commit + shallow recursive submodules (no history)
RUN mkdir -p /src && cd /src && git init -q && git remote add origin ${GPUAGENT_REPO} && \
    git fetch --depth 1 origin ${GPUAGENT_COMMIT} && git checkout -q FETCH_HEAD && \
    git submodule update --init --recursive --depth 1
# amdsmi: default COPY pre-staged; optional re-extract from tarball
COPY assets/amd_smi_lib/x86_64/RHEL9/lib/ /staged-amdsmi/      # default source
RUN if [ "${AMDSMI_FROM_TARBALL}" = "1" ]; then \
        mkdir -p /staged-amdsmi && curl -s "${ROCM_TARBALL_URL}" | \
        tar -xz -C /tmp/smi --wildcards --no-anchored 'amdsmi.h' 'libamd_smi.so*' && \
        cp -a /tmp/smi/lib/libamd_smi.so* /staged-amdsmi/ && \
        cp -a /tmp/smi/include/amd_smi/amdsmi.h /staged-amdsmi/ ; fi
# inject into clone vendor tree (derive SO version dynamically; fix symlinks) + apply patches
COPY patch/gpuagent/ /patches/
RUN <inject amdsmi into /src/sw/nic/third-party/rocm/amd_smi_lib/{include,x86_64/lib}> && \
    for p in /patches/*.patch; do git -C /src apply "$p"; done && \
    cd /src && make -C sw/nic/gpuagent all
# (abseil ranlib step runs here too if not expressible as a patch)

# runtime stage (existing) — replace ADD ./gpuagent/gpuctl with:
COPY --from=gpuagent-build /src/sw/nic/build/x86_64/sim/bin/gpuagent /home/amd/bin/gpuagent
COPY --from=gpuagent-build /src/sw/nic/build/x86_64/sim/bin/gpuctl   /home/amd/bin/gpuctl
```
- Keep the existing runtime amdsmi override (`ADD ./libamd_smi.so.<ver>`) sourced from `assets/`
  via `build_prep_docker.sh` — same single tarball source as the builder stage.

### Step 4 — docker/Makefile + build_prep wiring
- Pass `--build-arg GPUAGENT_REPO --build-arg GPUAGENT_COMMIT --build-arg AMDSMI_FROM_TARBALL
  --build-arg ROCM_TARBALL_URL --build-arg GPUAGENT_BUILDER_BASE_IMAGE` through to the image build.
- `build_prep_docker.sh`: drop the gpuagent tarball-extraction branch (binaries now built in-stage);
  keep amdsmi `.so` staging from `assets/`.
- The mock/sriov images still need `gpuagent_mock` / `gpuagent_gim` — builder stage must build
  those targets too (`make -C sw/nic/gpuagent all` builds all three) and the respective Dockerfiles
  COPY the matching binary.

### Step 5 — Retire host-side targets
Remove from `Makefile`/`Makefile.compile`/`Makefile.build`: `gpuagent-compile`,
`gpuagent-build-only`, `gpuagent-build`, `gpuagent-asset-copy`, `gpuagent-shell`,
`amdsmi-inject-gpuagent`, `docker-tarball-amdsmi`. Keep `amdsmi-from-tarball` + `amdsmi-sync-assets`
(used to refresh `assets/` when `AMDSMI_FROM_TARBALL=1` produces new libs).

## Verification

- `git submodule status` no longer lists gpuagent; `.gitmodules` has only `libgimsmi`.
- `make docker` (default, `AMDSMI_FROM_TARBALL=0`) builds the release image with NO 10.5 GB
  download; gpuagent binary present at `/home/amd/bin/gpuagent` in the image.
- `make docker AMDSMI_FROM_TARBALL=1 ROCM_TARBALL_URL=...` refreshes amdsmi from the tarball.
- In-image `gpuagent` starts with no `undefined symbol` (amdsmi 26.4.0 + abseil link both resolved).
- sha256 of the in-image `libamd_smi.so.26.4.0` == the one gpuagent linked against (single source).
- Mock smoke test: run `:mock` image, `curl /metrics`, confirm `amd_*` metrics.

## Risks / Limitations

- **In-image submodule clone** pulls grpc/abseil/boost/protobuf/libzmq recursively over HTTPS —
  network-heavy and subject to upstream availability at build time (no longer vendored).
- **abseil link fix** must be expressible as a `git apply` patch OR a build-step `ranlib`; if the
  upstream gpuagent layout shifts, the patch may need a refresh (it's pinned to GPUAGENT_COMMIT).
- **Build context size** — default path COPYs `assets/amd_smi_lib/.../RHEL9/lib/` into the build;
  fine. Avoid COPYing the whole repo into the builder stage.
- **Non-default toggle** (`AMDSMI_FROM_TARBALL=1`) streams 10.5 GB in-build; intended for occasional
  tarball/ROCm bumps only.
- **Other images** (azure, ainic, sriov-ub22) that relied on prebuilt gpuagent assets need the same
  multi-stage treatment or must keep consuming `assets/` — scope this plan to the RHEL9 release
  (+mock/sriov) images first.
- **Reproducibility** depends on the GPUAGENT_COMMIT pin; cloning branch tip would drift.

---

## Implementation Outcomes (2026-06-16)

**Branch:** `build/gpuagent-source-in-docker`
**Commits:**
- `e2c301cbf` — main build change (submodule removal, multi-stage Dockerfile, patch, plan)
- `80a06462e` — fix: align `docker-cicd` recipe with `docker`

### Build discoveries during execution

1. **gpuagent `ABS_DIR` is hardcoded** to `/usr/src/github.com/ROCm/gpu-agent/sw`, so the clone
   must live at `/usr/src/github.com/ROCm/gpu-agent` — not a generic `/src`. Added `ENV GPUAGENT_SRC`
   pointing to this path.

2. **`gopkglist` required before compile** — the gpuagent repo-level `Makefile` has a `gopkglist`
   target that `go install`s the proto plugins (`protoc-gen-gogofast`, etc.) needed for the Go-proto
   generation step (`gen-protos`). Must run `make gopkglist` before `make -C sw/nic/gpuagent all`.

3. **Shallow fetch needs full 40-char SHA** — `git fetch --depth 1 origin <short-sha>` is rejected
   by GitHub; the 40-char SHA is required. `GPUAGENT_COMMIT` updated accordingly.

4. **Legacy docker builder + `FROM ${ARG}` after first stage** — the classic builder (BuildKit off)
   resolves `FROM ${BASE_IMAGE}` only if `BASE_IMAGE` is declared as a global ARG before the **first**
   `FROM`. Added `ARG BASE_IMAGE=...` at the top of the file (pre-first-FROM) alongside
   `GPUAGENT_BUILDER_BASE_IMAGE`.

5. **abseil static archives ship with empty armap** — the gpuagent build produces abseil `.a` files
   with no symbol index (armap = 0 entries). RHEL9 `ld` requires the index (Ubuntu tolerated the
   missing index). Fixed by `find sw/nic/build -name '*.a' -exec ranlib {} \;` in the RUN step;
   the `--start-group` patch is also applied for the circular-dependency ordering.

6. **gpuagent unstripped = 84MB** — the old `gpuagent-asset-copy` ran `strip`; the new build stage
   did not. Added `strip` of all three C++ binaries (`gpuagent`, `gpuagent_gim`, `gpuagent_mock`)
   in the same RUN, reducing from 84MB to 28MB in-image.

7. **amdsmi version-agnostic symlinks** — `entrypoint.sh` hardcoded `LD_PRELOAD=libamd_smi.so.26`.
   Both the Dockerfile runtime symlinks and the entrypoint were made version-agnostic: the Dockerfile
   derives `libamd_smi.so.<maj>` and `libamd_smi.so` dynamically from the staged file; entrypoint
   uses `LD_PRELOAD=/home/amd/lib/libamd_smi.so`. No hardcoded `26`/`26.4.0` literals remain.

8. **`build_prep` deref'd symlink bloat** — `cp -vf libamd_smi.so.*` copies the `.so.26` symlink
   as a real 3.2MB file, doubling the staged size. Changed to `cp -vfL libamd_smi.so.*.*.*` (only
   the real versioned file), saving 3MB from the image layer.

9. **`docker-cicd` recipe drift** — `docker-cicd` in `docker/Makefile` was missing all four
   gpuagent build-args (`GPUAGENT_BUILDER_BASE_IMAGE`, `GPUAGENT_REPO`, `GPUAGENT_COMMIT`,
   `AMDSMI_FROM_TARBALL`) and the parent wasn't forwarding `AMDSMI_FROM_TARBALL`/`ROCM_TARBALL_URL`
   to the sub-make. Fixed in `80a06462e`.

### Image size

| Image | Size |
|---|---|
| Previous (prebuilt submodule gpuagent) | ~1.0 GB |
| After pivot, before strip fix | 1.25 GB (+84MB unstripped gpuagent) |
| After strip + dedup fix | **1.13 GB** |

### Hardware verification (dev GPU host)

Full 20-minute soak on a real AMD GPU host (`/dev/kfd` + `/dev/dri/renderD128`):

| Check | Result |
|---|---|
| Container Up | ✅ 22 min, no restart |
| `server` (PID 1) alive throughout | ✅ |
| `gpuagent` (PID 7) alive, PID stable | ✅ unchanged across all 4 samples |
| `/metrics` returns GPU metrics | ✅ 147 metrics, real values |
| Sample: `gpu_average_package_power` | 58W (AMD Radeon Graphics, gpu_id=0) |
| Sample: `gpu_clock` (system) | 1700 MHz |
| Log errors (panic/fatal/ErrZeroGPUs) | 0 |
| t+5m / t+10m / t+15m / t+20m | all green |

Host quirks encountered:
- Port 5000 occupied by local `registry:2` — do NOT publish to 5000.
- `docker run -p` fails with iptables MASQUERADE error (broken bridge) — verify via `docker exec curl`.
- `--group-add render` by name fails; use numeric GIDs (render=992, video=44).
- Metric prefix is `gpu_*` for bare `docker run` (no custom config mounted); `amd_*` prefix
  requires `MetricsFieldPrefix` set in `/etc/metrics/config.json`.

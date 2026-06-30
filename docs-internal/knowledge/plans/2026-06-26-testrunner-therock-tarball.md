# Test Runner: TheRock Tarball Support + MI350P Power-Cap RVS Folder Selection

- **Date:** 2026-06-26
- **Author:** yan.sun3@amd.com
- **Related PR(s):** #1410
- **Related issue(s) / JIRA:** N/A

## Context

The test runner Docker image previously installed ROCm and RVS exclusively via
`repo.radeon.com` dnf packages. This blocked use of TheRock nightly tarballs
(7.13+) since nightly builds are not published to the dnf repo.

MI350P ships in two TDP variants (450W and 600W) with separate RVS recipe
folders (`MI350P-450W`, `MI350P-600W`). The test runner had no mechanism to
select the correct folder — it would fall through to model-agnostic recipes.

## Approach

### TheRock tarball support (Dockerfile + Makefile)

Two new build args gate the install paths:
- `ROCM_TARBALL_URL` — empty = legacy `repo.radeon.com` dnf path (unchanged);
  non-empty = download TheRock tarball and prune to RVS runtime lib set.
- `RVS_TARBALL_URL` — empty = `dnf install rocm-validation-suite`; non-empty =
  download the RVS pre-built tarball and extract to `/opt/rocm`.

Two new scripts:
- `install-rocm-tarball.sh`: downloads TheRock tarball, prunes to libs required
  by RVS (`libamdhip64`, `librocblas`, `libhipblaslt`, `libhiprand`,
  `libroctx64`, `librocroller`, `libhsa-runtime64`, `librocm-core`, `libomp`,
  `rocm_sysdeps`), sets up `/opt/rocm` symlink and `libdrm_amdgpu.so` symlinks.
- `install-rvs-tarball.sh`: downloads RVS tarball, extracts to `/opt/rocm`,
  patches `level_1` confs to remove package-validation rcqt actions.

Both `Dockerfile` and `agfhc-dockerfile` use a single `if/else` RUN block;
legacy dnf path is fully preserved when both tarball URLs are empty.

### AGFHC v1.32.0

v1.32.0 auto-detects TheRock for ROCm ≥7.11. Required changes:
- Pass `--rocm-ver $(ROCM_MAJOR_MINOR)` — pruned TheRock install lacks
  `.info/version` so the installer aborts without an explicit version.
- Pass `--no-sign-check` to avoid GPG errors in container builds.
- Drop `rocm-smi-lib` from tarball-path dnf install (provided by TheRock prune).

### MI350P RVS folder selection (`pkg/testrunner/`)

New `case "MI350P":` in the existing RVS model-folder switch calls
`getMI350PRVSFolder(amdSMIPath)` which:
1. Runs `amd-smi static --limit --json`.
2. Reads `ppt0.socket_power_limit` for every GPU.
3. Returns `MI350P-600W` if all GPUs ≥ 600 W; otherwise `MI350P-450W`.
4. On any error logs one line and returns `MI350P-450W` as safe fallback.

### Alternatives considered

- **Selective tarball extraction** — TheRock has no per-package download for
  nightly builds; full tarball required. Docker layer cache amortizes cost.
- **sysfs fallback for power cap** — `power1_cap` in hwmon is available without
  `amd-smi`, but the JSON API is canonical. Can be added as secondary fallback.

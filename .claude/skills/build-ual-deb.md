---
name: build-ual-deb
description: Build the UAL Debian package (`amdgpuifoe-exporter_<ubuntu>_amd64.deb`) via `make debpkg-ual`. Host-only target — depends on `amdexporter` having been built first inside the build container. Trigger when the user asks to "build ual deb", "make ual deb", "debpkg-ual", or wants a packaged Ubuntu install of the IFOE exporter. Symmetric to the existing build-ual-rpm skill.
---

# Build UAL Debian package

Build the `amdgpuifoe-exporter_*~<ubuntu>_amd64.deb` produced by
`make debpkg-ual` (Makefile.package:264-275). The recipe modifies a
systemd unit and `gpuagent.conf` in-place, runs `debpkg`, then runs
`git checkout` to revert the in-place edits — a common
non-destructive pattern in this repo's packaging.

**Inputs:**
- Build container ID/name (e.g. `<user>_exporter-bld` from
  `make docker-shell`). Default: pick the running container matching
  `*_exporter-bld`.
- Username inside that container (defaults to invoking user).
- `UBUNTU_VERSION` env (e.g. `22.04` or `24.04`) — passed through to
  the recipe; defaults to whatever the build container is built for
  (`Makefile.package:259-262` shows the rename targets).

---

## Step 1 — Pre-req: amdexporter binary

`debpkg-ual` calls `debpkg` which depends on `amdexporter` (the
exporter binary) existing under `bin/`. Build it inside the container
if missing:

```bash
docker exec -u <username> -w /usr/src/github.com/ROCm/device-metrics-exporter <container> bash -lc \
  "export PATH=/usr/local/go/bin:\$HOME/go/bin:\$PATH; export GOPATH=\$HOME/go; \
   ls bin/amd-metrics-exporter >/dev/null 2>&1 || make amdexporter"
```

The same `PATH`+`GOPATH` rule as `build-ual-rpm` and `update-ual-proto` —
without them the recursive `/bin/sh` doesn't see `go`.

## Step 2 — Build the deb

`debpkg-ual` is host-only in spirit (it calls `dpkg-deb`), but the
`dpkg-deb` binary IS available inside the build container, so the same
`docker exec` invocation works:

```bash
docker exec -u <username> -w /usr/src/github.com/ROCm/device-metrics-exporter <container> bash -lc \
  "export PATH=/usr/local/go/bin:\$HOME/go/bin:\$PATH; export GOPATH=\$HOME/go; \
   make debpkg-ual UBUNTU_VERSION_NUMBER=${UBUNTU_VERSION:-22.04}"
```

Expected output: `bin/amdgpuifoe-exporter_<ubuntu>_amd64.deb` (~340 MB).

## Step 3 — Verify

```bash
ls -la bin/amdgpuifoe-exporter_*_amd64.deb
dpkg-deb -I bin/amdgpuifoe-exporter_*_amd64.deb | head -20
dpkg-deb -c bin/amdgpuifoe-exporter_*_amd64.deb | grep -E '(amd-metrics-exporter|gpuagent|gpuctl)$'
```

Confirm the systemd unit `amd-metrics-exporter.service` is present and
the `ExecStart` line contains `-monitor-ifoe=true` (Makefile.package:266
sed-rewrites this for UAL builds).

```bash
dpkg-deb --fsys-tarfile bin/amdgpuifoe-exporter_*_amd64.deb \
  | tar -xO ./usr/lib/systemd/system/amd-metrics-exporter.service \
  | grep ExecStart
```

## Pitfalls

- **`git checkout` is part of the recipe — three transient dirty files after a failed run.**
  Before `dpkg-deb` runs, `debpkg-ual` sed-edits these three files in
  place (Makefile.package:266-275); after success it `git checkout`s
  them back. If anything between the sed and the checkout fails
  (build error, stale `.git/index.lock`, interrupted run), the files
  stay dirty:
  - `debian/DEBIAN/control` — `Package: amdgpu-exporter` →
    `amdgpuifoe-exporter`
  - `debian/usr/lib/systemd/system/amd-metrics-exporter.service` —
    adds `-monitor-ifoe=true` to `ExecStart`
  - `debian/usr/local/etc/metrics/gpuagent.conf` — appends
    `PLATFORM=helios`

  These are transient build scaffolding, **not real code changes**.
  If you see *only* these three files modified after a failed
  `make debpkg-ual`, revert with targeted `git checkout <path>` —
  do not assume the user authored them. Check whether the `.deb`
  actually landed in `bin/` first; the build often succeeded and
  only the cleanup `git checkout` failed.

- **debpkg-ual is host-only but gated on container output.**
  `debpkg-ual` itself does not invoke docker (it runs `dpkg-deb`,
  `cp`, `strip`, etc. on the host), but it errors at
  Makefile.package:182-185 if `build/assets/${OS}/profilerlibs/` is
  missing. That dir is populated by `libcopy-assets-UBUNTU22` via a
  host-side `cp -rvf build/rocprofilerdeplib/
  build/assets/${OS}/profilerlibs/` (Makefile.compile:158).

  Shortcut when `build/rocprofilerdeplib/` already exists from a
  prior run — skip the container entirely and stage the assets
  manually:
  ```
  mkdir -p build/assets/UBUNTU22/profilerlibs/
  cp -rf build/rocprofilerdeplib/ build/assets/UBUNTU22/profilerlibs/
  ```
  Then `make debpkg-ual` runs pure-host with no TTY needed.
- **`make debpkg-clean` before re-building.** The `debian/` staging
  directory accumulates content from previous runs and can poison the
  next build (extra files end up inside the .deb).
- **TTY trap on dependent libcopy targets.** `debpkg-ual` does not
  itself need a TTY, but if you run a parent target (e.g.
  `make pkg`) that chains in `libcopy-assets-RHEL9` or its Debian
  cousin, that step fails non-interactively — see `build-ual-rpm`
  Step 1 for the manual `docker run` workaround.
- **Mount path is `…/ROCm/device-metrics-exporter`.** Host repo lives
  under `…/pensando/…` but the container mounts at `…/ROCm/…`.
- **Package rename.** Lines 258-262 of Makefile.package rename
  `amdgpuifoe-exporter_*~22.04_amd64.deb` to drop the `~`. If your
  caller is matching the `~<ubuntu>~` form, look at the post-rename
  filename instead.

## Cross-references

- RPM counterpart: `.claude/skills/build-ual-rpm.md`
- Install the produced deb: `.claude/skills/install-ual-package.md`
- Repo conventions / do-not-touch list: `CLAUDE.md`

# CLAUDE.md — Repo conventions for Claude / codie / agentq

Guardrails for AI agents working in this repo. Read once at session
start; re-read when in doubt.

## Files Claude must NOT edit

### `entrypoint.sh` (repo root)

**Owner: external jobd / CI infrastructure (outside this repo).**

This file is the dev-container helper that the internal jobd
pipeline uses to start `dockerd`, bind-mount the source tree, set
`vm.max_map_count`, and exec the build command. It is **NOT** the
runtime entrypoint of any released container — that is
`docker/entrypoint.sh`, which is what `docker/Dockerfile.*-release`
ADD's into the image.

**Why this matters:** during GPUOP-723, the wrong file was edited
twice:
- Commit `03f76cbc2` modified root `entrypoint.sh` thinking it was
  the runtime entrypoint — silently no-op for the container, but
  broke jobd CI semantics on the dev-container side.
- Commit `79336c35d` then deleted root `entrypoint.sh` on the
  reasoning that "no Dockerfile references it" — true for the
  in-repo Dockerfiles, but jobd CI consumes it directly outside
  the repo, so the deletion broke CI.

**Rule:** treat `entrypoint.sh` (root) as **read-only**. If a
runtime container behavior change is needed, edit
`docker/entrypoint.sh` instead. Never delete `entrypoint.sh`
(root). Never assume "no Dockerfile references it" means it's
unused — external CI does.

If you genuinely need to change root `entrypoint.sh`, get
explicit owner approval first (the jobd team).

### Other files Claude should not casually delete

- Anything under `vendor/` — generated; use `go mod vendor`.
- Anything under `pkg/*/gen/` and `pkg/amdgpu/mock_gen/` — protobuf
  / mock generated; regenerate via the proto build, don't hand-edit.
- Auto-generated `.pb.go` / `_grpc.pb.go` files anywhere.

## Two `entrypoint.sh` files — runtime vs CI

| File | Purpose | Edited by |
|------|---------|-----------|
| `entrypoint.sh` (repo root) | Dev-container helper for **internal jobd CI** — starts dockerd, mounts source, runs build | jobd team only — **off-limits** |
| `docker/entrypoint.sh` | Runtime entrypoint of the released container — handles `MONITOR_GPU`, `ENABLE_IFOE`, launches gpuagent + exporter | This is the one to edit for runtime-container behavior |

`docker/Dockerfile.*-release` all do `ADD ./entrypoint.sh
/home/amd/tools/entrypoint.sh` with build context `docker/`, so
they pick up `docker/entrypoint.sh`. Root `entrypoint.sh` is never
ADD'd into a release image.

## Two `.job.yml` files

| File | Purpose |
|------|---------|
| `.job.yml` (repo root) | Defines build targets (deb/rpm/docker images, helm charts, test runners) for jobd CI |
| `asset-build/.job.yml` | Defines the asset-push target — depends on the build targets in the root `.job.yml` and uploads bundled artifacts to the internal hourly-build store via `asset-push builds hourly-device-metrics-exporter` |

When enabling a new artifact in the asset bundle: (1) add the
build target's name to `asset-build/.job.yml` `build-dependencies`,
(2) make sure `asset-build/exporter-asset-push.sh` actually copies
the artifact into `BUNDLE_DIR` so `upload()` picks it up.

## Build context gotchas

- `make docker-cicd` (Makefile:396) runs from repo root and shells
  out to `make -C docker docker-cicd TOP_DIR=$(CURDIR)`. The build
  context inside `docker/Makefile` is `.` (i.e., `$TOP_DIR/docker/`),
  NOT the repo root. So `ADD ./<file>` in `docker/Dockerfile.*` is
  always relative to `docker/`, not the repo root.
- `docker/build_prep_docker.sh` stages assets (gpuagent, libamd_smi,
  rocpctl, etc.) into `docker/` from `assets/`/`bin/` BEFORE the
  docker build runs.
- `docker/build_post_docker.sh` removes those staged files AFTER
  the build. **If you run `docker build` manually without
  `make docker-cicd`, you must run `build_post_docker.sh` yourself
  to clean up — otherwise `docker/` is left with a pile of untracked
  binaries.**

## Internal vs public registries

- Default `DOCKER_REGISTRY` in `dev.env` is `registry.test.pensando.io:5000`
  (internal). Not reachable from arbitrary build hosts.
- For local builds outside the lab network, override `BASE_IMAGE`:
  `docker build --build-arg BASE_IMAGE=registry.access.redhat.com/ubi9/ubi-minimal:9.6 ...`
- `make libcopy-assets-*` targets pull from the internal registry
  too — no clean override exists; needs lab access.

## UAL / IFOE specifics

- `assets/gpuagent_ual.bin.gz` is the IFOE-aware gpuagent. Refresh
  via `UAL_REMOTE_SERVER=sw-dev3.pensando.io ./scripts/update_ual_assets.sh <version>`.
- `assets/gpuagent_static.bin.gz` is the non-IFOE prebuilt — older,
  smaller, no UAL gRPC.
- `docker/build_prep_docker.sh` defaults to bundling
  `gpuagent_ual.bin.gz` (`UAL=1` by default; `UAL=0` falls back).
- IFOE Prometheus metrics use the `amd_ifoe_*` prefix (the `amd_`
  comes from `MetricsFieldPrefix` in `example/config.json:4`).
  Never grep `/metrics` for `^ifoe_` — always `^amd_ifoe_`.
- The exporter logs to **`/var/log/exporter.log` inside the
  container**, NOT stdout. `docker logs <ctr>` is empty for the
  Go logger output. To inspect runtime logs:
  `docker exec <ctr> cat /var/log/exporter.log`. Stderr from the
  entrypoint script (e.g., `WARN: invalid ENABLE_IFOE=...`) IS
  captured by `docker logs`.
- IFOE end-to-end correctness validated by
  `.claude/commands/validate-ifoe-exporter.md` (slash command
  `/validate-ifoe-exporter <user>@<host>`) — cross-checks
  Prometheus output against `gpuctl show ual` for count + UUID +
  per-entity field parity.

## Test-port-allocation pattern

The exporter listens on TCP 5000. On shared lab hosts the hourly
build's container usually has 5000 bound. Test cases must
auto-discover a free port (5000–5100 range) — see
`.claude/skills/test-port-allocation.md`.

## Claude / agent shared knowledge in repo (REQUIRED for all PRs)

Any engineer using Claude (or codie / agentq) in this repo MUST contribute reusable artifacts back into the repo so the next session inherits the work. The convention is fixed:

- **`.claude/commands/`** — slash commands available to any engineer running Claude Code in this repo (e.g., `/validate-ifoe-exporter`).
- **`.claude/skills/`** — reusable Claude skills (`build-ual-rpm`, `build-ual-deb`, `build-exporter-image`, `deploy-exporter-container`, `install-ual-package`, `validate-ifoe-exporter`, `test-port-allocation`, …). Note `.gitignore` has a coarse `build*` rule that catches `.claude/skills/build-*.md` — use `git add -f` for those, or fix the `.gitignore` rule to anchor to root (`/build*`) in a separate cleanup.

**No `.claude/memory/` directory in the repo.** Engineer-private memory lives only at `~/.claude/projects/.../memory/` (per-engineer, never in git). When a memory file captures something the team should know, **fold it into the matching skill or this CLAUDE.md** rather than mirroring as a separate `.claude/memory/<file>.md`. Avoids duplication; keeps one source of truth per topic. Examples:

- A new "build gotcha for `make foo`" → add a Pitfalls section to `.claude/skills/build-foo.md`.
- A new "do not edit X" rule → extend the "Files Claude must NOT edit" section above in this CLAUDE.md.
- A new convention that doesn't fit any existing skill → either create a new skill for it or add a section to CLAUDE.md.

**Never** put credentials, host IPs, per-engineer absolute paths, or any other engineer-private state in tracked `.claude/` files — those belong only in engineer-private memory at `~/.claude/projects/.../memory/`. If an engineer's memory file mixes shared conventions with private state, split it before folding the shared portion into a skill or CLAUDE.md.

If you (Claude) added something useful to your engineer-private memory during a session, surface that in the PR review checklist: "Should this fold into a skill or CLAUDE.md so the team picks it up?".

## Codie pipeline files

- `.codie/project.yaml` — team-shared codie project config (in repo)
- `.codie/skills/` — team-shared codie skills (in repo)
- `codie-config.yaml` — per-engineer config with absolute paths;
  **gitignored**. Each engineer creates their own pointing to
  `.codie/`.
- `PROGRESS-*.md` — per-session codie state files; **gitignored**.

## When in doubt

Ask before deleting any file you didn't create in this session.
"No reference found in the repo" is not sufficient evidence that
a file is unused — internal tooling outside this repo (jobd CI,
mirror processes, asset-push, helm-release pipelines) consumes
files directly.

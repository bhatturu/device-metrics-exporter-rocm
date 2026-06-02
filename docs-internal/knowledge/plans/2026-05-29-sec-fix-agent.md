# Fix CVE: Go toolchain bump + /sec-fix skill

- **Date:** 2026-05-29
- **Author:** praveen
- **Related PR(s):** TBD

## Context

Trivy CI scan flagged 11 HIGH + 3 MEDIUM stdlib CVEs across all four Go
binaries (`amd-metrics-exporter`, `amdgpuhealth`, `metricsclient`,
`amd-test-runner`) — fixed by bumping the Go toolchain from `1.25.8` →
`1.25.10`. Added a `/sec-fix` skill+agent to automate future CVE triage and
toolchain bumps from a CI scan log URL.

## Approach

- Bump `go`/`toolchain` directives in `go.mod`, update `tools/base-image/Dockerfile`, `dev.env`, `box.rb` to new Go version.
- Rebuild and push the build container image with the new Go toolchain.
- Run `go mod tidy && go mod vendor` inside the dev container.
- New skill `.claude/skills/sec-fix/` + agent `.claude/agents/sec-fix-agent.md`: parses trivy gobinary log, auto-fixes stdlib CVEs, defers API-breaking third-party bumps to the user.
- Local verification via `trivy rootfs` on rebuilt binaries.

### Alternatives considered

- Fixing only `go.mod` without rebuilding the builder image — rejected: `go build` inside the container would still use the old baked-in Go binary.

## Scope

- **In scope:** stdlib HIGH/MEDIUM CVEs via toolchain bump; build container rebuild; `/sec-fix` skill+agent.
- **Out of scope:** `gpuctl` (upstream repo); AWS SDK v2 major version jump (API-breaking, deferred); OS-level base-image CVEs.

## Validation

- `git diff go.mod` — `go` and `toolchain` directives updated.
- `make unit-test` inside new container — no regressions.
- `.claude/bin/scan-gobinary-local.sh bin/` after `make all` — 0 HIGH/CRITICAL findings.

## Risks and rollback

- Unit test failure after bump → revert `go.mod`, `go.sum`, `vendor/`, `Dockerfile`, `dev.env`, `box.rb`.
- AWS SDK MEDIUM CVEs deferred — marked `TODO-CVE` in agent output for follow-up.

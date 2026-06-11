---
name: sec-fix
description: Use when fixing CVE vulnerabilities reported by trivy in the four Go binaries (amd-metrics-exporter, amdgpuhealth, metricsclient, amd-test-runner). Provide a CI scan log URL. Handles Go toolchain bump, build container image update, go mod vendor, and local trivy verification.
agent: sec-fix-agent
---

# sec-fix — CVE Remediation for Go Binaries

Orchestrates end-to-end CVE remediation for the four Go binaries produced by this repo. Delegates all execution to the `sec-fix-agent`.

## What this skill does

- Parses a trivy CI scan URL using `.claude/bin/parse-trivy-gobinary.sh` and groups findings by severity (HIGH/CRITICAL auto-fixed; MEDIUM/LOW flagged as `TODO-CVE` and deferred).
- Determines the minimum Go toolchain version that closes all HIGH/CRITICAL stdlib CVEs and presents a fix plan for user confirmation before touching any file.
- Updates four files atomically: `go.mod` (go + toolchain directives), `tools/base-image/Dockerfile` (Go download URL), `dev.env` (DOCKER_BUILDER_TAG increment), and `box.rb` (from tag).
- Builds and pushes the new builder image if the updated tag is not already in the registry, then runs `go mod tidy && go mod vendor` inside the dev container to update the vendor tree.
- Offers a post-fix verification checklist and optional local trivy re-scan; emits rollback instructions if unit tests fail.

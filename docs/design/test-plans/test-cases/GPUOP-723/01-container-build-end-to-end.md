---
topology: single-node
timeout: 1800
pass_criteria: "make docker-cicd exits 0 and produces docker/device-metrics-exporter-latest.tar.gz; image retagged with user-specific identifier for test hosts"
stability: stable
retries: 0
validation_groups: [post-test]
---

# Container build end-to-end (CI parity + user-specific retag)

## Purpose

Verify `make docker-cicd` (the same target the CI job
`build-device-metrics-exporter-docker-ubi9.6` runs in `.job.yml:67-69`)
builds the new IFOE-enabled image successfully with all Sub-task changes
applied (libmnl/libnl3 install, libamd_smi 26.3.0 ADD, ENV ENABLE_IFOE=true,
real gpuagent in assets), produces the expected artifact tarball under
`docker/`, and can be retagged with a user-specific identifier so test
deployments do not collide with the `latest` hourly build already running
on shared test hosts.

## Category

positive

## Prerequisites

- A clone of this repo with `bugfix/exporter-ifoe` branch checked out and all Sub-tasks merged. The test runs from the repo root regardless of where the repo is cloned (`$REPO_ROOT` is wherever you ran `git clone`; nothing in this test case assumes a specific filesystem location).
- `assets/gpuagent_ual.bin.gz` is the real (non-mock) UAL gpuagent (S1)
- `assets/amd_smi_lib/x86_64/RHEL9/lib/libamd_smi.so.26.3.0` present (S2)
- `docker/libamd_smi.so.26.3.0` present in build context (S2)
- Docker daemon running on build host
- `$USER` set in env (used to derive the user-specific tag)

## Parameters

| Parameter | Default | Range | Description |
|-----------|---------|-------|-------------|
| `EXPORTER_IMAGE_TAG` | `latest` | any docker tag | Tag baked into the image at build time (matches `.job.yml`) |
| `HOURLY_TAG_LABEL` | `latest` | any string | Value of the `HOURLY_TAG` label embedded in the image (matches `.job.yml`) |
| `USER_TAG` | `${USER}-GPUOP-723` | any docker tag | User-specific retag applied after build to avoid hourly-tag clobber on test hosts |

## Steps

1. **[host]** From the repo root: `git status && git rev-parse --abbrev-ref HEAD`
   Expected: clean working tree; current branch is `bugfix/exporter-ifoe`
2. **[host]** `HOURLY_TAG_LABEL=GPUOP-723 EXPORTER_IMAGE_TAG=latest make docker-cicd 2>&1 | tee /tmp/GPUOP-723-build.log`
   Expected: exit code 0; build log shows `--label HOURLY_TAG=GPUOP-723` passed to `docker build`
3. **[host]** `ls -la docker/device-metrics-exporter-latest.tar.gz`
   Expected: artifact present, recent mtime, non-zero size (matches `.job.yml:72` artifact path)
4. **[host]** `EXPORTER_IMAGE_NAME=$(grep '^EXPORTER_IMAGE_NAME' Makefile | head -1 | awk -F'?= ' '{print $2}'); EXPORTER_IMAGE_TAG=$(grep '^EXPORTER_IMAGE_TAG' Makefile | head -1 | awk -F'?= ' '{print $2}'); echo "$EXPORTER_IMAGE_NAME:$EXPORTER_IMAGE_TAG"`
   Expected: prints `device-metrics-exporter:latest` (the tag baked in by `make docker-cicd` defaults)
5. **[host]** `docker images device-metrics-exporter:latest --format '{{.Repository}}:{{.Tag}} {{.ID}} {{.CreatedSince}}'`
   Expected: image listed with current ID and recent creation
6. **[host]** `docker inspect --format '{{index .Config.Labels "HOURLY_TAG"}}' device-metrics-exporter:latest`
   Expected: prints `GPUOP-723` (matches what we passed in step 2)
7. **[host]** `USER_TAG="${USER}-GPUOP-723"; docker tag device-metrics-exporter:latest device-metrics-exporter:${USER_TAG} && docker images device-metrics-exporter:${USER_TAG}`
   Expected: retagged image listed; this is the tag used by all downstream test cases (TC02–TC05, TC09–TC11)
8. **[host]** `docker run --rm --entrypoint /bin/bash device-metrics-exporter:${USER_TAG} -c "ls -la /home/amd/lib/libamd_smi*"`
   Expected: lists `libamd_smi.so.26 -> libamd_smi.so.26.3.0` and the actual `libamd_smi.so.26.3.0`
9. **[host]** `docker run --rm --entrypoint /bin/bash device-metrics-exporter:${USER_TAG} -c "ldconfig -p | grep -E 'libmnl|libnl-3|libnl-genl'"`
   Expected: at least three matching lines (libmnl, libnl-3, libnl-genl-3) — confirms S3
10. **[host]** `docker inspect --format '{{json .Config.Env}}' device-metrics-exporter:${USER_TAG} | tr ',' '\n' | grep ENABLE_IFOE`
    Expected: shows `ENABLE_IFOE=true` (confirms S5)

(All commands run from the repo root; substitute your own clone location. No path in this test case is engineer- or workstation-specific.)

## Expected Result

`make docker-cicd` completes without error. `docker/device-metrics-exporter-latest.tar.gz` produced (matches CI artifact). Image carries `HOURLY_TAG` label, libamd_smi 26.3.0, IFOE runtime libs, and `ENABLE_IFOE=true` ENV. User-specific retag (`device-metrics-exporter:${USER}-GPUOP-723`) created so downstream tests on shared hosts won't collide with `latest` hourly builds.

## Failure Indicators

- `make docker-cicd` non-zero exit code
- `microdnf install` failure for libmnl / libnl3 / libnl3-cli (S3 broken)
- `docker/device-metrics-exporter-latest.tar.gz` not produced (artifact contract broken)
- Missing `libamd_smi.so.26.3.0` in `/home/amd/lib/` (S2 broken)
- `HOURLY_TAG` label missing or has unexpected value
- ENABLE_IFOE env not set (S5 broken)
- Image size delta > 100 MB vs prior release (suggests unintended bloat)

## Cleanup

- `docker rmi device-metrics-exporter:latest device-metrics-exporter:${USER}-GPUOP-723` (only if rebuilding)
- `rm /tmp/GPUOP-723-build.log`
- Keep `docker/device-metrics-exporter-latest.tar.gz` for distribution to test hosts (TC02 onward)

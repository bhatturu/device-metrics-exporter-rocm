---
name: build-exporter-image
description: Build the device-metrics-exporter container image via `cd docker && make docker`, optionally save it to a transferable `.tar.gz` via `make docker-save`. Trigger when the user asks to "build exporter image", "build container image", "make docker exporter", "save exporter image for testbed", or when an agent needs a locally-built image to feed deploy-exporter-container on a host without registry access.
---

# Build the device-metrics-exporter container image

The image build target is `docker` inside `docker/Makefile` (lines
10-18) — confusingly named (you'd expect `docker-build`). It runs
`build_prep_docker.sh`, then `docker build` against
`Dockerfile.exporter-release`, then `build_post_docker.sh`. To ship the
image to a testbed without a registry, follow with `docker-save`
(lines 41-49) which writes `<image-name>-<tag>.tar.gz`.

**Inputs:**
- `EXPORTER_IMAGE_TAG` — defaults to whatever the parent Makefile
  computes (commit-derived). Override for test images, e.g. `dev-ifoe-cont`.
- `BUILD_CONTAINER` — the build container the parent Makefile uses.
  In practice the host-side `docker build` doesn't need this — only
  the binary build inside the build container does. Step 0 covers it.

This is the **GPU exporter** image. SR-IOV (`docker-sriov`,
`docker-sriov-save`) and AINIC (`docker-ainic`) targets exist in the
same Makefile and follow the identical shape — substitute the target
name if you need them.

---

## Step 0 — Pre-req: binaries built

`docker/build_prep_docker.sh` expects `bin/amd-metrics-exporter`,
`bin/metricsclient`, `bin/amdgpuhealth`, and the prebuilt assets under
`assets/`. Build them inside the build container if missing:

```bash
docker exec -u <username> -w /usr/src/github.com/ROCm/device-metrics-exporter <container> bash -lc \
  "export PATH=/usr/local/go/bin:\$HOME/go/bin:\$PATH; export GOPATH=\$HOME/go; \
   make amdexporter"
```

(`amdexporter` aliases the binary build per `Makefile:370` —
`amdexporter: metricsclient amdgpuhealth`.)

## Step 1 — Build image (host-side)

```bash
make -C docker docker EXPORTER_IMAGE_TAG=${EXPORTER_IMAGE_TAG:-dev-$(date +%s)}
```

Internally invokes
`docker build --build-arg BASE_IMAGE=$(RHEL_BASE_MIN_IMAGE) ...
-t $(EXPORTER_IMAGE) . -f Dockerfile.exporter-release` from inside
`docker/`. `EXPORTER_IMAGE` resolves to
`$(DOCKER_REGISTRY)/$(EXPORTER_IMAGE_NAME):$(EXPORTER_IMAGE_TAG)` —
default registry/name come from the parent Makefile.

Verify:

```bash
docker images | grep device-metrics-exporter
```

## Step 2 — (optional) Save to tar.gz for testbed transfer

```bash
make -C docker docker-save EXPORTER_IMAGE_TAG=${EXPORTER_IMAGE_TAG}
```

Output: `docker/<exporter-image-name>-<tag>.tar.gz` (path is
`docker/` because docker-save writes relative to its own Makefile's CWD).

Transfer + load on testbed:

```bash
scp docker/<exporter-image-name>-<tag>.tar.gz <user>@<host>:/tmp/
ssh <user>@<host> "gunzip -c /tmp/<exporter-image-name>-<tag>.tar.gz | docker load"
```

After load, `deploy-exporter-container` can reference the loaded image
with `IMAGE_REGISTRY=<the-registry-tag-from-image>`.

## Pitfalls

- **Target is `docker`, not `docker-build`.** Easy to miss in
  autocomplete; intuitive name is wrong.
- **Always run from repo root via `make -C docker docker`** — calling
  `make docker` directly from `docker/` works too but loses parent-Makefile
  variable inheritance (`EXPORTER_IMAGE_NAME`, `DOCKER_REGISTRY`,
  `RHEL_BASE_MIN_IMAGE`, `ROCM_VERSION`).
- **`docker-save` writes to `docker/` CWD.** Not `bin/`. Easy to
  overlook when fishing for the artifact path.
- **`docker-cicd` adds `--label HOURLY_TAG=$(HOURLY_TAG_LABEL)`.** Use
  it for hourly-build parity; use plain `docker` for ad-hoc test images.
- **Mock vs release.** `docker-mock` (`Dockerfile.exporter-mock-release`)
  gives a binary that returns canned GPU state. Do **not** use it for
  IFOE validation tests — it doesn't exercise UAL.

## Cross-references

- Container Makefile: `docker/Makefile`
- Deploy the produced image: `deploy-exporter-container` skill
- Image transfer pattern + lab firmware/image flow: `docs/installation/docker.md`

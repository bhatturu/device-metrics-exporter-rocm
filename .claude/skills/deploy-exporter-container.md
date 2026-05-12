---
name: deploy-exporter-container
description: Deploy the device-metrics-exporter container on a target host via `docker run`, with dynamic port allocation, configmap mount, and IFOE env var control. Trigger when an agent needs to bring up a fresh exporter for a test (TC01/TC02 of GPUOP-723 and any future container test plan), or when the user asks "deploy exporter", "start exporter container", "run device-metrics-exporter on <host>". Use install-ual-package for the bare-metal .deb/.rpm path instead.
---

# Deploy device-metrics-exporter container

Bring up a fresh `device-metrics-exporter` container on a target host
using the deployment shape documented in
`docs/installation/docker.md`. Always pick a free host port (the
hourly-build container on shared lab hosts owns 5000 — see the
`test-port-allocation` skill).

**Inputs:**
- `TARGET` — `<user>@<host>` (SSH).
- `IMAGE_TAG` — exporter image tag (e.g. `v1.5.0` or a local build tag).
- `IMAGE_REGISTRY` — defaults to `rocm`. Use `local` for an image
  loaded from a `docker save` tar (see `build-exporter-image` skill).
- `TEST_LABEL` — short token for container-name isolation
  (e.g. `tc01-ifoe-on`).
- `IFOE` — `on` (default), `off` (sets `ENABLE_IFOE=false`), or
  `auto` (let the container detect). Used by GPUOP-722 negative tests.
- `CONFIGMAP_PATH` (optional) — local path to a `config.json` to mount
  at `/etc/metrics/config.json` inside the container. If omitted, the
  baked-in default config is used.

---

## Step 0 — Port discovery

Run on the target. See `test-port-allocation` skill for rationale —
this is the same script.

```bash
ssh ${TARGET} bash -lc "'
  EXPORTER_PORT=\$(comm -23 \\
    <(seq 5000 5100 | sort) \\
    <(ss -tnl 2>/dev/null | awk \"NR>1 {print \\\$4}\" | sed \"s/.*://\" | sort -u) \\
    | head -1)
  echo EXPORTER_PORT=\$EXPORTER_PORT
  [ -n \"\$EXPORTER_PORT\" ] || { echo no-free-port; exit 1; }
'"
```

Capture `EXPORTER_PORT` from output. Fail loudly if empty.

## Step 1 — (optional) Push configmap

If `CONFIGMAP_PATH` is set:

```bash
scp ${CONFIGMAP_PATH} ${TARGET}:/tmp/ame-${TEST_LABEL}-config.json
```

Edit the remote copy in-place to set `ServerPort` to the port the
container listens on internally (always **5000** — host-side mapping
is what `EXPORTER_PORT` sets).

## Step 2 — docker run

```bash
ssh ${TARGET} "
  docker rm -f ame-${TEST_LABEL}-\$USER 2>/dev/null
  docker run -d \\
    --name ame-${TEST_LABEL}-\$USER \\
    --device=/dev/kfd --device=/dev/dri \\
    -v /sys:/sys:ro \\
    $( [ -n \"${CONFIGMAP_PATH}\" ] && echo \"-v /tmp/ame-${TEST_LABEL}-config.json:/etc/metrics/config.json:ro\" ) \\
    -p ${EXPORTER_PORT}:5000 \\
    $( case ${IFOE} in
         off)  echo \"-e ENABLE_IFOE=false\" ;;
         on)   echo \"-e ENABLE_IFOE=true\"  ;;
         auto) ;;
       esac ) \\
    ${IMAGE_REGISTRY:-rocm}/device-metrics-exporter:${IMAGE_TAG}
"
```

Container name `ame-${TEST_LABEL}-${USER}` matches the test-port-allocation
isolation pattern so concurrent runs don't collide.

## Step 3 — Readiness check

Poll `/metrics` for up to 30s:

```bash
for i in $(seq 1 30); do
  if ssh ${TARGET} "curl -fsS http://localhost:${EXPORTER_PORT}/metrics > /dev/null"; then
    echo "exporter ready on port ${EXPORTER_PORT}"
    break
  fi
  sleep 1
done
```

If the loop exits without success, dump container logs and FAIL:

```bash
ssh ${TARGET} "docker logs ame-${TEST_LABEL}-\$USER" | tee deploy_logs_${TEST_LABEL}.txt
```

## Step 4 — Hand-off

Emit a one-line summary the caller (workflow, validate-ifoe-exporter
skill) can parse:

```
DEPLOY_OK host=<host> port=${EXPORTER_PORT} container=ame-${TEST_LABEL}-${USER} image=${IMAGE_REGISTRY:-rocm}/device-metrics-exporter:${IMAGE_TAG} ifoe=${IFOE}
```

## Cleanup

```bash
ssh ${TARGET} "docker stop ame-${TEST_LABEL}-\$USER && docker rm ame-${TEST_LABEL}-\$USER"
[ -n "${CONFIGMAP_PATH}" ] && ssh ${TARGET} "rm -f /tmp/ame-${TEST_LABEL}-config.json"
```

## Pitfalls

- **Hourly build owns 5000** on shared lab hosts. Step 0 is non-optional.
- **`/sys:/sys:ro` is required** for inband-ras (see `docs/installation/docker.md`).
- **Container always listens on 5000 internally**, regardless of host port.
  Configmap `ServerPort` must stay 5000 unless you change the container `-p` map too.
- **`--device=/dev/kfd --device=/dev/dri`** — without these the GPU
  fields silently emit zeros and tests pass meaninglessly.

## Cross-references

- Port discovery: `.claude/skills/test-port-allocation.md`
- Container shape: `docs/installation/docker.md`, `docs/configuration/docker.md`
- Validation gate to run after readiness: `validate-ifoe-exporter` skill
- GPUOP-723 test plan that exercises this path: `docs/design/test-plans/GPUOP-723-test-plan.md`

---
name: test-port-allocation
description: Reusable pattern for picking a free TCP port when running the device-metrics-exporter container in test cases. Avoids hard-coded port 5000 collisions on shared test hosts.
---

# Dynamic test-port allocation for device-metrics-exporter container tests

## Why

The exporter listens on TCP **5000** by default (see
`docs/installation/docker.md`). On shared CI / lab hosts, port 5000 is
frequently already bound by:

- The hourly-build `device-metrics-exporter` container running as part
  of normal monitoring on the host
- Another engineer's parallel test run
- An unrelated service that happens to use 5000

Test cases that hard-code `-p 5000:5000` and `curl http://localhost:5000/metrics`
will conflict in these conditions and either fail to start the container
or scrape the wrong (older) instance's metrics. Pick a free port at
test invocation time instead.

## Pattern

### Parameter

Every test case that runs the exporter container declares an
`EXPORTER_PORT` parameter with no default — the test must resolve
it dynamically:

```markdown
## Parameters

| Parameter | Default | Description |
|-----------|---------|-------------|
| `EXPORTER_PORT` | (auto, see Steps) | host TCP port forwarded to container 5000; auto-discovered to avoid collision with the hourly build |
```

### Step 0 — Port discovery (always the first step)

```bash
# Pick the first free port in the test range 5000-5100
EXPORTER_PORT=$(
  comm -23 \
    <(seq 5000 5100 | sort) \
    <(ss -tnl 2>/dev/null | awk 'NR>1 {print $4}' | sed 's/.*://' | sort -u) \
  | head -1
)
echo "EXPORTER_PORT=${EXPORTER_PORT}"
[ -n "${EXPORTER_PORT}" ] || { echo "no free port in 5000-5100"; exit 1; }
```

Expected: a single integer in `[5000, 5100]` printed; non-empty.

### Step N — docker run uses the discovered port

```bash
docker run -d \
  --name "ame-${TEST_LABEL}-${USER}" \
  --device=/dev/kfd --device=/dev/dri \
  -v /sys:/sys:ro \
  -p "${EXPORTER_PORT}:5000" \
  device-metrics-exporter:${IMAGE_TAG}
```

Note `-p "${EXPORTER_PORT}:5000"` — the **container** still listens on
5000 internally; only the host-side port is dynamic.

### Step M — scrape uses the discovered port

```bash
curl -s "http://localhost:${EXPORTER_PORT}/metrics"
```

### Cleanup

The container name embeds `${USER}` (for the same isolation reason as
the port — see `docs/design/test-plans/GPUOP-723-test-plan.md` Testbed
section) so cleanup targets only this run's container:

```bash
docker stop "ame-${TEST_LABEL}-${USER}" && docker rm "ame-${TEST_LABEL}-${USER}"
```

## When to use

- Any test case that runs `device-metrics-exporter` on a shared host
- Any new test plan generated via `/codie:story-test-plan` for this repo
- Any system / regression test that asserts on `/metrics`

## When you can skip

- Single-engineer dev box where you know nothing else is bound to 5000
- Hermetic CI runners that get a fresh sandbox per job (still
  recommended for hygiene, but optional)

## Two patterns, two contexts

This skill covers **test deployments** that bring up a fresh container.
For **validating an existing deployment** (e.g., the running hourly
build or a packaged install), the source of truth is the on-host
config:

```bash
ssh "${TEST_USER}@${TEST_HOST}" "cat /etc/metrics/config.json" \
  | jq -r '.ServerPort'
```

The `/validate-ifoe-exporter` slash command
(`.claude/commands/validate-ifoe-exporter.md`) uses this pattern —
it never assumes 5000 either, but reads the live `ServerPort`
instead of discovering a free one. Use whichever applies:

- **Bringing up your own test container**: discover a free port
  (top of this skill).
- **Validating someone else's running exporter**: read its port from
  `/etc/metrics/config.json`.

When TC02 of GPUOP-723 deploys a fresh container *and* validates with
the harness, it does both: discover a free port, then write that port
into `/etc/metrics/config.json` so the harness picks it up.

## Cross-references

- Default port + deployment shape: `docs/installation/docker.md`
- Container deployment configmap pattern: `docs/configuration/docker.md`
- IFOE metrics validation harness:
  `.claude/commands/validate-ifoe-exporter.md`
- First story to adopt this pattern: GPUOP-723
  (`docs/design/test-plans/test-cases/GPUOP-723/`)

#!/usr/bin/env bash
# SR-IOV exporter stress test runner.
# Runs on the target host (10.30.60.190). Assumes the SR-IOV image tarball has
# been loaded as `device-metrics-exporter-sriov:latest` and that gpuagent has a
# Unix socket on the host or inside the container.
#
# What it does:
#   1. Starts the SR-IOV exporter container (host network, /dev/gim-sim0 passthrough).
#   2. Verifies /metrics endpoint responds and contains SR-IOV-related fields.
#   3. Runs concurrent curl + gpuctl loops for DURATION seconds.
#   4. Counts request successes / failures, samples SR-IOV-tagged metric values,
#      and emits a final summary.
#
# Env knobs (override on invocation):
#   IMAGE       container image to run (default: device-metrics-exporter-sriov:latest)
#   PORT        exporter port to scrape (default: 5000)
#   DURATION    stress duration in seconds (default: 1800 = 30 min)
#   CURL_PAR    parallel curl loops (default: 8)
#   GPUCTL_PAR  parallel gpuctl loops (default: 2)
#   RUN_DIR     directory for run artifacts (default: /tmp/sriov-stress-<ts>)
set -uo pipefail

IMAGE="${IMAGE:-device-metrics-exporter-sriov:latest}"
PORT="${PORT:-5050}"
DURATION="${DURATION:-1800}"
CURL_PAR="${CURL_PAR:-4}"
GPUCTL_PAR="${GPUCTL_PAR:-2}"
TS="$(date +%Y%m%d-%H%M%S)"
RUN_DIR="${RUN_DIR:-/tmp/sriov-stress-${TS}}"
CONTAINER_NAME="${CONTAINER_NAME:-dme-sriov-stress}"
CONFIG_DIR="${CONFIG_DIR:-${RUN_DIR}/etc-metrics}"

# AGA_SMI_LAZY_INIT verification: the SR-IOV exporter is supposed to release
# /dev/gim-smi0 between scrapes. We poll the device on the host with lsof to
# confirm. Defaults are tuned for the leto host (passwordless sudo, lsof).
FD_PROBE_PAR="${FD_PROBE_PAR:-1}"
FD_PROBE_INTERVAL_MS="${FD_PROBE_INTERVAL_MS:-200}"
IDLE_CHECK_SECS="${IDLE_CHECK_SECS:-5}"
SMI_DEV="${SMI_DEV:-/dev/gim-smi0}"
# lsof must run as root to see holders of a 0600 root:root char device.
# Two ways to grant that without editing sudoers:
#   - SUDO_PASS=<pw>     -> we wrap as `sudo -S` and feed the password via stdin
#   - LSOF_CMD=<custom>  -> caller supplies a full command (e.g. for passwordless sudo or fuser)
# Default falls back to `sudo -n` (works only if NOPASSWD is configured).
if [[ -n "${LSOF_CMD:-}" ]]; then
  :
elif [[ -n "${SUDO_PASS:-}" ]]; then
  LSOF_CMD="sudo -S -p '' lsof ${SMI_DEV}"
  LSOF_STDIN="${SUDO_PASS}"
else
  LSOF_CMD="sudo -n lsof ${SMI_DEV}"
fi
LSOF_STDIN="${LSOF_STDIN:-}"
SKIP_FD_PROBE="${SKIP_FD_PROBE:-0}"
# verdict thresholds for the fd-probe metric — see plan file for rationale
FD_HELD_PCT_MAX="${FD_HELD_PCT_MAX:-50}"
FD_STREAK_MS_MAX="${FD_STREAK_MS_MAX:-5000}"

mkdir -p "${RUN_DIR}"
exec > >(tee -a "${RUN_DIR}/run.log") 2>&1

echo "[$(date -Is)] === SR-IOV stress test starting ==="
echo "  IMAGE=${IMAGE}"
echo "  PORT=${PORT}"
echo "  DURATION=${DURATION}s"
echo "  CURL_PAR=${CURL_PAR}  GPUCTL_PAR=${GPUCTL_PAR}  FD_PROBE_PAR=${FD_PROBE_PAR}"
echo "  SMI_DEV=${SMI_DEV}  LSOF_CMD='${LSOF_CMD}'  (stdin: ${LSOF_STDIN:+<redacted>}${LSOF_STDIN:-<empty>})"
echo "  RUN_DIR=${RUN_DIR}"
echo

# NOTE: do NOT auto-remove the container in the trap; we want the summary
# section below to be able to inspect it. Only kill load loops on signal.
KEEP_CONTAINER="${KEEP_CONTAINER:-1}"
cleanup() {
  echo "[$(date -Is)] cleanup: stopping load loops"
  if [[ -n "${LOAD_PIDS:-}" ]]; then
    # shellcheck disable=SC2086
    kill ${LOAD_PIDS} 2>/dev/null || true
  fi
  if [[ "${KEEP_CONTAINER}" != "1" ]]; then
    docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true
  fi
}
trap cleanup INT TERM

# --- 1. Start container ---------------------------------------------------
docker rm -f "${CONTAINER_NAME}" >/dev/null 2>&1 || true

DEV_ARGS=()
for d in /dev/gim-sim0 /dev/gim-smi0; do
  [[ -e "$d" ]] && DEV_ARGS+=(--device "$d")
done
[[ -d /dev/dri ]] && DEV_ARGS+=(-v /dev/dri:/dev/dri)

# Prepare config dir with our chosen port (host port 5000 is often taken on
# this lab host by the local docker registry).
mkdir -p "${CONFIG_DIR}"
cat > "${CONFIG_DIR}/config.json" <<EOF
{
  "ServerPort": ${PORT},
  "CommonConfig": {
    "MetricsFieldPrefix": "amd_",
    "HealthService": { "Enable": true, "PollingRate": "30s" },
    "Logging": { "Level": "info", "MaxFileSizeMB": 10, "MaxBackups": 3, "MaxAgeDays": 7 }
  }
}
EOF

echo "[$(date -Is)] docker run -d --network host --privileged ${DEV_ARGS[*]} -v ${CONFIG_DIR}:/etc/metrics --name ${CONTAINER_NAME} ${IMAGE}"
docker run -d --network host --privileged \
  "${DEV_ARGS[@]}" \
  -v "${CONFIG_DIR}":/etc/metrics \
  --name "${CONTAINER_NAME}" \
  "${IMAGE}" >/dev/null

echo "[$(date -Is)] waiting for /metrics on :${PORT} ..."
ready=0
for i in $(seq 1 30); do
  if curl -fsS --max-time 2 "http://[::1]:${PORT}/metrics" >/dev/null 2>&1; then
    ready=1; break
  fi
  sleep 2
done
if [[ "${ready}" -ne 1 ]]; then
  echo "[$(date -Is)] FATAL: exporter never came up"
  echo "--- container logs ---"
  docker logs "${CONTAINER_NAME}" 2>&1 | tail -80
  exit 1
fi
echo "[$(date -Is)] exporter is up"

# --- 1b. AGA_SMI_LAZY_INIT precheck --------------------------------------
# Confirm lazy-init env is set inside the container AND that we can probe the
# device on the host. If we can't probe, set SKIP_FD_PROBE=1 and continue —
# don't fail the whole stress run for an environmental gap.
LAZY_ENV=$(docker exec "${CONTAINER_NAME}" sh -c 'echo "${AGA_SMI_LAZY_INIT:-<unset>}"' 2>/dev/null || echo "?")
echo "[$(date -Is)] AGA_SMI_LAZY_INIT in container: ${LAZY_ENV}"

if [[ "${SKIP_FD_PROBE}" != "1" ]]; then
  if ! command -v lsof >/dev/null 2>&1; then
    echo "[$(date -Is)] lsof missing on host -> SKIP_FD_PROBE=1"
    SKIP_FD_PROBE=1
  elif ! timeout 5 bash -c "${LSOF_CMD}" <<<"${LSOF_STDIN}" >/dev/null 2>&1; then
    # lsof returns 1 with no output when nothing holds the device — that's
    # the expected "no holder" path. We need to distinguish that from a real
    # failure (sudo prompt, missing device, perms). Check rc explicitly.
    rc=$?
    if [[ $rc -ne 1 ]]; then
      echo "[$(date -Is)] lsof precheck failed (rc=${rc}) -> SKIP_FD_PROBE=1"
      SKIP_FD_PROBE=1
    fi
  fi
fi
[[ -e "${SMI_DEV}" ]] || { echo "[$(date -Is)] ${SMI_DEV} missing -> SKIP_FD_PROBE=1"; SKIP_FD_PROBE=1; }
echo "[$(date -Is)] SKIP_FD_PROBE=${SKIP_FD_PROBE}"

# --- 2. Field inventory ---------------------------------------------------
SNAP="${RUN_DIR}/metrics-initial.txt"
curl -fsS "http://[::1]:${PORT}/metrics" > "${SNAP}"
echo "[$(date -Is)] initial /metrics snapshot: $(wc -l < "${SNAP}") lines"

# Metrics that exercise the SR-IOV / GIM path. We don't fail if some are
# missing (firmware/driver may not surface every counter on every host) — we
# only assert that at least one shows up, and we log which ones are present.
SRIOV_PATTERNS=(
  'deployment_mode="hypervisor"'   # SR-IOV PF view tag (definitive SR-IOV marker)
  'gpu_partition_id='              # partition id label always emitted in SR-IOV
  'gpu_compute_partition_type='    # MPx compute partition type
  'gpu_memory_partition_type='     # MPx memory partition type
  '^amd_gpu_ecc_'                  # ECC counters via GIM SMI
  '^amd_gpu_temperature'           # temperature (GIM SMI path)
  '^amd_gpu_memory_'               # memory metrics
  '^amd_gpu_power_'                # power metrics
  '^amd_gpu_clock'                 # clocks via GIM SMI
)
echo
echo "=== SR-IOV-relevant field presence in initial snapshot ==="
for pat in "${SRIOV_PATTERNS[@]}"; do
  c=$(grep -cE "${pat}" "${SNAP}" || true)
  printf '  %-30s -> %d lines\n' "${pat}" "${c}"
done | tee "${RUN_DIR}/sriov-field-inventory.txt"

total_sriov=$(grep -cE "$(IFS='|'; echo "${SRIOV_PATTERNS[*]}")" "${SNAP}" || true)
if [[ "${total_sriov}" -eq 0 ]]; then
  echo "[$(date -Is)] FATAL: no SR-IOV-relevant metrics in /metrics"
  exit 2
fi
echo "[$(date -Is)] total SR-IOV-relevant metric lines: ${total_sriov}"

# --- 3. Background load loops --------------------------------------------
CURL_OK="${RUN_DIR}/curl.ok"
CURL_FAIL="${RUN_DIR}/curl.fail"
GPUCTL_OK="${RUN_DIR}/gpuctl.ok"
GPUCTL_FAIL="${RUN_DIR}/gpuctl.fail"
FD_TOTAL="${RUN_DIR}/fd.total"
FD_HELD="${RUN_DIR}/fd.held"
FD_STREAK="${RUN_DIR}/fd.streak_max_ms"
FD_HOLDERS_LOG="${RUN_DIR}/fd-holders.log"
: > "${CURL_OK}"; : > "${CURL_FAIL}"
: > "${GPUCTL_OK}"; : > "${GPUCTL_FAIL}"
: > "${FD_TOTAL}"; : > "${FD_HELD}"; : > "${FD_STREAK}"
: > "${FD_HOLDERS_LOG}"

curl_loop() {
  local id="$1" end="$2"
  local n=0 fails=0
  while [[ "$(date +%s)" -lt "${end}" ]]; do
    # --retry handles transient connect-refused under heavy concurrent fanout
    # (the exporter listen backlog can momentarily fill). --retry-all-errors
    # so we retry on ECONNREFUSED, not just HTTP 5xx.
    if curl -fsS --max-time 10 --retry 3 --retry-delay 0 --retry-all-errors \
            "http://[::1]:${PORT}/metrics" -o /dev/null 2>/dev/null; then
      n=$((n+1))
    else
      fails=$((fails+1))
    fi
  done
  echo "${id} ${n}" >> "${CURL_OK}"
  echo "${id} ${fails}" >> "${CURL_FAIL}"
}

gpuctl_loop() {
  local id="$1" end="$2"
  local n=0 fails=0
  while [[ "$(date +%s)" -lt "${end}" ]]; do
    if timeout 15 docker exec "${CONTAINER_NAME}" /home/amd/bin/gpuctl show gpu > "${RUN_DIR}/gpuctl-${id}.last" 2>&1; then
      n=$((n+1))
    else
      fails=$((fails+1))
    fi
    sleep 1
  done
  echo "${id} ${n}" >> "${GPUCTL_OK}"
  echo "${id} ${fails}" >> "${GPUCTL_FAIL}"
}

# Polls /dev/gim-smi0 on the host. In AGA_SMI_LAZY_INIT=1 mode the device
# should be UNHELD most of the time, with only brief holds while a scrape is
# in flight. We tally total/held and track the longest contiguous held streak
# (in milliseconds) — that streak is the single most diagnostic number, since
# a regression to persistent-init shows up as a streak ≈ entire test duration.
fd_probe_loop() {
  local id="$1" end="$2"
  local total=0 held=0
  local streak_ms=0 streak_max=0
  local interval_s
  # bash arithmetic is integer-only; build "0.NNN" form for sleep
  interval_s=$(awk -v ms="${FD_PROBE_INTERVAL_MS}" 'BEGIN{printf "%.3f", ms/1000.0}')
  while [[ "$(date +%s)" -lt "${end}" ]]; do
    total=$((total+1))
    local out rc
    out=$(timeout 3 bash -c "${LSOF_CMD}" <<<"${LSOF_STDIN}" 2>/dev/null) || rc=$? && rc=${rc:-0}
    if [[ -n "${out}" ]]; then
      held=$((held+1))
      streak_ms=$((streak_ms + FD_PROBE_INTERVAL_MS))
      if [[ "${streak_ms}" -gt "${streak_max}" ]]; then
        streak_max=${streak_ms}
      fi
      # Log just the data lines from lsof (skip the COMMAND header) so we
      # have evidence of WHO held it and when. One line per holding sample.
      printf '%s\t%s\n' "$(date -Is)" \
        "$(printf '%s' "${out}" | awk 'NR>1{print; exit}')" \
        >> "${FD_HOLDERS_LOG}"
    else
      streak_ms=0
    fi
    sleep "${interval_s}"
  done
  echo "${id} ${total}" >> "${FD_TOTAL}"
  echo "${id} ${held}"  >> "${FD_HELD}"
  echo "${id} ${streak_max}" >> "${FD_STREAK}"
}

END=$(( $(date +%s) + DURATION ))
echo
echo "[$(date -Is)] starting load: ${CURL_PAR} curl + ${GPUCTL_PAR} gpuctl, until $(date -d @${END} -Is)"

LOAD_PIDS=""
for i in $(seq 1 "${CURL_PAR}");   do curl_loop   "c${i}" "${END}" & LOAD_PIDS+=" $!"; done
for i in $(seq 1 "${GPUCTL_PAR}"); do gpuctl_loop "g${i}" "${END}" & LOAD_PIDS+=" $!"; done
if [[ "${SKIP_FD_PROBE}" != "1" ]]; then
  for i in $(seq 1 "${FD_PROBE_PAR}"); do fd_probe_loop "f${i}" "${END}" & LOAD_PIDS+=" $!"; done
fi

# --- 4. Periodic sampler --------------------------------------------------
SAMPLE_FILE="${RUN_DIR}/samples.tsv"
echo -e "ts\trss_kib\tcontainer_status\tmetrics_lines\tsriov_lines" > "${SAMPLE_FILE}"
while [[ "$(date +%s)" -lt "${END}" ]]; do
  now="$(date -Is)"
  # Use `timeout` everywhere so a hung docker call doesn't freeze the sampler
  # (and therefore the whole script). docker stats is notoriously slow under
  # cgroup-v1 + heavy concurrent exec load, so we skip it entirely.
  rss=$(timeout 5 docker exec "${CONTAINER_NAME}" sh -c 'ps -o rss= -p $(pgrep -f /home/amd/bin/server | head -1)' 2>/dev/null | tr -d ' \n' || echo "?")
  status=$(timeout 5 docker inspect -f '{{.State.Status}}' "${CONTAINER_NAME}" 2>/dev/null || echo "?")
  mfile="${RUN_DIR}/metrics-$(date +%H%M%S).txt"
  if timeout 8 curl -fsS --max-time 5 --retry 2 --retry-all-errors "http://[::1]:${PORT}/metrics" -o "${mfile}" 2>/dev/null; then
    mlines=$(wc -l < "${mfile}")
    slines=$(grep -cE "$(IFS='|'; echo "${SRIOV_PATTERNS[*]}")" "${mfile}" || true)
  else
    mlines=0; slines=0
  fi
  printf '%s\t%s\t%s\t%s\t%s\n' "${now}" "${rss:-?}" "${status}" "${mlines}" "${slines}" >> "${SAMPLE_FILE}"
  # keep only most recent N metrics dumps to save disk
  ls -1t "${RUN_DIR}"/metrics-*.txt 2>/dev/null | tail -n +20 | xargs -r rm -f
  sleep 60
done

# shellcheck disable=SC2086
wait ${LOAD_PIDS} 2>/dev/null || true

# --- 4b. Post-load idle fd check -----------------------------------------
# All load loops have exited — no more scrapes are in flight. In lazy-init
# mode the device MUST be released at this point. We probe three times spaced
# 200ms apart to avoid a single-sample flake from race timing.
IDLE_CHECK_RESULT="SKIP"
IDLE_CHECK_FILE="${RUN_DIR}/fd-idle-check.txt"
if [[ "${SKIP_FD_PROBE}" != "1" ]]; then
  echo "[$(date -Is)] sleeping ${IDLE_CHECK_SECS}s for idle fd check ..."
  sleep "${IDLE_CHECK_SECS}"
  IDLE_CHECK_RESULT="RELEASED"
  : > "${IDLE_CHECK_FILE}"
  for n in 1 2 3; do
    out=$(timeout 5 bash -c "${LSOF_CMD}" <<<"${LSOF_STDIN}" 2>/dev/null || true)
    printf 'sample %d @ %s\n%s\n---\n' "${n}" "$(date -Is)" "${out:-<empty>}" \
      >> "${IDLE_CHECK_FILE}"
    if [[ -n "${out}" ]]; then
      IDLE_CHECK_RESULT="HELD"
    fi
    sleep 0.2
  done
  echo "[$(date -Is)] idle fd check: ${IDLE_CHECK_RESULT}"
fi

# --- 5. Summary ----------------------------------------------------------
CURL_TOTAL=$(awk '{s+=$2} END {print s+0}' "${CURL_OK}")
CURL_BAD=$(awk '{s+=$2} END {print s+0}' "${CURL_FAIL}")
GPUCTL_TOTAL=$(awk '{s+=$2} END {print s+0}' "${GPUCTL_OK}")
GPUCTL_BAD=$(awk '{s+=$2} END {print s+0}' "${GPUCTL_FAIL}")

CRASHES=$(docker inspect -f '{{.RestartCount}}' "${CONTAINER_NAME}" 2>/dev/null || echo "?")
STATUS=$(docker inspect -f '{{.State.Status}}' "${CONTAINER_NAME}" 2>/dev/null || echo "?")

# fd probe aggregates
FD_TOTAL_SUM=$(awk '{s+=$2} END {print s+0}' "${FD_TOTAL}" 2>/dev/null)
FD_HELD_SUM=$(awk '{s+=$2}  END {print s+0}' "${FD_HELD}"  2>/dev/null)
FD_STREAK_MAX=$(awk '{if($2+0>m)m=$2+0} END {print m+0}' "${FD_STREAK}" 2>/dev/null)
if [[ "${FD_TOTAL_SUM:-0}" -gt 0 ]]; then
  FD_HELD_PCT=$(awk -v h="${FD_HELD_SUM}" -v t="${FD_TOTAL_SUM}" 'BEGIN{printf "%.1f", (h*100.0)/t}')
else
  FD_HELD_PCT="0.0"
fi

{
  echo "=== SR-IOV stress summary ($(date -Is)) ==="
  echo "Duration:            ${DURATION}s"
  echo "Image:               ${IMAGE}"
  echo "Container status:    ${STATUS}  restarts=${CRASHES}"
  echo "/metrics requests:   ok=${CURL_TOTAL}  fail=${CURL_BAD}"
  echo "gpuctl requests:     ok=${GPUCTL_TOTAL}  fail=${GPUCTL_BAD}"
  echo "Initial SR-IOV lines: ${total_sriov}"
  echo "Samples file:        ${SAMPLE_FILE}"
  echo "AGA_SMI_LAZY_INIT:   ${LAZY_ENV:-?} (in container)"
  if [[ "${SKIP_FD_PROBE}" == "1" ]]; then
    echo "fd probe:            SKIPPED (lsof unavailable, no sudo, or device missing)"
    echo "idle fd check:       SKIPPED"
  else
    echo "fd probe:            total=${FD_TOTAL_SUM} held=${FD_HELD_SUM} (held%=${FD_HELD_PCT}) max_streak_ms=${FD_STREAK_MAX}"
    echo "idle fd check:       ${IDLE_CHECK_RESULT}"
    echo "fd holders log:      ${FD_HOLDERS_LOG}"
    echo "fd idle check log:   ${IDLE_CHECK_FILE}"
  fi
  echo
  echo "--- per-loop curl results ---"
  paste "${CURL_OK}" "${CURL_FAIL}" | awk '{printf "  %s: ok=%s fail=%s\n",$1,$2,$4}'
  echo "--- per-loop gpuctl results ---"
  paste "${GPUCTL_OK}" "${GPUCTL_FAIL}" | awk '{printf "  %s: ok=%s fail=%s\n",$1,$2,$4}'
  echo
  echo "--- last 5 samples ---"
  tail -5 "${SAMPLE_FILE}"
  echo
  echo "--- last 40 lines of container log ---"
  docker logs --tail 40 "${CONTAINER_NAME}" 2>&1
} | tee "${RUN_DIR}/summary.txt"

# verdict
fail=0
if [[ "${STATUS}" != "running" ]]; then fail=1; echo "FAIL: container not running"; fi
if [[ "${CRASHES}" != "0" && "${CRASHES}" != "?" ]]; then fail=1; echo "FAIL: container restarted (${CRASHES})"; fi
if [[ "${CURL_BAD}" -gt 0 ]];   then fail=1; echo "FAIL: ${CURL_BAD} curl failures"; fi
if [[ "${GPUCTL_BAD}" -gt 0 ]]; then fail=1; echo "FAIL: ${GPUCTL_BAD} gpuctl failures"; fi
if [[ "${SKIP_FD_PROBE}" != "1" ]]; then
  # held% comparison via awk (bash can't do float compares)
  held_over=$(awk -v p="${FD_HELD_PCT}" -v m="${FD_HELD_PCT_MAX}" 'BEGIN{print (p+0 > m+0)?1:0}')
  if [[ "${held_over}" -eq 1 ]]; then
    fail=1; echo "FAIL: fd held% ${FD_HELD_PCT} > ${FD_HELD_PCT_MAX} (lazy-init regression?)"
  fi
  if [[ "${FD_STREAK_MAX:-0}" -gt "${FD_STREAK_MS_MAX}" ]]; then
    fail=1; echo "FAIL: fd held continuously for ${FD_STREAK_MAX}ms (> ${FD_STREAK_MS_MAX}ms threshold)"
  fi
  if [[ "${IDLE_CHECK_RESULT}" == "HELD" ]]; then
    fail=1; echo "FAIL: ${SMI_DEV} still held ${IDLE_CHECK_SECS}s after load stopped"
  fi
fi
if [[ "${fail}" -eq 0 ]]; then echo "PASS"; fi
exit "${fail}"

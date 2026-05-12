---
name: validate-ifoe-exporter
description: Validate that the `amdgpuifoe-exporter` Prometheus `/metrics` endpoint correctly reflects live UAL device, station, and port state from `gpuctl` on a remote host. Authoritative IFOE metric correctness gate. Trigger when an agent (testbed-ops, qa-lead) needs to validate IFOE metrics after install, deploy, or upgrade — and when the user asks "validate ifoe exporter", "check ifoe metrics", or names a host to validate. Skill form of the `/validate-ifoe-exporter` slash command — both share this body; the skill is the canonical form for agentq dispatch.
---

# Validate IFOE Exporter (skill form)

Validate that the `amdgpuifoe-exporter` Prometheus metrics endpoint
correctly reflects the live UAL device, station, and port state reported
by `gpuctl` on a remote system.

**Inputs:**
- `TARGET` — `<user>@<host>` for SSH (e.g. `prshanmug@mheliosr-1b114-f01-1.mnb.dcgpu`).

**Output:** `IFOE_EXPORTER_<HOSTNAME>_SUMMARY.md` in CWD with a PASS/FAIL verdict.

Read `ServerPort` from `/etc/metrics/config.json` on the target before
issuing any curl commands — never assume port 5000 (see project memory
`reference_test_host_ifoe_epic.md` and `test-port-allocation` skill for
the rationale). All metric greps use the `amd_*` prefix from
`MetricsFieldPrefix` — never `^ifoe_` or `^gpu_ual_ifoe_` (see
`project_metric_prefix.md`).

---

## Step 1 — Preflight checks

Run in parallel on the target:

1. SSH connectivity: `ssh ${TARGET} "echo connected"`
2. Package presence: `ssh ${TARGET} "sudo apt list --installed 2>/dev/null | grep amdgpuifoe"`
3. ServerPort: `ssh ${TARGET} "cat /etc/metrics/config.json"` — extract `.ServerPort`

Report SSH reachable yes/no, package name+version (or "NOT INSTALLED"),
and the metrics port. Stop on SSH or install failure.

## Step 2 — Collect data

```bash
ssh ${TARGET} "mkdir -p ~/validate_ifoe_logs && \
  curl -s localhost:<PORT>/metrics > ~/validate_ifoe_logs/metrics.txt && \
  sudo gpuctl show ual device  -y > ~/validate_ifoe_logs/ual_device.txt  2>&1 && \
  sudo gpuctl show ual station -y > ~/validate_ifoe_logs/ual_station.txt 2>&1 && \
  sudo gpuctl show ual port    -y > ~/validate_ifoe_logs/ual_port.txt    2>&1 && \
  echo done"

scp -r ${TARGET}:~/validate_ifoe_logs/ ./validate_ifoe_logs/
```

## Step 3 — Count validation

| Source | Count |
|---|---|
| `ual_device.txt`  | `grep -c '^- spec:'` |
| `ual_station.txt` | `grep -c '^- spec:'` |
| `ual_port.txt`    | `grep -c '^- spec:'` |
| `metrics.txt`     | `amd_ifoe_total_devices`, `amd_ifoe_total_stations`, `amd_ifoe_total_ports` |

PASS if all three counts match exactly. FAIL with delta if not.
Internal consistency: stations == devices × `capability.numualstation`,
ports == devices × `capability.numnetworkport`.

## Step 4 — UUID and label validation

Decode `spec.id` byte arrays from `gpuctl` output to UUID strings and
verify:

- device `spec.id`  → `device_uuid`  label
- device `status.gpu` → `gpu_uuid` label
- station `spec.id`  → `station_uuid` label, parented to correct device
- port `status.name` (e.g. `netport0`) → `port_name` label, parented to correct station

PASS if all sampled UUIDs/labels match. FAIL with the mismatched entry.

## Step 5 — Per-entity field value validation

### Ports (from `ual_port.txt` vs `metrics.txt`)

| gpuctl field | Prometheus metric | Mapping |
|---|---|---|
| `status.linkstate` | `amd_ifoe_port_link_state` | 0=NONE,1=UP,2=DOWN |
| `status.speed` | `amd_ifoe_port_speed` | 0=NONE,1=400G,2=800G |
| `status.linkupcount` | `amd_ifoe_port_link_up_count` | direct int |
| `status.linkupduration` | `amd_ifoe_port_link_up_duration_msec` | direct int |
| `status.linkdownduration` | `amd_ifoe_port_link_down_duration_msec` | direct int |
| `status.linktrainingdurationlatest` | `amd_ifoe_port_link_training_duration_latest_msec` | direct int |
| `status.linktrainingdurationavg` | `amd_ifoe_port_link_training_duration_avg_msec` | direct int |
| `stats.rxtotalbytes` | `amd_ifoe_rx_total_bytes` | direct int |
| `stats.rxtotalpackets` | `amd_ifoe_rx_total_packets` | direct int |
| `stats.txtotalbytes` | `amd_ifoe_tx_total_bytes` | direct int |
| `stats.txtotalpackets` | `amd_ifoe_tx_total_packets` | direct int |
| `stats.rxbadfcs` | `amd_ifoe_rx_bad_fcs` | direct int |
| `stats.txbadfcs` | `amd_ifoe_tx_bad_fcs` | direct int |
| `stats.rxpacketdropped` | `amd_ifoe_rx_packet_dropped` | direct int |
| `stats.feccodewordsymbolerrors0..15` | `amd_ifoe_fec_codeword_symbol_errors{0..15}` | per-lane int |
| `stats.feccodewordsymbolerroruncorrectable` | `amd_ifoe_fec_codeword_symbol_errors_uncorrectable` | direct int |
| `stats.rxfeccorrectedcodewords` | `amd_ifoe_rx_fec_corrected_codewords` | direct int |
| `stats.rxfecuncorrectedcodewords` | `amd_ifoe_rx_fec_uncorrected_codewords` | direct int |

### Stations (from `ual_station.txt` vs `metrics.txt`)

| gpuctl field | Prometheus metric | Mapping |
|---|---|---|
| `stats.txrequestpacketcount` | `amd_ifoe_station_tx_request_packets` | direct int |
| `stats.txresponsepacketcount` | `amd_ifoe_station_tx_response_packets` | direct int |
| `stats.rxrequestpacketcount` | `amd_ifoe_station_rx_request_packets` | direct int |
| `stats.rxresponsepacketcount` | `amd_ifoe_station_rx_response_packets` | direct int |
| `stats.streamremapstotal` | `amd_ifoe_station_stream_remaps_total` | direct int |
| `stats.streamremapsnetworkport0` | `amd_ifoe_station_stream_remaps_network_port0` | direct int |
| `stats.streamremapsnetworkport1` | `amd_ifoe_station_stream_remaps_network_port1` | direct int |

**Tolerance:** counter fields may differ between the gpuctl capture and
the Prometheus scrape (different timestamps). A delta is only a FAIL if
the counter is non-monotonic OR the delta exceeds 5% of the value.

PASS per field if all entities match within tolerance. FAIL with
entity, field, gpuctl value, and metric value.

## Step 6 — Missing-metric check (known gaps)

These gpuctl fields have **no** corresponding metric in the current
exporter — verify they are still absent:

| gpuctl field | Expected |
|---|---|
| `station.status.operstate` | NOT exported |
| `station.status.currentavailablebandwidth` | NOT exported |
| `station.spec.adminstate` | NOT exported |
| `port.spec.adminstate` | NOT exported |

If any of these ARE present, mark **NEWLY ADDED** (positive — the gap closed).

## Step 7 — Extra-metric check

Scan `metrics.txt` for any `amd_ifoe_*` metric not in the mapping
tables above. Flag as **UNEXPECTED METRIC**.

## Step 8 — Report

Write `IFOE_EXPORTER_<HOSTNAME>_SUMMARY.md` in CWD with sections:
Counts, UUIDs/Labels, Field Values (totals + per-failure detail),
Known Missing Metrics, Extra Metrics, **Overall: PASS/FAIL** verdict.

`<HOSTNAME>` is the host part of `${TARGET}` (after `@`), uppercased
without the domain suffix.

## Cross-references

- Slash command (same content): `.claude/commands/validate-ifoe-exporter.md` — points at this skill as canonical
- Port discovery rationale: `.claude/skills/test-port-allocation.md`
- Metric prefix convention: `memory/project_metric_prefix.md`
- Related skills (full lifecycle): `install-ual-package`, `deploy-exporter-container`

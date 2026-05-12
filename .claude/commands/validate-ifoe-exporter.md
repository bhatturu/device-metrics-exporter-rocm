# Validate IFOE Exporter

> **Canonical form for agent dispatch:** `.claude/skills/validate-ifoe-exporter.md` (skill). This slash command and that skill share the same body — agentq workflows should invoke the skill; humans can keep using the slash command.

Validate that the `amdgpuifoe-exporter` Prometheus metrics endpoint correctly reflects the live UAL device, station, and port state reported by `gpuctl` on a remote system.

## Usage

```
/validate-ifoe-exporter <user>@<host>
```

**Example:** `/validate-ifoe-exporter prshanmug@mheliosr-1b114-f01-1.mnb.dcgpu`

---

## Instructions

The target system is: **$ARGUMENTS**

Work through the steps below in order. Run remote commands via `ssh $ARGUMENTS "<cmd>"`. Read the ServerPort from `/etc/metrics/config.json` on the target before issuing any curl commands — never assume port 5000.

---

### Step 1 — Preflight checks

Run these in parallel:

1. Verify SSH connectivity: `ssh $ARGUMENTS "echo connected"`
2. Check package installed: `ssh $ARGUMENTS "sudo apt list --installed 2>/dev/null | grep amdgpuifoe"`
3. Read server port: `ssh $ARGUMENTS "cat /etc/metrics/config.json"` — extract `ServerPort`

Report:
- SSH reachable: yes/no
- Package name and version installed (or "NOT INSTALLED")
- Metrics port (use this port for all subsequent curl calls)

If SSH fails or the package is not installed, stop and report the failure clearly.

---

### Step 2 — Collect data

Run all four commands in parallel on the remote system, saving output to `~/validate_ifoe_logs/` on the remote:

```bash
ssh $ARGUMENTS "mkdir -p ~/validate_ifoe_logs && \
  curl -s localhost:<PORT>/metrics > ~/validate_ifoe_logs/metrics.txt && \
  sudo gpuctl show ual device -y > ~/validate_ifoe_logs/ual_device.txt 2>&1 && \
  sudo gpuctl show ual station -y > ~/validate_ifoe_logs/ual_station.txt 2>&1 && \
  sudo gpuctl show ual port -y > ~/validate_ifoe_logs/ual_port.txt 2>&1 && \
  echo done"
```

Then copy the logs locally:

```bash
scp -r $ARGUMENTS:~/validate_ifoe_logs/ ./validate_ifoe_logs/
```

---

### Step 3 — Count validation

From `ual_device.txt`: count entries (`grep -c '^- spec:'`)
From `ual_station.txt`: count entries
From `ual_port.txt`: count entries
From `metrics.txt`: extract `amd_ifoe_total_devices`, `amd_ifoe_total_stations`, `amd_ifoe_total_ports`

**PASS** if all three counts match exactly. **FAIL** with the delta if not.

Also verify internal consistency: stations should equal devices × stations-per-device (from `capability.numualstation`), ports should equal devices × ports-per-device (from `capability.numnetworkport`).

---

### Step 4 — UUID and label validation

**Device UUIDs:** The gpuctl output encodes UUIDs as byte arrays. Decode each `spec.id` byte array to a UUID string and verify it matches the `device_uuid` label in `metrics.txt`.

**GPU UUIDs:** Decode `status.gpu` byte arrays and verify they match the `gpu_uuid` label in `metrics.txt`.

**Station UUIDs:** For a representative sample (at least the first station per device), decode the station `spec.id` and verify it matches the `station_uuid` label and is correctly parented to the right `device_uuid`.

**Port names:** Verify `status.name` (e.g., `netport0`) appears as the `port_name` label associated with the correct `station_uuid`.

**PASS** if all sampled UUIDs and labels match. **FAIL** with the mismatched entry.

---

### Step 5 — Per-entity field value validation

Check the following fields exhaustively across all entities:

#### Ports (from `ual_port.txt` vs `metrics.txt`)

| gpuctl field | Prometheus metric | Expected mapping |
|---|---|---|
| `status.linkstate` | `amd_ifoe_port_link_state` | Direct integer: 0=NONE, 1=UP, 2=DOWN |
| `status.speed` | `amd_ifoe_port_speed` | Direct integer: 0=NONE, 1=400G, 2=800G |
| `status.linkupcount` | `amd_ifoe_port_link_up_count` | Direct integer |
| `status.linkupduration` | `amd_ifoe_port_link_up_duration_msec` | Direct integer |
| `status.linkdownduration` | `amd_ifoe_port_link_down_duration_msec` | Direct integer |
| `status.linktrainingdurationlatest` | `amd_ifoe_port_link_training_duration_latest_msec` | Direct integer |
| `status.linktrainingdurationavg` | `amd_ifoe_port_link_training_duration_avg_msec` | Direct integer |
| `stats.rxtotalbytes` | `amd_ifoe_rx_total_bytes` | Direct integer |
| `stats.rxtotalpackets` | `amd_ifoe_rx_total_packets` | Direct integer |
| `stats.txtotalbytes` | `amd_ifoe_tx_total_bytes` | Direct integer |
| `stats.txtotalpackets` | `amd_ifoe_tx_total_packets` | Direct integer |
| `stats.rxbadfcs` | `amd_ifoe_rx_bad_fcs` | Direct integer |
| `stats.txbadfcs` | `amd_ifoe_tx_bad_fcs` | Direct integer |
| `stats.rxpacketdropped` | `amd_ifoe_rx_packet_dropped` | Direct integer |
| `stats.feccodewordsymbolerrors0..15` | `amd_ifoe_fec_codeword_symbol_errors{0..15}` | Direct integer per lane |
| `stats.feccodewordsymbolerroruncorrectable` | `amd_ifoe_fec_codeword_symbol_errors_uncorrectable` | Direct integer |
| `stats.rxfeccorrectedcodewords` | `amd_ifoe_rx_fec_corrected_codewords` | Direct integer |
| `stats.rxfecuncorrectedcodewords` | `amd_ifoe_rx_fec_uncorrected_codewords` | Direct integer |

#### Stations (from `ual_station.txt` vs `metrics.txt`)

| gpuctl field | Prometheus metric | Expected mapping |
|---|---|---|
| `stats.txrequestpacketcount` | `amd_ifoe_station_tx_request_packets` | Direct integer |
| `stats.txresponsepacketcount` | `amd_ifoe_station_tx_response_packets` | Direct integer |
| `stats.rxrequestpacketcount` | `amd_ifoe_station_rx_request_packets` | Direct integer |
| `stats.rxresponsepacketcount` | `amd_ifoe_station_rx_response_packets` | Direct integer |
| `stats.streamremapstotal` | `amd_ifoe_station_stream_remaps_total` | Direct integer |
| `stats.streamremapsnetworkport0` | `amd_ifoe_station_stream_remaps_network_port0` | Direct integer |
| `stats.streamremapsnetworkport1` | `amd_ifoe_station_stream_remaps_network_port1` | Direct integer |

**Timing tolerance:** Counter fields may differ by a small delta between the gpuctl capture and Prometheus scrape (both were captured at different times). A difference is only a FAIL if it is not monotonically increasing or is more than 5% of the value.

**PASS** per field if all entities match within tolerance. **FAIL** with entity, field, gpuctl value, and metric value.

---

### Step 6 — Missing metric check

Verify the following fields from gpuctl output have **no corresponding metric** in `metrics.txt` (these are known gaps in the current exporter version):

| gpuctl field | Expected status |
|---|---|
| `station.status.operstate` | NOT exported (known gap) |
| `station.status.currentavailablebandwidth` | NOT exported (known gap) |
| `station.spec.adminstate` | NOT exported (known gap) |
| `port.spec.adminstate` | NOT exported (known gap) |

If any of these ARE present in `metrics.txt`, mark as **NEWLY ADDED** (positive finding — the gap was fixed).

---

### Step 7 — Extra metric check

Scan `metrics.txt` for any `amd_ifoe_*` metric that does not correspond to a known UAL proto field. If found, flag as **UNEXPECTED METRIC** for investigation.

---

### Step 8 — Report

Produce a structured summary in this format:

```
## IFOE Exporter Validation Report
**Host:** <host>
**Package:** <name + version>
**Metrics port:** <port>
**Date:** <today>

### Counts
| Entity    | gpuctl | Prometheus | Result |
...

### UUIDs / Labels
...

### Field Values
| Category | Total checked | Passed | Failed | Details |
...

### Known Missing Metrics
| Field | Status |
...

### Extra Metrics
...

### Overall: PASS / FAIL
<one-line verdict>
```

Save the report as `IFOE_EXPORTER_<HOSTNAME>_SUMMARY.md` in the current working directory, where `<HOSTNAME>` is extracted from the ssh argument.

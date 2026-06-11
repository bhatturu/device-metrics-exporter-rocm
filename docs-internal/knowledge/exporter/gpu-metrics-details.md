# GPU Metrics Details - AMD Device Metrics Exporter

This document provides detailed information about GPU metrics, including static vs dynamic metrics, ECC metrics, and testing methodologies.

---

## Static vs Dynamic Metrics

GPU metrics in the Device Metrics Exporter fall into two categories:

### Static Metrics

**Definition:** Metrics that remain constant or change very infrequently (typically only on GPU reconfiguration or reboot).

**Examples:**
- `GPU_TOTAL_VRAM` - Total VRAM capacity (constant)
- `SERIAL_NUMBER` - GPU serial number (constant)
- `GPU_UUID` - GPU unique identifier (constant)
- `DRIVER_VERSION` - Driver version (changes only on driver update)
- `CARD_MODEL` - GPU model name (constant)
- `FIRMWARE_VERSION` - GPU firmware version (changes only on firmware update)
- Partition configuration (changes only when manually reconfigured)

**Collection Strategy:**
- Collected once during initialization or infrequently
- Cached in memory to avoid unnecessary queries
- Updated only on specific events (driver reload, partition change)

**File:** [pkg/amdgpu/gpuagent/gpuagent.go](../../pkg/amdgpu/gpuagent/gpuagent.go) - `UpdateStaticMetrics()`

### Dynamic Metrics

**Definition:** Metrics that change based on workload activity and GPU state.

**Examples:**

**Critical Dynamic Metrics (change with workload):**
- Temperature metrics: `GPU_EDGE_TEMPERATURE`, `GPU_JUNCTION_TEMPERATURE`, `GPU_MEMORY_TEMPERATURE`
- Power metrics: `GPU_PACKAGE_POWER`, `GPU_AVERAGE_PACKAGE_POWER`
- Activity metrics: `GPU_GFX_ACTIVITY`, `GPU_UMC_ACTIVITY`, `GPU_GFX_BUSY_INSTANTANEOUS`
- VRAM usage: `GPU_USED_VRAM`, `GPU_FREE_VRAM`
- Clock speeds: `GPU_CLOCK` (varies with workload and power state)
- Profiler metrics: `GPU_PROF_SM_ACTIVE`, `GPU_PROF_TENSOR_ACTIVE_PERCENT`

**Non-Critical Dynamic Metrics:**
- PCIe bandwidth: `PCIE_BANDWIDTH`, `PCIE_RX`, `PCIE_TX`
- XGMI link throughput: `GPU_XGMI_LINK_RX`, `GPU_XGMI_LINK_TX`
- Throttling violations: `GPU_VIOLATION_THERMAL`, `GPU_VIOLATION_POWER`

**Collection Strategy:**
- Queried on every `/metrics` request
- Real-time values from GPU hardware
- Critical for monitoring workload behavior

**File:** [pkg/amdgpu/gpuagent/gpuagent.go](../../pkg/amdgpu/gpuagent/gpuagent.go) - `UpdateMetrics()`

---

## ECC Metrics - Special Case

### Overview

ECC (Error Correcting Code) metrics are a **special category** of GPU metrics with unique characteristics:

1. **Incremental Counters** - ECC error counts only increase (never decrease until GPU reset)
2. **Health-Critical** - Uncorrectable errors trigger GPU health state changes
3. **Platform-Specific** - Not all ECC error types supported on all platforms
4. **Block-Specific** - Each GPU functional block has separate ECC error counters

### ECC Error Types (19 Total)

**Correctable Errors** (Recoverable):
- `GPU_ECC_CORRECT_SDMA` - SDMA engine
- `GPU_ECC_CORRECT_GFX` - Graphics engine
- `GPU_ECC_CORRECT_MMHUB` - Memory hub
- `GPU_ECC_CORRECT_ATHUB` - Address translation hub
- `GPU_ECC_CORRECT_BIF` - Bus interface
- `GPU_ECC_CORRECT_HDP` - Host data path
- `GPU_ECC_CORRECT_XGMI_WAFL` - XGMI interconnect
- `GPU_ECC_CORRECT_DF` - Data fabric
- `GPU_ECC_CORRECT_SMN` - System management network
- `GPU_ECC_CORRECT_SEM` - SEM
- `GPU_ECC_CORRECT_MP0` - MP0 processor
- `GPU_ECC_CORRECT_MP1` - MP1 processor
- `GPU_ECC_CORRECT_FUSE` - Fuse controller
- `GPU_ECC_CORRECT_UMC` - Unified memory controller (critical)
- `GPU_ECC_CORRECT_MCA` - Machine check architecture
- `GPU_ECC_CORRECT_VCN` - Video codec engine
- `GPU_ECC_CORRECT_JPEG` - JPEG engine
- `GPU_ECC_CORRECT_IH` - Interrupt handler
- `GPU_ECC_CORRECT_MPIO` - Multi-port I/O

**Uncorrectable Errors** (Fatal):
- Same 19 blocks with `GPU_ECC_UNCORRECT_*` prefix
- **CRITICAL:** Any uncorrectable error marks GPU as unhealthy
- Typically requires GPU reset or node reboot to recover

### Health Threshold Configuration

ECC errors are evaluated against configurable thresholds for health determination:

**File:** `/etc/metrics/config.json`

```json
{
  "GPUConfig": {
    "HealthThresholds": {
      "GPU_ECC_UNCORRECT_SDMA": 0,
      "GPU_ECC_UNCORRECT_GFX": 0,
      "GPU_ECC_UNCORRECT_UMC": 0,
      "GPU_ECC_UNCORRECT_MMHUB": 0,
      "GPU_ECC_UNCORRECT_ATHUB": 0,
      "GPU_ECC_UNCORRECT_BIF": 0,
      "GPU_ECC_UNCORRECT_HDP": 0,
      "GPU_ECC_UNCORRECT_XGMI_WAFL": 0,
      "GPU_ECC_UNCORRECT_DF": 0,
      "GPU_ECC_UNCORRECT_SMN": 0,
      "GPU_ECC_UNCORRECT_SEM": 0,
      "GPU_ECC_UNCORRECT_MP0": 0,
      "GPU_ECC_UNCORRECT_MP1": 0,
      "GPU_ECC_UNCORRECT_FUSE": 0,
      "GPU_ECC_UNCORRECT_MCA": 0,
      "GPU_ECC_UNCORRECT_VCN": 0,
      "GPU_ECC_UNCORRECT_JPEG": 0,
      "GPU_ECC_UNCORRECT_IH": 0,
      "GPU_ECC_UNCORRECT_MPIO": 0
    }
  }
}
```

**Typical Thresholds:**
- **Uncorrectable errors:** 0 (any error = unhealthy)
- **Correctable errors:** Higher threshold (e.g., 100) or not monitored

**File:** [pkg/amdgpu/gpuagent/gpuagent_health.go](../../pkg/amdgpu/gpuagent/gpuagent_health.go)

### Platform Support

**Check ECC Support on Platform:**
```bash
amd-smi -ecc
```

**Documentation:** [AMD SMI CLI Tool - ECC](https://github.com/ROCm/rocm-systems/blob/develop/projects/amdsmi/docs/how-to/amdsmi-cli-tool.md)

**Output Example:**
```
GPU[0]: ECC ENABLED
  Supported blocks: SDMA, GFX, UMC, MMHUB, ATHUB, BIF, HDP
  Unsupported blocks: XGMI_WAFL, DF, SMN (platform limitation)
```

**Important Notes:**
- **Not all ECC blocks supported on all platforms** (MI2xx vs MI3xx differences)
- **Platform-specific handling required** in code (field logger for unsupported blocks)
- **Check platform docs** before adding new ECC metrics

---

## AFID Metrics - Special Case

### Overview

AFID (Accelerated Function ID) metrics track GPU function execution and are another **special category**:

**Characteristics:**
- **Process-Specific** - Track per-process GPU function usage
- **Dynamic** - Change rapidly with workload activity
- **Correlation** - Link GPU activity to specific applications/containers

**Note:** Details TBD - refer to implementation in `pkg/amdgpu/` when available.

---

## ECC Error Injection for Testing

### Overview

ECC error injection is critical for testing health monitoring, alert systems, and error handling. Two methods are available:

### Method 1: Mock ECC Injection (metricsclient)

**Purpose:** Software-based error injection for testing without real hardware errors

**Tool:** `metricsclient` utility (built with exporter)

**Location:** `tools/metricsclient/` - See [metricsclient-tool.md](metricsclient-tool.md) for complete usage

**Usage:**

metricsclient uses a **JSON file** for error configuration (not CLI flags).

**1. Create JSON error configuration:**
```bash
# Inject uncorrectable UMC error on GPU 0
cat > /tmp/ecc-error.json <<EOF
{
  "ID": "0",
  "Fields": ["GPU_ECC_UNCORRECT_UMC"],
  "Counts": [1]
}
EOF
```

**2. Inject error:**
```bash
metricsclient --ecc-file-path /tmp/ecc-error.json
```

**3. Clear error (set count to 0):**
```bash
cat > /tmp/ecc-clear.json <<EOF
{
  "ID": "0",
  "Fields": ["GPU_ECC_UNCORRECT_UMC"],
  "Counts": [0]
}
EOF

metricsclient --ecc-file-path /tmp/ecc-clear.json
```

**Multiple errors:**
```bash
cat > /tmp/ecc-multi.json <<EOF
{
  "ID": "0",
  "Fields": ["GPU_ECC_UNCORRECT_SDMA", "GPU_ECC_UNCORRECT_GFX"],
  "Counts": [1, 2]
}
EOF

metricsclient --ecc-file-path /tmp/ecc-multi.json
```

**Characteristics:**
- **Internal to DME** - Only affects DME's health service, not real GPU state
- **Reversible** - Can clear errors and restore healthy state
- **Safe** - No impact on GPU hardware or running workloads
- **Testing** - Used for device-plugin health monitoring tests
- **Limited scope** - Only tests DME logic, not driver/hardware behavior

**Implementation:**
- Proto: [pkg/amdgpu/proto/gpumetricssvc.proto](../../pkg/amdgpu/proto/gpumetricssvc.proto) - `GPUErrorRequest`, `SetError()` RPC
- Service: [pkg/amdgpu/metricsserver/metrics_svc.go](../../pkg/amdgpu/metricsserver/metrics_svc.go) - `SetError()` implementation
- Client: [tools/metricsclient/main.go](../../tools/metricsclient/main.go) - CLI tool

**Use Cases:**
- Test health service gRPC API
- Test device-plugin health monitoring integration
- Test K8s node label updates on GPU health changes
- Test alert/notification systems
- CI/CD automated testing

**Complete Documentation:** [metricsclient-tool.md](metricsclient-tool.md)

### Method 2: Real Hardware Error Injection (AMDGPURAS)

**Purpose:** Hardware-level error injection for comprehensive testing (driver, firmware, hardware)

**Tool:** `AMDGPURAS` (AMD GPU RAS Tool) - Black box proprietary tool

**Capabilities:**

AMDGPURAS can inject **real hardware errors** into the following GPU blocks:

| Block        | Description                    | Critical? |
|--------------|--------------------------------|-----------|
| SDMA         | SDMA engine                    | Yes       |
| GFX          | Graphics engine                | Yes       |
| MMHUB        | Memory hub                     | Yes       |
| ATHUB        | Address translation hub        | Yes       |
| BIF          | Bus interface                  | Yes       |
| HDP          | Host data path                 | Yes       |
| XGMI_WAFL    | XGMI interconnect              | Yes       |
| DF           | Data fabric                    | Yes       |
| SMN          | System management network      | No        |
| SEM          | SEM                            | No        |
| MP0          | MP0 processor                  | No        |
| MP1          | MP1 processor                  | No        |
| FUSE         | Fuse controller                | No        |
| UMC          | Unified memory controller      | **Critical** |
| MCA          | Machine check architecture     | Yes       |
| VCN          | Video codec engine             | No        |
| JPEG         | JPEG engine                    | No        |
| IH           | Interrupt handler              | No        |
| MPIO         | Multi-port I/O                 | Yes       |

**Platform Support:**

**IMPORTANT:** Not all blocks are supported on all platforms.

**Check Support:**
```bash
# Check which ECC blocks are supported on current platform
amd-smi -ecc

# Example output (MI300X):
GPU[0]: ECC ENABLED
  Supported blocks: SDMA, GFX, UMC, MMHUB, ATHUB, BIF, HDP, XGMI_WAFL
  Unsupported blocks: DF, SMN, SEM, MP0, MP1, FUSE (not available on this platform)
```

**Documentation:** [AMD SMI CLI - ECC Reference](https://github.com/ROCm/rocm-systems/blob/develop/projects/amdsmi/docs/how-to/amdsmi-cli-tool.md)

**Usage Example:**
```bash
# Inject correctable error to UMC block
amdgpuras --inject-error --gpu 0 --block UMC --error-type correctable --count 10

# Inject uncorrectable error to SDMA block (will mark GPU unhealthy)
amdgpuras --inject-error --gpu 0 --block SDMA --error-type uncorrectable --count 1

# WARNING: Uncorrectable errors may require GPU reset or node reboot!
```

**Characteristics:**
- **Hardware-level** - Actual errors injected into GPU hardware
- **Persistent** - Errors persist in hardware counters until GPU reset
- **Realistic** - Tests entire stack (hardware → driver → amd-smi → gpuagent → DME)
- **Risky** - Uncorrectable errors may crash GPU or node
- **Platform-specific** - Only supported blocks can be tested
- **Irreversible** - Cannot clear errors (requires GPU reset)

**Use Cases:**
- End-to-end RAS testing (driver + firmware + hardware)
- Validate amd-smi error reporting
- Test driver error handling and recovery
- Firmware RAS feature validation
- Production-like error scenario testing

**Safety Precautions:**
- **Test on dedicated hardware** - Not on production systems
- **Monitor carefully** - Uncorrectable errors can crash nodes
- **Document expected behavior** - Know what errors are recoverable
- **Plan recovery** - Have node reboot procedure ready
- **Check platform support** - Verify block supported before injection

### Metric Increment Relationship

**IMPORTANT:** Any metric that AMDGPURAS can inject errors for can be incremented using these tools (if platform supports the block).

**Metric Flow:**
```
AMDGPURAS (inject error)
  ↓
GPU Hardware (error counter incremented)
  ↓
AMD GPU Driver (reports error via RAS interface)
  ↓
amd-smi library (reads error counter)
  ↓
gpuagent (queries amd-smi)
  ↓
DME (exposes metric via Prometheus)
  ↓
Prometheus scrape (metric value increases)
```

**Testing Strategy:**
1. **Development/CI:** Use metricsclient for fast, safe testing
2. **Integration Testing:** Use AMDGPURAS on test hardware for realistic scenarios
3. **Pre-Production:** Use AMDGPURAS to validate full RAS stack
4. **Production:** Monitor real errors; use metricsclient only for testing monitoring systems

---

## CPER (Common Platform Error Record) Injection

### Overview

CPER is the UEFI standard format for hardware error reporting. GPU RAS errors are reported via CPER to the OS.

### CPER Injection Methods

Both metricsclient and AMDGPURAS can inject CPER-formatted errors:

**metricsclient:**
- Injects mock CPER records into DME's internal state
- Does not generate real CPER records in OS

**AMDGPURAS:**
- Generates real CPER records via GPU firmware
- CPER records visible in OS logs (`dmesg`, `/sys/firmware/efi/cper/`)
- Triggers APEI (ACPI Platform Error Interface) handlers

**CPER Record Types:**
- Correctable errors: Logged but do not trigger machine checks
- Uncorrectable errors: May trigger machine checks (system crash)
- Fatal errors: Always trigger system crash

**View CPER Records:**
```bash
# View CPER error records in kernel log
dmesg | grep -i cper

# View ACPI error records
cat /sys/firmware/acpi/tables/HEST
```

---

## Health Monitoring Integration

### ECC Error → Health State Flow

```
ECC Error Occurs
  ↓
Driver increments error counter
  ↓
amd-smi reports error count
  ↓
gpuagent queries via GPUCPERGet()
  ↓
DME health monitor loop (30s default)
  ↓
Evaluate against HealthThresholds
  ↓
If threshold exceeded:
  - Mark GPU unhealthy
  - Update gRPC health service
  - K8s: Add node label (gpu.amd.com/unhealthy=true)
  ↓
Device-plugin queries health service
  ↓
Device-plugin marks GPU unhealthy
  ↓
K8s scheduler stops scheduling pods to GPU
```

**File:** [pkg/amdgpu/gpuagent/gpuagent_health.go](../../pkg/amdgpu/gpuagent/gpuagent_health.go)

### Recovery Scenarios

**Correctable Errors:**
- Logged and counted
- GPU remains healthy (unless threshold exceeded)
- No immediate action required

**Uncorrectable Errors:**
- GPU marked unhealthy immediately (threshold = 0)
- **No automatic recovery** - GPU stays unhealthy
- **Recovery options:**
  1. **GPU reset:** `amd-smi --reset-gpu <gpu_id>` (may not clear errors)
  2. **Node reboot:** Only guaranteed way to clear error counters
  3. **Mock clear:** `metricsclient clear-error` (testing only, doesn't affect hardware)

**Production Reality:**
- Unhealthy → Healthy transition rarely occurs in production
- Once GPU has uncorrectable errors, node typically requires reboot
- Health monitoring is for detection, not automatic recovery

---

## Platform-Specific Considerations

### MI2xx vs MI3xx Differences

**ECC Block Support:**
- **MI2xx:** Subset of ECC blocks supported
- **MI3xx:** Extended ECC block coverage

**Check per Platform:**
```bash
# On MI200A
amd-smi -ecc
# Supported: SDMA, GFX, UMC, MMHUB, ATHUB

# On MI300X
amd-smi -ecc
# Supported: SDMA, GFX, UMC, MMHUB, ATHUB, BIF, HDP, XGMI_WAFL, DF
```

**Code Handling:**
- Use field logger for unsupported blocks
- Don't fail if platform doesn't support specific ECC type
- Document platform support in metrics documentation

---

## Related Documentation

- **Troubleshooting:** [troubleshooting.md](troubleshooting.md) - ECC error debugging
- **Health Monitoring:** [health-monitoring.md](health-monitoring.md) - Health service details
- **Configuration:** [configuration.md](configuration.md) - HealthThresholds configuration
- **AMD SMI Docs:** [AMD SMI CLI Tool](https://github.com/ROCm/rocm-systems/blob/develop/projects/amdsmi/docs/how-to/amdsmi-cli-tool.md)

---

**Last Updated:** 2026-04-06

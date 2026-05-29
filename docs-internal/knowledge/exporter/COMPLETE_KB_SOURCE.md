# Complete Knowledge Base - Source Document

This file contains the complete comprehensive knowledge base content for all remaining documentation files. Future agents should extract relevant sections to create individual focused documents.

**Source:** This content is derived from the approved plan at `~/.claude/plans/robust-stirring-harbor.md`

---

## GPU Exporter (gpu-exporter.md)

### Static vs Dynamic Metrics

**Static Metrics** (constant or rarely changing):
- GPU_TOTAL_VRAM, SERIAL_NUMBER, GPU_UUID, DRIVER_VERSION
- Collected once during initialization, cached in memory

**Dynamic Metrics** (change with workload):
- Temperature, Power, Activity, VRAM usage, Clock speeds, Profiler metrics
- Queried on every /metrics request
- Critical for monitoring workload behavior

### ECC Metrics - Special Case

**19 ECC Error Types** across GPU blocks:
- SDMA, GFX, MMHUB, ATHUB, BIF, HDP, XGMI_WAFL, DF, SMN, SEM, MP0, MP1, FUSE, UMC, MCA, VCN, JPEG, IH, MPIO

**Correctable vs Uncorrectable:**
- Correctable: Logged, GPU stays healthy (unless threshold exceeded)
- Uncorrectable: GPU marked unhealthy immediately, requires reset/reboot

**Platform Support:**
- Not all ECC blocks supported on all platforms
- Check: `amd-smi -ecc`
- MI2xx vs MI3xx have different block support

**Error Injection Methods:**

1. **metricsclient** (Mock injection - safe):
   ```bash
   # Uses JSON file for error configuration
   cat > /tmp/ecc.json <<EOF
   {"ID":"0","Fields":["GPU_ECC_UNCORRECT_UMC"],"Counts":[1]}
   EOF
   metricsclient --ecc-file-path /tmp/ecc.json
   ```
   - Internal to DME, reversible, testing only
   - Complete guide: docs-internal/knowledge/exporter/metricsclient-tool.md

2. **AMDGPURAS** (Real HW injection - risky):
   - Injects real errors into GPU hardware
   - Platform-specific support (check with amd-smi -ecc)
   - Tests entire RAS stack (hardware → driver → amd-smi → gpuagent → DME)
   - **WARNING:** Uncorrectable errors may crash node

**Documentation:** [AMD SMI CLI Tool](https://github.com/ROCm/rocm-systems/blob/develop/projects/amdsmi/docs/how-to/amdsmi-cli-tool.md)

### AFID Metrics - Special Case

Process-specific function tracking (TBD - refer to implementation)

### GPUAgent Library Architecture

**Reference:** [gpuagent/developerguide.md](../../gpuagent/developerguide.md)

**GPUAgent Library Structure:**

GPUAgent is a multi-layered library that provides both GPU and UAL (formerly IFOE) services:

1. **API Layer** (North Bound):
   - API definitions: `gpuagent/sw/nic/gpuagent/protos/`
   - Internal models: `gpuagent/sw/nic/gpuagent/api/`

2. **Service Layer:**
   - gRPC services: `gpuagent/sw/nic/gpuagent/svc/`
   - **Two Sub-Clients:**
     - **GPU Client** - GPU metrics and management
     - **UAL Client** - UAL/IFOE metrics and management

3. **Data Abstraction Layer (SMI):**
   - Layer: `gpuagent/sw/nic/gpuagent/api/smi/`
   - Responsible for populating/retrieving data from client libraries
   - Translates internal data models to protobuf payloads
   - Data client: amdsmi (`gpuagent/sw/nic/gpuagent/api/smi/amdsmi/smi_api.cc`)

### GPU Metrics Categories

**Critical Metrics Reference:** [internal/metricsmap.md](../../internal/metricsmap.md)

**Critical Metrics for Workload Evaluation:**

1. **Temperature Metrics:**
   - GPU_EDGE_TEMPERATURE (MI2xx)
   - GPU_JUNCTION_TEMPERATURE (MI3xx)
   - GPU_MEMORY_TEMPERATURE
   - GPU_HBM_TEMPERATURE (deprecated from driver 6.14.14)

2. **Power Metrics:**
   - GPU_PACKAGE_POWER (MI3xx)
   - GPU_AVERAGE_PACKAGE_POWER (MI2xx)

3. **Activity Metrics:**
   - GPU_GFX_ACTIVITY (unpartitioned GPU)
   - GPU_UMC_ACTIVITY
   - GPU_GFX_BUSY_INSTANTANEOUS (MI3xx, per-partition)
   - GPU_VCN_BUSY_INSTANTANEOUS (MI3xx, per-partition)

4. **VRAM Metrics:**
   - GPU_TOTAL_VRAM
   - GPU_USED_VRAM
   - GPU_FREE_VRAM

5. **Profiler Metrics:**
   - GPU_PROF_SM_ACTIVE (VALUBusy)
   - GPU_PROF_TENSOR_ACTIVE_PERCENT (MfmaUtil)
   - GPU_PROF_OCCUPANCY_PER_CU
   - GPU_PROF_OCCUPANCY_PER_ACTIVE_CU

**Non-Profiler Metrics (0-799):**
- Power, Temperature, Activity, Memory, Clock, ECC Errors (19 types)
- PCIe, XGMI, Violations, Health, Process Metrics

**Profiler Metrics (801-1200):**
- Raw Metrics: GRBM_GUI_ACTIVE, SQ_WAVES, CPC/CPF stats
- Derived Metrics: OccupancyPercent, TensorActivePercent, SimdUtilization

**Platform-Specific:**
- **MI2xx:** GPU_EDGE_TEMPERATURE, GPU_AVERAGE_PACKAGE_POWER
- **MI3xx:** GPU_JUNCTION_TEMPERATURE, GPU_PACKAGE_POWER, partition metrics

---

## ROCProfiler (rocprofiler.md)

### Architecture

**Component:** [pkg/amdgpu/rocprofiler/rocpclient.go](../../pkg/amdgpu/rocprofiler/rocpclient.go)

**Execution Model:**
- Wrapper around `rocpctl` command-line tool
- Command: `rocpctl -d <duration_us> -p <ptl_delay_ms> <metric_fields>`
- Parses JSON output into `amdgpu.GpuProfiler` protobuf

### Safety Mechanisms

1. **PTL (Peak Tops Limiter) Handling:**
   ```bash
   ROCPROFILER_DEVICE_LOCK_AT_START=1
   ```
   - Prevents PTL toggle overwhelm on PTL-enabled platforms
   - PTL toggle SLA: not within 1ms to avoid driver crashes
   - Solution: Turn off PTL once at first `context_start`, restore only on exit

2. **Caching Strategy:**
   - Cache duration: 10 seconds
   - Rationale: Avoid stressing hardware with frequent rocpctl calls

3. **Failure Handling:**
   - Threshold: 3 consecutive failures → profiler disabled
   - Fatal detection: String matching "dumped"/"aborted" → permanent disable
   - Timeout: 30 seconds per rocpctl execution

4. **Platform Restrictions:**
   - SR-IOV: Profiler disabled via `-sriov-enable` flag
   - Driver requirement: Only amdgpu driver platforms

### Configuration

```protobuf
message ProfilerConfig {
    uint64 SamplingInterval = 1;  // Default: 1000 us
    uint32 PtlDelay = 2;          // PTL delay (ms), 0 to disable
}
```

**Per-Node Control:**
```json
"ProfilerMetrics": {
    "all": false,
    "hostname1": false
}
```

---

## Health Monitoring (health-monitoring.md)

### Health Service Architecture

**Service Endpoint:** gRPC over Unix socket at `/var/lib/amd-metrics-exporter/amdgpu_device_metrics_exporter_grpc.socket`

**Proto:** [pkg/amdgpu/proto/gpumetricssvc.proto](../../pkg/amdgpu/proto/gpumetricssvc.proto)

```protobuf
service MetricsService {
    rpc GetGPUState(GPUGetRequest) returns (GPUStateResponse) {}
    rpc List(google.protobuf.Empty) returns (GPUStateResponse) {}
    rpc SetError(GPUErrorRequest) returns (GPUErrorResponse) {}
}

message GPUState {
    string ID = 1;
    string UUID = 2;
    string Health = 3;  // "healthy" or "unhealthy"
    repeated string AssociatedWorkload = 4;
    string Device = 5;  // PCIe Bus ID
}
```

### Health Monitoring Loop

**Location:** [pkg/amdgpu/gpuagent/gpuagent.go](../../pkg/amdgpu/gpuagent/gpuagent.go) - `StartMonitor()`

**Workflow:**
1. Poll interval from config (default: 30s, min: 30s, max: 24h)
2. Check agent connectivity via `isActive()`
3. Reconnect if inactive (with failure counter)
4. Process health validation (GPU metrics, ECC checks)
5. Update health states
6. Serve health status via gRPC (DME is server)
7. Reset failure counter on success
8. Exit DME after 3 failures if `-exit-on-agent-down` enabled

**K8s Integration:** GPU health set in node labels (unhealthy adds label, healthy removes)

### Health Determination Logic

**File:** [pkg/amdgpu/gpuagent/gpuagent_health.go](../../pkg/amdgpu/gpuagent/gpuagent_health.go)

1. Default: All GPUs start HEALTHY
2. Threshold Evaluation: Check 19 ECC error types
3. Mark Unhealthy: If uncorrectable error count exceeds threshold

**Example Thresholds:**
```json
"HealthThresholds": {
    "GPU_ECC_UNCORRECT_SDMA": 0,
    "GPU_ECC_UNCORRECT_GFX": 0,
    "GPU_ECC_UNCORRECT_UMC": 0
}
```

### Exit-On-Agent-Down Mechanism

**Flag:** `-exit-on-agent-down`

**Error Types:**
- `ErrAgentUnreachable` - gRPC connection fails
- `ErrZeroGPUs` - gpuagent reports 0 GPUs

**Failure Counter Logic:**
```go
const maxConsecutiveFailures = 3
consecutiveFailures := 0

if err := ga.reconnect(); err != nil {
    consecutiveFailures++
    if ga.exitOnAgentDown && consecutiveFailures >= 3 {
        ga.exitFn(1)  // os.Exit(1)
    }
}
```

**K8s Integration:**
- Pod restartPolicy: `Always` or `OnFailure`
- DME exits code 1 → K8s restarts → retries when driver ready

---

## NIC Exporter (nic-exporter.md)

### Design Philosophy

**CLI-Based Approach:** Uses system tools (nicctl, ethtool, rdma) instead of direct system calls

**Plugin Architecture:**
1. **NICCtl Client** - Port, LIF, QP statistics via `nicctl`
2. **Ethtool Client** - Ethernet stats via `ethtool`
3. **RDMA Stats Client** - RoCE stats via `rdma statistic`

### NIC Data Structures

```go
type NIC struct {
    Index, UUID, ProductName, SerialNumber string
    EthBDF, FirmwareVersion string
    Ports map[string]*Port
    Lifs  map[string]*Lif
}

type NetDevice struct {
    IntfName, IntfAlias, RoceDevName string
    PCIeBusId, PodName string
}
```

### CLI Clients

**NICCtl:**
- `nicctl show port statistics -j` → Port metrics
- `nicctl show port statistics --rate -j` → Rate metrics
- 54+ fields: RX/TX frames, octets, errors, FEC

**Ethtool:**
- `ethtool -S <interface>`
- Host and pod interfaces (via nsenter)
- 100+ fields: packet counts, frame distribution, errors

**RDMA Stats:**
- `rdma statistic -j` (20s timeout)
- 93+ fields: RDMA packets, errors, atomic ops

### Pod→Device Discovery

**Mapping Flow:**
1. Pod + namespace → Container ID (K8s API)
2. Container ID → PID (CRI: containerd/crio)
3. PID → Network namespace → interfaces
4. Interface → LIF → NIC → PCIe address

**Cache Strategy:**
- Pod→PID: LRU (size=100)
- Pod→NetDevice: LRU (size=100)

---

## Configuration (configuration.md)

### Configuration Files

**Main Config:** `/etc/metrics/config.json`

**Schema:** [pkg/exporter/proto/exporterconfig.proto](../../pkg/exporter/proto/exporterconfig.proto)

**Example:** [example/config.json](../../example/config.json)

### Configuration Structure

```protobuf
message MetricConfig {
    uint32 ServerPort = 1;
    GPUMetricConfig GPUConfig = 2;
    NICMetricConfig NICConfig = 4;
    IFOEMetricConfig IFOEConfig = 5;
    CommonConfig CommonConfig = 3;
}

message CommonConfig {
    string MetricsFieldPrefix = 1;
    HealthServiceConfig HealthService = 2;
    LoggingConfig Logging = 3;
}
```

### Dynamic Config Reloading

**File Watcher:** [pkg/exporter/exporter.go](../../pkg/exporter/exporter.go) - `foreverWatcher()`

**Workflow:**
1. Monitor config directory
2. On modification (3-second debounce):
   - Stop server
   - Reload config
   - Reinit registry
   - Start server
3. No service restart required

---

## Build System (build-system.md)

### Docker Build Environment

```bash
make docker-shell
```

**Creates:** `docker.io/rocm/device-metrics-exporter-builder:v1.0`
**Mounts:** Current directory to `/device-metrics-exporter`

**Container Reuse:**
```bash
docker ps | grep device-metrics-exporter-builder
docker exec -it <container_id> bash
```

### Key Build Targets

**Development:**
- `make default` - Build everything in Docker
- `make docker-shell` - Interactive build environment
- `make gen` - Generate protobuf code
- `make copyrights` - Add copyright headers (may fail, ignore if tracked files OK)
- `make all` - Build exporter binary

**Testing:**
- `make unit-test` - Run tests in `pkg/`
- `make e2e` - End-to-end tests (mock exporter)

**Images:**
- `make docker` - Standard exporter
- `make docker-sriov` - SR-IOV variant
- `make docker-ainic` - AI NIC variant

**Packages:**
- `make profiler-libdependent-assets` - Build profiler libs (once)
- `make pkg` - All Debian packages
- `make rpmpkg` - RPM packages (RHEL 9)

**Helm:**
- `make helm` - Build Helm chart
- `make helm-docs` - Generate docs

### Environment Variables

```bash
DOCKER_REGISTRY=docker.io/rocm
EXPORTER_IMAGE_TAG=latest
UBUNTU_VERSION=jammy  # or noble
ROCM_VERSION=6.4.1
```

### Git Submodules

**gpuagent:**
- Repo: https://github.com/ROCm/gpu-agent.git
- Build: `make gpuagent-build && make gpuagent-compile`
- Output: `assets/gpuagent`
- Builder: RHEL9-based (Ubuntu deprecated)

**libamdsmi:**
- Repo: https://github.com/ROCm/amdsmi.git
- Branch: release/7.2
- Build: `make amdsmi-build && make amdsmi-compile`

**rocprofilerclient:**
- Repo: https://github.com/ROCm/rocm-systems
- Build: `ROCM_VERSION=6.4.1 make rocprofiler-compile`

---

## Deployment (deployment.md)

### 1. Kubernetes

**Detection:** `KUBERNETES_SERVICE_HOST` or kubelet socket

**Service Ports:**
- Metrics: 5000 (GPU), 5001 (NIC)
- Health: Unix socket

**Build:** `make docker`

### 2. Debian Package

**Detection:** Systemd service files in `/usr/lib/systemd/system/`

**Config:** `/etc/metrics/config.json`

**Build:** `make profiler-libdependent-assets && make pkg`

**Ubuntu:** 22.04 (jammy), 24.04 (noble)

### 3. RPM Package

**Build:** `make rpmpkg` (RHEL 9)

### 4. SR-IOV Mode

**Detection:** `-sriov-enable` flag

**Build:** `make docker-sriov`

**Limitations:** Profiler disabled

---

## CI/CD (ci-cd.md)

### Build Jobs (.job.yml)

**Debian Packages:**
- `build-debian-package-ub22.04` - Ubuntu 22.04
- `build-debian-package-ub24.04` - Ubuntu 24.04
- `build-nic-debian-package` - NIC exporter

**RPM:**
- `build-rpm-package-rhel9` - RHEL 9

**Docker:**
- `build-device-metrics-exporter-docker-ubi9.6` - Standard
- `build-device-metrics-exporter-docker-sriov-ubi9.6` - SR-IOV
- `build-device-metrics-exporter-docker-ainic-ubi9.6` - AI NIC
- `e2e-build-mock-exporter` - Mock testing

**Helm:**
- `build-helm-gpu-artifact` - GPU charts (internal + public)
- `build-helm-nic-artifact` - NIC charts (internal + public)

### Test Targets

- `device-metrics-sanity` - Unit tests
- `api-checks` - API validation
- `copyright-checks` - Copyright headers
- `docs-lint` - Documentation linting

### Asset Build Pipeline

**copy-device-metrics-exporter-artifacts:**
- Script: `./asset-build/exporter-asset-push.sh`
- Publishes: packages, images, helm charts

---

## Adding New Metrics (adding-metrics.md)

### GPU Non-Profiler Metrics

**Steps:**

1. Add to gpuagent proto: `gpuagent/sw/nic/gpuagent/protos/gpu.proto`
2. Implement in gpuagent: Query amdsmi for field
3. Copy proto to DME: `pkg/amdgpu/proto/gpu.proto`
4. Add to exporterconfig: `pkg/exporter/proto/exporterconfig.proto`
5. Register Prometheus metric: `pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go`
6. Populate metric: `pkg/amdgpu/gpuagent/gpuagent_gpu.go`
7. Update config: [example/config.json](../../example/config.json)
8. Rebuild: `make gen && make all`

### GPU Profiler Metrics

**Enum Range:** 801-1200

**Steps:**
1. Add to exporterconfig (range 801-1200)
2. Update rocpctl invocation
3. Register metric
4. Populate metric

### NIC Metrics

**Enum Ranges:**
- Port Stats: 2-99
- LIF Stats: 100-199
- RDMA Stats: 200-299
- QP Stats: 300-399
- Ethtool Stats: 500-699

**Steps:**
1. Add to exporterconfig (appropriate range)
2. Add to proto (nic/rdmastats/ethtool)
3. Update CLI client
4. Register metric
5. Update config
6. Rebuild

---

## Critical File Paths (critical-paths.md)

### Entry Points
- `/cmd/exporter/main.go` - Application entry
- `/pkg/exporter/proto/exporterconfig.proto` - Config schema
- `/example/config.json` - Config template

### GPU
- `/pkg/amdgpu/gpuagent/gpuagent.go` - Main client
- `/pkg/amdgpu/gpuagent/gpuagent_gpu.go` - GPU metrics
- `/pkg/amdgpu/gpuagent/gpuagent_health.go` - Health logic
- `/pkg/amdgpu/gpuagent/gpuagent_ifoe.go` - UAL metrics
- `/pkg/amdgpu/rocprofiler/rocpclient.go` - ROCProfiler
- `/pkg/amdgpu/proto/gpumetricssvc.proto` - Health service

### NIC
- `/pkg/amdnic/nicagent/nicagent.go` - NIC coordinator
- `/pkg/amdnic/nicagent/nicctl_client.go` - NICCtl client
- `/pkg/amdnic/nicagent/ethtool_client.go` - Ethtool client
- `/pkg/amdnic/nicagent/rdma_stats_client.go` - RDMA client
- `/pkg/amdnic/nicagent/nicagent_metrics.go` - Metric registration (137KB)

### Scheduler
- `/pkg/exporter/scheduler/scheduler.go` - Interface
- `/pkg/exporter/scheduler/k8s.go` - K8s PodResources
- `/pkg/exporter/scheduler/slurm.go` - SLURM watcher
- `/pkg/client/k8s.go` - K8s API client

### Core
- `/pkg/exporter/exporter.go` - Main orchestrator
- `/pkg/exporter/npd.go` - NPD integration (/gpumetrics, /inbandraserrors)
- `/pkg/exporter/globals/constants.go` - Socket paths, ports

### Build
- `/Makefile` - Top-level targets
- `/.job.yml` - CI/CD build jobs
- `/asset-build/.job.yml` - Artifact publishing

### Documentation

**User-Facing** (`/docs/` - Sphinx-generated):
- `/docs/index.md` - Main entry with compatibility matrix
- `/docs/configuration/` - Configuration guides (settings, metrics lists, troubleshooting)
- `/docs/installation/` - Installation guides (Helm, Docker, GPU operator, Debian)
- `/docs/integrations/` - Integration guides (Prometheus, Grafana, SLURM, health monitoring)
- `/docs/releasenotes.md` - Release notes

**Developer**:
- `/docs/developerguide.md` - Build instructions
- `/internal/metricsmap.md` - Metrics mapping
- `/gpuagent/developerguide.md` - GPUAgent architecture

---

**Last Updated:** 2026-04-05

This document serves as the comprehensive source for all knowledge base content. Future agents should extract relevant sections to create individual focused documentation files.

# AMD Device Metrics Exporter - AI Agent Instructions

## Architecture Overview

This is a complex multi-component system for collecting AMD GPU/NIC telemetry and exposing it as Prometheus metrics. The architecture consists of:

- **Main Exporter** (`pkg/exporter/`): Prometheus metrics server that orchestrates data collection from GPU/NIC agents
- **GPU Agent** (`gpuagent/`): C++ service using AMD SMI libraries to collect GPU metrics via gRPC
- **NIC Agent** (`pkg/amdnic/`): Network interface monitoring component  
- **ROCProfiler Client** (`rocprofilerclient/`): Performance profiling metrics collection
- **Multiple SMI Libraries**: `libamdsmi/` (main AMD SMI), `libgimsmi/` (virtualization support)

Key data flow: GPU Agent (C++) ← gRPC → Main Exporter (Go) → Prometheus endpoint

## Critical Build System

This project uses a complex multi-stage Docker-based build system with OS-specific variants:

### Key Build Commands
```bash
# Enter containerized build environment (REQUIRED for development)
make docker-shell

# Build everything (run inside docker-shell)
make all

# Build specific components when modified
make amdsmi-compile-all      # Rebuild AMD SMI library
make gpuagent-compile-full   # Rebuild GPU agent  
make rocprofiler-compile     # Rebuild profiler client

# Create packages
make pkg                     # Both .deb and .rpm packages
make docker                  # Standard container image
```

### Build Environment Setup
- **Always use `make docker-shell`** for development - direct builds often fail due to complex dependencies
- Multi-OS support: Ubuntu 22.04/24.04, RHEL9, Azure Linux 3.0
- Uses `dev.env` for build configuration overrides
- GPU agent requires specific C++ toolchain and AMD headers

## Configuration System

Configuration is JSON-based with metric field selection:

```json
{
  "ServerPort": 5000,
  "CommonConfig": { "MetricsFieldPrefix": "amd_" },
  "GPUConfig": {
    "Fields": ["GPU_PACKAGE_POWER", "GPU_TEMPERATURE", ...],
    "ExtraPodLabels": ["workload", "nodepool"]
  }
}
```

- Metric fields are selectively enabled via `GPUConfig.Fields` array
- See `internal/metricsmap.md` for metric mappings between exporter ↔ GPU agent ↔ AMD SMI
- Platform-specific metrics (MI2xx vs MI3xx) documented in metricsmap

## Development Patterns

### Package Structure
- `cmd/exporter/`: Main exporter binary entry point
- `cmd/testrunner/`: Test runner for validation scenarios  
- `pkg/exporter/`: Core prometheus metrics collection logic
- `pkg/amdgpu/gpuagent/`: gRPC client for GPU agent communication
- `pkg/client/`: Kubernetes API client for pod/node metadata

### Testing & Debugging
- Use `metricsclient` tool for debugging: `bin/metricsclient -get -id 1`
- Mock ECC errors: `metricsclient -ecc-file-path ecc.json` 
- E2E tests in `test/e2e/` and `test/k8s-e2e/`
- Relaxed flag parsing: Set `AMD_EXPORTER_RELAXED_FLAGS_PARSING=1`

### Key Integration Points
- **gRPC Communication**: GPU agent exposes Unix socket at `/sockets/amdgpu_device_metrics_exporter_grpc.socket`
- **Kubernetes Integration**: Auto-discovery of pod labels, node metadata via K8s API
- **Container Deployment**: Requires `--device=/dev/dri --device=/dev/kfd` for GPU access
- **Helm Charts**: Available in `helm-charts/` with configurable values

### Common Gotchas
- GPU agent must be built before exporter container - uses precompiled binaries in `assets/`
- Submodules must be updated: `git submodule update --init --recursive`
- Multi-architecture builds use different base images and library paths
- Metric field names are inconsistent between components - always check `internal/metricsmap.md`

### File Modification Impact
- Changes to `libamdsmi/` → Rebuild with `make amdsmi-compile-all`
- Changes to `gpuagent/` → Rebuild with `make gpuagent-compile-full` 
- Changes to Go code → Standard `make all` sufficient
- Changes to protobuf → Regenerate with protoc (included in `make all`)

When working on metrics collection, always test with actual GPU hardware or use mock data injection via `metricsclient` tool.
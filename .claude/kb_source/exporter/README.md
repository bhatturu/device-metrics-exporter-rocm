# AMD Device Metrics Exporter - Knowledge Base

This directory contains comprehensive documentation for AI agents working with the AMD Device Metrics Exporter project.

## Knowledge Base Structure

### ✅ Created Documentation

1. **[architecture.md](architecture.md)** - Complete architecture overview (CREATED)
   - Three-tier architecture with detailed diagrams
   - Component interactions and data flows
   - GPUAgent library internals
   - NIC exporter design
   - Request flow diagrams

2. **[build-system.md](build-system.md)** - Build system and development environment
   - Docker build environment (docker-shell)
   - Makefile structure and targets
   - Environment variables and configuration
   - External dependencies (submodules)
   - Build workflow and artifact creation

3. **[configuration.md](configuration.md)** - Configuration system
   - Config schema and structure
   - GPU metric configuration (120+ fields)
   - NIC metric configuration (200+ fields)
   - Dynamic config reloading
   - Per-node profiler control
   - Health threshold configuration

4. **[deployment.md](deployment.md)** - Deployment modes and integration
   - Kubernetes deployment (helm charts, DaemonSet)
   - Debian/RPM packages (systemd services)
   - SR-IOV mode
   - Scheduler integrations (K8s PodResources, SLURM)

5. **[ci-cd.md](ci-cd.md)** - CI/CD pipeline
   - Internal build jobs (.job.yml)
   - Asset publishing pipeline (asset-build/.job.yml)
   - Test targets and sanity checks
   - Artifact outputs and versioning

### Component Deep Dives

6. **[gpu-exporter.md](gpu-exporter.md)** - GPU exporter details
   - GPUAgent library architecture
   - gRPC service clients (GPU, UAL, Events)
   - GPU metrics categories (0-799, 801-1200)
   - Platform-specific metrics (MI2xx, MI3xx)
   - Mock mode for testing

7. **[rocprofiler.md](rocprofiler.md)** - ROCProfiler integration
   - rocpctl command execution
   - PTL (Peak Tops Limiter) handling
   - Caching strategy and failure handling
   - Platform restrictions
   - Configuration and per-node control

8. **[health-monitoring.md](health-monitoring.md)** - Health service
   - Health service architecture (gRPC over Unix socket)
   - Health monitoring loop (30s default)
   - ECC threshold evaluation
   - Exit-on-agent-down mechanism
   - K8s integration and node labels

9. **[nic-exporter.md](nic-exporter.md)** - NIC exporter details
   - CLI-based architecture
   - Three clients (NICCtl, Ethtool, RDMA Stats)
   - Pod→Device discovery and mapping
   - Workload correlation
   - Health service

### Operational Guides

10. **[troubleshooting.md](troubleshooting.md)** - Comprehensive troubleshooting (CREATED)
    - ✅ Driver load timing issues (primary failure mode)
    - ✅ GPU agent communication failures
    - ✅ ROCProfiler failures
    - ✅ Configuration errors
    - ✅ Scheduler client failures
    - ✅ NIC exporter failures
    - ✅ Diagnostic commands and logs

11. **adding-metrics.md** - Adding new metrics (See COMPLETE_KB_SOURCE.md)
    - GPU non-profiler metrics (step-by-step)
    - GPU profiler metrics (enum ranges)
    - NIC metrics (CLI tool integration)
    - Proto definitions and code generation
    - Testing and validation

12b. **[pr-guidelines.md](pr-guidelines.md)** - PR guidelines (CREATED)
    - ✅ Scope discipline (in/out, no asset or generated-file churn)
    - ✅ PR description template (Summary / Files / Test plan / Risk)
    - ✅ Commit message format + safety rules (no force-push, no `--no-verify`)
    - ✅ Sensitive-data scrub table + final grep recipe
    - ✅ Pre-flight checklist

12a. **[sriov-deployment.md](sriov-deployment.md)** - SR-IOV / GIM deployment knowledge (CREATED)
    - ✅ Required `docker run` device list (5 devices, not just `/dev/dri`)
    - ✅ `AGA_SMI_LAZY_INIT=1` lazy SMI lifecycle — intent, where set, how to verify
    - ✅ Metric prefix gotcha (`gpu_*` not `amd_gpu_*` in SR-IOV image)
    - ✅ Diagnostic recipe for "exporter HTTP 200 but no metrics"
    - ✅ Surgical single-asset refresh recipe (just `gpuagent_sriov_static.bin.gz`)

12. **[partition-vm-environments.md](partition-vm-environments.md)** - Partition and VM environment metrics (CREATED)
    - ✅ Deployment environments (Hypervisor vs Baremetal)
    - ✅ GPU partition modes (SPX, CPX, DPX, QPX)
    - ✅ Physical sensor metrics (primary partition only)
    - ✅ Per-partition metrics (all partitions)
    - ✅ Metric availability matrix format
    - ✅ Testing in partition environments
    - ✅ PRD requirements for partition support

13. **[critical-paths.md](critical-paths.md)** - File path reference
    - Entry points and configuration files
    - GPU exporter components
    - NIC exporter components
    - Scheduler integration
    - Core infrastructure
    - Build system files
    - Documentation files

## Using This Knowledge Base

### For Debugging Agents

When creating debugging agents, focus on:
- **[troubleshooting.md](troubleshooting.md)** - Comprehensive failure scenarios
- **[gpu-metrics-details.md](gpu-metrics-details.md)** - ECC error injection, testing tools
- **[partition-vm-environments.md](partition-vm-environments.md)** - Partition/VM metric availability
- **[architecture.md](architecture.md)** - Understanding component interactions
- **[health-monitoring.md](health-monitoring.md)** - Health service and monitoring
- **[critical-paths.md](critical-paths.md)** - Key files to investigate

### For Feature Planning Agents

When creating feature planning agents, focus on:
- **[architecture.md](architecture.md)** - Understanding existing design
- **[gpu-metrics-details.md](gpu-metrics-details.md)** - Static/dynamic metrics, special cases (ECC, AFID)
- **[partition-vm-environments.md](partition-vm-environments.md)** - Partition support requirements
- **[adding-metrics.md](adding-metrics.md)** - Metric addition workflow
- **[build-system.md](build-system.md)** - Build and test process
- **[deployment.md](deployment.md)** - Deployment considerations

### For Build/CI Agents

When creating build/CI agents, focus on:
- **[build-system.md](build-system.md)** - Complete build workflow
- **[ci-cd.md](ci-cd.md)** - CI/CD pipeline details
- **[deployment.md](deployment.md)** - Package creation

## Design Principles

The AMD Device Metrics Exporter follows these core principles:

1. **Graceful Degradation** - Unsupported features are marked, not failed
2. **Self-Recovery** - Auto-restart, reconnection logic
3. **Security** - Unix sockets, no network exposure
4. **Extensibility** - Modular clients, builder pattern
5. **Runtime Configurability** - Dynamic config reload

## Quick Reference

- **Main Project:** repo root of `pensando/device-metrics-exporter`
- **Quick Guide:** [CLAUDE.md](../../CLAUDE.md)
- **Example Config:** [example/config.json](../../example/config.json)
- **Build Entry:** `make docker-shell` → `make all`
- **Config Schema:** [pkg/exporter/proto/exporterconfig.proto](../../pkg/exporter/proto/exporterconfig.proto)

---

**Last Updated:** 2026-04-06

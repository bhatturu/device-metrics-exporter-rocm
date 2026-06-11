# Product Requirements Documents (PRDs)

This directory contains Product Requirements Documents for GPU metrics features in the Device Metrics Exporter.

## Purpose

PRDs serve as the design and implementation specification for adding new GPU metrics to the exporter. Each PRD documents:

- Metric specification and purpose
- Proto definitions and implementation details
- Driver and platform requirements
- Testing requirements
- Acceptance criteria

## Directory Structure

```
docs-internal/knowledge/prds/
├── README.md                           # This file
├── templates/
│   └── gpu-metric-prd-template.md     # Standard PRD template
├── 2026/
│   ├── Q1/                             # Q1 2026 PRDs
│   └── Q2/                             # Q2 2026 PRDs
└── implemented/                        # Archived PRDs (completed features)
```

## PRD Naming Convention

Format: `PRD-GPU-[YYYYMMDD]-[NN]-[metric-name-slug].md`

- **YYYYMMDD**: Date created
- **NN**: Sequential number (resets daily)
- **metric-name-slug**: Descriptive kebab-case name

Examples:
- `PRD-GPU-20260405-01-memory-bandwidth.md`
- `PRD-GPU-20260405-02-throttle-reason.md`
- `PRD-GPU-20260406-03-simd-efficiency.md`

## Creating a New PRD

### Using the PRD Agent (Recommended)

The easiest way to create a PRD is using the PRD Agent:

```
User: "I want to add GPU [metric name]"
```

The agent will:
1. Analyze the codebase to check if the metric exists
2. Ask targeted questions about requirements
3. Generate a comprehensive PRD automatically
4. Update this index

### Manual Creation

1. Copy the template:
   ```bash
   cp docs-internal/knowledge/prds/templates/gpu-metric-prd-template.md docs-internal/knowledge/prds/2026/Q2/PRD-GPU-YYYYMMDD-NN-metric-name.md
   ```

2. Fill in all sections following the template structure

3. Add entry to this README (see Index section below)

## PRD Status Workflow

1. **Draft** - Initial creation, under development
2. **In Review** - Submitted for stakeholder review
3. **Approved** - Approved for implementation
4. **Implemented** - Feature completed and merged

Once implemented, move the PRD to `implemented/` directory for archival.

## Index of PRDs

### 2026 Q2

- **[PRD-GPU-20260406-01](./2026/Q2/PRD-GPU-20260406-01-ecc-deferred-errors.md)** - ECC Deferred Error Count Metrics - Status: Draft - Author: praveen with Claude PRD Agent

<!-- Template for new entries:
- **[PRD-GPU-YYYYMMDD-NN](./2026/Q2/PRD-GPU-YYYYMMDD-NN-metric-name.md)** - [Metric Name] - Status: [Draft/In Review/Approved/Implemented] - Author: [Name]
-->

### 2026 Q1

*No PRDs created yet*

### Implemented (Archived)

*No completed PRDs yet*

## Quick Reference

### Metric Type Classification

| Type | Index Range | Description | Collection Method |
|------|-------------|-------------|-------------------|
| **Non-Profiler** | 0-800 | Standard GPU metrics | AMD-SMI API, sysfs |
| **Profiler** | 801-1200 | Performance analysis metrics | ROCProfiler (rocpctl) |

### Special Metric Categories

| Category | Description | Characteristics |
|----------|-------------|----------------|
| **Critical Metrics** | Essential for workload evaluation | Temperature, Power, Activity, VRAM, Profiler metrics |
| **ECC Metrics** | Error Correcting Code errors | 19 error types, health-critical, **partition 0 only** |
| **Static Metrics** | Values constant regardless of workload | ECC error counts, device info, capabilities |
| **Dynamic Metrics** | Values change with workload activity | Power, temperature, utilization, profiler metrics |

### Key Proto Files

| File | Purpose |
|------|---------|
| `pkg/amdgpu/proto/gpu.proto` | GPU metrics proto definitions (GPUStats, GPUStatus) |
| `pkg/exporter/proto/exporterconfig.proto` | Metric field enums and configuration |
| `pkg/amdgpu/proto/rocprofiler.proto` | Profiler metrics structure |
| `pkg/amdgpu/proto/gpumetricssvc.proto` | Health service proto |

### GPUAgent Submodule Files

| File | Purpose | Driver Variant |
|------|---------|----------------|
| `gpuagent/sw/nic/gpuagent/protos/gpu.proto` | GPU metrics proto in gpuagent | Both |
| `gpuagent/sw/nic/gpuagent/api/smi/amdsmi/smi_api.cc` | AMD-SMI implementation | Baremetal |
| `gpuagent/sw/nic/gpuagent/api/smi/gimamdsmi/smi_api.cc` | GIM AMD-SMI implementation | SR-IOV/Hypervisor |

**Note**: Changes to gpuagent require PR in git@github.com:ROCm/gpu-agent.git

### Implementation Files

| File | Purpose |
|------|---------|
| `pkg/amdgpu/gpuagent/gpuagent_gpu_metrics.go` | Prometheus metric registration |
| `pkg/amdgpu/gpuagent/gpuagent_gpu.go` | Metric collection logic |
| `pkg/amdgpu/rocprofiler/rocpclient.go` | ROCProfiler integration |
| `pkg/amdgpu/gpuagent/gpuagent_test.go` | Unit tests |

### Documentation Files

| File | Purpose | Required for PRD |
|------|---------|------------------|
| `docs/configuration/metricslist.md` | User-facing metrics catalog | ✓ Yes |
| `docs/index.md` | Compatibility matrix | If driver/platform changed |
| `internal/metricsmap.md` | Internal metric mappings + Critical Metrics list | ✓ Yes |
| `docs-internal/knowledge/exporter/gpu-metrics-details.md` | Static/dynamic metrics, ECC injection, special cases | For ECC/special cases |
| `example/config.json` | Configuration examples | ✓ Yes |
| `docs/releasenotes.md` | Release notes | ✓ Yes |

### Testing Tools

| Tool | Purpose | Use Case |
|------|---------|----------|
| **metricsclient** | Mock ECC error injection | Safe, reversible testing (recommended) |
| **AMDGPURAS** | Real HW error injection | Risky, platform-specific (use with caution) |
| **amd-smi** | Reference validation tool | Metric accuracy comparison |
| **rocpctl** | Profiler metrics collection | Profiler metric validation |

### Platform Support

| Platform | Series | Examples | Partition Support |
|----------|--------|----------|-------------------|
| MI2xx | CDNA 2 | MI210, MI250, MI250X | Non-partitioned (SPX) |
| MI3xx | CDNA 3 | MI300A, MI300X | SPX, CPX (8), DPX (2), QPX (4) |

**Important Notes:**
- **ECC Metrics**: Only available on primary partition 0 (hardware/AMD-SMI limitation)
- **Partitioned GPUs**: Secondary partitions (1-7) do not expose ECC error counts
- **Non-partitioned (SPX)**: All metrics available on single partition

### Driver Requirements

| Deployment Mode | Driver Type | Minimum Version | Implementation |
|----------------|-------------|-----------------|----------------|
| **Baremetal** | amdgpu | 6.4.x+ | amdsmi (standard) |
| **SR-IOV/Hypervisor** | GIM | 8.3.0.K+ | gimamdsmi (SR-IOV variant) |

**Note**: When adding metrics that require gpuagent changes, consider both driver variants.

## PRD Requirements Checklist

When creating a PRD, ensure it includes:

### Core Sections (10 required)
- [ ] 1. Metric Overview (name, description, purpose, critical classification)
- [ ] 2. Technical Specification (proto definitions, data specs, example output)
- [ ] 3. Driver and Platform Requirements (AMD-SMI, drivers, platforms, partition support)
- [ ] 4. Implementation Plan (proto changes, implementation files, checklist)
- [ ] 5. Testing Requirements (unit, integration, SR-IOV, performance, **system test validation**)
- [ ] 6. Documentation Updates (user docs, developer docs, config examples)
- [ ] 7. Acceptance Criteria (complete deliverables checklist)
- [ ] 8. Known Limitations (platform support, dependencies, constraints)
- [ ] 9. References (AMD-SMI docs, code references, related docs)
- [ ] 10. Approval Sign-off (stakeholder approvals, timeline)

### System Test Validation (Section 5.5+)
- [ ] Functional validation (config behavior, /metrics output, workload scenarios, static vs dynamic)
- [ ] Metric accuracy validation (reference tool, comparison method, tolerance)
- [ ] Negative test cases (disabled metric, unsupported platforms, driver compatibility)
- [ ] Platform-specific validation (MI2xx vs MI3xx, Hypervisor vs Baremetal, partition modes)
- [ ] Special cases (ECC injection with metricsclient/AMDGPURAS, profiler cache, health thresholds)

### Documentation Requirements
- [ ] metricslist.md entry with Hypervisor/Baremetal columns
- [ ] internal/metricsmap.md mapping + Critical Metrics list (if applicable)
- [ ] example/config.json configuration example
- [ ] docs/index.md compatibility matrix (if driver/platform changed)
- [ ] docs-internal/knowledge/exporter/gpu-metrics-details.md reference (for ECC/special cases)
- [ ] Release notes entry

### Validation Checks
- [ ] No duplicate enum indices in exporterconfig.proto
- [ ] Naming conventions followed (Prometheus snake_case, Proto PascalCase, Enum UPPER_SNAKE)
- [ ] Proto field numbers don't conflict
- [ ] Driver variant support documented (amdsmi and gimamdsmi if gpuagent changes required)
- [ ] Partition support documented (primary partition 0 for ECC metrics)
- [ ] Critical metric classification justified

## Changelog

- **2026-04-06**: Added partition support, driver requirements, testing tools, system test validation requirements
- **2026-04-05**: PRD Agent created

## Related Documentation

- [CLAUDE.md](../../CLAUDE.md) - Project quick reference
- [PRD Agent](../prd-agent.md) - Interactive PRD generation agent
- [GPU Metrics Details](../exporter/gpu-metrics-details.md) - Static/dynamic metrics, ECC error injection, special cases
- [Developer Guide](../../docs/developerguide.md) - Build and development instructions
- [Metrics List](../../docs/configuration/metricslist.md) - User-facing metrics catalog
- [Metrics Map](../../internal/metricsmap.md) - Internal metric mappings and critical metrics list
- [Release Notes](../../docs/releasenotes.md) - Version history

## Support

For questions about PRDs or the PRD Agent:
- Review existing PRDs in this directory for examples
- Use the PRD Agent: "help me add a GPU metric"
- Consult the PRD template: `docs-internal/knowledge/prds/templates/gpu-metric-prd-template.md`
- Check the Developer Guide: `docs/developerguide.md`

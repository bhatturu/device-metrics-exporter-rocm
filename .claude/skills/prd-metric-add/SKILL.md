---
name: PRD Metric Add
description: This skill should be used when adding new GPU metrics to the Device Metrics Exporter. Guides through discovery, requirements gathering, PRD generation, and documentation updates.
version: 2.0.0
---

# PRD Metric Add Workflow

Complete workflow for adding new GPU metrics to the AMD Device Metrics Exporter.

**Note**: For complex scenarios, the `prd-agent` orchestrates this workflow with full validation.

## Workflow Phases

### Phase 1: Discovery

**Objective**: Determine if metric exists and find next available index

1. **Extract metric name** from user's request
2. **Search proto files**: `gpu.proto`, `exporterconfig.proto`, `rocprofiler.proto`
3. **Check documentation**: `metricslist.md`, `network-metricslist.md`
4. **Find next index**:
   - Non-profiler: 0-800
   - Profiler: 801-1200

### Phase 2: Requirements Gathering (11 Questions)

1. **Proto Status**: Existing field vs new field vs profiler
2. **Metric Type**: Standard / Profiler / ECC / AFID
3. **AMD-SMI Version**: Which version introduced this field?
4. **Driver Requirements**: Type, version, limitations
5. **Platform Support**: MI2xx / MI3xx / all platforms
6. **Health Service**: Include in health monitoring?
7. **Data Specifications**: Type, unit, range, collection method
8. **Purpose**: Use case, target audience, decisions enabled
9. **Critical Classification**: Is this a critical metric?
10. **System Test Validation**: How to validate in tests?
11. **Partition/VM Support**: Hypervisor, baremetal, VF, guest VM

### Phase 3: PRD Generation

**File Location**: `docs-internal/knowledge/prds/YYYY/QN/PRD-GPU-YYYYMMDD-NN-metric-name.md`

**PRD Structure** (11 sections):
1. Header (ID, date, status, target release)
2. Overview (purpose, why needed, use cases)
3. Requirements (functional, non-functional, platform)
4. Technical Specifications (proto, Prometheus, data specs)
5. Dependencies (AMD-SMI, driver, ROCm versions)
6. Implementation Details (code locations, collection flow)
7. Special Cases (ECC, profiler, partition/VM)
8. Documentation Updates (4 files)
9. Testing and Validation (unit, system, integration)
10. Success Criteria
11. Open Questions

### Phase 4: Documentation Updates

**Four files to update**:

1. **`docs/configuration/metricslist.md`**:
   ```markdown
   | Prometheus Metric | Description | Hypervisor | Baremetal | Type | Unit |
   |-------------------|-------------|------------|-----------|------|------|
   | `gpu_new_metric` | Description | Yes | Yes | Gauge | Unit |
   ```

2. **`internal/metricsmap.md`**:
   ```markdown
   | GPU_NEW_METRIC | new_metric | gpu_new_metric | Notes |
   ```

3. **`example/config.json`**:
   ```json
   {
     "name": "GPU_NEW_METRIC",
     "enabled": true,
     "description": "..."
   }
   ```

4. **`docs/releasenotes.md`**:
   ```markdown
   ## [vX.Y.Z] - TBD
   ### Added
   - New metric: `gpu_new_metric` - Description
   ```

### Phase 5: Validation

**Six validation categories**:
1. **Index**: No duplicates, correct range, sequential
2. **Naming**: snake_case (Prometheus), PascalCase (proto field), UPPER_SNAKE_CASE (enum)
3. **Dependencies**: AMD-SMI version, driver version, platform support
4. **Documentation**: Correct structure, accurate columns, user-friendly descriptions
5. **Critical Metrics**: Justification, category match, no duplicates
6. **Special Cases**: ECC (19 types), Profiler (801-1200, cache), VM/Partition support

## Special Metric Categories

| Category | Index Range | Characteristics | Examples |
|----------|-------------|-----------------|----------|
| **Standard** | 0-800 | AMD-SMI API, every scrape | Power, Temperature |
| **Profiler** | 801-1200 | rocpctl, 10s cache | GPU_PROF_* |
| **ECC** | 0-800 | Health-critical, 19 types | Correctable/Uncorrectable errors |
| **AFID** | 0-800 | Process-specific (TBD) | Function tracking |

## Naming Conventions

| Context | Format | Example |
|---------|--------|---------|
| Prometheus | snake_case | `gpu_memory_bandwidth` |
| Proto Field | PascalCase | `MemoryBandwidth` |
| Proto Enum | UPPER_SNAKE_CASE | `GPU_MEMORY_BANDWIDTH` |

## Critical Files

| File | Purpose |
|------|---------|
| `pkg/amdgpu/proto/gpu.proto` | Field definitions |
| `pkg/exporter/proto/exporterconfig.proto` | Exportable metric enum |
| `docs/configuration/metricslist.md` | User metrics catalog |
| `internal/metricsmap.md` | Developer mapping |
| `example/config.json` | Config template |
| `docs-internal/knowledge/prds/` | PRD documents |

## Integration with PRD Agent

The `prd-agent` handles:
- Automated codebase analysis
- Interactive 11-question framework
- PRD generation with validation
- Documentation updates
- Special case handling (ECC, profiler, AFID)

**When to use agent**:
- Multiple related metrics
- Complex special cases
- First-time metric addition
- Validation failures

## Summary

This skill provides a structured 5-phase workflow for adding GPU metrics: Discovery → Requirements (11 questions) → PRD Generation → Documentation Updates → Validation.

**Outcome**: Comprehensive PRD + Updated documentation + Clear implementation roadmap

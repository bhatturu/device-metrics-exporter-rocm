---
name: prd-metric-implementation
description: Implements GPU metric additions following PRDs created by prd-metric-add skill. Only for GPU metric feature additions.
agent: implementation-agent
triggerPhrases:
  - implement prd
  - implement the prd
  - implement metric prd
  - add metrics from prd
  - execute prd
  - implement gpu metric
  - implement the gpu metric
  - code the metric
---

# PRD Metric Implementation Skill

This skill implements GPU metric additions by following Product Requirements Documents (PRDs) **created by the `/prd-metric-add` skill**.

**Scope**: This skill is ONLY for implementing GPU metric additions. It is not a general-purpose PRD implementation tool.

## When to Use

Use this skill when you want to:
- Implement a GPU metric defined in a PRD created by `/prd-metric-add`
- Add multiple related GPU metrics from a PRD
- Execute the implementation plan from a metric addition PRD
- Translate a metric PRD into working code across GPUAgent and Device-Metrics-Exporter

## What This Skill Does

The PRD Implementation Agent will:

1. **Read and analyze** the specified PRD document
2. **Extract metric specifications** (names, types, sources, categories)
3. **Make coordinated changes** across:
   - GPUAgent submodule (6 files: proto, C++, CLI)
   - Device-Metrics-Exporter (2 files: proto, Go metrics)
   - Example configs (3 files: JSON, YAML)
4. **Validate** the implementation
5. **Provide build/test guidance**

## How to Use

### Option 1: Specify PRD by name
```
/prd-metric-implementation PRD-001-ecc-deferred-errors
```

### Option 2: Natural language
```
Implement the ECC deferred errors PRD
```

### Option 3: From PRD directory
```
Add the metrics defined in .claude/prds/PRD-001-ecc-deferred-errors.md
```

**Note**: This skill expects PRDs in `.claude/prds/` that were created by `/prd-metric-add`. For general feature requests, use `/prd-metric-add` first to generate the PRD.

## What You'll Need

Before running this skill:
- ✅ A PRD document in `.claude/prds/` (use `/prd-metric-add` to create one)
- ✅ Understanding of what metrics to add
- ✅ Write access to both GPUAgent submodule and main repo

## Implementation Coverage

The agent will modify these file types:

### GPUAgent Submodule (`gpuagent/`)
- `gpu.proto` - Protocol buffer definitions
- `aga_gpu.hpp` - C++ struct definitions  
- `smi_api.cc` (baremetal) - AMD-SMI integration (amdsmi/)
- `smi_api.cc` (SR-IOV) - AMD-SMI integration (gimamdsmi/)
- `gpu_to_proto.hpp` - Struct to proto mapping
- `gpu.go` - CLI display formatting

### Device-Metrics-Exporter (main repo)
- `exporterconfig.proto` - Metric enum definitions
- `gpuagent_gpu_metrics.go` - Prometheus metric registration and export
- `test/k8s-e2e/exporter_test.go` - Integration tests (real hardware)

### Example Configurations
- `example/config-gpu.json`
- `example/config.json`
- `example/configmap.yaml`

## Example Workflow

```
User: "I want to implement the ECC deferred errors PRD"

Agent:
1. Reads PRD from .claude/prds/PRD-001-ecc-deferred-errors.md
2. Extracts: 19 deferred error fields across 6 ECC blocks
3. Creates task list for all file modifications
4. Implements changes:
   - Adds fields to gpu.proto
   - Updates C++ structs
   - Populates from AMD-SMI
   - Registers Prometheus metrics
   - Updates configs
5. Validates compilation
6. Provides build commands and next steps
```

## After Implementation

The agent will provide:
- ✅ Summary of all changes made
- ✅ Build commands to validate (`make gen`, `make all`)
- ✅ Suggested commit message
- ✅ Testing recommendations
- ✅ Next steps (PR creation, integration tests, etc.)

## Tips

- **Have a PRD ready**: Use `/prd-metric-add` first if you don't have one
- **Verify proto field numbers**: PRDs may have incorrect indices - check actual proto file
- **Review changes**: Agent will show all modifications before committing
- **Test thoroughly**: Build and test gpuagent + exporter after implementation
- **One metric category at a time**: Don't mix ECC, temperature, and performance metrics in one PRD
- **Check both AMD-SMI implementations**: Verify field availability before implementing SR-IOV support

## Implementation Best Practices

### Proto Field Numbering
- **CRITICAL**: Proto field numbers in `gpu.proto` must be sequential from the last existing field
- PRDs may show incorrect field numbers - always verify the actual next available index
- Example: If last field is index 68, new fields start at 69 (NOT 92 as some PRDs may state)

### C++ Header Files (`aga_gpu.hpp`)
- **Group related fields together** - don't just append at the end
- For ECC metrics: Group correctable, uncorrectable, and deferred errors for each block together
- Follow existing patterns in the struct for consistency
- Maintain alphabetical/logical ordering within groups

### Variable Declaration Order (C++ files)
- Follow gpuagent coding guidelines: declare variables in **ascending order by line length**
- Example: `uint64_t total_deferred_count = 0;` comes before `uint64_t total_correctable_count = 0;`
- This applies to all `.cc` files in gpuagent submodule

### SR-IOV Implementation
- **Check AMD-SMI header compatibility FIRST**: Verify the AMD-SMI field exists in corresponding header
- Compare `amdsmi.h` in both:
  - `nic/third-party/rocm/amd_smi_lib/include/` (baremetal)
  - `nic/third-party/rocm/gim_amd_smi_lib/include/` (SR-IOV)
- Implementation is **not always identical** - some fields may only be available in baremetal
- If field is unavailable in SR-IOV, handle gracefully or skip
- **When fields are available in both**: Apply same logic to both implementations:
  - `amdsmi/smi_api.cc` (baremetal)
  - `gimamdsmi/smi_api.cc` (SR-IOV/hypervisor)

### Mock/Proto Conversion
- Don't forget `gpu_to_proto.hpp` - required for mock support and testing
- Add `set_*()` calls for all new proto fields in `aga_gpu_api_stats_to_proto()`
- Follow the same pattern as existing fields

### Test Requirements
- **k8s-e2e tests**: Use real hardware - verify metrics are **exported** when enabled
- **NO mock data injection** in k8s-e2e - these are real hardware integration tests
- Test should check metric **presence** in /metrics endpoint, not specific values
- Values may be 0 on real hardware - that's expected and valid
- Enable fields in configmap and verify they appear in Prometheus output
- Pattern: Update config → reload → verify metric exists in output

## Related Skills

- `/prd-metric-add` - Create a PRD for a new GPU metric (run this first)
- `/builder` - Build gpuagent and exporter binaries after implementation
- `/simplify` - Review and optimize the generated code

---

## Workflow Integration

This skill is part of a two-step workflow for adding GPU metrics:

1. **Step 1**: Use `/prd-metric-add` to create a PRD document
2. **Step 2**: Use `/prd-metric-implementation` to implement the PRD

**Ready to implement a GPU metric PRD?** Just say "implement [PRD name]" or use `/prd-metric-implementation`!

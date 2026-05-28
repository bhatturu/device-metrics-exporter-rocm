---
name: prd-dev-workflow
description: Complete PRD development workflow through implementation, testing, building, verification, and documentation phases with progress tracking and resumption support
tags: [prd, workflow, implementation, testing, building, verification, documentation]
agent: task-tracker
---

# PRD Development Workflow

Orchestrates the complete development lifecycle for implementing Product Requirements Documents (PRDs) with tracked phases and user approval gates.

## When to Use This Skill

Trigger this skill when:
- You want to implement a PRD through a structured, tracked workflow
- You need to ensure testing, building, and verification are all completed
- You want to track progress and resume work later
- User says: "implement PRD [name]", "start PRD workflow", "work on PRD"

**Note**: For quick GPU metric implementation without full workflow, use `/prd-metric-implementation` instead.

## Usage

### Start New Workflow
```
/prd-dev-workflow <PRD-file-path>
```

Example:
```
/prd-dev-workflow .claude/prds/PRD-GPU-TEMPERATURE-RANGE.md
```

### Resume Existing Workflow
```
/prd-dev-workflow resume <PRD-ID>
```

Example:
```
/prd-dev-workflow resume PRD-GPU-TEMPERATURE-RANGE
```

## Workflow Phases

The workflow guides you through 5 sequential phases:

### Phase 1: Implementation & Development
- Reads and analyzes the PRD
- For GPU metrics: delegates to `implementation-agent`
- For other features: asks clarifying questions and implements directly
- **Gate**: User must approve before proceeding to testing

### Phase 2: Test Planning & Development
- Assesses applicable test types (unit, E2E mock, K8s E2E real HW)
- Identifies test gaps and suggests test scenarios
- Creates test files in appropriate locations
- **Gate**: User must approve test coverage before proceeding to building

### Phase 3: Building
- Determines which components need to be built
- Uses `builder` skill to build gpuagent, exporter, or docker images
- Ensures all modules compile successfully
- **Gate**: User must approve successful builds before proceeding to verification

### Phase 4: Verification
- Runs applicable tests (unit, E2E mock, K8s E2E)
- For real HW tests: asks for explicit environment details
- Handles test failures with user input
- **Gate**: User must approve verification results before proceeding to documentation

### Phase 5: Documentation
- Uses `doc-agent` to update user-facing documentation based on PRD
- Updates metrics catalog, configuration reference, installation guides, etc.
- Follows Sphinx documentation standards
- **Gate**: User must approve documentation changes to complete workflow

## Progress Tracking

Progress is tracked in `.claude/prd_task_tracker/<PRD-ID>-status.json` with:
- Current phase and completion status
- Timestamps for phase start/completion
- Notes and metadata for each phase
- List of commits, files modified, tests added

## Resumption

Workflows can be paused and resumed later:
- Status is saved after each phase
- Use `resume` command to pick up where you left off
- All context (PRD, changes, test results) is preserved

## Example Session

```
User: /prd-dev-workflow .claude/prds/PRD-GPU-TEMPERATURE-RANGE.md

Agent: Starting PRD workflow for GPU Temperature Range metric...

[Phase 1: Implementation]
- Analyzing PRD requirements
- Delegating to implementation-agent
- Implementing metric in gpuagent and exporter
- ✓ Implementation complete

Review changes and approve to proceed to testing? [Yes/No]

User: Yes

[Phase 2: Test Planning]
- Identified test types: unit, e2e-mock
- Created unit tests in pkg/amdgpu/gpuagent/temp_range_test.go
- Created E2E mock test in test/e2e/gpu_temp_range_test.go
- ✓ Test planning complete

Review test coverage and approve to proceed to building? [Yes/No]

User: Yes

[Phase 3: Building]
- Building gpuagent (C++ changes detected)
- Building exporter (Go changes detected)
- ✓ All builds successful

Approve to proceed to verification? [Yes/No]

User: Yes

[Phase 4: Verification]
- Running unit tests: PASSED
- Running E2E mock tests: PASSED
- K8s E2E real HW: Not applicable (no HW available)
- ✓ Verification complete

Approve to proceed to documentation? [Yes/No]

User: Yes

[Phase 5: Documentation]
- Delegating to doc-agent with PRD context
- Updated docs/configuration/metricslist.md (added GPU_TEMP_RANGE metric)
- Updated docs/configuration/configuration-settings.md (added config examples)
- ✓ Documentation complete

Workflow complete! Summary:
- PRD: PRD-GPU-TEMPERATURE-RANGE
- Files modified: 8 (gpuagent: 4, exporter: 4)
- Tests added: 12 unit tests, 3 E2E tests
- All tests passing
- Documentation updated: 2 files
- Ready for PR creation
```

## Advanced Options

The skill will ask upfront questions to customize the workflow:
- Which test types apply to this PRD?
- Should we build docker images or just binaries?
- Are there real hardware test environments available?

## Integration with Existing Skills

- Extends `/prd-metric-implementation` with test/build/verify phases
- Uses `/builder` for building components
- Can invoke other agents as needed (implementation-agent, builder-agent)

## Benefits

- **Structured**: Ensures no steps are skipped
- **Tracked**: Progress saved and resumable
- **Gated**: User approval required at each phase
- **Comprehensive**: Covers full dev lifecycle
- **Flexible**: Works with any PRD type

---

**Location**: `.claude/skills/prd-dev-workflow/`
**Agent**: `task-tracker`
**Created**: 2026-04-08

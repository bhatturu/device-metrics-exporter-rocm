# PRD Workflow Status Directory

This directory contains workflow status files for PRD implementation tracking.

## Purpose

The `/prd-dev-workflow` skill creates and updates JSON status files here to track progress through the 5-phase development workflow:
1. Implementation & Development
2. Test Planning & Development
3. Building
4. Verification
5. Documentation

## File Format

Each PRD workflow has a status file named `<PRD-ID>-status.json`:

```json
{
  "prd_id": "PRD-GPU-TEMPERATURE-RANGE",
  "prd_file": "docs-internal/knowledge/prds/PRD-GPU-TEMPERATURE-RANGE.md",
  "started_at": "2026-04-08T10:30:00Z",
  "current_phase": "implementation",
  "phases": {
    "implementation": {
      "status": "in_progress",
      "started_at": "2026-04-08T10:30:00Z",
      "completed_at": null,
      "notes": "Using implementation-agent for GPU metric addition"
    },
    "testing": {
      "status": "pending",
      "test_types": [],
      "started_at": null,
      "completed_at": null,
      "notes": ""
    },
    "building": {
      "status": "pending",
      "build_targets": [],
      "started_at": null,
      "completed_at": null,
      "notes": ""
    },
    "verification": {
      "status": "pending",
      "test_results": {},
      "started_at": null,
      "completed_at": null,
      "notes": ""
    },
    "documentation": {
      "status": "pending",
      "docs_updated": [],
      "started_at": null,
      "completed_at": null,
      "notes": ""
    }
  },
  "metadata": {
    "prd_type": "gpu_metric",
    "components_affected": ["gpuagent", "exporter"],
    "commits": []
  }
}
```

## Status Values

- `pending` - Not started yet
- `in_progress` - Currently working on this phase
- `completed` - User approved and phase finished
- `blocked` - Waiting on external dependency
- `skipped` - User chose to skip this phase

## Resumption

To resume a workflow:
```
/prd-dev-workflow resume <PRD-ID>
```

The agent will read the status file and continue from the last incomplete phase.

## Maintenance

- Status files are automatically created when starting a new workflow
- Files are updated after each phase transition
- Completed workflows can be archived or deleted manually
- Files are ignored by git (see .gitignore)

## Example Status Files

- `PRD-GPU-TEMPERATURE-RANGE-status.json` - GPU temperature range metric workflow
- `PRD-NIC-BANDWIDTH-status.json` - NIC bandwidth tracking workflow
- `PRD-UAL-MEMORY-status.json` - UAL memory metric workflow

---

**Created**: 2026-04-08
**Used by**: `/prd-dev-workflow` skill and `task-tracker`

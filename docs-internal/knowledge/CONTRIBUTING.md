# Contributing — Per-PR Plan File Requirement

Every PR targeting `main` must be associated with a plan file. A PR satisfies this if EITHER:

1. The PR body references an existing file path under `docs-internal/knowledge/` — for example:
   ```
   Plan: docs-internal/knowledge/plans/2026-05-28-my-feature.md
   ```
   The referenced file must already exist on the base branch (you cannot self-satisfy by adding it in the same diff under a different subdirectory).
2. The PR diff adds, modifies, or renames a file under `docs-internal/knowledge/plans/` (TEMPLATE.md and README.md do not count).

Plan files go in [`docs-internal/knowledge/plans/`](plans/). Copy [`TEMPLATE.md`](plans/TEMPLATE.md) to get started. Name files `YYYY-MM-DD-short-description.md`.

This rule is enforced by `.github/workflows/pr-plan-check.yml` as a required status check. The intent is to preserve design context across reviewers and AI assistants.

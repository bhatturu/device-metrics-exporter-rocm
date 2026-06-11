# Require per-PR plan file and gate in CI

- **Date:** 2026-05-28
- **Author:** praveen
- **Related PR(s):** TBD
- **Related issue(s) / JIRA:** N/A

## Context

Design context for non-trivial PRs is often scattered across PR comments,
Slack threads, and reviewer memory. AI assistants picking up a PR mid-review
lack the "why" behind decisions. This change introduces a lightweight
requirement: every PR to `main` must be associated with a durable plan file
so that context is preserved across reviewers and sessions.

The pattern is adopted from pensando/gpu-operator PR #1464, which introduced
the same gate for that repo.

## Approach

- Plan files live under `docs-internal/knowledge/plans/` (mirroring gpu-operator).
- A PR satisfies the rule if EITHER:
  1. The PR body references an existing file under `docs-internal/knowledge/`
     (the file must exist on the base branch — cannot self-satisfy by adding
     it elsewhere in the same diff), OR
  2. The PR diff adds, modifies, or renames a file under
     `docs-internal/knowledge/plans/` (TEMPLATE.md and README.md excluded).
- A GitHub Actions workflow (`.github/workflows/pr-plan-check.yml`) enforces
  the rule on every PR targeting `main` and must be added as a required status
  check in branch protection settings.
- `CLAUDE.md` documents the rule so contributors and AI assistants encounter
  it before opening a PR.

### Alternatives considered

- **jobd-based gate** — rejected; jobd is for builds, not PR policy checks.
  GitHub Actions integrates natively with branch protection required status checks.
- **`.claude/kb_source/plans/`** — rejected in favor of `docs-internal/knowledge/plans/`
  to stay consistent with gpu-operator and allow non-AI tooling to find plans
  without knowing about Claude conventions.
- **Gate on all branches** — rejected; `collab-*` branches are integration
  branches for ongoing epics already covered by a plan. Gating them adds
  friction without benefit.

## Scope

- **In scope:** workflow file, TEMPLATE.md, CLAUDE.md rule, self-satisfying plan file.
- **Out of scope:** migrating existing PRs retroactively, enforcing plan quality.

## Validation

- Open a test PR to `main` without a plan file — workflow must fail.
- Open a test PR that adds a file under `docs-internal/knowledge/plans/` — workflow must pass.
- Open a test PR whose body references an existing plan path — workflow must pass.

## Risks and rollback

- Known risks: contributors may write minimal throwaway plan files to satisfy
  the gate. Mitigated by code review; the gate enforces presence, not quality.
- Rollback plan: delete the workflow file or remove it from required status checks.

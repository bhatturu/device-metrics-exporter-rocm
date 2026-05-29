# Claude Code Setup Notes

Reference for how this repo's Claude Code config is structured and why. Based on the [Claude Code best practices guide](https://code.claude.com/en/best-practices). Done on 2026-05-26.

## What lives where

| Path | Purpose | Committed? |
|---|---|---|
| `CLAUDE.md` | Workflow rules + non-obvious gotchas (~60 lines, lean by design) | yes |
| `.claude/settings.json` | Team-shared permission allowlist + hook wiring | yes |
| `.claude/settings.local.json` | Personal/machine allowlist (Jira MCP, jobd webfetch) | **no** (gitignored) |
| `.claude/hooks/protect-generated.sh` | PreToolUse: blocks edits to generated/vendored/jobd-owned files | yes |
| `.claude/hooks/gofmt-on-edit.sh` | PostToolUse: runs `gofmt -w` on every `.go` edit | yes |
| `.claude/skills/<name>/SKILL.md` | Project skills (auto-discovered, uniform naming) | yes |
| `.claude/agents/` | Project subagents | yes |
| `docs-internal/knowledge/exporter/` | Deep architecture / troubleshooting docs (loaded on demand, NOT every session) | yes |
| `docs-internal/knowledge/prds/`, `.claude/prd_task_tracker/` | PRD workflow artifacts | yes |
| `CLAUDE.local.md` | Personal project notes (if you create one) | **no** (gitignored) |

## Design rules we chose

1. **CLAUDE.md stays small.** Anything Claude can derive from code, the README, or a SKILL.md frontmatter does NOT go here. Only non-guessable bash commands, gotchas, and "don't touch" rules.
2. **Architecture/components live in `docs-internal/knowledge/`, not CLAUDE.md.** Loaded on demand by Claude when actually needed; doesn't burn context every session.
3. **`settings.json` is team-shared, `settings.local.json` is personal.** The shared file only allowlists read-only or local-only operations (no `git push`, no `gh pr create`, no `docker run`).
4. **Hooks are deterministic guardrails, not advice.** Rules that MUST happen every time (gofmt, blocking generated-code edits) go in hooks, not CLAUDE.md.
5. **Skills uniformly use `<name>/SKILL.md`.** Auto-discovery works either casing but consistency makes the listing readable.
6. **Project-specific personal skills migrate into the repo.** If a skill is 100% about this codebase, it belongs in `.claude/skills/`, not `~/.claude/skills/`, so the team gets it.

## Permission allowlist (shared)

In `.claude/settings.json`. Read-only: git status/diff/log/show/branch/ls-files/blame; gh pr view/list/diff/checks, gh issue view/list, gh api, gh run view/list; docker ps/images/logs/inspect. Go toolchain (test/vet/build/mod tidy/mod download, gofmt, goimports, golangci-lint). Project: `Bash(make:*)` (broad — all targets are in-repo Makefile). WebFetch: github, instinct.docs.amd.com, code.claude.com, docs.claude.com.

**NOT allowlisted (still prompts):** `git commit/push/checkout/reset`, `gh pr create/merge/close`, `docker run/build/rm`, anything that hits lab hardware.

## Hook coverage

- **Block edits to:** root `entrypoint.sh` (jobd-owned, runtime is `docker/entrypoint.sh`), `*.pb.go`, `pkg/*/gen/**`, `vendor/**`, `libamdsmi/**`, `libgimsmi/**`. **Allow:** the `gpuagent/` submodule (actively edited).
- **Auto-format:** `gofmt -w` on every `.go` edit. Skips silently if `gofmt` not on PATH (e.g., inside a container without Go).
- **Session-end capture:** non-trivial sessions (≥5 tool uses) get their JSONL transcript copied to `.claude/kb_source/_pending/` (gitignored) for later review. Run `/curate-learnings` to distill durable, non-obvious findings into `.claude/kb_source/learnings.md`. Most sessions produce zero kept entries — that's correct.

## Maintenance checklist (when adding stuff later)

- New skill → `.claude/skills/<name>/SKILL.md`. Use a `description` that starts with "Use when..." for reliable auto-dispatch.
- New non-obvious gotcha → add to `CLAUDE.md` (~2 lines). If it's >5 lines, put it in `docs-internal/knowledge/` and link from CLAUDE.md.
- New tool/CLI you want auto-approved → add to `.claude/settings.json` `permissions.allow`. Keep entries scoped (`Bash(tool subcommand:*)` not `Bash(tool *)`).
- New deterministic rule ("always do X after Y") → write a hook in `.claude/hooks/`, register it in `settings.json`, test it with synthetic input.
- New MCP server → `.mcp.json` at repo root (and commit it if team-wide).

## What we deliberately did NOT do

- No `init` re-run — repo already had a working CLAUDE.md to refine, not regenerate.
- No `goimports` hook — it can fight the build-container's vendored toolchain on import order. Plain `gofmt` is safer.
- No `go vet` PostToolUse hook — too slow, too noisy, runs per-edit. Belongs in pre-commit or CI.
- No `SessionStart` git-status print — Claude can ask for it on demand, no need to burn context every session.
- No agent-team / parallel-session orchestration — single-Claude flow is the common case for this repo.

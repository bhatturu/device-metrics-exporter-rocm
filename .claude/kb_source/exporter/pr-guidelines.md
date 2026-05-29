# PR Guidelines

How to produce small, reviewable PRs against this repo. Mirrors what good reviewers expect and what the project's `CLAUDE.md` already mandates (no out-of-scope edits, no generated-file churn, no sensitive data).

---

## 1. Scope discipline

A PR must contain only changes that trace directly to its stated goal.

**In-scope examples:**
- The minimal source files that implement the feature/fix.
- Config files that the feature depends on (e.g. systemd unit, `gpuagent.conf`).
- KB / docs that document the new behavior **if** they were created or updated as part of this work and are themselves sanitized.

**Out-of-scope — exclude or split into a separate PR:**
- `assets/*.bin.gz`, `assets/*.gobin` — binary artifacts. Bumps to these come from external repo rebuilds (e.g. `gpu-agent`) and have their own review path. Don't fold them into a feature PR even if your local working tree shows them as modified.
- `pkg/**/gen/**`, `pkg/**/mock_gen/**` — generated protobuf / mocks. The project enforces this via the `.claude/hooks/protect-generated.sh` PreToolUse hook; do not stage them.
- `jobd`-owned files (`entrypoint.sh` in repo root, jobd configs) — see `CLAUDE.md`.
- Unrelated incidental edits in adjacent code, comments, or formatting.

If `git status` shows files you didn't intentionally touch, **don't stage them** with `git add -A` / `git add .`. List each file by name in `git add`.

---

## 2. Compact, easy-to-review

- Target **one logical change per commit**. A PR can carry multiple commits if they tell a coherent story (e.g. `feat:` + `docs:`).
- Keep total PR diff under ~300 lines of human-authored change where possible. Larger refactors should be split.
- Avoid drive-by improvements. If you spot dead code or a smell, open a separate issue or PR.

---

## 3. PR description format

Required sections, in this order:

```markdown
## Summary
1-3 bullet points stating WHAT changed and WHY. Reviewer-first phrasing.

## Files changed
Bulleted list of files with a one-line note per file about its role.

## Test plan
- Specific commands run
- Concrete pre/post observations (counters, log lines, return codes)
- For HW-touching changes: a validation table (check → result)
- Scrub all sensitive data (see Section 5)

## Risk / rollback
One paragraph: blast radius, who is affected, how to revert.
```

A minimal one-line PR with no test plan is unacceptable for anything that touches binaries, configs, or runtime behavior.

---

## 4. Commit message format

Match existing project style (see `git log --oneline`):

- Lowercase prefix tag: `feat:`, `fix:`, `docs:`, `chore:`, `build:`, `ci:`.
- Subject line ≤ 70 chars, imperative mood ("add", "fix", not "added"/"fixes").
- Body wrapped at ~72 chars, explains WHY.
- Trailers:
  - `Co-Authored-By: <name> <email>` if pair-authored or AI-assisted.
  - `Signed-off-by:` if the project DCO requires it.

Use a HEREDOC to preserve formatting:

```bash
git commit -m "$(cat <<'EOF'
feat(sriov): enable AGA_SMI_LAZY_INIT for SR-IOV exporter

Frees /dev/gim-smi0 between metric scrapes by enabling the
per-request open/use/close path in the GIM gpuagent.

Co-Authored-By: ...
EOF
)"
```

Never use `git commit --amend` or `git push --force` on a published branch unless explicitly requested. Don't pass `--no-verify` to skip hooks.

---

## 5. Sensitive data — MUST be scrubbed everywhere

Nothing in the PR diff, PR description, commit messages, KB docs, or `.claude/skills/` content may contain:

| Category | Examples | Replace with |
|---|---|---|
| Personal usernames | unix logins, GitHub handles | `<username>`, `<user>`, or generic |
| Lab/test-host IPs and hostnames | `10.x.y.z`, `lab-host-42` | `<test-host>` |
| Credentials | passwords, API tokens, SSH keys | omit entirely |
| Internal registry hostnames | `registry.internal.corp:5000` | `<DOCKER_REGISTRY>` placeholder + note that it comes from `dev.env` |
| GPU UUIDs / serial numbers | `14ff740f-0000-...` | `<gpu-uuid>` |
| Container IDs / image digests | `sha256:abc…` | omit or use generic `<image>` |
| Live PII | email addresses, employee IDs | omit |

Run a final scrub grep before opening the PR:

```bash
git diff origin/main...HEAD | grep -iE "<your-username>|<internal-registry-pattern>|<lab-ip-prefix>"
```

If it returns anything, fix it before pushing.

---

## 6. Pre-flight checklist

Before `gh pr create`, confirm:

- [ ] `git diff origin/main...HEAD` reads only as in-scope changes.
- [ ] No `assets/*` binary churn unless that IS the PR's purpose.
- [ ] No `pkg/**/gen/**` or `mock_gen/**` files staged.
- [ ] No untracked files accidentally staged (`git status` shows expected `M` / `A` lines only).
- [ ] Commit message follows Section 4.
- [ ] PR body has Summary / Files changed / Test plan / Risk sections.
- [ ] Sensitive-data grep returns empty (Section 5).
- [ ] Branch pushed with `-u origin/<branch>` so the PR can find it.

---

## 7. After the PR is open

- Add reviewers and labels via `gh pr edit` if not auto-assigned.
- If CI fails, fix forward with a NEW commit (no force-push).
- Respond to review comments inline. Don't squash until merge if the project preserves history.
- Don't merge your own PR unless the project explicitly allows it.

---

## 8. Reference: project enforcement points

- `CLAUDE.md` (repo root) — surgical-changes rule, do-not-touch list.
- `.claude/hooks/protect-generated.sh` — PreToolUse hook that blocks generated-file edits.
- `dev.env` — single source of truth for registry / base-image values; never hardcode these in commits.
- `docker/Dockerfile.sriov.*`, `debian-sriov/`, `Makefile.package` — the SR-IOV deployment surfaces (touch them together when changing SR-IOV behavior).

---
name: dev-pipeline
description: Transforms a technical specification document into reviewed, tested, and documented code with a pull request. Orchestrates analyst → architect → coder + test-designer → test-writer → validator → reviewer → doc-updater → pr-agent. Use when the user asks to "run the dev pipeline", "implement this spec", or "build from spec".
allowed-tools: Read Write Glob Agent
argument-hint: [path/to/spec.md] [branch-name]
effort: max
---

# Dev Pipeline

Orchestrate the full pipeline from technical specification to merged pull request.

```!
echo "Repo root: $(git rev-parse --show-toplevel 2>/dev/null || echo '(not a git repo)')"
echo "Branch:    $(git branch --show-current 2>/dev/null || echo '(unknown)')"
```

## Pipeline Overview

```
spec.md
  │
  ▼
[analyst]          → .pipeline/requirements.md
  │
  ▼
[architect]        → .pipeline/architecture.md
  │
  ├─────────────────────────┐
  ▼                         ▼
[coder]            [test-designer]
  │                         │ → .pipeline/test_plan.md
  └──────────┬──────────────┘
             ▼
        [test-writer]       → test files in repo
             │
             ▼
        [validator] ──FAIL──→ [coder] or [test-writer]
             │ PASS          (validator rounds reset each reviewer cycle)
             ▼
        [reviewer]  ──CHANGES──→ [coder] (max 2 reviewer rounds total)
             │ APPROVE
             ▼
        [doc-updater]       → updated docs in repo
             │
             ▼
        [pr-agent]          → PR URL + CI monitored
```

## Handoff convention

Before calling each agent, write a JSON handoff file to `.pipeline/`. The agent reads this file from the path passed as its argument. Each agent also writes its report to `.pipeline/<agent>_report.md` for use by downstream agents.

---

## Steps

### 0. Setup

1. **Parse arguments**:
   - `$ARGUMENTS[0]` — path to a spec document **or** an inline description (required)
   - `$ARGUMENTS[1]` — branch name (optional; default derived below)

2. **Determine spec source** — two modes:

   **File mode**: `$ARGUMENTS[0]` is a path to an existing file.
   - Verify the file exists and is readable. Stop if not.
   - `spec_file` = `$ARGUMENTS[0]`
   - Default branch name: `feat/<spec-filename-without-extension>`

   **Inline mode**: `$ARGUMENTS[0]` is not an existing file path (treat as a plain-text description).
   - Also applies when the pipeline is invoked from conversation context (no explicit argument) — use the user's request text as the description.
   - Write the description to `.pipeline/spec.md` (create `.pipeline/` first).
   - `spec_file` = `.pipeline/spec.md`
   - Default branch name: `feat/pipeline-<timestamp>` (e.g. `feat/pipeline-20260417`)

3. **Create a git worktree** for the branch:
   - Worktree path: `<repo_root>/../pipeline-worktree-<branch-name>`
   - Run: `git worktree add -b <branch_name> <worktree_path>`
   - All subsequent file writes (code, tests, docs, `.pipeline/`) happen inside `<worktree_path>`.
   - Use `<worktree_path>` as `repo_root` in every downstream handoff.

4. **Create** `.pipeline/` directory inside the worktree — all intermediate artifacts go here.

---

### 1. Analyst

Call the `analyst` agent with `spec_file` as its argument. Write its output to `.pipeline/requirements.md`.

**Stop if** the analyst emits an ERROR or lists more than 3 open questions — show the questions to the user and ask for clarification.

---

### 2. Architect

Write `.pipeline/handoff_architect.json`:
```json
{ "requirements_file": ".pipeline/requirements.md" }
```
Call the `architect` agent with `.pipeline/handoff_architect.json`. Write its output to `.pipeline/architecture.md`.

---

### 3. Coder + Test Designer (parallel)

Write `.pipeline/handoff_coder.json`:
```json
{
  "requirements_file": ".pipeline/requirements.md",
  "architecture_file": ".pipeline/architecture.md",
  "repo_root": "<repo_root>"
}
```

Write `.pipeline/handoff_test_designer.json` (same content).

Call `coder` with `.pipeline/handoff_coder.json` and `test-designer` with `.pipeline/handoff_test_designer.json` **simultaneously**. Wait for both.

- Save coder's returned file list as `code_files`.
- Save `test-designer` output to `.pipeline/test_plan.md`.

---

### 4. Test Writer

Write `.pipeline/handoff_test_writer.json`:
```json
{
  "test_plan_file": ".pipeline/test_plan.md",
  "code_files": <code_files list from step 3>,
  "repo_root": "<repo_root>"
}
```
Call `test-writer` with `.pipeline/handoff_test_writer.json`. Save its returned file list as `test_files` and its notes as `validator_notes`.

---

### 5. Validator loop (max 3 rounds per reviewer cycle)

Write `.pipeline/handoff_validator.json`:
```json
{
  "test_files": <test_files list>,
  "repo_root": "<repo_root>",
  "validator_notes": "<notes from test-writer>"
}
```
Call `validator` with `.pipeline/handoff_validator.json`. Write its output to `.pipeline/validator_report.md`.

- **If PASS** → proceed to step 6.
- **If FAIL** → read the routing summary:
  - Routed to `coder` → update `.pipeline/handoff_coder.json` to add a `failure_details` field with the validator report, call `coder` again
  - Routed to `test-writer` → update `.pipeline/handoff_test_writer.json` with `failure_details`, call `test-writer` again
  - Routed to `analyst` → **stop and ask the user** to clarify the requirement
  - After fixes, re-run `validator` (update the handoff file)
- **After 3 failed rounds** → stop and show `.pipeline/validator_report.md` to the user.

---

### 6. Reviewer

Write `.pipeline/handoff_reviewer.json`:
```json
{
  "requirements_file": ".pipeline/requirements.md",
  "architecture_file": ".pipeline/architecture.md",
  "code_files": <code_files>,
  "test_files": <test_files>,
  "validator_report": ".pipeline/validator_report.md"
}
```
Call `reviewer` with `.pipeline/handoff_reviewer.json`. Write its output to `.pipeline/reviewer_report.md`.

- **If APPROVE** → proceed to step 7.
- **If REQUEST CHANGES** → send BLOCKING issues back to `coder` (add to handoff as `review_issues`), then return to step 5. Validator round counter **resets** for the new reviewer cycle. Maximum **2 reviewer rounds** total — stop and show the report if still failing.

---

### 7. Doc Updater

Write `.pipeline/handoff_doc_updater.json`:
```json
{
  "repo_root": "<repo_root>",
  "reviewer_verdict": "APPROVE"
}
```
Call `doc-updater` with `.pipeline/handoff_doc_updater.json`. Save its list of updated/created doc files as `doc_files`.

---

### 8. PR Agent

Write `.pipeline/handoff_pr_agent.json`:
```json
{
  "repo_root": "<worktree_path>",
  "branch_name": "<branch_name>",
  "worktree_path": "<worktree_path>",
  "code_files": <code_files>,
  "test_files": <test_files>,
  "doc_files": <doc_files>,
  "artifact_dir": ".pipeline",
  "requirements_file": ".pipeline/requirements.md",
  "reviewer_report": ".pipeline/reviewer_report.md",
  "reviewer_verdict": "APPROVE"
}
```

Call `pr-agent` with `.pipeline/handoff_pr_agent.json`. The pr-agent handles: reading PR description content, cleanup of `.pipeline/`, commit, push, PR creation, and CI monitoring.

Report the PR URL to the user when done.

After the PR is created (or if the pipeline is stopped early due to an error), remove the worktree:
```
git worktree remove --force <worktree_path>
```

---

## Error Handling

| Situation | Action |
|---|---|
| File mode: spec file not found | Stop immediately, tell the user |
| Inline mode: no argument and no conversation context | Stop, ask the user what to build |
| Worktree creation fails | Stop, tell the user (branch may already exist) |
| Analyst finds >3 open questions | Stop, show the questions to the user |
| Validator fails 3 times in a cycle | Stop, show last report to the user |
| Reviewer requests changes twice | Stop, show review to the user |
| CI fails 3 times | Stop, show CI log to the user |
| Any agent emits ERROR | Stop, show the error to the user |

## Rules

- Never commit `.pipeline/` artifacts to the repo
- Never push to `main` or `master`
- Always run coder and test-designer in parallel (step 3)
- Show progress to the user after each major step
- Validator round counters reset per reviewer cycle, not globally

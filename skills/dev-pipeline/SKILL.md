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
[analyst]          → requirements.md
  │
  ▼
[architect]        → architecture.md
  │
  ├─────────────────────────┐
  ▼                         ▼
[coder]            [test-designer]
  │                         │
  └──────────┬──────────────┘
             ▼
        [test-writer]       → test files
             │
             ▼
        [validator] ──FAIL──→ [coder] or [test-writer]
             │ PASS
             ▼
        [reviewer]  ──CHANGES──→ [coder]
             │ APPROVE
             ▼
        [doc-updater]       → updated docs
             │
             ▼
        [pr-agent]          → PR URL + CI monitored
```

## Steps

### 0. Setup

1. **Parse arguments**:
   - `$0` — path to the spec document (required)
   - `$1` — branch name (optional; default: `feat/<spec-filename-without-extension>`)

2. **Verify** the spec file exists and is readable.

3. **Create a pipeline working directory**: `<repo_root>/.pipeline/` — all intermediate artifacts go here.

---

### 1. Analyst

Call the `analyst` agent with the spec path. Write the output to `.pipeline/requirements.md`.

**Stop if** the analyst emits an ERROR or lists more than 3 open questions — ask the user to clarify the spec first.

---

### 2. Architect

Call the `architect` agent with `.pipeline/requirements.md`. Write the output to `.pipeline/architecture.md`.

---

### 3. Coder + Test Designer (parallel)

Call both agents simultaneously:
- `coder` — receives `.pipeline/requirements.md` + `.pipeline/architecture.md` + `repo_root`. Writes source files directly to the repo. Returns a file list.
- `test-designer` — receives same inputs. Writes `.pipeline/test_plan.md`.

Wait for both to complete before proceeding.

---

### 4. Test Writer

Call `test-writer` with `.pipeline/test_plan.md` + coder's file list + `repo_root`. Writes test files directly to the repo. Returns a file list.

---

### 5. Validator loop (max 3 rounds)

Call `validator` with the test file list + `repo_root` + test-writer notes.

- **If PASS** → proceed to step 6.
- **If FAIL** → read the routing summary:
  - Files routed to `coder` → call `coder` again with the failure details + original inputs
  - Files routed to `test-writer` → call `test-writer` again with the failure details
  - Files routed to `analyst` → **stop and ask the user** to clarify the requirement
  - After fixes, re-run `validator`
- **After 3 failed rounds** → stop and show the last validator report to the user.

---

### 6. Reviewer

Call `reviewer` with requirements + architecture + code files + test files + validator report.

- **If APPROVE** → proceed to step 7.
- **If REQUEST CHANGES** → send BLOCKING issues back to `coder`, then return to step 5 (validator loop). Maximum 2 reviewer rounds total.

---

### 7. Doc Updater

Call `doc-updater` with `repo_root` + `reviewer_verdict: APPROVE`. Collect the list of updated/created doc files.

---

### 8. PR Agent

Call `pr-agent` with:
- `repo_root`
- `branch_name`
- `code_files` — from coder
- `test_files` — from test-writer
- `artifact_files` — everything under `.pipeline/` (do NOT include doc files — those must be committed)
- `requirements_file` — `.pipeline/requirements.md`
- `reviewer_report` — `.pipeline/reviewer_report.md`
- `reviewer_verdict` — `APPROVE`

The pr-agent handles cleanup, commit, push, PR creation, and CI monitoring.

---

### 9. Cleanup

Delete the `.pipeline/` directory after the PR is successfully created.

Report the PR URL to the user.

## Error Handling

| Situation | Action |
|---|---|
| Spec file not found | Stop immediately, tell the user |
| Analyst finds >3 open questions | Stop, show the questions to the user |
| Validator fails 3 times | Stop, show last report to the user |
| Reviewer requests changes twice | Stop, show review to the user |
| CI fails 3 times | Stop, show CI log to the user |
| Any agent emits ERROR | Stop, show the error to the user |

## Rules

- Never commit `.pipeline/` artifacts to the repo
- Never push to `main` or `master`
- Always run coder and test-designer in parallel (step 3)
- Show progress to the user after each major step

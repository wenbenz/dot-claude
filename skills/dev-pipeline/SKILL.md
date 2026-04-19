---
name: dev-pipeline
description: Transforms a specification or ticket into reviewed, tested, and documented code with a pull request. Accepts a local spec file, GitHub/Linear/Jira ticket URL, or inline description. Orchestrates planner → coder + test-writer → validator → reviewer → doc-patcher → pr-agent. Use when the user asks to "run the dev pipeline", "implement this spec", "implement this ticket", or "build from spec".
allowed-tools: Read Write Glob Bash Agent
argument-hint: [url-or-file-or-description] [branch-name]
effort: max
---

# Dev Pipeline

Orchestrate the full pipeline from technical specification to merged pull request.

```!
echo "Repo root: $(git rev-parse --show-toplevel 2>/dev/null || echo '(not a git repo)')"
echo "Branch:    $(git branch --show-current 2>/dev/null || echo '(unknown)')"
```

## Pipeline Overview

```mermaid
flowchart TD
    input([spec / ticket URL / inline text])
    plan[(".pipeline/plan.md")]
    vreport[(".pipeline/validator_report.md")]
    rreport[(".pipeline/reviewer_report.md")]

    input --> planner[planner]
    planner --> plan

    plan --> coder[coder]
    plan --> testwriter[test-writer]

    coder --> validator[validator]
    testwriter --> validator
    validator --> vreport

    vreport -->|"FAIL (max 5 rounds)"| coder
    vreport -->|"FAIL (max 5 rounds)"| testwriter
    vreport -->|PASS| reviewer[reviewer]
    reviewer --> rreport

    rreport -->|APPROVE| docpatcher[doc-patcher]
    rreport -->|"CHANGES (max 1 retry)"| coder

    docpatcher --> pragent[pr-agent]
    pragent --> pr([PR URL])
```

## Handoff convention

Before calling each agent, write a JSON handoff file to `<pipeline_dir>/`. Agent reads it from the absolute path passed as argument; writes its report to `<pipeline_dir>/<agent>_report.md` for downstream agents.

---

## Steps

### 0. Setup

1. **Parse arguments**:
   - `$ARGUMENTS[0]` — ticket URL, file path, or inline description (required)
   - `$ARGUMENTS[1]` — branch name (optional; default derived below)

2. **Determine input source** — three modes:

   **Ticket URL mode**: `$ARGUMENTS[0]` is a URL.
   - `https://github.com/*/issues/*` → `gh issue view <number> --repo <org/repo> --json title,body,comments`
   - `https://linear.app/*` → fetch via Linear MCP if available; else stop and ask user to paste content
   - `*atlassian.net/browse/*` → fetch via Jira MCP if available; else stop and ask user to paste content
   - Write fetched content to `<pipeline_dir>/spec.md`; `spec_content` = fetched JSON/text; `spec_file` = `<pipeline_dir>/spec.md`
   - Default branch: derived from ticket ID or title (e.g. `feat/eng-123-add-dark-mode`)

   **File mode**: `$ARGUMENTS[0]` is path to existing file.
   - Verify readable; stop if not.
   - `spec_file` = `$ARGUMENTS[0]`; `spec_content` = null
   - Default branch: `feat/<spec-filename-without-extension>`

   **Inline mode**: `$ARGUMENTS[0]` is neither URL nor existing file path (also applies when invoked from conversation context — use user's request text).
   - Write description to `<pipeline_dir>/spec.md`; `spec_file` = `<pipeline_dir>/spec.md`; `spec_content` = null
   - Default branch: `feat/pipeline-<timestamp>` (e.g. `feat/pipeline-20260417`)

3. **Create git worktree** following worktree-workflow rule (check existing worktrees first, reuse if found). Derive `<worktree_path>` from where worktree is placed.
   - Use `<worktree_path>` as `repo_root` in every downstream handoff.

4. **Set `pipeline_dir`** = `<worktree_path>/.pipeline`. Create directory now.

---

### 1. Planner

Write `<pipeline_dir>/handoff_planner.json`:
```json
{
  "spec_file": "<spec_file or null>",
  "content": "<spec_content or null>",
  "repo_root": "<repo_root>"
}
```
Call `planner` with `<pipeline_dir>/handoff_planner.json`. Write output to `<pipeline_dir>/plan.md`.

**Stop if** planner emits: `ERROR`, >3 open questions, or `BREAKDOWN REQUIRED` — show to user.

---

### 2. Coder + Test Writer (parallel)

Write `<pipeline_dir>/handoff_coder.json`:
```json
{
  "requirements_file": "<pipeline_dir>/plan.md",
  "architecture_file": "<pipeline_dir>/plan.md",
  "repo_root": "<repo_root>"
}
```

Write `<pipeline_dir>/handoff_test_writer.json`:
```json
{
  "requirements_file": "<pipeline_dir>/plan.md",
  "code_files": [],
  "repo_root": "<repo_root>"
}
```

Call `coder` and `test-writer` simultaneously. Save coder's file list as `code_files`; test-writer's list as `test_files` and notes as `validator_notes`.

---

### 3. Validator loop (max 5 rounds)

Write `<pipeline_dir>/handoff_validator.json`:
```json
{
  "test_files": <test_files list>,
  "repo_root": "<repo_root>",
  "validator_notes": "<notes from test-writer>"
}
```
Call `validator`; write output to `<pipeline_dir>/validator_report.md`.

- **PASS** → step 4.
- **FAIL** → read routing: `coder` → add `failure_details` to coder handoff, re-call coder; `test-writer` → add to test-writer handoff, re-call; `analyst` → stop and ask user. Re-run validator after fixes.
- **5 failed rounds** → stop, show report to user.

---

### 4. Reviewer

Write `<pipeline_dir>/handoff_reviewer.json`:
```json
{
  "requirements_file": "<pipeline_dir>/plan.md",
  "architecture_file": "<pipeline_dir>/plan.md",
  "code_files": <code_files>,
  "test_files": <test_files>,
  "validator_report": "<pipeline_dir>/validator_report.md"
}
```
Call `reviewer`; write output to `<pipeline_dir>/reviewer_report.md`.

- **APPROVE** → step 5.
- **REQUEST CHANGES** → send BLOCKING issues to `coder` (add as `review_issues`), reset validator counter, return to step 3. Max 1 retry — stop and show report if still failing.

---

### 5. Doc Patcher

Write `<pipeline_dir>/handoff_doc_patcher.json`:
```json
{
  "code_files": <code_files>,
  "repo_root": "<repo_root>"
}
```
Call `doc-patcher`; save updated file list as `doc_files`.

---

### 6. PR Agent

Write `<pipeline_dir>/handoff_pr_agent.json`:
```json
{
  "repo_root": "<worktree_path>",
  "branch_name": "<branch_name>",
  "worktree_path": "<worktree_path>",
  "code_files": <code_files>,
  "test_files": <test_files>,
  "doc_files": <doc_files>,
  "artifact_dir": "<pipeline_dir>",
  "requirements_file": "<pipeline_dir>/plan.md",
  "reviewer_report": "<pipeline_dir>/reviewer_report.md",
  "reviewer_verdict": "APPROVE"
}
```

Call `pr-agent` (handles cleanup, commit, push, PR creation, CI monitoring). Report PR URL to user.

After PR created (or on early stop), remove worktree:
```
git worktree remove --force <worktree_path>
```

---

## Error Handling

| Situation | Action |
|---|---|
| Spec file not found | Stop, tell user |
| No argument and no conversation context | Stop, ask user |
| Worktree creation fails | Stop, tell user |
| Planner >3 open questions | Stop, show questions |
| Planner >3 requirements | Stop, show breakdown |
| Validator fails 5 times | Stop, show last report |
| Reviewer requests changes twice | Stop, show review |
| CI fails 3 times | Stop, show CI log |
| Any agent emits ERROR | Stop, show error |

## Rules

- Never commit `<pipeline_dir>/` artifacts
- Never push to `main` or `master`
- Always run coder and test-writer in parallel (step 2)
- Show progress after each major step
- Validator round counter resets per reviewer cycle, not globally

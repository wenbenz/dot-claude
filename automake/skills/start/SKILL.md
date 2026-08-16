---
name: start
description: Turns a spec, ticket, or feature/bugfix request into reviewed, tested, documented code with a pull request, tracked as a durable SQLite-backed issue. Trigger proactively — without the user needing to invoke a skill by name — for any request that describes a feature, bugfix, or change with acceptance-criteria-shaped scope, references a ticket, or touches 2+ files / introduces a new module. Do NOT trigger for trivial single-file/single-line edits (typos, renames, one-line tweaks) or iteration on a diff already open in the conversation — handle those as normal inline edits. Also use when the user explicitly asks to "run the pipeline", "implement this spec/ticket", or "build from spec".
allowed-tools: Read Write Glob Bash Agent
argument-hint: [url-or-file-or-description] [branch-name]
effort: max
---

# Start

Orchestrate the full pipeline from technical specification to merged pull request. Every run is a durable `issues` row in the global SQLite state at `~/.claude/automake/state.db`; every agent invocation is a `work` row; `issues.status` only ever changes through `automake-db issue transition`, which enforces the configured topology as a DFA — a `(status, event)` pair not in the topology's transitions map is rejected outright, not just discouraged in prose.

```!
# check_redundant_build MARKER_PATH BINARY_PATH
# Only called from the branch where BINARY_PATH is about to be (re)built — i.e. it
# was NOT already present. Returns 0 (suspicious/redundant) only when a marker from
# an earlier successful build already exists AND names this exact binary path;
# returns 1 (not redundant — the normal first-build case) when there is no marker
# yet, or it names a different path.
check_redundant_build() {
  marker_path="$1"
  binary_path="$2"
  [ -f "$marker_path" ] || return 1
  recorded_path=$(sed -n '1p' "$marker_path" 2>/dev/null)
  [ "$recorded_path" = "$binary_path" ] || return 1
  return 0
}

# record_build_marker MARKER_PATH BINARY_PATH — call only after a successful build.
record_build_marker() {
  marker_path="$1"
  binary_path="$2"
  size_mtime=$(stat -c '%s %Y' "$binary_path" 2>/dev/null || stat -f '%z %m' "$binary_path" 2>/dev/null)
  printf '%s\n%s\n' "$binary_path" "$size_mtime" > "$marker_path"
}

# infer_plugin_root JSON_PATH PLUGIN_KEY
# Given installed_plugins.json's path and a "<plugin>@<marketplace>" key, prints
# that key's first installPath to stdout and returns 0, or prints nothing and
# returns 1 on any failure (file missing/unreadable, no matching key, or
# malformed/truncated JSON with no extractable installPath after the key).
# Pure: never reads $HOME/$CLAUDE_PLUGIN_ROOT itself, only its arguments — takes
# no dependency on ambient state so it stays independently testable.
# Relies on installed_plugins.json being one-field-per-line (as Claude Code
# itself writes it) — no jq/python3, awk/sed/grep only.
infer_plugin_root() {
  json_path="$1"
  plugin_key="$2"
  [ -r "$json_path" ] || return 1
  result=$(awk -v key="\"$plugin_key\"" '
    index($0, key) { found=1; next }
    found && match($0, /"installPath"[[:space:]]*:[[:space:]]*"[^"]*"/) {
      val = substr($0, RSTART, RLENGTH)
      sub(/^"installPath"[[:space:]]*:[[:space:]]*"/, "", val)
      sub(/"$/, "", val)
      print val
      exit
    }
  ' "$json_path" 2>/dev/null)
  [ -n "$result" ] || return 1
  printf '%s\n' "$result"
  return 0
}

INSTALLED_PLUGINS_JSON="$HOME/.claude/plugins/installed_plugins.json"
PLUGIN_KEY="automake@ben9"

if [ -z "${CLAUDE_PLUGIN_ROOT:-}" ]; then
  if INFERRED_PLUGIN_ROOT=$(infer_plugin_root "$INSTALLED_PLUGINS_JSON" "$PLUGIN_KEY"); then
    export CLAUDE_PLUGIN_ROOT="$INFERRED_PLUGIN_ROOT"
    echo "automake-db: inferred \$CLAUDE_PLUGIN_ROOT -> $CLAUDE_PLUGIN_ROOT (from $INSTALLED_PLUGINS_JSON)"
  fi
fi

BINARY="${CLAUDE_PLUGIN_ROOT:-}/cli/automake-db/automake-db"
MARKER="${CLAUDE_PLUGIN_ROOT:-}/cli/automake-db/.build_marker"

if [ -z "${CLAUDE_PLUGIN_ROOT:-}" ]; then
  echo "automake-db: BUILD FAILED — \$CLAUDE_PLUGIN_ROOT is unset/empty, and inference from $INSTALLED_PLUGINS_JSON also failed (file missing/unreadable, or no \"$PLUGIN_KEY\" entry with an installPath) — stop and show this to the user, and invoke healer (trigger=missing_env_var, issue_id=null, context={var_name: CLAUDE_PLUGIN_ROOT, installed_plugins_json: $INSTALLED_PLUGINS_JSON, plugin_key: $PLUGIN_KEY}) per ## Self-Healing"
elif [ -x "$BINARY" ]; then
  echo "automake-db: already built"
elif ! command -v go >/dev/null 2>&1; then
  echo "automake-db: BUILD FAILED — go not found on \$PATH — stop and show this to the user"
else
  if check_redundant_build "$MARKER" "$BINARY"; then
    REDUNDANT_BUILD=1
  else
    REDUNDANT_BUILD=0
  fi
  if go build -C "$CLAUDE_PLUGIN_ROOT/cli/automake-db" -o "$BINARY" .; then
    echo "automake-db: built"
    record_build_marker "$MARKER" "$BINARY"
    if [ "$REDUNDANT_BUILD" = "1" ]; then
      echo "automake-db: redundant build — a prior successful build for this exact binary path was already recorded, but a rebuild fired anyway — invoke healer (trigger=redundant_build, issue_id=<current issue or null>, context={prior_marker: $MARKER, current_build: $BINARY}) per ## Self-Healing; non-blocking, pipeline continues"
    fi
  else
    echo "automake-db: BUILD FAILED — stop and show this to the user"
  fi
fi
```

## Should this trigger?

Trigger automatically for:
- A feature or bugfix request with definable scope (the kind of thing that would naturally become a ticket)
- A request that references a ticket/issue URL or ID
- A change that will plausibly touch 2+ files, or introduces a new module/endpoint/component

Do **not** trigger — handle as a normal inline edit instead — for:
- A single-file, mechanically-scoped change (typo, rename, one-line tweak, config value change)
- Follow-up iteration on a diff already open in the current conversation
- Debugging/exploration where the user hasn't asked for a committed change yet

When genuinely unsure, err toward inline editing — spinning up a worktree and opening a draft PR for a one-line fix is worse than occasionally asking the user whether they wanted the full pipeline.

## Pipeline

```mermaid
stateDiagram-v2
    [*] --> queued
    blocked --> planning : resume
    coding --> validating : success
    doc_patching --> pr_open : success
    planning --> blocked : breakdown_required
    planning --> blocked : error
    planning --> coding : success
    pr_open --> pr_open : ci_fail
    pr_open --> failed : ci_max_rounds
    pr_open --> done : ci_pass
    queued --> planning : start
    reviewing --> doc_patching : approve
    reviewing --> coding : changes
    reviewing --> failed : max_retries
    validating --> blocked : fail_analyst
    validating --> coding : fail_coder
    validating --> coding : fail_test_writer
    validating --> failed : max_rounds
    validating --> reviewing : pass
    done --> [*]
    failed --> [*]
```

Generated by `automake-db topology mermaid` against `automake/cli/automake-db/topology.default.json` — regenerate and re-paste after editing the topology, don't hand-edit this diagram.

| State | Agent(s) |
|---|---|
| `planning` | `planner` |
| `coding` | `coder`, `test-writer` (parallel) |
| `validating` | `validator` |
| `reviewing` | `reviewer` |
| `doc_patching` | `doc-patcher` |
| `pr_open` | `pr-agent` |

`ticket-analyst` belongs to `plan-tickets`, not this pipeline.

## The `automake-db` CLI

Every state change and every agent invocation goes through the `automake-db` binary, invoked by its full path:

```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" <args>
```

The setup block at the top of this skill builds that binary before anything else runs, but only if it isn't already there — the pipeline makes ~15 calls per run, and re-invoking `go build` on every single one costs about 2.5s a call for no benefit. A plugin update lands in a new cache directory (its path is content-addressed), so there's no stale-binary case to worry about: a rebuild only happens the first time a given checkout runs. If the build fails, **stop and show the user** — every step below depends on it. Do not fall back to `go run`.

It is the only thing that writes `issues.status` — never update pipeline progress by editing files or by reasoning about it in prose. See `automake/README.md` for the full command reference. The commands used below:

- `automake-db issue list --ticket <ticket>` / `issue create --description TEXT --ticket TICKET` — resumability lookup and issue creation
- `automake-db issue get <id>` — current status
- `automake-db issue transition [--config PATH] <id> <event>` — the only way status changes
- `automake-db work start --id ID --agent NAME --repo PATH [--branch B] [--worktree W] [--context TEXT]` — call before every agent invocation; prints a run id
- `automake-db work finish [--output TEXT] [--pr URL] <run>` — call after every agent invocation
- `automake-db work list --id ID` — inspect prior runs (used for resumability)

If `<repo_root>/.claude/automake/topology.json` exists, pass `--config <that path>` to every `automake-db` call below; otherwise omit `--config` to use the shipped default.

If `issue transition` exits nonzero, **stop the pipeline and show the error to the user** — it means the event mapping below doesn't match reality, not a normal pipeline outcome.

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
   - `ticket` = the URL itself; `spec_content` = fetched JSON/text
   - Default branch: derived from ticket ID or title (e.g. `feat/eng-123-add-dark-mode`)

   **File mode**: `$ARGUMENTS[0]` is path to existing file.
   - Verify readable; stop if not.
   - `ticket` = absolute path to the file; `spec_file` = `$ARGUMENTS[0]`
   - Default branch: `feat/<spec-filename-without-extension>`

   **Inline mode**: `$ARGUMENTS[0]` is neither URL nor existing file path (also applies when invoked from conversation context — use the user's request text).
   - `ticket` = null (no external reference); `spec_content` = the description text
   - Default branch: `feat/pipeline-<timestamp>` (e.g. `feat/pipeline-20260417`)

3. **Resolve `repo_root`**: `git rev-parse --show-toplevel` from the current directory.

4. **Check for a resumable issue** (skip if `ticket` is null — inline requests with no external reference always start fresh):
   ```
   "$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue list --ticket <ticket>
   ```
   If a result has a non-terminal `status` (not `done`/`failed`):
   - `issue_id` = its `id`; current status = its `status`
   - `"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work list --id <issue_id>` → take the most recent row's `repo`, `branch`, `worktree`
   - If `worktree` still exists on disk and matches `repo_root`, reuse it; otherwise recreate the worktree at that `branch` (following the worktree-workflow rule)
   - Skip to the step matching the resumed status (table below); use the most recent `output` for that state's agent as the handoff content where a prior step's output is needed (e.g. resuming at `coding` needs the `planner` row's `output` as `plan.md`)
   - **Resuming from `blocked`**: show the user why it was blocked (from the relevant agent's last report) and ask for clarification before calling `"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue transition <issue_id> resume` and re-entering at Planner (step 1) with the clarification appended to the handoff.

   | Resumed status | Re-enter at |
   |---|---|
   | `queued` | Fire `issue transition <issue_id> start`, then step 1 |
   | `planning` | Step 1 |
   | `coding` | Step 2 |
   | `validating` | Step 3 |
   | `reviewing` | Step 4 |
   | `doc_patching` | Step 5 |
   | `pr_open` | Step 6 |

   `queued` is the common case when `plan-tickets` registered the ticket but no
   pipeline has run for it yet: the issue exists with zero `work` rows, so
   there is nothing to recover from `work list` — treat it as a fresh start
   (create the worktree in step 5 as if it were a new issue) that happens to
   reuse the existing `issue_id`.

   Otherwise (no resumable issue): create one —
   ```
   "$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue create --description "<one-line summary of the request>" --ticket <ticket or omit>
   ```
   `issue_id` = the printed id. Its status is now the topology's initial state (`queued`); immediately:
   ```
   "$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue transition <issue_id> start
   ```
   → status is now `planning`.

5. **Create git worktree** (new issues, and issues resumed from `queued`) following worktree-workflow rule (check existing worktrees first, reuse if found). Derive `<worktree_path>` from where worktree is placed.
   - Use `<worktree_path>` as `repo_root` in every downstream handoff.

6. **Set `pipeline_dir`** = `<worktree_path>/.pipeline`. Create directory now. (This still holds JSON handoff files and Markdown reports — payload between agents, not status; the DB tracks status and history, not the payload itself.)

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
```
run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent planner --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_planner.json)
```
Call `planner` with `<pipeline_dir>/handoff_planner.json`. Write output to `<pipeline_dir>/plan.md`.
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output <pipeline_dir>/plan.md $run
```

- Output is a normal requirements+architecture doc → `"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue transition <issue_id> success` → step 2.
- Output is `ERROR`, or lists >3 open questions → `... issue transition <issue_id> error` → status is now `blocked`; show the questions/error to the user and stop (see resumption path in step 0.4).
- Output is `BREAKDOWN REQUIRED` → `... issue transition <issue_id> breakdown_required` → status is now `blocked`; show the suggested breakdown to the user and stop.

---

### 2. Coder + Test Writer (parallel, test-writer conditional on the planner's Test Strategy)

Read the `## Test Strategy` `Decision:` line from `<pipeline_dir>/plan.md`. It is `WRITE_TESTS` or `SKIP_TESTS`. **If the field is missing, malformed, or ambiguous, treat it as `WRITE_TESTS`** — never silently skip tests because the field couldn't be parsed.

Write `<pipeline_dir>/handoff_coder.json`:
```json
{
  "requirements_file": "<pipeline_dir>/plan.md",
  "architecture_file": "<pipeline_dir>/plan.md",
  "repo_root": "<repo_root>"
}
```

Start coder's run:
```
coder_run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent coder --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_coder.json)
```

**If `WRITE_TESTS`** — write `<pipeline_dir>/handoff_test_writer.json`:
```json
{
  "requirements_file": "<pipeline_dir>/plan.md",
  "code_files": [],
  "repo_root": "<repo_root>"
}
```
and start test-writer's run *before* calling either agent, so both are in flight together:
```
testwriter_run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent test-writer --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_test_writer.json)
```
Call `coder` and `test-writer` simultaneously. Save coder's file list as `code_files`; test-writer's list as `test_files` and notes as `validator_notes`.
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output "<coder's Files Written/Modified summary>" $coder_run
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output "<test-writer's Test Files Written summary>" $testwriter_run
```

**If `SKIP_TESTS`** — do not start or call test-writer at all (no `work start` row for an agent that didn't run). Call only `coder`. Set `test_files = []` and `validator_notes` to the planner's Test Strategy `Reasoning:` line verbatim, so the validator and reviewer see *why* there are no tests instead of guessing.
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output "<coder's Files Written/Modified summary>" $coder_run
```

Once coder (and test-writer, if it ran) finish (join point — `coding` is one state covering both agents, whether or not test-writer participated this round):
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue transition <issue_id> success
```
→ step 3. (Either agent emitting `ERROR` is not a DFA event — stop and show the error per the Rules below, same as today.)

---

### 3. Validator loop

Write `<pipeline_dir>/handoff_validator.json`:
```json
{
  "test_files": <test_files list>,
  "repo_root": "<repo_root>",
  "validator_notes": "<notes from test-writer>"
}
```
```
run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent validator --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_validator.json)
```
Call `validator`; write output to `<pipeline_dir>/validator_report.md`.
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output <pipeline_dir>/validator_report.md $run
```

- **PASS** → `issue transition <issue_id> pass` → step 4.
- **FAIL, routed to `coder`** → `issue transition <issue_id> fail_coder`. If the CLI reports "limit reached ... auto-routing via max_rounds", status is now `failed` — stop, show the report (this is the 5-round cap firing). Otherwise status is `coding` — add `failure_details` to the coder handoff, re-run step 2's `work start`/coder/`work finish` sequence for the coder only (test-writer isn't re-run unless also routed), then **fire `issue transition <issue_id> success` again** to return the issue to `validating` before re-entering step 3.
- **FAIL, routed to `test-writer`** → `issue transition <issue_id> fail_test_writer`, same round-limit behavior, re-run test-writer only, then fire `issue transition <issue_id> success` again before re-entering step 3.

> Every retry round goes back through `coding → validating`. Re-entering step 3
> while the issue is still in `coding` makes the next `pass`/`fail_*` an illegal
> transition, which the CLI rejects and the rules below turn into a hard stop.
- **FAIL, routed to `analyst`** (ambiguous requirements) → `issue transition <issue_id> fail_analyst` → status is `blocked` — stop and ask the user (see step 0.4 resumption path).

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
```
run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent reviewer --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_reviewer.json)
```
Call `reviewer`; write output to `<pipeline_dir>/reviewer_report.md`.
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output <pipeline_dir>/reviewer_report.md $run
```

- **APPROVE** → `issue transition <issue_id> approve` → step 5.
- **REQUEST CHANGES** → `issue transition <issue_id> changes`. If the CLI reports the limit auto-routed to `max_retries`, status is now `failed` — stop, show the report (this is the "2 total review attempts" cap firing). Otherwise status is `coding` — send BLOCKING issues to `coder` (add as `review_issues`) and return to step 2.

---

### 5. Doc Patcher

Write `<pipeline_dir>/handoff_doc_patcher.json`:
```json
{
  "code_files": <code_files>,
  "repo_root": "<repo_root>"
}
```
```
run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent doc-patcher --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_doc_patcher.json)
```
Call `doc-patcher`; save updated file list as `doc_files`.
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish --output "<Docs Updated summary>" $run
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" issue transition <issue_id> success
```
→ step 6.

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
```
run=$("$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work start --id <issue_id> --agent pr-agent --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <pipeline_dir>/handoff_pr_agent.json)
```
Call `pr-agent` (handles cleanup, commit, push, PR creation, and — internally, per its own contract — up to 3 CI fix rounds before returning a single final verdict).
```
"$CLAUDE_PLUGIN_ROOT/cli/automake-db/automake-db" work finish $run --output "<pr-agent's report>" --pr <PR URL>
```
Put `$run` before the flags here, and **omit `--pr` entirely when pr-agent produced no PR URL** — a bare trailing `--pr` swallows `$run` as its value, and the command then fails for want of a positional argument, losing the finish record.

- **Status: SUCCESS** → `issue transition <issue_id> ci_pass` → status is `done`. Report the PR URL to the user.
- **Status: FAIL** → `issue transition <issue_id> ci_max_rounds` → status is `failed` (pr-agent already exhausted its internal retry budget before reporting FAIL, so this always maps straight to the terminal failure event, not the `ci_fail` self-loop — the loop stays available in the topology for a future pr-agent variant that reports per-round instead of self-managing retries — such a variant must record a `work start` row per CI round, or the `pr_open.ci_fail` limit has nothing to count and the loop is uncapped). Stop, show the last CI log.

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
| `issue transition`/`work start`/`work finish` exits nonzero | Stop, show the CLI's stderr — this is an event-mapping bug, not a normal outcome — and invoke `healer` (`trigger=cli_error`, `context={cmd, stderr, exit_code, state}`) before stopping, per `## Self-Healing` (non-blocking; doesn't delay the stop) |
| `CLAUDE_PLUGIN_ROOT` unset/empty and inference from `installed_plugins.json` also fails (setup block) | The setup block first tries to self-heal inline by inferring `CLAUDE_PLUGIN_ROOT` from `$HOME/.claude/plugins/installed_plugins.json`'s `automake@ben9` entry — this only reaches `healer` when that inference itself fails (file missing/unreadable/no matching entry). Stop, show the message — and invoke `healer` (`trigger=missing_env_var`, `issue_id=null`, `context={var_name, installed_plugins_json, plugin_key}`) per `## Self-Healing`; there's no `pipeline_dir`/`issue_id` yet, so show `healer`'s report to the user directly alongside the stop message |
| Planner emits `ERROR` or >3 open questions | `issue transition <id> error` → `blocked`; show to user |
| Planner emits `BREAKDOWN REQUIRED` | `issue transition <id> breakdown_required` → `blocked`; show breakdown |
| Validator fails 5 rounds | Auto-routed to `failed` by the CLI; show last report |
| Reviewer requests changes twice | Auto-routed to `failed` by the CLI; show review |
| pr-agent reports FAIL | `issue transition <id> ci_max_rounds` → `failed`; show CI log |
| Any agent emits `ERROR` outside the above | Stop, show error (not a DFA event — this is a bug in the agent call, not a pipeline outcome) |
| Agent output malformed/self-contradictory but not a literal `ERROR`, plausibly traceable to ambiguous/contradictory wording in its own `.md` | Handle the immediate situation as today (unchanged) — additionally invoke `healer` (`trigger=bad_directive`, `context={agent_name, agent_md_path, handoff, raw_output}`) per `## Self-Healing` |
| User expresses dissatisfaction (rejects the PR, asks for a redo, explicit quality complaint) — mid-pipeline or after `done`/`failed` | Does not change `issues.status` or resume/re-run anything by itself — invoke `healer` (`trigger=user_dissatisfaction`, `context={complaint_text, reviewer_report, pr_url, recent_work_rows}`) per `## Self-Healing`; separately, address the user's actual request (redo/fix) through the normal resumption path if they want the work itself redone |

## Self-Healing

`start` (and, for `user_dissatisfaction`, whichever agent is operating the conversation) invokes the `healer` agent (`automake/agents/healer.md`) out-of-band on five triggers. `healer` is **not** a topology state: it fires no `issue transition`, is never a declared agent for any state, and its invocation never replaces or delays the existing stop-and-show-user behavior above — it runs *in addition to* that behavior, not instead of it, and every existing happy-path step is otherwise unchanged.

**Triggers**:

| Trigger | Fires when | `issue_id` |
|---|---|---|
| `cli_error` | `issue transition`/`work start`/`work finish` exits nonzero | current issue, or `null` if the failing call was `issue create` itself |
| `missing_env_var` | `CLAUDE_PLUGIN_ROOT` (or another orchestrator-required var) is unset/empty in the setup block **and** the setup block's own inline inference of it from `$HOME/.claude/plugins/installed_plugins.json` (the `automake@ben9` entry's `installPath`) also failed — this is the post-inference-failure fallback, not the first line of defense; a successful inference self-heals inline and never reaches `healer` | `null` (no issue exists yet) |
| `redundant_build` | the setup block reports `built` (not `already built`) and `check_redundant_build` finds a prior successful build already recorded for the same binary path (see the setup block's `.build_marker` above) | current issue, or `null` before one exists |
| `bad_directive` | an agent's output is malformed/self-contradictory in a way plausibly traceable to ambiguous wording in its own `.md`, not already covered by an `ERROR`/DFA-event row above | current issue |
| `user_dissatisfaction` | the user rejects the PR, asks for a redo, or otherwise explicitly complains about quality — as distinct from a plain "also add X" follow-up, which is scope expansion, not dissatisfaction, and must never trigger `healer` | current issue, or the last known issue if the pipeline already finished |

**Invocation contract**:

Write `<pipeline_dir>/handoff_healer.json` (or, before a `pipeline_dir` exists, a path under the plugin cache):
```json
{
  "trigger": "<one of the above>",
  "issue_id": <int or null>,
  "repo_root": "<repo_root>",
  "plugin_root": "$CLAUDE_PLUGIN_ROOT",
  "context": { "...": "trigger-specific — see automake/agents/healer.md" },
  "evidence_files": ["<paths healer should Read first>"],
  "report_path": "<only when issue_id is null>"
}
```

- **`issue_id` non-null** — bracket the call like every other agent: `work start --id <issue_id> --agent healer --repo <repo_root> --branch <branch_name> --worktree <worktree_path> --context <handoff path>`, then call `healer`, then `work finish --output <report>`. **Never fire `issue transition` around it** — `healer` is deliberately absent from every state's `agents` list in the topology, so these `work` rows never count toward any round/retry limit.
- **`issue_id` null** — skip the `work start`/`work finish` pair entirely (there is no issue yet). `healer` writes its report to `report_path` instead, and the orchestrator shows that report to the user directly, alongside whatever stop message is already firing.
- Every invocation is non-blocking with respect to pipeline control flow: `healer` never changes `issues.status`, never re-routes the current step, and never itself pauses a step that wasn't already stopping for its own existing reason (`cli_error`/`missing_env_var` already stop the pipeline today — `healer` runs in addition to that stop, not as its cause; `redundant_build`/`bad_directive`/`user_dissatisfaction` never stopped the pipeline before and still don't).

See `automake/agents/healer.md` for the full diagnosis/fix/escalate contract and its `FIXED`/`PROPOSED`/`ESCALATE`/`NO_ACTION` output format.

## Rules

- Never commit `<pipeline_dir>/` artifacts
- Never push to `main` or `master`
- Always run coder and test-writer in parallel (step 2)
- Show progress after each major step
- Every `issue transition` call must be preceded by the `work start`/`work finish` pair for the agent whose output produced the event — the CLI's round-limit derivation depends on that history existing
- Rate-limit `healer`: at most one invocation per (`trigger`, `issue_id`-or-`null`) combination per orchestrator run. Before invoking, check for a marker file under `pipeline_dir` (e.g. `<pipeline_dir>/.healer_invoked_<trigger>_<issue_id-or-null>`; before a `pipeline_dir` exists, fall back to a path under the plugin cache) — if it already exists this run, skip the invocation. Write the marker immediately after invoking, whether `healer`'s report is `FIXED`, `PROPOSED`, `ESCALATE`, or `NO_ACTION`. This exists to stop a `healer` fix that doesn't stick from causing a retrigger loop within the same run — it is not a cross-run dedup (that's `healer`'s own job, per its Steps, via `work list`/prior `healer_report.md`)

#!/usr/bin/env bash
# preamble.sh — builds the automake-db binary and prints best-effort
# diagnostics for the `start` skill's setup block.
#
# Invoked from SKILL.md's `!` fenced block, after an inline guard there has
# already confirmed $CLAUDE_PLUGIN_ROOT is non-empty (that guard can't live
# here — it gates locating this very script).
#
# Convention (preserved from the prior inline block): the calling agent
# detects failure by pattern-matching the printed text below, not this
# script's exit status, since `!` blocks run non-interactively with no
# stdin. This script still exits nonzero on failure for its own internal
# clarity, but callers must not rely on that.
#
# Must not mutate the invoking shell's $PATH or other env vars beyond this
# script's own process.

set -u

# Cap how much captured `go build` stderr we ever print — this text is
# injected into every pipeline run's context, so unbounded compiler output
# (e.g. from a deeply broken checkout) shouldn't be reproduced verbatim.
MAX_BUILD_STDERR_CHARS=4000

# --- print_ambient_repo_diagnostics ----------------------------------------
# Pure output, no control-flow impact. These lines reflect the harness's
# ambient cwd at skill-load time — before Step 0 has parsed arguments or
# resolved the pipeline's real target repo_root (Step 0.3) — so they are
# explicitly diagnostic-only, not authoritative about what the pipeline will
# operate on.
print_ambient_repo_diagnostics() {
  local repo_root branch
  repo_root=$(git rev-parse --show-toplevel 2>/dev/null || echo '(not a git repo)')
  branch=$(git branch --show-current 2>/dev/null || echo '(unknown)')
  echo "Ambient cwd repo root (diagnostic only, not necessarily the pipeline's target — see Step 0.3): $repo_root"
  echo "Ambient cwd branch (diagnostic only, not necessarily the pipeline's target — see Step 0.3): $branch"
}

# --- resolve_go -------------------------------------------------------------
# Finds a usable `go` binary: $PATH first, then a fixed list of common
# install locations, first match wins. Does not mutate $PATH. Prints the
# resolved path on stdout and returns 0, or prints nothing and returns 1 if
# none of the candidates are executable. Records which candidates were
# tried in $RESOLVE_GO_TRIED for the caller to report on failure.
resolve_go() {
  local candidates=() path_go c
  path_go=$(command -v go 2>/dev/null || true)
  if [ -n "$path_go" ]; then
    candidates+=("$path_go")
  else
    candidates+=("go (not found on \$PATH)")
  fi
  candidates+=(
    "/usr/local/go/bin/go"
    "$HOME/go/bin/go"
    "/usr/lib/go/bin/go"
    "/opt/go/bin/go"
  )

  RESOLVE_GO_TRIED=""
  for c in "${candidates[@]}"; do
    RESOLVE_GO_TRIED="${RESOLVE_GO_TRIED}${RESOLVE_GO_TRIED:+, }${c}"
    if [ -x "$c" ]; then
      echo "$c"
      return 0
    fi
  done
  return 1
}

# --- build_automake_db -------------------------------------------------------
# Composes go resolution + the actual build, capturing stderr so a real
# compile failure is diagnosable instead of collapsing to a generic message.
build_automake_db() {
  local plugin_root="$1"
  local go_bin build_err status

  if ! go_bin=$(resolve_go); then
    echo "automake-db: BUILD FAILED — go not found (checked: ${RESOLVE_GO_TRIED}) — stop and show this to the user"
    return 1
  fi

  build_err=$("$go_bin" build -C "$plugin_root/cli/automake-db" -o "$plugin_root/cli/automake-db/automake-db" . 2>&1)
  status=$?

  if [ "$status" -eq 0 ]; then
    echo "automake-db: built"
    return 0
  fi

  if [ "${#build_err}" -gt "$MAX_BUILD_STDERR_CHARS" ]; then
    build_err="${build_err:0:$MAX_BUILD_STDERR_CHARS}
... (truncated, ${#build_err} chars total)"
  fi

  echo "automake-db: BUILD FAILED — stop and show this to the user"
  echo "$build_err"
  return 1
}

print_ambient_repo_diagnostics
build_automake_db "$CLAUDE_PLUGIN_ROOT"

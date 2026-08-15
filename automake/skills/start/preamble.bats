#!/usr/bin/env bats
# preamble.bats — black-box tests for preamble.sh and the $CLAUDE_PLUGIN_ROOT
# guard in SKILL.md's `!` fenced setup block.
#
# Run: bats preamble.bats  (or `bats .` from this directory)
#
# Everything here treats preamble.sh as a black box: it's invoked as a real
# subprocess with controlled env vars and a fake `go` stub on PATH, and
# assertions are made on stdout/stderr/exit status — this mirrors the
# script's own documented contract (callers pattern-match printed text, not
# exit status, since `!` blocks run non-interactively).

setup() {
  SCRIPT_DIR="$(cd "$(dirname "$BATS_TEST_FILENAME")" && pwd)"
  PREAMBLE="$SCRIPT_DIR/preamble.sh"
  SKILL_MD="$SCRIPT_DIR/SKILL.md"
  REAL_GIT_DIR="$(dirname "$(command -v git)")"
  TMP="$(mktemp -d)"
}

teardown() {
  rm -rf "$TMP"
}

# Writes a fake `go` stub to $1 that records its own invocation path to
# $FAKE_GO_TRACE_FILE (if set) and either succeeds — touching the file named
# by its `-o` flag — or fails per $FAKE_GO_MODE / $FAKE_GO_STDERR_MSG /
# $FAKE_GO_STDERR_SIZE. Lets us test go-resolution and build-failure handling
# without needing a real Go toolchain or the actual automake-db sources.
write_fake_go() {
  local dest="$1"
  mkdir -p "$(dirname "$dest")"
  cat > "$dest" <<'STUB'
#!/usr/bin/env bash
set -u
[ -n "${FAKE_GO_TRACE_FILE:-}" ] && echo "$0" >> "$FAKE_GO_TRACE_FILE"
outfile="" prev=""
for a in "$@"; do
  [ "$prev" = "-o" ] && outfile="$a"
  prev="$a"
done
if [ "${FAKE_GO_MODE:-success}" = "fail" ]; then
  if [ -n "${FAKE_GO_STDERR_SIZE:-}" ]; then
    head -c "$FAKE_GO_STDERR_SIZE" /dev/zero | tr '\0' 'x' 1>&2
  else
    echo "${FAKE_GO_STDERR_MSG:-FAKE COMPILE ERROR}" 1>&2
  fi
  exit 1
fi
[ -n "$outfile" ] && { : > "$outfile"; chmod +x "$outfile"; }
exit 0
STUB
  chmod +x "$dest"
}

# Pulls the ```! ... ``` fenced block out of SKILL.md so the $CLAUDE_PLUGIN_ROOT
# guard (which can't live inside preamble.sh — it gates locating that very
# script) gets tested against the actual shipped source, not a copy of it.
extract_skill_preamble_block() {
  awk '/^```!$/{f=1; next} /^```$/{if (f) exit} f' "$SKILL_MD"
}

@test "PATH go takes precedence over \$HOME/go/bin fallback" {
  local pathbin="$TMP/pathbin" fakehome="$TMP/home" trace="$TMP/trace"
  mkdir -p "$pathbin" "$fakehome/go/bin"
  write_fake_go "$pathbin/go"
  write_fake_go "$fakehome/go/bin/go"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$fakehome" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root FAKE_GO_TRACE_FILE="$trace" \
    bash "$PREAMBLE"

  [[ "$output" == *"automake-db: built"* ]]
  grep -q "$pathbin/go" "$trace"
  ! grep -q "$fakehome/go/bin/go" "$trace"
}

@test "falls back to \$HOME/go/bin/go when PATH has no go" {
  # resolve_go checks /usr/local/go/bin/go before $HOME/go/bin/go; if the
  # former is a real install on this host, it shadows our fake and we can't
  # isolate the $HOME fallback specifically.
  if [ -x /usr/local/go/bin/go ]; then
    skip "/usr/local/go/bin/go exists on this host — shadows the \$HOME/go/bin fallback candidate"
  fi
  local fakehome="$TMP/home" trace="$TMP/trace"
  mkdir -p "$fakehome/go/bin"
  write_fake_go "$fakehome/go/bin/go"

  run env -i PATH="$REAL_GIT_DIR" HOME="$fakehome" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root FAKE_GO_TRACE_FILE="$trace" \
    bash "$PREAMBLE"

  [[ "$output" == *"automake-db: built"* ]]
  grep -q "$fakehome/go/bin/go" "$trace"
}

@test "reports go-not-found with checked candidates when none exist" {
  for c in /usr/local/go/bin/go /usr/lib/go/bin/go /opt/go/bin/go; do
    if [ -x "$c" ]; then
      skip "$c exists on this host — can't isolate resolve_go from real filesystem state"
    fi
  done

  run env -i PATH="$REAL_GIT_DIR" HOME="$TMP/emptyhome" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root \
    bash "$PREAMBLE"

  [[ "$output" == *"BUILD FAILED — go not found"* ]]
}

@test "build failure surfaces the real go stderr, not a generic message" {
  local pathbin="$TMP/bin"
  mkdir -p "$pathbin"; write_fake_go "$pathbin/go"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$TMP/home" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root FAKE_GO_MODE=fail \
    FAKE_GO_STDERR_MSG="FAKE COMPILE ERROR: syntax error at line 42" \
    bash "$PREAMBLE"

  [[ "$output" == *"automake-db: BUILD FAILED"* ]]
  [[ "$output" == *"FAKE COMPILE ERROR: syntax error at line 42"* ]]
}

@test "oversized build stderr is capped and noted as truncated" {
  local pathbin="$TMP/bin"
  mkdir -p "$pathbin"; write_fake_go "$pathbin/go"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$TMP/home" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root FAKE_GO_MODE=fail \
    FAKE_GO_STDERR_SIZE=5000 \
    bash "$PREAMBLE"

  [[ "$output" == *"(truncated, 5000 chars total)"* ]]
  # Measure the longest contiguous run of 'x' rather than a total count —
  # other output (e.g. mktemp-generated tmp paths) can incidentally contain
  # stray 'x' characters unrelated to the truncated build error.
  run_len=$(printf '%s' "$output" | grep -oE 'x+' | awk '{ if (length($0) > m) m = length($0) } END { print m+0 }')
  [ "$run_len" -gt 0 ]
  [ "$run_len" -le 4000 ]
}

@test "successful build prints no failure banner" {
  local pathbin="$TMP/bin"
  mkdir -p "$pathbin"; write_fake_go "$pathbin/go"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$TMP/home" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root \
    bash "$PREAMBLE"

  [[ "$output" == *"automake-db: built"* ]]
  [[ "$output" != *"BUILD FAILED"* ]]
}

@test "ambient repo diagnostics are clearly marked non-authoritative in a real repo" {
  local pathbin="$TMP/bin"
  mkdir -p "$pathbin"; write_fake_go "$pathbin/go"
  ( cd "$TMP" && env -i PATH="$REAL_GIT_DIR" HOME="$TMP/home" git init -q . )
  local expected_root expected_branch
  expected_root="$(cd "$TMP" && git rev-parse --show-toplevel)"
  expected_branch="$(cd "$TMP" && git branch --show-current)"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$TMP/home" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root \
    bash -c 'cd "$1" && bash "$2"' _ "$TMP" "$PREAMBLE"

  [[ "$output" == *"Ambient cwd repo root (diagnostic only, not necessarily the pipeline's target — see Step 0.3): $expected_root"* ]]
  [[ "$output" == *"Ambient cwd branch (diagnostic only, not necessarily the pipeline's target — see Step 0.3): $expected_branch"* ]]
}

@test "ambient repo diagnostics say '(not a git repo)' outside any repo, still non-authoritative" {
  local pathbin="$TMP/bin"
  mkdir -p "$pathbin"; write_fake_go "$pathbin/go"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$TMP/home" \
    CLAUDE_PLUGIN_ROOT=/tmp/fake-plugin-root \
    bash -c 'cd "$1" && bash "$2"' _ "$TMP" "$PREAMBLE"

  [[ "$output" == *"Ambient cwd repo root (diagnostic only, not necessarily the pipeline's target — see Step 0.3): (not a git repo)"* ]]
}

@test "unset \$CLAUDE_PLUGIN_ROOT produces a named error, never a raw chdir failure" {
  local block; block="$(extract_skill_preamble_block)"

  run env -i PATH="$REAL_GIT_DIR" bash -c "$block"

  [[ "$output" == *"CLAUDE_PLUGIN_ROOT is unset/empty"* ]]
  [[ "$output" != *"chdir"* ]]
}

@test "set \$CLAUDE_PLUGIN_ROOT reaches and runs preamble.sh" {
  mkdir -p "$TMP/skills/start"
  cp "$PREAMBLE" "$TMP/skills/start/preamble.sh"
  local pathbin="$TMP/bin"
  mkdir -p "$pathbin"; write_fake_go "$pathbin/go"
  local block; block="$(extract_skill_preamble_block)"

  run env -i PATH="$REAL_GIT_DIR:$pathbin" HOME="$TMP/home" \
    CLAUDE_PLUGIN_ROOT="$TMP" bash -c "$block"

  [[ "$output" == *"automake-db: built"* ]]
}

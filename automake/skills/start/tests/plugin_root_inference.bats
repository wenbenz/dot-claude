#!/usr/bin/env bats
#
# Tests for infer_plugin_root(), the CLAUDE_PLUGIN_ROOT-inference helper
# planned for automake/skills/start/SKILL.md's setup block (REQ-001,
# .pipeline/plan.md's Architecture: "infer_plugin_root (new function, setup
# block)" and Testability Notes).
#
#   infer_plugin_root <json_path> <plugin_key>
#     - prints the resolved installPath to stdout and exits 0 on success
#     - prints nothing (empty stdout) and exits 1 on ANY failure: missing
#       file, unreadable file, no matching plugin_key entry, or
#       malformed/truncated JSON (REQ-001 edge cases: "never partially
#       succeed on garbage")
#     - when plugin_key's value is a JSON array with multiple install
#       records, resolves to the FIRST installPath found for that key
#       (.pipeline/plan.md's Open Questions, resolved: "use the first
#       installPath found in the array")
#
# OUT OF SCOPE FOR THIS SUITE (see plan's Testability Notes: "the
# orchestration layer ... is not separately unit-tested, matching the
# existing convention"):
#   - whether the setup block calls infer_plugin_root at all when
#     CLAUDE_PLUGIN_ROOT is already set (REQ-001's "byte-for-byte identical
#     to today" AC) -- infer_plugin_root itself never reads
#     $CLAUDE_PLUGIN_ROOT/$HOME, so there's nothing to unit-test there.
#   - the BUILD FAILED message / healer context payload construction
#     (REQ-002) -- that's orchestration-layer prose/wiring, not this
#     function's behavior.
#
# SOURCING STRATEGY:
# The coder's edit to SKILL.md (adding infer_plugin_root to its setup
# `` ```! `` block) may not exist yet -- this suite is written to run either
# way, following the same convention as tests/build_marker.bats. setup_file()
# below extracts infer_plugin_root() directly out of SKILL.md's setup block
# (brace-counted, so the rest of that imperative block is never
# sourced/executed) and uses it if found; otherwise it falls back to the
# local reference implementation in tests/lib/plugin_root_stub.sh, which
# mirrors the contract described above only. Re-run this suite once the
# coder's implementation lands to exercise the real function instead of the
# stub -- teardown_file() below prints a reminder when the stub was used.

setup_file() {
  SKILL_MD="$BATS_TEST_DIRNAME/../SKILL.md"
  SETUP_BLOCK="$BATS_FILE_TMPDIR/setup_block.sh"
  EXTRACTED="$BATS_FILE_TMPDIR/extracted_infer_plugin_root.sh"

  # Pull the first ```! ... ``` fenced (setup) block out of SKILL.md.
  awk '
    /^```!/ { inblock=1; next }
    inblock && /^```/ { inblock=0; next }
    inblock { print }
  ' "$SKILL_MD" > "$SETUP_BLOCK" 2>/dev/null || true

  # Extract just the infer_plugin_root() function definition (brace
  # counting) so sourcing it can never execute the surrounding setup-block
  # logic (env-var checks, `go build`, etc.) as a side effect.
  awk '
    /^[[:space:]]*(function[[:space:]]+)?infer_plugin_root[[:space:]]*\(\)/ { found=1 }
    found {
      print
      o = gsub(/\{/, "{")
      c = gsub(/\}/, "}")
      depth += o - c
      if (depth == 0 && (o + c) > 0) exit
    }
  ' "$SETUP_BLOCK" > "$EXTRACTED" 2>/dev/null || true

  if [ -s "$EXTRACTED" ]; then
    USING_REAL_IMPL=1
    # shellcheck disable=SC1090
    source "$EXTRACTED"
  else
    USING_REAL_IMPL=0
  fi
  export USING_REAL_IMPL

  # Always source the local stub too -- it only defines infer_plugin_root
  # itself when one wasn't already sourced above (see the stub file's
  # `declare -f` guard), matching build_marker.bats's convention of always
  # sourcing tests/lib/build_marker_stub.sh regardless of extraction result.
  # shellcheck disable=SC1090
  source "$BATS_TEST_DIRNAME/lib/plugin_root_stub.sh"

  FIXTURES="$BATS_TEST_DIRNAME/fixtures"
  export FIXTURES

  # bats-core runs setup_file() and each @test in separate subshells, so
  # functions defined only here (whether sourced from SKILL.md above or
  # from the stub) are invisible to the tests unless explicitly exported.
  export -f infer_plugin_root
}

teardown_file() {
  if [ "$USING_REAL_IMPL" != "1" ]; then
    echo "# NOTE: automake/skills/start/SKILL.md does not yet define infer_plugin_root() -- this suite ran against the local reference stub in tests/lib/plugin_root_stub.sh. Re-run once the coder's implementation lands to validate the real function." >&3
  fi
}

@test "infer_plugin_root prints the installPath and exits 0 for a valid single-entry fixture" {
  run infer_plugin_root "$FIXTURES/plugin_root_single.json" "automake@ben9"
  [ "$status" -eq 0 ]
  [ "$output" = "/home/user/.claude/plugins/cache/automake-ben9-abc123" ]
}

@test "infer_plugin_root resolves to the first installPath when the plugin key has multiple install records" {
  run infer_plugin_root "$FIXTURES/plugin_root_multi.json" "automake@ben9"
  [ "$status" -eq 0 ]
  [ "$output" = "/home/user/.claude/plugins/cache/automake-ben9-user-scope" ]
}

@test "infer_plugin_root fails with empty stdout and exit 1 when the JSON file does not exist" {
  run infer_plugin_root "$BATS_TEST_TMPDIR/does_not_exist.json" "automake@ben9"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "infer_plugin_root fails with empty stdout and exit 1 when the file exists but is unreadable" {
  if [ "$(id -u)" = "0" ]; then
    skip "running as root -- permission bits don't block reads in this sandbox, so this case can't be exercised"
  fi
  unreadable="$BATS_TEST_TMPDIR/unreadable.json"
  cp "$FIXTURES/plugin_root_single.json" "$unreadable"
  chmod 000 "$unreadable"

  run infer_plugin_root "$unreadable" "automake@ben9"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "infer_plugin_root fails with empty stdout and exit 1 when the JSON has no matching plugin key" {
  run infer_plugin_root "$FIXTURES/plugin_root_missing_key.json" "automake@ben9"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "infer_plugin_root fails with empty stdout and exit 1 on malformed/truncated JSON" {
  run infer_plugin_root "$FIXTURES/plugin_root_malformed.json" "automake@ben9"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "infer_plugin_root fails with empty stdout and exit 1 on an empty file" {
  run infer_plugin_root "$FIXTURES/plugin_root_empty.json" "automake@ben9"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

@test "infer_plugin_root fails with empty stdout and exit 1 for a plugin key that isn't present anywhere in a well-formed file" {
  # Distinct from the "missing key" fixture above: this queries a key that
  # doesn't exist in ANY plugin's JSON, not just absent from a
  # single-plugin fixture -- guards against a naive implementation that
  # only checks "is the file valid JSON" rather than "did we actually find
  # this key".
  run infer_plugin_root "$FIXTURES/plugin_root_single.json" "totally-unknown-plugin@nowhere"
  [ "$status" -eq 1 ]
  [ -z "$output" ]
}

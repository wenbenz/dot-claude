#!/usr/bin/env bats
#
# Tests for check_redundant_build(), the build-marker / redundant-build
# detection helper planned for automake/skills/start/SKILL.md's setup
# block (REQ-001c, .pipeline/plan.md's Architecture: "Build marker" and
# Testability Notes).
#
#   check_redundant_build <marker_path> <binary_path>
#     - exit 1 (not redundant) when <marker_path> does not exist yet
#       (first build for a freshly-installed plugin cache dir; REQ-001
#       edge case: "must not false-positive on the very first build...
#       no prior marker")
#     - exit 1 (not redundant) when <marker_path> exists but records a
#       *different* binary path (stale marker from an older,
#       content-addressed cache dir -- not comparable; REQ-001 edge case:
#       "only fires when a marker from an earlier successful build for
#       the SAME path already exists")
#     - exit 0 (redundant) when <marker_path> exists and records the
#       exact same <binary_path> (REQ-001 AC: "a prior successful build
#       for the same ... path was already recorded")
#
# NOTE ON A PLAN INCONSISTENCY (flagged for reviewer/validator):
# .pipeline/plan.md's "Testability Notes" section describes the three
# branches as "marker-present-and-matching (no healer trigger)" /
# "marker-present-but-stale/mismatched (healer trigger)", which reads as
# the inverse of REQ-001's own Acceptance Criteria and Edge Cases text
# quoted above (matching path -> trigger; no-marker-or-mismatched path ->
# no trigger). This suite follows the normative Acceptance
# Criteria/Edge Cases wording, since it is internally consistent, and
# names every case explicitly so a human can quickly compare against
# whatever the coder actually implements.
#
# SOURCING STRATEGY:
# The coder's edit to SKILL.md (adding check_redundant_build to its
# setup `` ```! `` block) may not exist yet -- this suite is written to
# run either way. setup_file() below extracts check_redundant_build()
# directly out of SKILL.md's setup block (brace-counted, so the rest of
# that imperative block -- e.g. the real `go build` call -- is never
# sourced/executed) and uses it if found; otherwise it falls back to the
# local reference implementation in tests/lib/build_marker_stub.sh, which
# mirrors the interface described above only. Re-run this suite once the
# coder's implementation lands to exercise the real function instead of
# the stub -- teardown_file() below prints a reminder when the stub was
# used.

setup_file() {
  SKILL_MD="$BATS_TEST_DIRNAME/../SKILL.md"
  SETUP_BLOCK="$BATS_FILE_TMPDIR/setup_block.sh"
  EXTRACTED="$BATS_FILE_TMPDIR/extracted_check_redundant_build.sh"

  # Pull the first ```! ... ``` fenced (setup) block out of SKILL.md.
  awk '
    /^```!/ { inblock=1; next }
    inblock && /^```/ { inblock=0; next }
    inblock { print }
  ' "$SKILL_MD" > "$SETUP_BLOCK" 2>/dev/null || true

  # Extract just the check_redundant_build() function definition (brace
  # counting) so sourcing it can never execute the surrounding build
  # logic as a side effect.
  awk '
    /^[[:space:]]*(function[[:space:]]+)?check_redundant_build[[:space:]]*\(\)/ { found=1 }
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

  # write_build_marker() is always sourced from the local stub: the plan
  # calls marker-writing "a separate concern" from check_redundant_build
  # but does not name a function for it yet, so there is nothing to
  # extract from SKILL.md. It only defines check_redundant_build itself
  # when one wasn't already sourced above (see the stub file's guard).
  # shellcheck disable=SC1090
  source "$BATS_TEST_DIRNAME/lib/build_marker_stub.sh"

  # bats-core runs setup_file() and each @test in separate subshells, so
  # functions defined only here (whether sourced from SKILL.md above or
  # from the stub) are invisible to the tests unless explicitly exported.
  export -f check_redundant_build
  export -f write_build_marker
}

teardown_file() {
  if [ "$USING_REAL_IMPL" != "1" ]; then
    echo "# NOTE: automake/skills/start/SKILL.md does not yet define check_redundant_build() -- this suite ran against the local reference stub in tests/lib/build_marker_stub.sh. Re-run once the coder's implementation lands to validate the real function." >&3
  fi
}

setup() {
  MARKER="$BATS_TEST_TMPDIR/.build_marker"
  BINARY="$BATS_TEST_TMPDIR/automake-db"
  printf '#!/bin/sh\necho stub-binary\n' > "$BINARY"
  chmod +x "$BINARY"
}

@test "first build with no existing marker file is not flagged as redundant" {
  [ ! -f "$MARKER" ]
  run check_redundant_build "$MARKER" "$BINARY"
  [ "$status" -eq 1 ]
}

@test "check_redundant_build alone never creates the marker file (writing is a separate concern)" {
  [ ! -f "$MARKER" ]
  check_redundant_build "$MARKER" "$BINARY" || true
  [ ! -f "$MARKER" ]
}

@test "write_build_marker records the built binary's path after a first build" {
  write_build_marker "$MARKER" "$BINARY"
  [ -f "$MARKER" ]
  [ "$(sed -n '1p' "$MARKER")" = "$BINARY" ]
}

@test "build with a marker already recorded for the exact same binary path is flagged as redundant" {
  write_build_marker "$MARKER" "$BINARY"
  run check_redundant_build "$MARKER" "$BINARY"
  [ "$status" -eq 0 ]
}

@test "build with a marker recorded for a different (stale) binary path is not flagged as redundant" {
  other_binary="$BATS_TEST_TMPDIR/other-cache-dir/automake-db"
  mkdir -p "$(dirname "$other_binary")"
  printf '#!/bin/sh\necho other\n' > "$other_binary"
  chmod +x "$other_binary"
  write_build_marker "$MARKER" "$other_binary"

  run check_redundant_build "$MARKER" "$BINARY"
  [ "$status" -eq 1 ]
}

@test "check_redundant_build treats an empty/corrupt marker file as not redundant rather than erroring" {
  : > "$MARKER"
  run check_redundant_build "$MARKER" "$BINARY"
  [ "$status" -eq 1 ]
}

@test "a freshly-written marker for one path does not flag a differently-pathed binary built afterward, then a repeat build of that same path is flagged" {
  # Simulates: plugin update lands in a new content-addressed cache dir
  # (new path) -> first build there is NOT redundant; a second "built"
  # outcome reported later for that SAME new path (e.g. a race, or the
  # cached binary check failing unexpectedly) IS flagged.
  first_path_binary="$BATS_TEST_TMPDIR/cache-v1/automake-db"
  mkdir -p "$(dirname "$first_path_binary")"
  printf '#!/bin/sh\necho v1\n' > "$first_path_binary"
  chmod +x "$first_path_binary"
  write_build_marker "$MARKER" "$first_path_binary"

  run check_redundant_build "$MARKER" "$BINARY"
  [ "$status" -eq 1 ]

  write_build_marker "$MARKER" "$BINARY"
  run check_redundant_build "$MARKER" "$BINARY"
  [ "$status" -eq 0 ]
}

#!/usr/bin/env bash
# Reference implementation of the build-marker helpers described in
# .pipeline/plan.md (REQ-001c / Architecture: "Build marker" / Testability
# Notes), used only until automake/skills/start/SKILL.md's setup block
# defines the real check_redundant_build() function itself.
#
# tests/build_marker.bats sources the real function straight out of
# SKILL.md when it can find one (brace-extracted from the setup block) and
# only falls back to the check_redundant_build defined below when it can't
# -- see setup_file() in that .bats file. write_build_marker() below is
# always used as-is: the plan calls marker-writing "a separate concern"
# from check_redundant_build itself but does not name a function for it,
# so there is nothing to extract from SKILL.md yet. Once the coder lands
# a real marker-writing function, point write_build_marker's test calls at
# it (or delete this stub) instead of leaving this duplicate around.
#
# Marker file format (matches the coder's real record_build_marker() in
# SKILL.md's setup block -- deliberately trivial, no schema/CLI change per
# the plan's Open Questions resolution):
#   line 1: <absolute path to the built automake-db binary> (raw, no key=)
#   line 2: <binary's size in bytes> <binary's mtime, seconds since epoch>

if ! declare -f check_redundant_build >/dev/null 2>&1; then
  # check_redundant_build <marker_path> <binary_path>
  #   returns 0 (redundant -- flag it) when <marker_path> exists and
  #     records the exact same <binary_path> (REQ-001 AC: "a prior
  #     successful build for the same ... path was already recorded")
  #   returns 1 (not redundant) when <marker_path> is absent (first
  #     build for a freshly-installed plugin cache dir) or when it
  #     exists but records a *different* path (stale marker from an
  #     older content-addressed cache dir -- per REQ-001's edge case,
  #     "only fires when a marker from an earlier successful build for
  #     the same path already exists")
  check_redundant_build() {
    local marker_path="$1" binary_path="$2" recorded_path
    [ -f "$marker_path" ] || return 1
    recorded_path=$(awk -F'=' '$1=="path"{print $2; exit}' "$marker_path" 2>/dev/null)
    [ -n "$recorded_path" ] && [ "$recorded_path" = "$binary_path" ]
  }
fi

# write_build_marker <marker_path> <binary_path>
#   Records <binary_path>'s current path/size/mtime into <marker_path>,
#   overwriting any prior contents. Not itself part of the
#   check_redundant_build interface named in the plan -- kept separate so
#   tests can exercise "did a marker get written" independently of "was
#   this build flagged as redundant". Mirrors record_build_marker() in
#   SKILL.md's setup block exactly (raw path on line 1, "size mtime" on
#   line 2) so it matches what check_redundant_build's `sed -n '1p'` read
#   actually expects.
write_build_marker() {
  local marker_path="$1" binary_path="$2" size_mtime
  size_mtime=$(stat -c '%s %Y' "$binary_path" 2>/dev/null || stat -f '%z %m' "$binary_path" 2>/dev/null)
  printf '%s\n%s\n' "$binary_path" "$size_mtime" > "$marker_path"
}

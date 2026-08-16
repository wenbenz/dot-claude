#!/usr/bin/env bash
# Reference implementation of infer_plugin_root(), the pure JSON-scan helper
# planned for automake/skills/start/SKILL.md's setup block (REQ-001,
# .pipeline/plan.md's Architecture: "infer_plugin_root (new function, setup
# block)" and Testability Notes), used only until SKILL.md's setup block
# defines the real function itself.
#
# tests/plugin_root_inference.bats sources the real function straight out of
# SKILL.md when it can find one (brace-extracted from the `` ```! `` setup
# block) and only falls back to the implementation below when it can't --
# see setup_file() in that .bats file. Re-run that suite once the coder's
# implementation lands to exercise the real function instead of this stub.
#
# Contract (per plan pseudocode):
#   infer_plugin_root <json_path> <plugin_key>
#     - prints the resolved installPath to stdout and returns 0 on success
#     - prints nothing and returns 1 on ANY failure: missing file, unreadable
#       file, no matching plugin_key, or malformed/truncated JSON (never
#       partially succeeds on garbage)
#     - when plugin_key's value is a JSON array with multiple install
#       records, resolves to the FIRST installPath found for that key (see
#       plan's Open Questions resolution)
#
# Extraction approach (no jq/python3, per the plan's Constraints): scan
# line-by-line for a line containing "<plugin_key>", then from that point on
# take the first subsequent line matching "installPath"[:space:]*:[:space:]*
# "..." and extract the quoted value. Relies on installed_plugins.json being
# human-formatted (one field per line), same assumption the plan calls out.
if ! declare -f infer_plugin_root >/dev/null 2>&1; then
  infer_plugin_root() {
    local json_path="$1" plugin_key="$2" result

    [ -n "$json_path" ] || return 1
    [ -r "$json_path" ] || return 1

    result=$(awk -v key="\"${plugin_key}\"" '
      index($0, key) { found = 1 }
      found && match($0, /"installPath"[[:space:]]*:[[:space:]]*"[^"]*"/) {
        line = substr($0, RSTART, RLENGTH)
        sub(/^"installPath"[[:space:]]*:[[:space:]]*"/, "", line)
        sub(/"$/, "", line)
        print line
        exit
      }
    ' "$json_path" 2>/dev/null)

    [ -n "$result" ] || return 1
    printf '%s\n' "$result"
    return 0
  }
fi

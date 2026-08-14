// Command automake-db is the deterministic state layer for the automake
// pipeline: it owns the SQLite database at $AUTOMAKE_DB (default
// ~/.claude/automake/state.db) and is the only thing that changes
// issues.status, enforcing the configured topology as a DFA — a
// (state, event) pair not in the topology's transitions map is rejected.
package main

import (
	"fmt"
	"os"
)

const usage = `usage: automake-db <command> [args]

commands:
  init                                  create the DB and schema if missing
  topology validate [--config PATH]     check the topology is a valid DFA
  topology show     [--config PATH]     print the resolved topology as JSON
  topology mermaid  [--config PATH]     render the topology as stateDiagram-v2

  issue create --description TEXT [--ticket TICKET] [--config PATH]
  issue get <id>
  issue list [--status STATE] [--ticket TICKET]
  issue transition [--config PATH] <id> <event>

  dep add <id> <dependency-id>
  dep list <id>

  work start --id ID --agent NAME --repo PATH [--branch B] [--worktree W] [--context TEXT]
  work finish [--output TEXT] [--pr URL] <run>
  work list --id ID

Note: flags must come before positional arguments (e.g. "issue transition
--config x.json 1 start", not "... 1 start --config x.json") — this is a
constraint of Go's standard flag parser, not a stylistic choice.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "init":
		err = cmdInit(os.Args[2:])
	case "topology":
		err = dispatchTopology(os.Args[2:])
	case "issue":
		err = dispatchIssue(os.Args[2:])
	case "dep":
		err = dispatchDep(os.Args[2:])
	case "work":
		err = dispatchWork(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprint(os.Stderr, usage)
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintln(os.Stderr, "automake-db:", err)
		os.Exit(1)
	}
}

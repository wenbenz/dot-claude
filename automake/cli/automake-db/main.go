// Command automake-db is the deterministic state layer for the automake
// pipeline: it owns the SQLite database at $AUTOMAKE_DB (default
// ~/.claude/automake/state.db) and is the only thing that changes
// issues.status, enforcing the configured topology as a DFA — a
// (state, event) pair not in the topology's transitions map is rejected.
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "automake-db",
		Short: "Deterministic state layer for the automake pipeline",
		// Domain errors (e.g. "issue not found") are common and expected;
		// dumping the full command tree's usage on every one of them (the
		// cobra default) would bury the actual message. Genuinely bad
		// invocations still get cobra's own usage output on the specific
		// subcommand, since Args validators/unknown-flag errors happen
		// before RunE and aren't affected by this.
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.AddCommand(
		newInitCmd(),
		newTopologyCmd(),
		newIssueCmd(),
		newDepCmd(),
		newWorkCmd(),
	)
	return root
}

func main() {
	initViper()
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "automake-db:", err)
		os.Exit(1)
	}
}

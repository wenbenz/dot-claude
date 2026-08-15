package main

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

type Issue struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Ticket      string `json:"ticket,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type WorkRow struct {
	ID       int64  `json:"id"`
	IssueID  int64  `json:"issue_id"`
	Agent    string `json:"agent"`
	Context  string `json:"context,omitempty"`
	Started  string `json:"started"`
	Finished string `json:"finished,omitempty"`
	Output   string `json:"output,omitempty"`
	Repo     string `json:"repo"`
	Branch   string `json:"branch,omitempty"`
	Worktree string `json:"worktree,omitempty"`
	PR       string `json:"pr,omitempty"`
}

func printJSON(v any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

// ---- init ----

func newInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "create the DB and schema if missing",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()
			path, _ := dbPath()
			fmt.Println(path)
			return nil
		},
	}
}

// ---- topology ----

func newTopologyCmd() *cobra.Command {
	topology := &cobra.Command{
		Use:   "topology",
		Short: "inspect or validate the DFA topology config",
	}
	// Shared by all three subcommands, so it's a persistent flag on the
	// parent rather than repeated on each -- only one subcommand runs per
	// process, so binding all three closures to the same variable is safe.
	var configPath string
	topology.PersistentFlags().StringVar(&configPath, "config", "", "path to topology config")

	topology.AddCommand(&cobra.Command{
		Use:   "validate",
		Short: "check the topology is a valid DFA",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, topo, err := loadTopologyForCLI(configPath)
			if err != nil {
				return err
			}
			errs := topo.Validate()
			if len(errs) > 0 {
				for _, e := range errs {
					fmt.Fprintln(os.Stderr, "-", e)
				}
				return fmt.Errorf("topology %s is not a valid DFA (%d issue(s))", path, len(errs))
			}
			fmt.Printf("OK: %s is a valid DFA\n", path)
			return nil
		},
	})
	topology.AddCommand(&cobra.Command{
		Use:   "show",
		Short: "print the resolved topology as JSON",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, topo, err := loadTopologyForCLI(configPath)
			if err != nil {
				return err
			}
			return printJSON(topo)
		},
	})
	topology.AddCommand(&cobra.Command{
		Use:   "mermaid",
		Short: "render the topology as stateDiagram-v2",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, topo, err := loadTopologyForCLI(configPath)
			if err != nil {
				return err
			}
			fmt.Print(topo.Mermaid())
			return nil
		},
	})
	return topology
}

func loadTopologyForCLI(configPath string) (string, *Topology, error) {
	return ResolveTopology(configPath)
}

// ---- issue ----

func newIssueCmd() *cobra.Command {
	issue := &cobra.Command{
		Use:   "issue",
		Short: "manage issue lifecycle rows",
	}
	issue.AddCommand(
		newIssueCreateCmd(),
		newIssueGetCmd(),
		newIssueListCmd(),
		newIssueTransitionCmd(),
	)
	return issue
}

func newIssueCreateCmd() *cobra.Command {
	var description, ticket, configPath string
	cmd := &cobra.Command{
		Use:   "create",
		Short: "create a new issue in the topology's initial state",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if description == "" {
				return fmt.Errorf("--description is required")
			}
			source, topo, err := ResolveTopology(configPath)
			if err != nil {
				return err
			}
			if topo.Initial == "" {
				return fmt.Errorf("topology %s has no initial state", source)
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			res, err := db.Exec(`INSERT INTO issues (status, description, ticket) VALUES (?, ?, ?)`,
				topo.Initial, description, nullIfEmpty(ticket))
			if err != nil {
				return err
			}
			id, err := res.LastInsertId()
			if err != nil {
				return err
			}
			fmt.Println(id)
			return nil
		},
	}
	cmd.Flags().StringVar(&description, "description", "", "issue description (required)")
	cmd.Flags().StringVar(&ticket, "ticket", "", "external ticket URL/ID, or local file path")
	cmd.Flags().StringVar(&configPath, "config", "", "path to topology config")
	return cmd
}

func scanIssue(row *sql.Row) (*Issue, error) {
	var iss Issue
	var ticket sql.NullString
	if err := row.Scan(&iss.ID, &iss.Status, &iss.Description, &ticket, &iss.CreatedAt); err != nil {
		return nil, err
	}
	iss.Ticket = ticket.String
	return &iss, nil
}

func newIssueGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "print an issue as JSON",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue id %q: %w", args[0], err)
			}
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			row := db.QueryRow(`SELECT id, status, description, ticket, created_at FROM issues WHERE id = ?`, id)
			iss, err := scanIssue(row)
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					return fmt.Errorf("issue %d not found", id)
				}
				return err
			}
			return printJSON(iss)
		},
	}
}

func newIssueListCmd() *cobra.Command {
	var status, ticket string
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list issues, optionally filtered",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			query := `SELECT id, status, description, ticket, created_at FROM issues WHERE 1=1`
			var qargs []any
			if status != "" {
				query += ` AND status = ?`
				qargs = append(qargs, status)
			}
			if ticket != "" {
				query += ` AND ticket = ?`
				qargs = append(qargs, ticket)
			}
			query += ` ORDER BY id`

			rows, err := db.Query(query, qargs...)
			if err != nil {
				return err
			}
			defer rows.Close()

			issues := []Issue{}
			for rows.Next() {
				var iss Issue
				var t sql.NullString
				if err := rows.Scan(&iss.ID, &iss.Status, &iss.Description, &t, &iss.CreatedAt); err != nil {
					return err
				}
				iss.Ticket = t.String
				issues = append(issues, iss)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			return printJSON(issues)
		},
	}
	cmd.Flags().StringVar(&status, "status", "", "filter by status")
	cmd.Flags().StringVar(&ticket, "ticket", "", "filter by ticket")
	return cmd
}

func newIssueTransitionCmd() *cobra.Command {
	var configPath string
	cmd := &cobra.Command{
		Use:   "transition <id> <event>",
		Short: "fire an event, transitioning the issue if legal",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue id %q: %w", args[0], err)
			}
			event := args[1]

			_, topo, err := ResolveTopology(configPath)
			if err != nil {
				return err
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			next, err := applyTransition(db, topo, id, event)
			if err != nil {
				return err
			}
			fmt.Println(next)
			return nil
		},
	}
	cmd.Flags().StringVar(&configPath, "config", "", "path to topology config")
	return cmd
}

// applyTransition is the DFA enforcement point: it resolves (status, event)
// against the topology, applies any configured limit, and writes the new
// status. Returns the new status.
//
// The write is a compare-and-swap on the status that was read, not a blind
// UPDATE by id. Two orchestrators can resolve to the same issue -- that is
// exactly what resumability-by-ticket makes possible -- and a blind write
// would let a `pass` and a `fail_coder` both succeed, last writer winning,
// silently destroying the guarantee the whole design rests on. A CAS turns
// that race into a visible error instead.
func applyTransition(db *sql.DB, topo *Topology, id int64, event string) (string, error) {
	var status, createdAt string
	row := db.QueryRow(`SELECT status, created_at FROM issues WHERE id = ?`, id)
	if err := row.Scan(&status, &createdAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("issue %d not found", id)
		}
		return "", err
	}

	evs, ok := topo.Transitions[status]
	if !ok {
		return "", fmt.Errorf("illegal transition: state %q has no defined transitions (issue %d unchanged)", status, id)
	}

	effectiveEvent := event
	if lim, ok := topo.Limits[status+"."+event]; ok {
		count, err := countSinceCycle(db, id, topo.Agents[status], lim.CycleAgent, createdAt)
		if err != nil {
			return "", err
		}
		// A zero count means no run of this state's agent was recorded since
		// the cycle boundary, so the cap can never advance and the loop it
		// guards is effectively uncapped. That only happens when the caller
		// skipped the `work start` that every transition is required to
		// follow, which is silent and self-inflicted -- say so.
		if count == 0 {
			fmt.Fprintf(os.Stderr, "warning: limit %s.%s is configured, but no %s work row exists since the cycle boundary, "+
				"so the limit cannot advance. Every transition must be preceded by `work start` for the agent whose output produced the event.\n",
				status, event, strings.Join(topo.Agents[status], "/"))
		}
		if count >= lim.Max {
			fmt.Fprintf(os.Stderr, "limit reached (%d/%d for %s.%s) — auto-routing via %q\n",
				count, lim.Max, status, event, lim.OnExceed)
			effectiveEvent = lim.OnExceed
		}
	}

	next, ok := evs[effectiveEvent]
	if !ok {
		return "", fmt.Errorf("illegal transition: no event %q from state %q (issue %d unchanged)", effectiveEvent, status, id)
	}

	res, err := db.Exec(`UPDATE issues SET status = ? WHERE id = ? AND status = ?`, next, id, status)
	if err != nil {
		return "", err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return "", err
	}
	if n == 0 {
		return "", fmt.Errorf("conflict: issue %d was no longer in state %q when the transition was applied "+
			"(another run changed it concurrently; issue unchanged)", id, status)
	}
	return next, nil
}

// countSinceCycle counts work rows for the given agents, started at or after
// the cycle boundary. The boundary is the most recent work row for
// cycleAgent, or the issue's created_at if cycleAgent hasn't run yet (or is
// unset) — see the Limit doc comment in topology.go.
func countSinceCycle(db *sql.DB, issueID int64, stateAgents []string, cycleAgent string, issueCreatedAt string) (int, error) {
	cycleStart := issueCreatedAt
	if cycleAgent != "" {
		var maxStarted sql.NullString
		row := db.QueryRow(`SELECT MAX(started) FROM work WHERE issue_id = ? AND agent = ?`, issueID, cycleAgent)
		if err := row.Scan(&maxStarted); err != nil {
			return 0, err
		}
		if maxStarted.Valid {
			cycleStart = maxStarted.String
		}
	}
	// A limit on an agentless state can never fire. `topology validate`
	// rejects that config, but a topology can be passed straight to
	// `issue transition` without ever being validated, so refuse loudly here
	// rather than returning a zero count that silently disables the cap.
	if len(stateAgents) == 0 {
		return 0, fmt.Errorf("limit is configured on a state with no agents, so its count would always be zero; run `automake-db topology validate`")
	}
	placeholders := make([]string, len(stateAgents))
	qargs := []any{issueID}
	for i, a := range stateAgents {
		placeholders[i] = "?"
		qargs = append(qargs, a)
	}
	qargs = append(qargs, cycleStart)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM work WHERE issue_id = ? AND agent IN (%s) AND started >= ?`,
		strings.Join(placeholders, ","))
	var count int
	if err := db.QueryRow(query, qargs...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ---- dependencies ----

func newDepCmd() *cobra.Command {
	dep := &cobra.Command{
		Use:   "dep",
		Short: "manage issue dependencies",
	}
	dep.AddCommand(newDepAddCmd(), newDepListCmd())
	return dep
}

func newDepAddCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "add <id> <dependency-id>",
		Short: "record that <id> depends on <dependency-id>",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue id %q: %w", args[0], err)
			}
			dep, err := strconv.ParseInt(args[1], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid dependency id %q: %w", args[1], err)
			}
			if id == dep {
				return fmt.Errorf("an issue cannot depend on itself")
			}
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()
			// The CHECK constraint only stops the one-hop case (id == dep);
			// a longer cycle would be accepted and then deadlock anything
			// that walks dependencies looking for ready work.
			cyclic, err := dependsOn(db, dep, id)
			if err != nil {
				return err
			}
			if cyclic {
				return fmt.Errorf("dependency cycle: issue %d already depends on issue %d, directly or transitively", dep, id)
			}
			_, err = db.Exec(`INSERT INTO dependencies (id, dependency) VALUES (?, ?)`, id, dep)
			return err
		},
	}
}

// dependsOn reports whether from reaches to by following dependency edges.
func dependsOn(db *sql.DB, from, to int64) (bool, error) {
	seen := map[int64]bool{from: true}
	queue := []int64{from}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if cur == to {
			return true, nil
		}
		rows, err := db.Query(`SELECT dependency FROM dependencies WHERE id = ?`, cur)
		if err != nil {
			return false, err
		}
		var next []int64
		for rows.Next() {
			var d int64
			if err := rows.Scan(&d); err != nil {
				rows.Close()
				return false, err
			}
			next = append(next, d)
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return false, err
		}
		rows.Close()
		for _, d := range next {
			if !seen[d] {
				seen[d] = true
				queue = append(queue, d)
			}
		}
	}
	return false, nil
}

func newDepListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list <id>",
		Short: "list an issue's dependencies",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid issue id %q: %w", args[0], err)
			}
			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			rows, err := db.Query(`SELECT dependency FROM dependencies WHERE id = ? ORDER BY dependency`, id)
			if err != nil {
				return err
			}
			defer rows.Close()

			deps := []int64{}
			for rows.Next() {
				var d int64
				if err := rows.Scan(&d); err != nil {
					return err
				}
				deps = append(deps, d)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			return printJSON(deps)
		},
	}
}

// ---- work ----

func newWorkCmd() *cobra.Command {
	work := &cobra.Command{
		Use:   "work",
		Short: "record agent work runs against an issue",
	}
	work.AddCommand(newWorkStartCmd(), newWorkFinishCmd(), newWorkListCmd())
	return work
}

func newWorkStartCmd() *cobra.Command {
	var id int64
	var agent, repo, branch, worktree, context string
	cmd := &cobra.Command{
		Use:   "start",
		Short: "record the start of an agent run against an issue; prints the new run id",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 || agent == "" || repo == "" {
				return fmt.Errorf("usage: automake-db work start --id ID --agent NAME --repo PATH [--branch B] [--worktree W] [--context TEXT]")
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			var exists bool
			if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM issues WHERE id = ?)`, id).Scan(&exists); err != nil {
				return err
			}
			if !exists {
				return fmt.Errorf("issue %d not found", id)
			}

			res, err := db.Exec(`INSERT INTO work (issue_id, agent, context, started, repo, branch, worktree)
				VALUES (?, ?, ?, ?, ?, ?, ?)`,
				id, agent, nullIfEmpty(context), nowRFC3339(), repo, nullIfEmpty(branch), nullIfEmpty(worktree))
			if err != nil {
				return err
			}
			run, err := res.LastInsertId()
			if err != nil {
				return err
			}
			fmt.Println(run)
			return nil
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "issue id (required)")
	cmd.Flags().StringVar(&agent, "agent", "", "agent name (required)")
	cmd.Flags().StringVar(&repo, "repo", "", "repo path (required)")
	cmd.Flags().StringVar(&branch, "branch", "", "branch name")
	cmd.Flags().StringVar(&worktree, "worktree", "", "worktree path")
	cmd.Flags().StringVar(&context, "context", "", "handoff context given to the agent (path or inline text)")
	return cmd
}

func newWorkFinishCmd() *cobra.Command {
	var output, pr string
	cmd := &cobra.Command{
		Use:   "finish <run>",
		Short: "record the end of an agent run",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			run, err := strconv.ParseInt(args[0], 10, 64)
			if err != nil {
				return fmt.Errorf("invalid run id %q: %w", args[0], err)
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			res, err := db.Exec(`UPDATE work
				SET finished = ?,
				    output = COALESCE(NULLIF(?, ''), output),
				    pr = COALESCE(NULLIF(?, ''), pr)
				WHERE id = ?`, nowRFC3339(), output, pr, run)
			if err != nil {
				return err
			}
			n, err := res.RowsAffected()
			if err != nil {
				return err
			}
			if n == 0 {
				return fmt.Errorf("run %d not found", run)
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&output, "output", "", "agent's report (path or inline text)")
	cmd.Flags().StringVar(&pr, "pr", "", "PR URL, once known")
	return cmd
}

func newWorkListCmd() *cobra.Command {
	var id int64
	cmd := &cobra.Command{
		Use:   "list",
		Short: "list an issue's work runs",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if id == 0 {
				return fmt.Errorf("usage: automake-db work list --id ID")
			}

			db, err := openDB()
			if err != nil {
				return err
			}
			defer db.Close()

			rows, err := db.Query(`SELECT id, issue_id, agent, context, started, finished, output, repo, branch, worktree, pr
				FROM work WHERE issue_id = ? ORDER BY id`, id)
			if err != nil {
				return err
			}
			defer rows.Close()

			items := []WorkRow{}
			for rows.Next() {
				var w WorkRow
				var context, finished, output, branch, worktree, pr sql.NullString
				if err := rows.Scan(&w.ID, &w.IssueID, &w.Agent, &context, &w.Started, &finished, &output, &w.Repo, &branch, &worktree, &pr); err != nil {
					return err
				}
				w.Context, w.Finished, w.Output = context.String, finished.String, output.String
				w.Branch, w.Worktree, w.PR = branch.String, worktree.String, pr.String
				items = append(items, w)
			}
			if err := rows.Err(); err != nil {
				return err
			}
			return printJSON(items)
		},
	}
	cmd.Flags().Int64Var(&id, "id", 0, "issue id (required)")
	return cmd
}

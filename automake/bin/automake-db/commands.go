package main

import (
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Issue struct {
	ID          int64  `json:"id"`
	Status      string `json:"status"`
	Description string `json:"description"`
	Ticket      string `json:"ticket,omitempty"`
	CreatedAt   string `json:"created_at"`
}

type WorkRow struct {
	Run      int64  `json:"run"`
	ID       int64  `json:"id"`
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

func cmdInit(args []string) error {
	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()
	path, _ := dbPath()
	fmt.Println(path)
	return nil
}

// ---- topology ----

func dispatchTopology(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: automake-db topology <validate|show|mermaid>")
	}
	fs := flag.NewFlagSet("topology "+args[0], flag.ExitOnError)
	configPath := fs.String("config", "", "path to topology config")
	fs.Parse(args[1:])

	path, err := resolveTopologyPath(*configPath)
	if err != nil {
		return err
	}
	topo, err := LoadTopology(path)
	if err != nil {
		return err
	}

	switch args[0] {
	case "validate":
		errs := topo.Validate()
		if len(errs) > 0 {
			for _, e := range errs {
				fmt.Fprintln(os.Stderr, "-", e)
			}
			return fmt.Errorf("topology %s is not a valid DFA (%d issue(s))", path, len(errs))
		}
		fmt.Printf("OK: %s is a valid DFA\n", path)
		return nil
	case "show":
		return printJSON(topo)
	case "mermaid":
		fmt.Print(topo.Mermaid())
		return nil
	default:
		return fmt.Errorf("unknown topology subcommand %q", args[0])
	}
}

// ---- issue ----

func dispatchIssue(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: automake-db issue <create|get|list|transition>")
	}
	switch args[0] {
	case "create":
		return cmdIssueCreate(args[1:])
	case "get":
		return cmdIssueGet(args[1:])
	case "list":
		return cmdIssueList(args[1:])
	case "transition":
		return cmdIssueTransition(args[1:])
	default:
		return fmt.Errorf("unknown issue subcommand %q", args[0])
	}
}

func cmdIssueCreate(args []string) error {
	fs := flag.NewFlagSet("issue create", flag.ExitOnError)
	description := fs.String("description", "", "issue description (required)")
	ticket := fs.String("ticket", "", "external ticket URL/ID, or local file path")
	configPath := fs.String("config", "", "path to topology config")
	fs.Parse(args)

	if *description == "" {
		return fmt.Errorf("--description is required")
	}
	path, err := resolveTopologyPath(*configPath)
	if err != nil {
		return err
	}
	topo, err := LoadTopology(path)
	if err != nil {
		return err
	}
	if topo.Initial == "" {
		return fmt.Errorf("topology %s has no initial state", path)
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	res, err := db.Exec(`INSERT INTO issues (status, description, ticket) VALUES (?, ?, ?)`,
		topo.Initial, *description, nullIfEmpty(*ticket))
	if err != nil {
		return err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	fmt.Println(id)
	return nil
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

func cmdIssueGet(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: automake-db issue get <id>")
	}
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
		if err == sql.ErrNoRows {
			return fmt.Errorf("issue %d not found", id)
		}
		return err
	}
	return printJSON(iss)
}

func cmdIssueList(args []string) error {
	fs := flag.NewFlagSet("issue list", flag.ExitOnError)
	status := fs.String("status", "", "filter by status")
	ticket := fs.String("ticket", "", "filter by ticket")
	fs.Parse(args)

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	query := `SELECT id, status, description, ticket, created_at FROM issues WHERE 1=1`
	var qargs []any
	if *status != "" {
		query += ` AND status = ?`
		qargs = append(qargs, *status)
	}
	if *ticket != "" {
		query += ` AND ticket = ?`
		qargs = append(qargs, *ticket)
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
}

func cmdIssueTransition(args []string) error {
	fs := flag.NewFlagSet("issue transition", flag.ExitOnError)
	configPath := fs.String("config", "", "path to topology config")
	fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 2 {
		return fmt.Errorf("usage: automake-db issue transition [--config PATH] <id> <event>")
	}
	id, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid issue id %q: %w", rest[0], err)
	}
	event := rest[1]

	path, err := resolveTopologyPath(*configPath)
	if err != nil {
		return err
	}
	topo, err := LoadTopology(path)
	if err != nil {
		return err
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var status, createdAt string
	row := db.QueryRow(`SELECT status, created_at FROM issues WHERE id = ?`, id)
	if err := row.Scan(&status, &createdAt); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("issue %d not found", id)
		}
		return err
	}

	evs, ok := topo.Transitions[status]
	if !ok {
		return fmt.Errorf("illegal transition: state %q has no defined transitions (issue %d unchanged)", status, id)
	}

	effectiveEvent := event
	if lim, ok := topo.Limits[status+"."+event]; ok {
		count, err := countSinceCycle(db, id, topo.Agents[status], lim.CycleAgent, createdAt)
		if err != nil {
			return err
		}
		if count >= lim.Max {
			fmt.Fprintf(os.Stderr, "limit reached (%d/%d for %s.%s) — auto-routing via %q\n",
				count, lim.Max, status, event, lim.OnExceed)
			effectiveEvent = lim.OnExceed
		}
	}

	next, ok := evs[effectiveEvent]
	if !ok {
		return fmt.Errorf("illegal transition: no event %q from state %q (issue %d unchanged)", effectiveEvent, status, id)
	}

	if _, err := db.Exec(`UPDATE issues SET status = ? WHERE id = ?`, next, id); err != nil {
		return err
	}
	fmt.Println(next)
	return nil
}

// countSinceCycle counts work rows for the given agents, started at or after
// the cycle boundary. The boundary is the most recent work row for
// cycleAgent, or the issue's created_at if cycleAgent hasn't run yet (or is
// unset) — see the Limit doc comment in topology.go.
func countSinceCycle(db *sql.DB, issueID int64, stateAgents []string, cycleAgent string, issueCreatedAt string) (int, error) {
	cycleStart := issueCreatedAt
	if cycleAgent != "" {
		var maxStarted sql.NullString
		row := db.QueryRow(`SELECT MAX(started) FROM work WHERE id = ? AND agent = ?`, issueID, cycleAgent)
		if err := row.Scan(&maxStarted); err != nil {
			return 0, err
		}
		if maxStarted.Valid {
			cycleStart = maxStarted.String
		}
	}
	if len(stateAgents) == 0 {
		return 0, nil
	}
	placeholders := make([]string, len(stateAgents))
	qargs := []any{issueID}
	for i, a := range stateAgents {
		placeholders[i] = "?"
		qargs = append(qargs, a)
	}
	qargs = append(qargs, cycleStart)
	query := fmt.Sprintf(`SELECT COUNT(*) FROM work WHERE id = ? AND agent IN (%s) AND started >= ?`,
		strings.Join(placeholders, ","))
	var count int
	if err := db.QueryRow(query, qargs...).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}

// ---- dependencies ----

func dispatchDep(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: automake-db dep <add|list>")
	}
	switch args[0] {
	case "add":
		return cmdDepAdd(args[1:])
	case "list":
		return cmdDepList(args[1:])
	default:
		return fmt.Errorf("unknown dep subcommand %q", args[0])
	}
}

func cmdDepAdd(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("usage: automake-db dep add <id> <dependency-id>")
	}
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
	_, err = db.Exec(`INSERT INTO dependencies (id, dependency) VALUES (?, ?)`, id, dep)
	return err
}

func cmdDepList(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("usage: automake-db dep list <id>")
	}
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
}

// ---- work ----

func dispatchWork(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("usage: automake-db work <start|finish|list>")
	}
	switch args[0] {
	case "start":
		return cmdWorkStart(args[1:])
	case "finish":
		return cmdWorkFinish(args[1:])
	case "list":
		return cmdWorkList(args[1:])
	default:
		return fmt.Errorf("unknown work subcommand %q", args[0])
	}
}

func cmdWorkStart(args []string) error {
	fs := flag.NewFlagSet("work start", flag.ExitOnError)
	id := fs.Int64("id", 0, "issue id (required)")
	agent := fs.String("agent", "", "agent name (required)")
	repo := fs.String("repo", "", "repo path (required)")
	branch := fs.String("branch", "", "branch name")
	worktree := fs.String("worktree", "", "worktree path")
	context := fs.String("context", "", "handoff context given to the agent (path or inline text)")
	fs.Parse(args)

	if *id == 0 || *agent == "" || *repo == "" {
		return fmt.Errorf("usage: automake-db work start --id ID --agent NAME --repo PATH [--branch B] [--worktree W] [--context TEXT]")
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	var exists bool
	if err := db.QueryRow(`SELECT EXISTS(SELECT 1 FROM issues WHERE id = ?)`, *id).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("issue %d not found", *id)
	}

	res, err := db.Exec(`INSERT INTO work (id, agent, context, started, repo, branch, worktree)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		*id, *agent, nullIfEmpty(*context), nowRFC3339(), *repo, nullIfEmpty(*branch), nullIfEmpty(*worktree))
	if err != nil {
		return err
	}
	run, err := res.LastInsertId()
	if err != nil {
		return err
	}
	fmt.Println(run)
	return nil
}

func cmdWorkFinish(args []string) error {
	fs := flag.NewFlagSet("work finish", flag.ExitOnError)
	output := fs.String("output", "", "agent's report (path or inline text)")
	pr := fs.String("pr", "", "PR URL, once known")
	fs.Parse(args)
	rest := fs.Args()
	if len(rest) != 1 {
		return fmt.Errorf("usage: automake-db work finish [--output TEXT] [--pr URL] <run>")
	}
	run, err := strconv.ParseInt(rest[0], 10, 64)
	if err != nil {
		return fmt.Errorf("invalid run id %q: %w", rest[0], err)
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
		WHERE run = ?`, nowRFC3339(), *output, *pr, run)
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
}

func cmdWorkList(args []string) error {
	fs := flag.NewFlagSet("work list", flag.ExitOnError)
	id := fs.Int64("id", 0, "issue id (required)")
	fs.Parse(args)
	if *id == 0 {
		return fmt.Errorf("usage: automake-db work list --id ID")
	}

	db, err := openDB()
	if err != nil {
		return err
	}
	defer db.Close()

	rows, err := db.Query(`SELECT run, id, agent, context, started, finished, output, repo, branch, worktree, pr
		FROM work WHERE id = ? ORDER BY run`, *id)
	if err != nil {
		return err
	}
	defer rows.Close()

	items := []WorkRow{}
	for rows.Next() {
		var w WorkRow
		var context, finished, output, branch, worktree, pr sql.NullString
		if err := rows.Scan(&w.Run, &w.ID, &w.Agent, &context, &w.Started, &finished, &output, &w.Repo, &branch, &worktree, &pr); err != nil {
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
}

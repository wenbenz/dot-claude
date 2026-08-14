CREATE TABLE IF NOT EXISTS issues (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  status      TEXT NOT NULL,
  description TEXT NOT NULL,
  ticket      TEXT,
  created_at  TIMESTAMPTZ NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%fZ','now'))
);

CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_ticket ON issues(ticket);

CREATE TABLE IF NOT EXISTS dependencies (
  id         INTEGER NOT NULL REFERENCES issues(id),
  dependency INTEGER NOT NULL REFERENCES issues(id),
  PRIMARY KEY (id, dependency),
  CHECK (id != dependency)
);

-- The primary key already indexes (id, dependency) for forward lookups
-- (dep list <id>); this covers the reverse direction (which issues depend
-- on a given one), not currently exposed by the CLI but cheap to keep ready.
CREATE INDEX IF NOT EXISTS idx_dependencies_dependency ON dependencies(dependency);

CREATE TABLE IF NOT EXISTS work (
  id       INTEGER PRIMARY KEY AUTOINCREMENT,
  issue_id INTEGER NOT NULL REFERENCES issues(id),
  agent    TEXT NOT NULL,
  context  TEXT,
  started  TIMESTAMPTZ NOT NULL,
  finished TIMESTAMPTZ,
  output   TEXT,
  repo     TEXT NOT NULL,
  branch   TEXT,
  worktree TEXT,
  pr       TEXT
);

-- Matches countSinceCycle's two query shapes: "most recent started for
-- (issue_id, agent)" and "count of started >= boundary for (issue_id,
-- agent...)" -- both filter issue_id and agent before ranging on started.
CREATE INDEX IF NOT EXISTS idx_work_issue_agent_started ON work(issue_id, agent, started);

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// Limit caps how many times an event may fire from a state before the CLI
// auto-routes to on_exceed instead. Max counts total occurrences of the
// state's agent(s) running since the cycle boundary, not "retries" — see
// automake/README.md for the reasoning (matches SKILL.md's plain-English
// "max N rounds"/"max N retries" limits exactly).
//
// CycleAgent names the agent whose most recent work row marks the start of
// the current cycle (e.g. "reviewer" resets the validator round count after
// a review sends a fix back for another pass). Empty means "since the issue
// was created" — i.e. a single global count for the whole issue.
type Limit struct {
	Max        int    `json:"max"`
	OnExceed   string `json:"on_exceed"`
	CycleAgent string `json:"cycle_agent"`
}

// Topology is the DFA: legal states, which agent(s) run in each, and the
// (state, event) -> state transition table that is the sole mechanism for
// changing issues.status. "Configurable and extendable" means editing this
// file — adding a state, its agents, and its transitions — not the CLI.
type Topology struct {
	Initial     string                       `json:"initial"`
	Terminal    []string                     `json:"terminal"`
	Agents      map[string][]string          `json:"agents"`
	Transitions map[string]map[string]string `json:"transitions"`
	Limits      map[string]Limit             `json:"limits"`
}

// resolveTopologyPath picks the config to load: an explicit --config flag
// wins, then $AUTOMAKE_TOPOLOGY, then the shipped default. Per-repo overrides
// (<repo_root>/.claude/automake/topology.json) are the orchestrating skill's
// job to detect and pass in via --config — the CLI itself stays repo-agnostic.
func resolveTopologyPath(flagVal string) (string, error) {
	if flagVal != "" {
		return flagVal, nil
	}
	if env := os.Getenv("AUTOMAKE_TOPOLOGY"); env != "" {
		return env, nil
	}
	return defaultTopologyPath()
}

// defaultTopologyPath locates the shipped default config. The binary is
// `go install`ed, so it cannot be found relative to the executable ($GOBIN is
// nowhere near the plugin) — and the build-time source path is not reliable
// either: it is baked in at compile time, so an installed binary keeps
// pointing at whichever checkout built it, which breaks as soon as the plugin
// is updated or reinstalled into a different cache directory (or was built
// from a temp clone). $CLAUDE_PLUGIN_ROOT is the only locator that stays
// correct across reinstalls, so it is tried first; the source-relative path
// remains as a fallback for `go run` and local development, and is only used
// if it actually still exists.
func defaultTopologyPath() (string, error) {
	var tried []string

	if root := os.Getenv("CLAUDE_PLUGIN_ROOT"); root != "" {
		p := filepath.Join(root, "config", "topology.default.json")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		tried = append(tried, p+" (from $CLAUDE_PLUGIN_ROOT)")
	}

	if _, thisFile, _, ok := runtime.Caller(0); ok {
		// thisFile is .../automake/bin/automake-db/topology.go
		p := filepath.Join(filepath.Dir(thisFile), "..", "..", "config", "topology.default.json")
		if _, err := os.Stat(p); err == nil {
			return p, nil
		}
		tried = append(tried, p+" (build-time source location)")
	}

	return "", fmt.Errorf("cannot locate the default topology config; tried:\n  %s\n"+
		"set $CLAUDE_PLUGIN_ROOT to the automake plugin directory, set $AUTOMAKE_TOPOLOGY "+
		"to a config file, or pass --config PATH", strings.Join(tried, "\n  "))
}

func LoadTopology(path string) (*Topology, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading topology config %s: %w", path, err)
	}
	var t Topology
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("parsing topology config %s: %w", path, err)
	}
	return &t, nil
}

// Validate checks the config is a well-formed DFA: every transition target
// and agents-map key is a declared state, every limit refers to a real
// (state, event) pair and a real on_exceed event, and every non-terminal
// state is reachable from initial and has at least one way out. Returns a
// human-readable defect list; empty means the topology is sound.
func (t *Topology) Validate() []string {
	var errs []string

	if t.Initial == "" {
		errs = append(errs, "initial state is not set")
	}

	states := map[string]bool{}
	if t.Initial != "" {
		states[t.Initial] = true
	}
	for _, s := range t.Terminal {
		states[s] = true
	}
	for s, evs := range t.Transitions {
		states[s] = true
		for _, target := range evs {
			states[target] = true
		}
	}
	for s := range t.Agents {
		states[s] = true
	}

	terminalSet := map[string]bool{}
	for _, s := range t.Terminal {
		terminalSet[s] = true
	}

	for _, s := range sortedKeys(states) {
		if terminalSet[s] {
			continue
		}
		if len(t.Transitions[s]) == 0 {
			errs = append(errs, fmt.Sprintf("state %q is non-terminal but has no outgoing transitions", s))
		}
	}

	for _, s := range sortedKeys(t.Agents) {
		if !states[s] {
			errs = append(errs, fmt.Sprintf("agents map references undeclared state %q", s))
		}
		if len(t.Agents[s]) == 0 {
			errs = append(errs, fmt.Sprintf("agents map for state %q is empty", s))
		}
	}

	declaredAgents := map[string]bool{}
	for _, agents := range t.Agents {
		for _, a := range agents {
			declaredAgents[a] = true
		}
	}

	for _, key := range sortedLimitKeys(t.Limits) {
		lim := t.Limits[key]
		state, event, ok := splitLimitKey(key)
		if !ok {
			errs = append(errs, fmt.Sprintf("limit key %q must be in the form state.event", key))
			continue
		}
		evs, ok := t.Transitions[state]
		if !ok {
			errs = append(errs, fmt.Sprintf("limit %q refers to unknown state %q", key, state))
			continue
		}
		if _, ok := evs[event]; !ok {
			errs = append(errs, fmt.Sprintf("limit %q refers to undeclared event %q on state %q", key, event, state))
		}
		if _, ok := evs[lim.OnExceed]; !ok {
			errs = append(errs, fmt.Sprintf("limit %q on_exceed %q is not a declared event on state %q", key, lim.OnExceed, state))
		}
		if lim.Max <= 0 {
			errs = append(errs, fmt.Sprintf("limit %q has non-positive max %d", key, lim.Max))
		}
		// A limit counts work rows for the state's agents. With no agents
		// declared the count is always 0, so the cap silently never fires and
		// the state loops forever — the exact drift this command exists to
		// catch, so it is an error rather than a warning.
		if len(t.Agents[state]) == 0 {
			errs = append(errs, fmt.Sprintf("limit %q is on state %q, which declares no agents; its count would always be 0 and the limit could never fire", key, state))
		}
		// An unknown cycle_agent is never found in work history, so the cycle
		// boundary silently falls back to the issue's creation time and the
		// limit counts globally instead of per cycle — firing far earlier than
		// the config says, with no error anywhere.
		if lim.CycleAgent != "" && !declaredAgents[lim.CycleAgent] {
			errs = append(errs, fmt.Sprintf("limit %q has cycle_agent %q, which is not an agent of any declared state", key, lim.CycleAgent))
		}
	}

	if t.Initial != "" && states[t.Initial] {
		reached := map[string]bool{t.Initial: true}
		queue := []string{t.Initial}
		for len(queue) > 0 {
			s := queue[0]
			queue = queue[1:]
			for _, target := range t.Transitions[s] {
				if !reached[target] {
					reached[target] = true
					queue = append(queue, target)
				}
			}
		}
		for _, s := range sortedKeys(states) {
			if !reached[s] {
				errs = append(errs, fmt.Sprintf("state %q is unreachable from initial state %q", s, t.Initial))
			}
		}
	}

	return errs
}

// Mermaid renders the topology as a stateDiagram-v2 block so docs can embed
// it directly instead of hand-maintaining a diagram that can drift from the
// enforced config.
func (t *Topology) Mermaid() string {
	var b strings.Builder
	b.WriteString("stateDiagram-v2\n")
	if t.Initial != "" {
		fmt.Fprintf(&b, "    [*] --> %s\n", t.Initial)
	}
	for _, s := range sortedKeys(t.Transitions) {
		for _, e := range sortedKeys(t.Transitions[s]) {
			fmt.Fprintf(&b, "    %s --> %s : %s\n", s, t.Transitions[s][e], e)
		}
	}
	terminal := append([]string{}, t.Terminal...)
	sort.Strings(terminal)
	for _, s := range terminal {
		fmt.Fprintf(&b, "    %s --> [*]\n", s)
	}
	return b.String()
}

func splitLimitKey(key string) (state, event string, ok bool) {
	idx := strings.LastIndex(key, ".")
	if idx < 0 {
		return "", "", false
	}
	return key[:idx], key[idx+1:], true
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func sortedLimitKeys(m map[string]Limit) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

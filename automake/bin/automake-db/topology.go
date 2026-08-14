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

func defaultTopologyPath() (string, error) {
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		return "", fmt.Errorf("cannot resolve automake-db source location")
	}
	// thisFile is .../automake/bin/automake-db/topology.go
	dir := filepath.Dir(thisFile)
	return filepath.Join(dir, "..", "..", "config", "topology.default.json"), nil
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

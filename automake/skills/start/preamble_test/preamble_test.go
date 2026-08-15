// Package preamble_test exercises automake/skills/start/preamble.sh (and the
// tiny inline $CLAUDE_PLUGIN_ROOT guard left behind in SKILL.md's `!` block)
// as black-box subprocesses, per the plan's Testability Notes recommendation
// to extract the setup logic into a companion script so PATH,
// $CLAUDE_PLUGIN_ROOT, and cwd become injectable.
//
// Requirement coverage:
//   - REQ-001 (go PATH fallback + captured build stderr):
//     TestResolveGoAndBuild_PrefersPathOverFallback
//     TestResolveGoAndBuild_FallsBackWhenPathHasNoGo
//     TestResolveGo_ReportsNotFoundWhenNoCandidateExists
//     TestResolveGo_SkipsNonExecutableCandidate
//     TestBuildAutomakeDB_SucceedsWithRealGo
//     TestBuildAutomakeDB_SurfacesCapturedStderrOnFailure
//     TestBuildAutomakeDB_TruncatesVeryLongStderr
//   - REQ-002 ($CLAUDE_PLUGIN_ROOT unset guard):
//     TestPluginRootGuard_UnsetReportsExplicitError
//     TestPluginRootGuard_SetDelegatesToPreambleScript
//   - REQ-003 (diagnostic-only repo-root/branch wording):
//     TestAmbientDiagnostics_InsideGitRepo
//     TestAmbientDiagnostics_OutsideGitRepo
//     TestAmbientDiagnostics_WordingIsMarkedDiagnosticOnly
//
// Convention preserved from the script/skill itself: failure is detected by
// pattern-matching printed stdout text, not process exit status (`!` blocks
// run non-interactively with no stdin) — so these tests assert on stdout
// content, not exit codes.
package preamble_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Path/location helpers
// ---------------------------------------------------------------------------

// startDir returns the absolute path to automake/skills/start, computed
// relative to this test file so the suite works regardless of the caller's
// working directory or module layout.
func startDir(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file's own location via runtime.Caller")
	}
	dir, err := filepath.Abs(filepath.Join(filepath.Dir(thisFile), ".."))
	if err != nil {
		t.Fatalf("resolving automake/skills/start dir: %v", err)
	}
	return dir
}

func preambleScriptPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(startDir(t), "preamble.sh")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("automake/skills/start/preamble.sh not found (expected companion script from the plan's Testability Notes): %v", err)
	}
	return p
}

func skillMDPath(t *testing.T) string {
	t.Helper()
	p := filepath.Join(startDir(t), "SKILL.md")
	if _, err := os.Stat(p); err != nil {
		t.Fatalf("automake/skills/start/SKILL.md not found: %v", err)
	}
	return p
}

// ---------------------------------------------------------------------------
// Subprocess execution helper
// ---------------------------------------------------------------------------

type result struct {
	stdout   string
	stderr   string
	exitCode int
}

// runBashScript runs `bash <script>` with a fully-controlled environment and
// working directory, and captures stdout/stderr separately (the scripts
// under test intentionally write diagnostics to stdout, so tests assert on
// stdout — see convention note in the package doc).
func runBashScript(t *testing.T, script string, dir string, env []string) result {
	t.Helper()
	cmd := exec.Command("bash", script)
	cmd.Dir = dir
	cmd.Env = env
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("failed to run %s: %v", script, err)
		}
	}
	return result{stdout: stdout.String(), stderr: stderr.String(), exitCode: exitCode}
}

// runInlineScript writes literal shell source (e.g. extracted from a
// Markdown fenced block) to a temp file and runs it the same way.
func runInlineScript(t *testing.T, source string, dir string, env []string) result {
	t.Helper()
	tmp := filepath.Join(t.TempDir(), "inline.sh")
	if err := os.WriteFile(tmp, []byte(source), 0o755); err != nil {
		t.Fatalf("writing inline script: %v", err)
	}
	return runBashScript(t, tmp, dir, env)
}

// ---------------------------------------------------------------------------
// Environment construction helpers
// ---------------------------------------------------------------------------

// minimalPathDirs are directories that, on this repo's supported dev/CI
// environments, provide bash/git/coreutils but (verified at package-load
// time by ensureNoGoOnMinimalPath) never provide a `go` binary themselves.
// Tests that need to hide `go` from $PATH build their env's PATH from these
// plus, optionally, one prepended stub/real-go directory.
var minimalPathDirs = []string{"/usr/bin", "/bin", "/usr/local/bin", "/usr/sbin", "/sbin"}

func minimalPath() string {
	return strings.Join(minimalPathDirs, ":")
}

func dirHasExecutable(dir, name string) bool {
	p := filepath.Join(dir, name)
	info, err := os.Stat(p)
	if err != nil {
		return false
	}
	return !info.IsDir() && info.Mode()&0o111 != 0
}

// ensureNoGoOnMinimalPath guards every test that relies on minimalPath()
// being go-less. If some environment's /usr/bin, /bin, etc. ship a `go`
// binary directly (unlike this suite's dev sandbox), the affected tests
// skip with a clear reason instead of silently asserting the wrong thing.
func ensureNoGoOnMinimalPath(t *testing.T) {
	t.Helper()
	for _, d := range minimalPathDirs {
		if dirHasExecutable(d, "go") {
			t.Skipf("environment ships a `go` binary directly in %s, which this suite's PATH-hiding tests assume is go-less; adjust minimalPathDirs for this environment", d)
		}
	}
}

// fixedFallbackCandidates mirrors resolve_go's hardcoded, non-PATH,
// non-$HOME candidate list in automake/skills/start/preamble.sh (kept in
// sync manually — there are only two such entries).
var fixedFallbackCandidates = []string{"/usr/lib/go/bin/go", "/opt/go/bin/go"}

// hostRealGoFallback returns the path of whichever of resolve_go's fixed,
// real-filesystem fallback candidates (checked in the script's own priority
// order: /usr/local/go/bin/go, then the two fixedFallbackCandidates above)
// actually exists on this machine, or "" if none do. $HOME/go/bin/go is
// deliberately excluded — that slot is always test-controlled instead.
func hostRealGoFallback(t *testing.T) string {
	t.Helper()
	candidates := append([]string{"/usr/local/go/bin/go"}, fixedFallbackCandidates...)
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() && info.Mode()&0o111 != 0 {
			return c
		}
	}
	return ""
}

func writeExecutableFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("writing %s: %v", path, err)
	}
}

// writeGoStub writes a fake `go` binary that, when invoked (regardless of
// arguments — it does not need to understand `build -C ... -o ...`),
// appends markerTag plus its args to $MARKER_FILE (if markerTag != ""),
// writes stderrMsg to stderr (if non-empty), and exits with exitCode. This
// decouples resolution tests from real compilation.
func writeGoStub(t *testing.T, path string, exitCode int, stderrMsg, markerTag string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	if markerTag != "" {
		b.WriteString(fmt.Sprintf("echo \"%s $*\" >> \"$MARKER_FILE\"\n", markerTag))
	}
	if stderrMsg != "" {
		b.WriteString("cat >&2 <<'STUBSTDERR'\n")
		b.WriteString(stderrMsg)
		b.WriteString("\nSTUBSTDERR\n")
	}
	b.WriteString(fmt.Sprintf("exit %d\n", exitCode))
	writeExecutableFile(t, path, b.String())
}

// writeBuildableFixture writes a minimal, dependency-free Go module at
// <pluginRoot>/cli/automake-db so a *real* `go build` can succeed (broken=
// false) or fail with a genuine compiler error (broken=true) offline,
// without touching the real automake-db module (which pulls in cobra/
// viper/sqlite and would make build-outcome tests slow/network-dependent).
func writeBuildableFixture(t *testing.T, pluginRoot string, broken bool) {
	t.Helper()
	dir := filepath.Join(pluginRoot, "cli", "automake-db")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir fixture module dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module automake-db-fixture\n\ngo 1.18\n"), 0o644); err != nil {
		t.Fatalf("writing fixture go.mod: %v", err)
	}
	main := "package main\n\nfunc main() {}\n"
	if broken {
		// Deliberate, stable compile error (undefined identifier) so the
		// captured stderr contains a distinctive, greppable token.
		main = "package main\n\nfunc main() {\n\tundefinedSymbolXYZ()\n}\n"
	}
	if err := os.WriteFile(filepath.Join(dir, "main.go"), []byte(main), 0o644); err != nil {
		t.Fatalf("writing fixture main.go: %v", err)
	}
}

func systemGoDir(t *testing.T) string {
	t.Helper()
	p, err := exec.LookPath("go")
	if err != nil {
		t.Skip("no `go` binary resolvable via this test process's own $PATH; skipping real-build test")
	}
	abs, err := filepath.Abs(p)
	if err != nil {
		t.Fatalf("resolving go path: %v", err)
	}
	return filepath.Dir(abs)
}

// realGoEnvExtras returns env entries needed for a *real* `go build` to run
// hermetically inside a per-test temp HOME, without network access or
// interference from the invoking user's real Go environment.
func realGoEnvExtras(tempRoot string) []string {
	return []string{
		"GOCACHE=" + filepath.Join(tempRoot, "gocache"),
		"GOPATH=" + filepath.Join(tempRoot, "gopath"),
		"GOTOOLCHAIN=local",
		"GOPROXY=off",
	}
}

func gitInitWithCommit(t *testing.T, dir string) {
	t.Helper()
	run := func(args ...string) string {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=preamble-test", "GIT_AUTHOR_EMAIL=preamble-test@example.com",
			"GIT_COMMITTER_NAME=preamble-test", "GIT_COMMITTER_EMAIL=preamble-test@example.com",
		)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %s failed: %v\n%s", strings.Join(args, " "), err, out)
		}
		return strings.TrimSpace(string(out))
	}
	run("init", "-q")
	run("commit", "--allow-empty", "-q", "-m", "init")
}

func gitShowToplevel(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel in %s: %v", dir, err)
	}
	return strings.TrimSpace(string(out))
}

func gitCurrentBranch(t *testing.T, dir string) string {
	t.Helper()
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("git branch --show-current in %s: %v", dir, err)
	}
	return strings.TrimSpace(string(out))
}

func mustContain(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Errorf("%s: expected output to contain %q, got:\n%s", context, needle, haystack)
	}
}

func mustNotContain(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Errorf("%s: expected output NOT to contain %q, got:\n%s", context, needle, haystack)
	}
}

// ---------------------------------------------------------------------------
// REQ-001: go PATH fallback + captured build stderr
// ---------------------------------------------------------------------------

// go present on $PATH: used directly, and the fallback list must not even
// be consulted (edge case from the plan: "fallback search must not run").
func TestResolveGoAndBuild_PrefersPathOverFallback(t *testing.T) {
	tempRoot := t.TempDir()
	home := filepath.Join(tempRoot, "home")
	stubDir := filepath.Join(tempRoot, "pathstub")
	marker := filepath.Join(tempRoot, "marker.log")
	pluginRoot := filepath.Join(tempRoot, "plugin")

	writeGoStub(t, filepath.Join(stubDir, "go"), 0, "", "PATH_GO")
	// A working fallback stub too — if resolve_go ever invoked it, that
	// would be a bug (fallback search must not run when PATH has go).
	writeGoStub(t, filepath.Join(home, "go", "bin", "go"), 0, "", "FALLBACK_GO")

	env := []string{
		"PATH=" + stubDir + ":" + minimalPath(),
		"HOME=" + home,
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"MARKER_FILE=" + marker,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	mustContain(t, res.stdout, "automake-db: built", "PATH-resolved go build")

	markerContent, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("expected the PATH go stub to have run and written a marker file, got: %v", err)
	}
	mustContain(t, string(markerContent), "PATH_GO", "resolved-binary marker")
	mustNotContain(t, string(markerContent), "FALLBACK_GO", "resolved-binary marker (fallback must not have run)")
}

// go absent from $PATH, present at a fallback location: build must still
// succeed instead of failing. Where possible this uses whichever *real*
// go install this host already has at one of resolve_go's fixed fallback
// paths (proving genuine end-to-end resolution+build); if the host has none
// of those, it falls back to a controlled $HOME/go/bin/go stub instead.
func TestResolveGoAndBuild_FallsBackWhenPathHasNoGo(t *testing.T) {
	ensureNoGoOnMinimalPath(t)
	tempRoot := t.TempDir()
	pluginRoot := filepath.Join(tempRoot, "plugin")

	if real := hostRealGoFallback(t); real != "" {
		// This host already has a real go at one of resolve_go's fixed
		// fallback locations — exercise the real resolution+build path.
		writeBuildableFixture(t, pluginRoot, false)
		env := append([]string{
			"PATH=" + minimalPath(),
			"HOME=" + os.Getenv("HOME"),
			"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
			"GIT_CEILING_DIRECTORIES=" + tempRoot,
		}, realGoEnvExtras(tempRoot)...)
		res := runBashScript(t, preambleScriptPath(t), tempRoot, env)
		mustContain(t, res.stdout, "automake-db: built",
			fmt.Sprintf("build via real fallback go at %s (PATH had none)", real))
		return
	}

	// No fixed fallback go on this host: prove the $HOME/go/bin/go slot
	// specifically is checked and used via a controlled stub.
	home := filepath.Join(tempRoot, "home")
	marker := filepath.Join(tempRoot, "marker.log")
	writeGoStub(t, filepath.Join(home, "go", "bin", "go"), 0, "", "FALLBACK_GO")

	env := []string{
		"PATH=" + minimalPath(),
		"HOME=" + home,
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"MARKER_FILE=" + marker,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)
	mustContain(t, res.stdout, "automake-db: built", "build via $HOME/go/bin/go fallback stub")

	markerContent, err := os.ReadFile(marker)
	if err != nil {
		t.Fatalf("expected the $HOME/go/bin/go fallback stub to have run: %v", err)
	}
	mustContain(t, string(markerContent), "FALLBACK_GO", "resolved-binary marker")
}

// No go anywhere: PATH clean and (as far as this process can tell) none of
// resolve_go's fixed fallback locations exist either. This dynamically
// skips on any host that happens to have a real go toolchain installed at
// one of those fixed paths (e.g. /usr/local/go/bin/go), since there is no
// portable, unprivileged way to hide a real filesystem entry from a
// hardcoded absolute-path check without root/mount-namespace tooling, which
// this test harness intentionally does not use.
func TestResolveGo_ReportsNotFoundWhenNoCandidateExists(t *testing.T) {
	ensureNoGoOnMinimalPath(t)
	if real := hostRealGoFallback(t); real != "" {
		t.Skipf("host has a real go at fixed fallback path %s; cannot simulate a fully go-less environment without root/mount-namespace access — known environment-dependent limitation, see test-writer report", real)
	}

	tempRoot := t.TempDir()
	home := filepath.Join(tempRoot, "home") // deliberately empty, no go/bin/go
	pluginRoot := filepath.Join(tempRoot, "plugin")
	if err := os.MkdirAll(home, 0o755); err != nil {
		t.Fatal(err)
	}

	env := []string{
		"PATH=" + minimalPath(),
		"HOME=" + home,
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	mustContain(t, res.stdout, "automake-db: BUILD FAILED", "go-not-found output")
	mustContain(t, res.stdout, "go not found", "go-not-found output")
	mustContain(t, res.stdout, "checked:", "go-not-found output should name the checked locations")
	// Must be visually distinguishable from a real build-error failure.
	mustNotContain(t, res.stdout, "automake-db: built", "go-not-found output")
}

// A fallback candidate exists but isn't executable: resolve_go must skip it
// (not crash / not treat it as found) rather than hard-failing differently.
// Same environment-dependent skip rationale as the not-found test above:
// this only isolates cleanly when no *earlier* fixed fallback candidate
// (checked before $HOME in resolve_go's priority order) is a real go.
func TestResolveGo_SkipsNonExecutableCandidate(t *testing.T) {
	ensureNoGoOnMinimalPath(t)
	if real := hostRealGoFallback(t); real != "" {
		t.Skipf("host has a real go at fixed fallback path %s, which is checked before $HOME/go/bin/go, so a non-executable $HOME candidate can't be isolated here — known environment-dependent limitation, see test-writer report", real)
	}

	tempRoot := t.TempDir()
	home := filepath.Join(tempRoot, "home")
	pluginRoot := filepath.Join(tempRoot, "plugin")
	nonExec := filepath.Join(home, "go", "bin", "go")
	if err := os.MkdirAll(filepath.Dir(nonExec), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(nonExec, []byte("#!/usr/bin/env bash\necho should-never-run\n"), 0o644); err != nil { // note: 0644, NOT executable
		t.Fatal(err)
	}

	env := []string{
		"PATH=" + minimalPath(),
		"HOME=" + home,
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	// Skipped (not found/not usable), not crashed, and not misreported as
	// having succeeded.
	mustContain(t, res.stdout, "go not found", "non-executable fallback candidate should be skipped, not used")
	mustNotContain(t, res.stdout, "should-never-run", "the non-executable candidate must never actually run")
}

// Genuine `go build` success via real go on $PATH.
func TestBuildAutomakeDB_SucceedsWithRealGo(t *testing.T) {
	goDir := systemGoDir(t)
	tempRoot := t.TempDir()
	pluginRoot := filepath.Join(tempRoot, "plugin")
	writeBuildableFixture(t, pluginRoot, false)

	env := append([]string{
		"PATH=" + goDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}, realGoEnvExtras(tempRoot)...)
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	mustContain(t, res.stdout, "automake-db: built", "real successful build")
	if _, err := os.Stat(filepath.Join(pluginRoot, "cli", "automake-db", "automake-db")); err != nil {
		t.Errorf("expected the built binary to exist after a reported success: %v", err)
	}
}

// A build failure's literal stderr must be surfaced, not just the generic
// "BUILD FAILED" line. Uses a go stub with deterministic stderr so the
// assertion doesn't depend on any particular compiler's exact wording.
func TestBuildAutomakeDB_SurfacesCapturedStderrOnFailure(t *testing.T) {
	tempRoot := t.TempDir()
	stubDir := filepath.Join(tempRoot, "pathstub")
	pluginRoot := filepath.Join(tempRoot, "plugin")
	const distinctiveError = "./main.go:4:2: undefined: undefinedSymbolXYZ"

	writeGoStub(t, filepath.Join(stubDir, "go"), 1, distinctiveError, "")

	env := []string{
		"PATH=" + stubDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	mustContain(t, res.stdout, "automake-db: BUILD FAILED", "build failure output")
	mustContain(t, res.stdout, distinctiveError, "build failure output must include the literal captured stderr")
	mustNotContain(t, res.stdout, "automake-db: built", "build failure output")
}

// Also validates the same requirement against a *genuine* compile error
// from the real go compiler (the plan's edge case: "Build fails for a
// genuine compile error ... full/near-full stderr must still be shown"),
// as a companion to the stub-based test above.
func TestBuildAutomakeDB_SurfacesCapturedStderrOnFailure_RealCompileError(t *testing.T) {
	goDir := systemGoDir(t)
	tempRoot := t.TempDir()
	pluginRoot := filepath.Join(tempRoot, "plugin")
	writeBuildableFixture(t, pluginRoot, true)

	env := append([]string{
		"PATH=" + goDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}, realGoEnvExtras(tempRoot)...)
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	mustContain(t, res.stdout, "automake-db: BUILD FAILED", "real compile-error output")
	mustContain(t, res.stdout, "undefinedSymbolXYZ", "real compile-error output must include the genuine compiler stderr")
}

// The coder's script caps captured stderr (MAX_BUILD_STDERR_CHARS) per the
// plan's "avoid dumping unbounded stderr" constraint; a very long failure
// must still print a bounded, clearly-truncated version rather than either
// silently dropping it or reproducing it unbounded.
func TestBuildAutomakeDB_TruncatesVeryLongStderr(t *testing.T) {
	tempRoot := t.TempDir()
	stubDir := filepath.Join(tempRoot, "pathstub")
	pluginRoot := filepath.Join(tempRoot, "plugin")
	longMsg := strings.Repeat("E", 20000)

	writeGoStub(t, filepath.Join(stubDir, "go"), 1, longMsg, "")

	env := []string{
		"PATH=" + stubDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runBashScript(t, preambleScriptPath(t), tempRoot, env)

	mustContain(t, res.stdout, "automake-db: BUILD FAILED", "truncated-failure output")
	if len(res.stdout) >= len(longMsg) {
		t.Errorf("expected truncated output (< %d chars of stub stderr alone) but got %d total stdout chars", len(longMsg), len(res.stdout))
	}
	if !strings.Contains(res.stdout, "truncat") { // matches "truncated"/"truncation" either wording
		t.Errorf("expected output to indicate the stderr was truncated, got:\n%.200s...", res.stdout)
	}
}

// ---------------------------------------------------------------------------
// REQ-003: ambient cwd repo-root/branch diagnostics must be non-misleading
// ---------------------------------------------------------------------------

func TestAmbientDiagnostics_InsideGitRepo(t *testing.T) {
	tempRoot := t.TempDir()
	repoDir := filepath.Join(tempRoot, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	gitInitWithCommit(t, repoDir)
	expectedRoot := gitShowToplevel(t, repoDir)
	expectedBranch := gitCurrentBranch(t, repoDir)

	stubDir := filepath.Join(tempRoot, "pathstub")
	writeGoStub(t, filepath.Join(stubDir, "go"), 0, "", "")
	pluginRoot := filepath.Join(tempRoot, "plugin")

	env := []string{
		"PATH=" + stubDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + repoDir,
	}
	res := runBashScript(t, preambleScriptPath(t), repoDir, env)

	mustContain(t, res.stdout, "diagnostic only", "ambient repo-root line")
	mustContain(t, res.stdout, "Step 0.3", "ambient repo-root line should cross-reference Step 0.3 as the source of truth")
	mustContain(t, res.stdout, expectedRoot, "ambient repo-root line should report the actual toplevel")
	mustContain(t, res.stdout, expectedBranch, "ambient branch line should report the actual current branch")
	mustNotContain(t, res.stdout, "(not a git repo)", "ambient repo-root line inside a real repo")
}

func TestAmbientDiagnostics_OutsideGitRepo(t *testing.T) {
	tempRoot := t.TempDir()
	plainDir := filepath.Join(tempRoot, "not-a-repo")
	if err := os.MkdirAll(plainDir, 0o755); err != nil {
		t.Fatal(err)
	}

	stubDir := filepath.Join(tempRoot, "pathstub")
	writeGoStub(t, filepath.Join(stubDir, "go"), 0, "", "")
	pluginRoot := filepath.Join(tempRoot, "plugin")

	env := []string{
		"PATH=" + stubDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		// Prevents git from discovering a real repo above plainDir, in case
		// the OS temp dir happens to be nested inside one.
		"GIT_CEILING_DIRECTORIES=" + plainDir,
	}
	res := runBashScript(t, preambleScriptPath(t), plainDir, env)

	mustContain(t, res.stdout, "(not a git repo)", "ambient repo-root line outside any repo")
	mustContain(t, res.stdout, "(unknown)", "ambient branch line outside any repo")
	mustContain(t, res.stdout, "diagnostic only", "ambient lines should still be marked diagnostic-only, not implying a blocking problem")
}

// Static assertion on the exact wording contract (REQ-003 AC): both lines
// must read as best-effort/diagnostic and cross-reference Step 0.3,
// independent of any particular cwd.
func TestAmbientDiagnostics_WordingIsMarkedDiagnosticOnly(t *testing.T) {
	data, err := os.ReadFile(preambleScriptPath(t))
	if err != nil {
		t.Fatalf("reading preamble.sh: %v", err)
	}
	src := string(data)
	mustContain(t, src, "diagnostic only", "preamble.sh source: repo-root/branch echo wording")
	mustContain(t, src, "Step 0.3", "preamble.sh source: repo-root/branch echo wording should reference Step 0.3")
}

// ---------------------------------------------------------------------------
// REQ-002: $CLAUDE_PLUGIN_ROOT unset guard (stays inline in SKILL.md per the
// plan's Testability Notes — module 1 gates locating preamble.sh itself, so
// it cannot be extracted into the script). Per the plan, this is
// "low-risk"/not mandated for automated coverage, but since the coder kept
// it to the recommended ~3-line inline guard (not a full inline rewrite),
// it can still be exercised by extracting the literal fenced-block text out
// of SKILL.md's Markdown and exec'ing it directly — the fallback technique
// the plan explicitly sanctions for exactly this situation.
// ---------------------------------------------------------------------------

var guardBlockRE = regexp.MustCompile("(?s)```!\n(.*?)\n```")

func extractGuardBlock(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(skillMDPath(t))
	if err != nil {
		t.Fatalf("reading SKILL.md: %v", err)
	}
	m := guardBlockRE.FindSubmatch(data)
	if m == nil {
		t.Fatal("could not find a fenced ```! block in SKILL.md — has the setup preamble format changed? (extraction regex may need updating)")
	}
	return string(m[1])
}

func TestPluginRootGuard_UnsetReportsExplicitError(t *testing.T) {
	guard := extractGuardBlock(t)
	tempRoot := t.TempDir()
	env := []string{"PATH=" + minimalPath()} // CLAUDE_PLUGIN_ROOT deliberately omitted

	res := runInlineScript(t, guard, tempRoot, env)

	mustContain(t, res.stdout, "automake-db: BUILD FAILED", "unset $CLAUDE_PLUGIN_ROOT guard output")
	mustContain(t, res.stdout, "CLAUDE_PLUGIN_ROOT", "unset $CLAUDE_PLUGIN_ROOT guard output should name the variable")
	mustContain(t, res.stdout, "unset", "unset $CLAUDE_PLUGIN_ROOT guard output should say it's unset/empty")
	mustContain(t, res.stdout, "stop and show this to the user", "unset $CLAUDE_PLUGIN_ROOT guard output")
	mustNotContain(t, res.stdout, "automake-db: built", "unset $CLAUDE_PLUGIN_ROOT guard output must not proceed to a build attempt")
}

func TestPluginRootGuard_SetDelegatesToPreambleScript(t *testing.T) {
	guard := extractGuardBlock(t)
	tempRoot := t.TempDir()
	pluginRoot := filepath.Join(tempRoot, "plugin")

	// Build a fixture plugin root containing a real copy of preamble.sh at
	// the exact relative path the guard invokes it from
	// ($CLAUDE_PLUGIN_ROOT/skills/start/preamble.sh), so this test tracks
	// whatever the coder's script actually does rather than a duplicated
	// hardcoded copy.
	real, err := os.ReadFile(preambleScriptPath(t))
	if err != nil {
		t.Fatalf("reading real preamble.sh: %v", err)
	}
	fixtureScript := filepath.Join(pluginRoot, "skills", "start", "preamble.sh")
	writeExecutableFile(t, fixtureScript, string(real))

	stubDir := filepath.Join(tempRoot, "pathstub")
	writeGoStub(t, filepath.Join(stubDir, "go"), 0, "", "")

	env := []string{
		"PATH=" + stubDir + ":" + minimalPath(),
		"HOME=" + filepath.Join(tempRoot, "home"),
		"CLAUDE_PLUGIN_ROOT=" + pluginRoot,
		"GIT_CEILING_DIRECTORIES=" + tempRoot,
	}
	res := runInlineScript(t, guard, tempRoot, env)

	mustContain(t, res.stdout, "automake-db: built", "set $CLAUDE_PLUGIN_ROOT guard output should delegate to preamble.sh and reach a normal build result")
	mustNotContain(t, res.stdout, "CLAUDE_PLUGIN_ROOT is unset", "set $CLAUDE_PLUGIN_ROOT guard output must not fire the unset guard")
}

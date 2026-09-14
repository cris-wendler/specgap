package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func withTasks(t *testing.T) {
	t.Helper()
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	t.Cleanup(func() { os.Unsetenv("SPECGAP_TASKS") })
}

// The agent that solves the cache task, as a shell command.
func copyIn(t *testing.T, name string) string {
	t.Helper()
	return "cp " + filepath.Join(repoRoot(t), "testdata", name) + " cache.go"
}

func TestRunNeedsATaskAndAnAgent(t *testing.T) {
	withTasks(t)
	cases := map[string][]string{
		"no arguments":             {},
		"no agent":                 {"cache"},
		"no task":                  {"--agent", "true"},
		"two tasks":                {"cache", "extra", "--agent", "true"},
		"no attempts":              {"cache", "--agent", "true", "--runs", "0"},
		"a task that is not there": {"nope", "--agent", "true"},
	}
	for what, args := range cases {
		if err := run(args); err == nil {
			t.Errorf("%s was accepted", what)
		}
	}
}

func TestRunGradesWhatTheAgentLeftBehind(t *testing.T) {
	withTasks(t)
	out := captureStdout(t, func() {
		if err := run([]string{"cache", "--agent", copyIn(t, "correct.go.txt")}); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{"visible  100%", "hidden   100%"} {
		if !strings.Contains(out, want) {
			t.Errorf("a correct implementation did not report %q:\n%s", want, out)
		}
	}
}

// Each attempt gets a workspace nothing has seen, so one cannot be
// finishing another one's work.
func TestEachAttemptStartsFromNothing(t *testing.T) {
	withTasks(t)
	out := captureStdout(t, func() {
		if err := run([]string{"cache", "--agent", copyIn(t, "naive.go.txt"), "--runs", "3"}); err != nil {
			t.Fatal(err)
		}
	})
	if strings.Count(out, "attempt ") != 3 {
		t.Errorf("expected three attempts:\n%s", out)
	}
	if !strings.Contains(out, "3 of 3   TestLenDoesNotCountExpiredEntries") {
		t.Errorf("the spread does not say how often it failed:\n%s", out)
	}
	if !strings.Contains(out, "lowest 60%") || !strings.Contains(out, "highest 60%") {
		t.Errorf("the spread is missing:\n%s", out)
	}
}

// An agent that gives up still leaves work behind, and what it left is
// what matters. Throwing the attempt away would hide a partial answer.
func TestAnAgentThatFailsIsStillGraded(t *testing.T) {
	withTasks(t)
	var attempts []Attempt
	out := captureStdout(t, func() {
		if err := run([]string{"cache", "--agent", copyIn(t, "naive.go.txt") + "; exit 3", "--json"}); err != nil {
			t.Fatal(err)
		}
	})
	if err := json.Unmarshal([]byte(out), &attempts); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, out)
	}
	if len(attempts) != 1 {
		t.Fatalf("%d attempts", len(attempts))
	}
	if attempts[0].Error == "" {
		t.Error("the agent exited 3 and nothing recorded it")
	}
	if attempts[0].VisiblePassed != 8 {
		t.Errorf("the work it left behind was not graded: %+v", attempts[0])
	}
}

func TestResultsAreAppendedOneAttemptPerLine(t *testing.T) {
	withTasks(t)
	path := filepath.Join(t.TempDir(), "results.jsonl")
	for i := 0; i < 2; i++ {
		captureStdout(t, func() {
			if err := run([]string{"cache", "--agent", copyIn(t, "naive.go.txt"), "--results", path}); err != nil {
				t.Fatal(err)
			}
		})
	}
	b, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(string(b)), "\n")
	if len(lines) != 2 {
		t.Fatalf("%d lines, want 2", len(lines))
	}
	for _, line := range lines {
		var a Attempt
		if err := json.Unmarshal([]byte(line), &a); err != nil {
			t.Fatalf("line is not an attempt: %v", err)
		}
		if a.Task != "cache" || a.HiddenTotal == 0 || a.StartedAt.IsZero() {
			t.Errorf("thin record: %+v", a)
		}
	}
}

func TestKeptWorkspacesSurviveAndAreSeparate(t *testing.T) {
	withTasks(t)
	dir := t.TempDir()
	captureStdout(t, func() {
		if err := run([]string{"cache", "--agent", copyIn(t, "naive.go.txt"), "--runs", "2", "--keep", dir}); err != nil {
			t.Fatal(err)
		}
	})
	for _, want := range []string{"cache-1", "cache-2"} {
		if _, err := os.Stat(filepath.Join(dir, want, "cache.go")); err != nil {
			t.Errorf("%s was not kept: %v", want, err)
		}
	}
}

// The workspace an agent works in must never hold the tests that grade it.
func TestTheAgentNeverSeesTheHiddenTests(t *testing.T) {
	withTasks(t)
	dir := t.TempDir()
	captureStdout(t, func() {
		if err := run([]string{"cache", "--agent", "ls -a > listing.txt", "--keep", dir}); err != nil {
			t.Fatal(err)
		}
	})
	b, err := ioutil.ReadFile(filepath.Join(dir, "cache-1", "listing.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(b), "zz_hidden") {
		t.Fatalf("the agent could see the hidden tests:\n%s", b)
	}
	// And they are gone again after grading.
	if _, err := os.Stat(filepath.Join(dir, "cache-1", "zz_hidden_test.go")); !os.IsNotExist(err) {
		t.Error("a hidden test was left in the workspace after grading")
	}
}

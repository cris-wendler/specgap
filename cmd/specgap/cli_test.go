package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestListPrintsEveryTaskWithItsSummary(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	out := captureStdout(t, func() {
		if err := list(); err != nil {
			t.Fatal(err)
		}
	})
	all, err := tasks()
	if err != nil {
		t.Fatal(err)
	}
	for _, task := range all {
		if !strings.Contains(out, task.Name) {
			t.Errorf("the list leaves out %s:\n%s", task.Name, out)
		}
		if !strings.Contains(out, strings.TrimSpace(task.Summary)) {
			t.Errorf("the list leaves out the summary of %s", task.Name)
		}
	}
}

func TestTaskLineHoldsTheNameAndTheSummary(t *testing.T) {
	line := Task{Name: "a-task", Summary: "  what it is.  "}.String()
	if !strings.HasPrefix(line, "a-task") {
		t.Errorf("the name is not first: %q", line)
	}
	if !strings.HasSuffix(line, "what it is.") {
		t.Errorf("the summary is not trimmed: %q", line)
	}
}

// The exit codes are what a script reads, so they are checked against
// the built command rather than against the functions behind it.
func TestTheCommandAnswersAndExitsCorrectly(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "specgap")
	build := exec.Command("go", "build", "-o", bin, ".")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}

	cases := []struct {
		args     []string
		code     int
		contains string
	}{
		{nil, 2, "specgap tasks"},
		{[]string{"nonsense"}, 2, "no command named nonsense"},
		{[]string{"--help"}, 0, "specgap grade"},
		{[]string{"help"}, 0, "specgap prepare"},
		{[]string{"grade"}, 1, "needs a task and a directory"},
		{[]string{"prepare", "no-such-task", "somewhere"}, 1, "no task named no-such-task"},
	}
	for _, c := range cases {
		cmd := exec.Command(bin, c.args...)
		cmd.Env = append(os.Environ(), "SPECGAP_TASKS="+repoRoot(t))
		out, err := cmd.CombinedOutput()
		code := 0
		if ee, ok := err.(*exec.ExitError); ok {
			code = ee.ExitCode()
		} else if err != nil {
			t.Fatalf("%v: %v", c.args, err)
		}
		if code != c.code {
			t.Errorf("%v exited %d, want %d\n%s", c.args, code, c.code, out)
		}
		if !strings.Contains(string(out), c.contains) {
			t.Errorf("%v does not say %q:\n%s", c.args, c.contains, out)
		}
	}
}

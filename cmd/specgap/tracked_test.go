package main

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Every source file has to be tracked by Git.
//
// The ignore rule for the built executable was written as `specgap`,
// which matches the directory `cmd/specgap` as well, so the whole
// program was hidden from Git. That was found once and fixed by anchoring
// the rule to the root, and a later change put it back. Nothing noticed,
// because Git keeps tracking a file it already tracks even after the file
// starts matching an ignore rule. Only new files disappear, quietly, at
// the moment somebody adds them.
//
// Two did: the test that proves a score is never rounded up to a hundred,
// and this package's own workspace comparison. Both were written, both
// were committed with everything else, and neither reached the
// repository.
func TestEverySourceFileIsTracked(t *testing.T) {
	out, err := exec.Command("git", "ls-files").Output()
	if err != nil {
		t.Skipf("git is not available here: %v", err)
	}
	tracked := map[string]bool{}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		tracked[filepath.ToSlash(name)] = true
	}
	if len(tracked) == 0 {
		t.Fatal("git tracks no files at all, so this test checks nothing")
	}

	// Source, tasks and schemas: everything the repository is for.
	roots := []string{".", "../../tasks", "../../testdata"}
	checked := 0
	for _, root := range roots {
		matches, gerr := filepath.Glob(filepath.Join(root, "*"))
		if gerr != nil {
			t.Fatal(gerr)
		}
		for _, path := range matches {
			ext := filepath.Ext(path)
			if ext != ".go" && ext != ".json" && ext != ".md" && ext != ".txt" {
				continue
			}
			rel, rerr := filepath.Rel("../..", path)
			if rerr != nil {
				continue
			}
			checked++
			if !tracked[filepath.ToSlash(rel)] {
				t.Errorf("%s is in the working tree and Git does not track it", filepath.ToSlash(rel))
			}
		}
	}
	if checked < 10 {
		t.Fatalf("only %d files were checked, so this test checks nothing", checked)
	}
}

// The rule that hid the source is anchored, so it names the executable at
// the root and not every directory that shares its name.
func TestTheIgnoreRuleIsAnchoredToTheRoot(t *testing.T) {
	out, err := exec.Command("git", "check-ignore", "-v", "cmd/specgap/main.go").CombinedOutput()
	if err == nil {
		t.Errorf("cmd/specgap/main.go is ignored: %s", strings.TrimSpace(string(out)))
	}
	// And the executable itself is still ignored, which is what the rule
	// is for.
	if err := exec.Command("git", "check-ignore", "-q", "specgap").Run(); err != nil {
		t.Error("the built executable at the root is no longer ignored")
	}
}

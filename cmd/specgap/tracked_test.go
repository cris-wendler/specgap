package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// repoRoot is where Git commands have to run from. Run from a package
// directory, git ls-files lists only that directory, which made the
// first version of this test report every file outside it as untracked.
func gitRoot(t *testing.T) string {
	t.Helper()
	out, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("git is not available here: %v", err)
	}
	return strings.TrimSpace(string(out))
}

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
	root := gitRoot(t)

	out, err := exec.Command("git", "-C", root, "ls-files").Output()
	if err != nil {
		t.Fatal(err)
	}
	tracked := map[string]bool{}
	for _, name := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if name != "" {
			tracked[filepath.ToSlash(name)] = true
		}
	}
	if len(tracked) == 0 {
		t.Fatal("git tracks no files at all, so this test checks nothing")
	}

	checked := 0
	err = filepath.Walk(root, func(path string, info os.FileInfo, werr error) error {
		if werr != nil {
			return nil
		}
		if info.IsDir() {
			switch info.Name() {
			case ".git", "dist", "work", "node_modules":
				return filepath.SkipDir
			}
			return nil
		}
		switch filepath.Ext(info.Name()) {
		case ".go", ".json", ".md", ".txt", ".yml", ".sh":
		default:
			return nil
		}
		rel, rerr := filepath.Rel(root, path)
		if rerr != nil {
			return nil
		}
		rel = filepath.ToSlash(rel)
		checked++
		if !tracked[rel] {
			t.Errorf("%s is in the working tree and Git does not track it", rel)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if checked < 20 {
		t.Fatalf("only %d files were checked, so this test checks nothing", checked)
	}
}

// The rule that hid the source is anchored, so it names the executable at
// the root and not every directory that shares its name.
func TestTheIgnoreRuleIsAnchoredToTheRoot(t *testing.T) {
	root := gitRoot(t)

	// --no-index is required. Without it check-ignore says nothing about
	// a file Git already tracks, so this would have stayed quiet through
	// exactly the regression it exists to catch: the rule went back to
	// matching the source directory while every file in it was tracked,
	// and only the next new file would have vanished.
	out, err := exec.Command("git", "-C", root, "check-ignore", "--no-index", "-v", "cmd/specgap/main.go").CombinedOutput()
	if err == nil {
		t.Errorf("cmd/specgap/main.go is ignored: %s", strings.TrimSpace(string(out)))
	}
	// The executable itself is still hidden, which is what the rule is
	// for. It is named rather than built here, because check-ignore
	// answers about a path whether or not anything is at it.
	if err := exec.Command("git", "-C", root, "check-ignore", "--no-index", "-q", "specgap").Run(); err != nil {
		t.Error("the built executable at the root is no longer ignored")
	}
}

package main

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeFile(t *testing.T, dir, rel, body string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func snapOf(t *testing.T, dir string) map[string]fileState {
	t.Helper()
	s, err := snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestWorkSeesWhatWasAddedChangedAndRemoved(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "keep.txt", "one\ntwo\n")
	writeFile(t, dir, "gone.txt", "a\nb\nc\n")
	before := snapOf(t, dir)

	writeFile(t, dir, "keep.txt", "one\ntwo\nthree\nfour\n")
	writeFile(t, dir, "new.txt", "x\n")
	if err := os.Remove(filepath.Join(dir, "gone.txt")); err != nil {
		t.Fatal(err)
	}
	w := compare(before, snapOf(t, dir))

	if len(w.Added) != 1 || w.Added[0] != "new.txt" {
		t.Errorf("added %v", w.Added)
	}
	if len(w.Changed) != 1 || w.Changed[0] != "keep.txt" {
		t.Errorf("changed %v", w.Changed)
	}
	if len(w.Removed) != 1 || w.Removed[0] != "gone.txt" {
		t.Errorf("removed %v", w.Removed)
	}
	if w.Touched() != 3 {
		t.Errorf("touched %d, want 3", w.Touched())
	}
	// keep.txt gained two lines, new.txt brought one, gone.txt took three.
	if w.LinesAdded != 3 || w.LinesRemoved != 3 {
		t.Errorf("lines in %d out %d, want 3 and 3", w.LinesAdded, w.LinesRemoved)
	}
}

// A file rewritten to the same length is still a change. Counting only
// lines would call this nothing happening.
func TestAFileRewrittenToTheSameLengthIsStillChanged(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one\ntwo\n")
	before := snapOf(t, dir)
	writeFile(t, dir, "a.txt", "three\nfour\n")

	w := compare(before, snapOf(t, dir))
	if len(w.Changed) != 1 {
		t.Fatalf("changed %v, want the rewritten file", w.Changed)
	}
	if w.Touched() != 1 {
		t.Errorf("touched %d, want 1", w.Touched())
	}
}

func TestDoingNothingIsRecordedAsNothing(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one\n")
	before := snapOf(t, dir)

	w := compare(before, snapOf(t, dir))
	if w.Touched() != 0 || w.Churn() != 0 {
		t.Errorf("an untouched workspace reports %s", w.describe())
	}
	if w.describe() != "changed nothing" {
		t.Errorf("describe is %q", w.describe())
	}
}

// A repository of its own, or a directory of build output, is not work
// the agent chose to do. Walking into them would drown the count.
func TestBookkeepingDirectoriesAreNotCounted(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "a.txt", "one\n")
	before := snapOf(t, dir)

	for _, rel := range []string{".git/objects/ab/cdef", "node_modules/left-pad/index.js", "__pycache__/x.pyc"} {
		writeFile(t, dir, rel, "noise\n")
	}
	if w := compare(before, snapOf(t, dir)); w.Touched() != 0 {
		t.Errorf("bookkeeping files were counted as work: %v", w.Added)
	}
}

// The line counts are a measure of text. A binary has lines only by
// accident, and counting them would make one image outweigh a rewrite.
func TestABinaryFileCountsAsTouchedButNotAsLines(t *testing.T) {
	dir := t.TempDir()
	before := snapOf(t, dir)
	writeFile(t, dir, "blob.bin", "\x00\x01\x02\n\x00\n")

	w := compare(before, snapOf(t, dir))
	if w.Touched() != 1 {
		t.Fatalf("touched %d, want the binary counted as a file", w.Touched())
	}
	if w.Churn() != 0 {
		t.Errorf("churn %d, want a binary to contribute no lines", w.Churn())
	}
}

func TestDescribeReadsAsASentence(t *testing.T) {
	w := Work{Added: []string{"a"}, Changed: []string{"b", "c"}, LinesAdded: 1, LinesRemoved: 12}
	got := w.describe()
	for _, want := range []string{"1 file added", "2 files changed", "1 line in", "12 lines out"} {
		if !strings.Contains(got, want) {
			t.Errorf("describe is %q, missing %q", got, want)
		}
	}
	if strings.Contains(got, "1 files") || strings.Contains(got, "1 lines") {
		t.Errorf("describe counts a single thing as plural: %q", got)
	}
}

// An attempt must record what the agent did and nothing else.
func TestAnAttemptRecordsWhatTheAgentDid(t *testing.T) {
	withTasks(t)
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}

	a, err := attempt(task, "printf 'package cache\\n' > agent_made_this.go", "", 0)
	if err != nil {
		t.Fatal(err)
	}

	if len(a.Work.Added) != 1 || a.Work.Added[0] != "agent_made_this.go" {
		t.Fatalf("added %v, want only the file the agent wrote", a.Work.Added)
	}
	if a.Work.Touched() != 1 {
		t.Errorf("touched %d, want 1: %s", a.Work.Touched(), a.Work.describe())
	}
}

// Grading is not the agent's work, and the reading has to be taken
// before any of it happens.
//
// The first version of this test planted an ordinary hidden test and
// asserted the planted file was not counted. It passed with the reading
// deliberately moved to after grading, because the grader takes its
// planted files away again and the Go build cache is not in the
// workspace. A test that cannot fail for the thing it names is worth
// less than no test, so this one uses a hidden test that leaves a file
// behind when it runs, which is the case the ordering actually governs.
func TestGradingIsNotCountedAsTheAgentsWork(t *testing.T) {
	dir, commit := pinnedRepo(t)
	// A hidden test that writes a file while it is being graded.
	writeFile(t, dir, "messy_test.go", "package pinned\n\nimport (\n\t\"io/ioutil\"\n\t\"testing\"\n)\n\n"+
		"func TestLeavesSomethingBehind(t *testing.T) {\n"+
		"\tioutil.WriteFile(\"the_grader_wrote_this.txt\", []byte(\"x\\n\"), 0644)\n}\n")
	commitAll(t, dir, "a hidden test that writes while it runs")
	commit = headOf(t, dir)

	task := pinnedTask(dir, commit)
	task.Hidden = map[string]string{"messy_test.go": "messy_test.go"}

	a, err := attempt(task, "true", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.HiddenTotal == 0 {
		t.Fatal("the hidden test never ran, so nothing was written and this proves nothing")
	}
	for _, name := range a.Work.Added {
		if name == "the_grader_wrote_this.txt" {
			t.Fatal("a file written while grading was recorded as the agent's work, " +
				"so every attempt would be credited with work it did not do")
		}
	}
	if a.Work.Touched() != 0 {
		t.Errorf("an agent that did nothing recorded %s", a.Work.describe())
	}
}

// An agent that changes nothing has to read as nothing, or the measure
// cannot tell a failure to act from a small change.
func TestAnAgentThatDidNothingRecordsNothing(t *testing.T) {
	withTasks(t)
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}
	a, err := attempt(task, "true", "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Work.Touched() != 0 || a.Work.Churn() != 0 {
		t.Errorf("an agent that did nothing recorded %s", a.Work.describe())
	}
}

// The reason this exists: two attempts can reach the same score having
// done visibly different amounts of work, and the score cannot say so.
func TestTwoAttemptsCanScoreTheSameAndDifferInWork(t *testing.T) {
	small := Attempt{HiddenPassed: 4, HiddenTotal: 4, Work: Work{Changed: []string{"a"}, LinesAdded: 3}}
	large := Attempt{HiddenPassed: 4, HiddenTotal: 4, Work: Work{
		Changed: []string{"a", "b", "c", "d"}, LinesAdded: 300, LinesRemoved: 120}}

	if small.hiddenScore() != large.hiddenScore() {
		t.Fatal("this test needs two attempts that scored the same")
	}
	if small.Work.Touched() == large.Work.Touched() {
		t.Fatal("this test needs two attempts that did different amounts")
	}
	if !sameHidden([]Attempt{small, large}) {
		t.Error("attempts with one score are not recognised as agreeing")
	}
}

package main

import (
	"io/ioutil"
	"os/exec"
	"path/filepath"
	"testing"
)

// The environment is only worth anything if it separates an
// implementation that answers the visible tests from one that is
// actually right. If both scored the same, there would be nothing to
// measure and the grader would be the broken part.
func gradeImplementation(t *testing.T, name string) (visible, hidden result) {
	t.Helper()
	dir := t.TempDir()
	if err := prepare([]string{"cache", dir}); err != nil {
		t.Fatal(err)
	}
	src, err := ioutil.ReadFile(filepath.Join("..", "..", "testdata", name))
	if err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "cache.go"), src, 0644); err != nil {
		t.Fatal(err)
	}
	visible, err = runTests(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}
	names, remove, err := task.plant(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer remove()
	hidden, err = runTests(dir, names)
	if err != nil {
		t.Fatal(err)
	}
	return visible, hidden
}

// Working only from the visible tests produces something that answers
// all of them and still gets composition wrong. That gap is the whole
// measurement.
func TestTheVisibleTestsCanBePassedWithoutBeingRight(t *testing.T) {
	visible, hidden := gradeImplementation(t, "naive.go.txt")
	if visible.score() != 100 {
		t.Fatalf("visible %0.f%%, want 100: %v", visible.score(), visible.Failed)
	}
	if hidden.score() == 100 {
		t.Fatal("the hidden tests found nothing, so the environment measures nothing")
	}
	if len(hidden.Failed) < 2 {
		t.Errorf("only %d hidden tests failed, which is a thin signal: %v", len(hidden.Failed), hidden.Failed)
	}
}

// And an implementation that is right passes both, so the hidden tests
// are asking for correctness rather than for something impossible.
func TestACorrectImplementationPassesBoth(t *testing.T) {
	visible, hidden := gradeImplementation(t, "correct.go.txt")
	if visible.score() != 100 {
		t.Fatalf("visible %0.f%%: %v", visible.score(), visible.Failed)
	}
	if hidden.score() != 100 {
		t.Fatalf("hidden %0.f%%, so a hidden test asks for something the specification does not: %v", hidden.score(), hidden.Failed)
	}
}

// Code that does not build passes nothing. A grader that reported an
// empty run as a perfect score would be worse than no grader.
func TestCodeThatDoesNotBuildScoresNothing(t *testing.T) {
	dir := t.TempDir()
	if err := prepare([]string{"cache", dir}); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "cache.go"), []byte("package cache\nthis is not go\n"), 0644); err != nil {
		t.Fatal(err)
	}
	r, err := runTests(dir, []string{"TestSetAndGet", "TestMissingKey"})
	if err != nil {
		t.Fatal(err)
	}
	if r.score() != 0 {
		t.Fatalf("score %0.f%% for code that does not compile", r.score())
	}
}

// A task nobody can solve measures the person who wrote it. Every task
// ships the change that solves it, and the environment applies that
// change and requires full marks on both suites. If a hidden test asks
// for something the specification does not say, this is what fails.
func TestTheReferenceSolutionPassesEverything(t *testing.T) {
	if testing.Short() {
		t.Skip("copies a repository and runs its whole suite")
	}
	task, err := loadTask("zeroturn-threshold")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := task.repoPath(); err != nil {
		t.Skipf("%v", err)
	}
	dir := t.TempDir()
	if err := prepare([]string{"zeroturn-threshold", dir}); err != nil {
		t.Fatal(err)
	}
	patch, err := filepath.Abs(filepath.Join("..", "..", "tasks", "zeroturn-threshold", "reference.patch"))
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("patch", "-p1", "-s", "-i", patch)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("the reference change no longer applies, so the task has moved under it: %v\n%s", err, out)
	}

	visible, err := runTests(dir, nil)
	if err != nil {
		t.Fatal(err)
	}
	if visible.score() != 100 {
		t.Errorf("the reference solution fails %d visible tests: %v", len(visible.Failed), visible.Failed)
	}
	names, remove, err := task.plant(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer remove()
	hidden, err := runTests(dir, names)
	if err != nil {
		t.Fatal(err)
	}
	if hidden.score() != 100 {
		t.Errorf("a hidden test asks for something the reference solution does not do: %v", hidden.Failed)
	}
}

// And doing nothing has to score badly, or the grader cannot tell an
// untouched workspace from finished work.
func TestAnUntouchedWorkspaceFailsTheAcceptanceTests(t *testing.T) {
	if testing.Short() {
		t.Skip("copies a repository and runs its whole suite")
	}
	task, err := loadTask("zeroturn-threshold")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := task.repoPath(); err != nil {
		t.Skipf("%v", err)
	}
	dir := t.TempDir()
	if err := prepare([]string{"zeroturn-threshold", dir}); err != nil {
		t.Fatal(err)
	}
	names, remove, err := task.plant(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer remove()
	hidden, err := runTests(dir, names)
	if err != nil {
		t.Fatal(err)
	}
	if hidden.score() == 100 {
		t.Fatal("a workspace where nothing was done scored full marks")
	}
	if len(hidden.Failed) < 3 {
		t.Errorf("only %v failed, so the task is not being checked for having been done", hidden.Failed)
	}
}

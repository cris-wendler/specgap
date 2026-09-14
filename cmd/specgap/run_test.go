package main

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// workspace writes a tiny module with the given test file, so the
// runner can be exercised without a task.
func workspace(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte("module w\n\ngo 1.17\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "w_test.go"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return dir
}

// runSuite2 is runSuite with the default runner, which is what most of
// these tests exercise.
func runSuite2(dir string, only []string) (result, error) { return runSuite(dir, "", only) }

func TestScoreOfNothingIsNothing(t *testing.T) {
	if (result{}).score() != 0 {
		t.Fatal("an empty run scored above zero")
	}
}

func TestScoreCountsPassesOutOfTheWhole(t *testing.T) {
	r := result{Passed: []string{"a", "b", "c"}, Failed: []string{"d"}}
	if r.total() != 4 {
		t.Fatalf("total %d", r.total())
	}
	if r.score() != 75 {
		t.Fatalf("score %v, want 75", r.score())
	}
}

func TestRunTestsSeparatesPassesFromFailures(t *testing.T) {
	dir := workspace(t, `package w

import "testing"

func TestOne(t *testing.T)   {}
func TestTwo(t *testing.T)   {}
func TestThree(t *testing.T) { t.Fatal("no") }
`)
	r, err := runSuite(dir, "", nil)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(r.Passed, ",") != "TestOne,TestTwo" {
		t.Errorf("passed %v", r.Passed)
	}
	if strings.Join(r.Failed, ",") != "TestThree" {
		t.Errorf("failed %v", r.Failed)
	}
}

// Asking for named tests must run those and leave the rest out of the
// count, or a hidden score would include tests nobody hid.
func TestRunTestsRunsOnlyWhatItWasAskedFor(t *testing.T) {
	dir := workspace(t, `package w

import "testing"

func TestWanted(t *testing.T)   {}
func TestUnwanted(t *testing.T) { t.Fatal("this must not be counted") }
`)
	r, err := runSuite(dir, "", []string{"TestWanted"})
	if err != nil {
		t.Fatal(err)
	}
	if r.total() != 1 || len(r.Passed) != 1 || r.Passed[0] != "TestWanted" {
		t.Fatalf("passed %v failed %v", r.Passed, r.Failed)
	}
}

// Code that does not compile passes nothing. A grader that reported an
// empty run as a perfect score would be worse than no grader, and this
// is the case that produces an empty run.
func TestCodeThatDoesNotCompileFailsEverythingAskedFor(t *testing.T) {
	dir := workspace(t, "package w\n\nthis is not go\n")
	r, err := runSuite(dir, "", []string{"TestA", "TestB"})
	if err != nil {
		t.Fatal(err)
	}
	if r.score() != 0 || r.total() != 2 {
		t.Fatalf("score %v over %d tests", r.score(), r.total())
	}
}

// A test that panics is a failure, not a test that never reported.
func TestAPanicCountsAsAFailure(t *testing.T) {
	dir := workspace(t, `package w

import "testing"

func TestBoom(t *testing.T) { panic("boom") }
`)
	r, err := runSuite(dir, "", []string{"TestBoom"})
	if err != nil {
		t.Fatal(err)
	}
	if r.score() != 0 {
		t.Fatalf("a panic scored %v", r.score())
	}
}

func TestGradeNeedsATaskAndADirectory(t *testing.T) {
	for _, args := range [][]string{{}, {"cache"}, {"cache", "a", "b"}} {
		if err := grade(args); err == nil {
			t.Errorf("grade(%v) was accepted", args)
		}
	}
}

func TestPrepareNeedsATaskAndADirectory(t *testing.T) {
	for _, args := range [][]string{{}, {"cache"}, {"cache", "a", "b"}} {
		if err := prepare(args); err == nil {
			t.Errorf("prepare(%v) was accepted", args)
		}
	}
}

// The machine readable output is what a harness reads, so its shape is
// a promise rather than a convenience.
func TestJSONOutputCarriesBothScoresAndTheFailures(t *testing.T) {
	os.Setenv("SPECGAP_TASKS", repoRoot(t))
	defer os.Unsetenv("SPECGAP_TASKS")

	dir := t.TempDir()
	if err := prepare([]string{"cache", dir}); err != nil {
		t.Fatal(err)
	}
	src, err := ioutil.ReadFile(filepath.Join(repoRoot(t), "testdata", "naive.go.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if err := ioutil.WriteFile(filepath.Join(dir, "cache.go"), src, 0644); err != nil {
		t.Fatal(err)
	}

	out := captureStdout(t, func() {
		if err := grade([]string{"--json", "cache", dir}); err != nil {
			t.Fatal(err)
		}
	})
	var report struct {
		Task                        string
		VisiblePassed, VisibleTotal int
		HiddenPassed, HiddenTotal   int
		HiddenFailures              []string
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("output is not JSON: %v\n%s", err, out)
	}
	if report.Task != "cache" {
		t.Errorf("task %q", report.Task)
	}
	if report.VisibleTotal == 0 || report.HiddenTotal == 0 {
		t.Errorf("visible %d hidden %d", report.VisibleTotal, report.HiddenTotal)
	}
	if len(report.HiddenFailures) != report.HiddenTotal-report.HiddenPassed {
		t.Errorf("%d failures listed, %d implied", len(report.HiddenFailures), report.HiddenTotal-report.HiddenPassed)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	saved := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = saved
	b, err := ioutil.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

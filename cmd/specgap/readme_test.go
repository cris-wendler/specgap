package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func readme(t *testing.T) string {
	t.Helper()
	b, err := ioutil.ReadFile(filepath.Join(repoRoot(t), "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

// A task nobody reading the README knows about might as well not be
// here. This was added because a task shipped undocumented: the edit
// that was meant to describe it matched nothing and changed nothing,
// and the commit went ahead saying it had.
func TestEveryTaskIsInTheReadme(t *testing.T) {
	withTasks(t)
	all, err := tasks()
	if err != nil {
		t.Fatal(err)
	}
	if len(all) == 0 {
		t.Fatal("no tasks found")
	}
	text := readme(t)
	for _, task := range all {
		if !strings.Contains(text, task.Name) {
			t.Errorf("the README never mentions the task %q", task.Name)
		}
	}
}

// And a task named there has to exist, so a task that is removed or
// renamed does not leave the README describing something that is gone.
func TestTheReadmeNamesNoTaskThatIsGone(t *testing.T) {
	withTasks(t)
	all, err := tasks()
	if err != nil {
		t.Fatal(err)
	}
	present := map[string]bool{}
	for _, task := range all {
		present[task.Name] = true
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot(t), "tasks"))
	if err != nil {
		t.Fatal(err)
	}
	_ = entries
	// Task names appear in the README inside backticks.
	text := readme(t)
	for _, chunk := range strings.Split(text, "`") {
		if !strings.Contains(chunk, "-") && chunk != "cache" {
			continue
		}
		if strings.ContainsAny(chunk, " \n/.") {
			continue
		}
		if looksLikeTaskName(chunk) && !present[chunk] {
			t.Errorf("the README names a task %q that does not exist", chunk)
		}
	}
}

// looksLikeTaskName keeps the check to words shaped like a task rather
// than every quoted word in the document.
func looksLikeTaskName(s string) bool {
	if s == "" || strings.HasPrefix(s, "-") {
		return false
	}
	for _, r := range s {
		if !(r >= 'a' && r <= 'z') && r != '-' {
			return false
		}
	}
	return strings.Contains(s, "-") || s == "cache"
}

// Every command the program answers has to be in the command list, or
// somebody reading it does not know the program can do it.
func TestEveryCommandIsInTheReadme(t *testing.T) {
	text := readme(t)
	for _, command := range []string{"tasks", "run", "prepare", "grade"} {
		if !strings.Contains(text, "specgap "+command) {
			t.Errorf("the README does not show `specgap %s`", command)
		}
	}
}

// The sample run in the README is program output kept by hand beside the
// program. It had already drifted: the document showed three of five
// hidden tests failing with two named, when the task had gained a sixth
// and a third was failing. Nobody reading it would have known, because
// nothing compared the two.
//
// So this runs the task the README quotes and requires the document to
// carry what came out. The timings are left alone, because they are the
// one part that is different every run.
func TestTheSampleRunInTheReadmeIsWhatTheProgramPrints(t *testing.T) {
	withTasks(t)
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}
	// The same implementation the README says the output came from.
	a, err := attempt(task, copyIn(t, "naive.go.txt"), "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if a.Error != "" {
		t.Fatalf("the sample agent did not finish: %s", a.Error)
	}

	text := readme(t)
	lines := []string{
		fmt.Sprintf("visible  %s%%  %d of %d", showPercent(a.visibleScore()), a.VisiblePassed, a.VisibleTotal),
		fmt.Sprintf("hidden   %s%%  %d of %d", showPercent(a.hiddenScore()), a.HiddenPassed, a.HiddenTotal),
		"work     " + a.Work.describe(),
	}
	for _, line := range lines {
		if !strings.Contains(text, line) {
			t.Errorf("the README does not show %q, which is what the run prints", line)
		}
	}
	if len(a.HiddenFailures) == 0 {
		t.Fatal("the sample implementation passed everything, so the README sample is not this run")
	}
	for _, name := range a.HiddenFailures {
		if !strings.Contains(text, name) {
			t.Errorf("the README does not name %s among the tests the agent never saw", name)
		}
	}
}

package main

import (
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

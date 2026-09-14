// Command specgap poses a programming task to a coding agent and grades
// the result against tests the agent never saw.
//
// The visible tests are the only description of the job the agent has,
// and an agent works to the description it is given. The hidden tests
// introduce no new feature: each asks what happens where two things that
// were described separately have to hold at once. The gap between the
// two scores is what this measures.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const usage = `specgap tasks                    list the tasks
specgap prepare <task> <dir>     write a workspace for an agent to work in
specgap grade <task> <dir>       run both suites and report the gap
specgap run <task> --agent ...   prepare, run an agent, and grade, any number of times

The workspace holds everything the agent may see. The hidden tests are
planted only to grade, and removed again afterwards.
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(2)
	}
	var err error
	switch os.Args[1] {
	case "tasks":
		err = list()
	case "prepare":
		err = prepare(os.Args[2:])
	case "grade":
		err = grade(os.Args[2:])
	case "run":
		err = run(os.Args[2:])
	case "-h", "--help", "help":
		fmt.Print(usage)
		return
	default:
		fmt.Fprintf(os.Stderr, "specgap has no command named %s\n", os.Args[1])
		os.Exit(2)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// root finds the directory holding the tasks, by walking up from where
// the command was started, so it works from the repository and from a
// package inside it. SPECGAP_TASKS says where they are for anyone
// running the command from somewhere else.
func root() (string, error) {
	if env := os.Getenv("SPECGAP_TASKS"); env != "" {
		if _, err := os.Stat(filepath.Join(env, "tasks")); err != nil {
			return "", fmt.Errorf("SPECGAP_TASKS is %s, which has no tasks directory", env)
		}
		return env, nil
	}
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "tasks")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("no tasks directory above %s: run specgap from the repository", dir)
		}
		dir = parent
	}
}

func list() error {
	all, err := tasks()
	if err != nil {
		return err
	}
	for _, t := range all {
		fmt.Println(t)
	}
	return nil
}

func prepare(args []string) error {
	if len(args) != 2 {
		return fmt.Errorf("prepare needs a task and a directory")
	}
	t, err := loadTask(args[0])
	if err != nil {
		return err
	}
	if err := t.write(args[1]); err != nil {
		return err
	}
	fmt.Printf("workspace %s\n", args[1])
	fmt.Printf("task      %s\n", t.Name)
	if t.Kind == "repo" {
		// The repository is named relative to this one, so it has to be
		// resolved before anything is looked up inside it.
		repo, err := t.repoPath()
		if err != nil {
			return err
		}
		var removed []string
		for _, path := range t.Hidden {
			if _, err := os.Stat(filepath.Join(repo, path)); err == nil {
				removed = append(removed, path)
			}
		}
		sort.Strings(removed)
		if len(removed) > 0 {
			fmt.Printf("removed   %s\n", strings.Join(removed, ", "))
		}
	}
	return nil
}

type result struct {
	Passed []string
	Failed []string
}

func (r result) total() int { return len(r.Passed) + len(r.Failed) }
func (r result) score() float64 {
	if r.total() == 0 {
		return 0
	}
	return 100 * float64(len(r.Passed)) / float64(r.total())
}

// runTests runs the suite in dir and reports which tests passed. A
// package that does not build counts every test asked for as failed,
// because code that does not compile passes nothing.
func runTests(dir string, only []string) (result, error) {
	args := []string{"test", "-json", "-count=1", "./..."}
	if len(only) > 0 {
		args = append(args, "-run", "^("+strings.Join(only, "|")+")$")
	}
	cmd := exec.Command("go", args...)
	cmd.Dir = dir
	out, _ := cmd.Output()

	var r result
	seen := map[string]string{}
	for _, line := range strings.Split(string(out), "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var ev struct{ Action, Test string }
		if json.Unmarshal([]byte(line), &ev) != nil || ev.Test == "" {
			continue
		}
		switch ev.Action {
		case "pass", "fail":
			seen[ev.Test] = ev.Action
		}
	}
	for name, action := range seen {
		if action == "pass" {
			r.Passed = append(r.Passed, name)
		} else {
			r.Failed = append(r.Failed, name)
		}
	}
	for _, name := range only {
		if _, ran := seen[name]; !ran {
			r.Failed = append(r.Failed, name)
		}
	}
	sort.Strings(r.Passed)
	sort.Strings(r.Failed)
	return r, nil
}

var testName = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]*)\(`)

func grade(args []string) error {
	fs := flag.NewFlagSet("grade", flag.ContinueOnError)
	asJSON := fs.Bool("json", false, "print machine readable output")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 2 {
		return fmt.Errorf("grade needs a task and a directory")
	}
	t, err := loadTask(fs.Arg(0))
	if err != nil {
		return err
	}
	dir := fs.Arg(1)

	visible, err := runTests(dir, nil)
	if err != nil {
		return err
	}
	names, remove, err := t.plant(dir)
	if err != nil {
		return err
	}
	defer remove()
	hidden, err := runTests(dir, names)
	if err != nil {
		return err
	}

	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(struct {
			Task                        string
			VisiblePassed, VisibleTotal int
			HiddenPassed, HiddenTotal   int
			HiddenFailures              []string
		}{t.Name, len(visible.Passed), visible.total(),
			len(hidden.Passed), hidden.total(), hidden.Failed})
	}

	fmt.Println("SPECGAP")
	fmt.Printf("task     %s\n", t.Name)
	fmt.Printf("visible  %3.0f%%  %d of %d   the tests the agent could see\n",
		visible.score(), len(visible.Passed), visible.total())
	fmt.Printf("hidden   %3.0f%%  %d of %d   the tests it could not\n",
		hidden.score(), len(hidden.Passed), hidden.total())
	fmt.Printf("gap      %3.0f points\n", visible.score()-hidden.score())
	if len(hidden.Failed) > 0 {
		fmt.Println("\nfailed on what it never saw:")
		for _, name := range hidden.Failed {
			fmt.Printf("  %s\n", name)
		}
	}
	return nil
}

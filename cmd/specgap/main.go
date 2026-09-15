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
		var removed, gone []string
		for _, path := range t.Hidden {
			if _, err := os.Stat(filepath.Join(repo, path)); err == nil {
				removed = append(removed, path)
			}
		}
		for _, path := range t.Remove {
			if _, err := os.Stat(filepath.Join(repo, path)); err == nil {
				gone = append(gone, path)
			}
		}
		sort.Strings(removed)
		sort.Strings(gone)
		if len(removed) > 0 {
			fmt.Printf("hidden    %s\n", strings.Join(removed, ", "))
		}
		if len(gone) > 0 {
			fmt.Printf("removed   %s, and not restored\n", strings.Join(gone, ", "))
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

// namePattern finds the tests a hidden file defines, which differs by
// language: Go names them with a function prefix, Python with one too,
// and both are read from the source rather than from the tool.
func namePattern(runner string) *regexp.Regexp {
	if runner == "pytest" {
		return pytestName
	}
	return goName
}

var (
	goName     = regexp.MustCompile(`(?m)^func (Test[A-Za-z0-9_]*)\(`)
	pytestName = regexp.MustCompile(`(?m)^def (test_[A-Za-z0-9_]*)\(`)
)

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

	visible, err := runSuite(dir, t.Runner, nil)
	if err != nil {
		return err
	}
	names, remove, err := t.plant(dir)
	if err != nil {
		return err
	}
	defer remove()
	hidden, err := runSuite(dir, t.Runner, names)
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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Attempt is one agent's work on one task, scored.
type Attempt struct {
	Task           string    `json:"task"`
	Agent          string    `json:"agent"`
	StartedAt      time.Time `json:"startedAt"`
	Seconds        float64   `json:"seconds"`
	VisiblePassed  int       `json:"visiblePassed"`
	VisibleTotal   int       `json:"visibleTotal"`
	HiddenPassed   int       `json:"hiddenPassed"`
	HiddenTotal    int       `json:"hiddenTotal"`
	HiddenFailures []string  `json:"hiddenFailures,omitempty"`
	Error          string    `json:"error,omitempty"`
}

func (a Attempt) visibleScore() float64 { return percent(a.VisiblePassed, a.VisibleTotal) }
func (a Attempt) hiddenScore() float64  { return percent(a.HiddenPassed, a.HiddenTotal) }

func percent(part, whole int) float64 {
	if whole == 0 {
		return 0
	}
	return 100 * float64(part) / float64(whole)
}

const runUsage = `specgap run <task> [flags]

  --agent <command>   the agent to run in the workspace, as a shell command
  --runs <n>          how many attempts, default 1
  --keep <dir>        keep the workspaces under this directory
  --results <file>    append each attempt to this file as JSON
  --json              print the attempts as JSON rather than a report

The agent runs with the workspace as its working directory and is given
no arguments. One attempt is one fresh workspace, so attempts cannot see
each other's work.
`

func run(args []string) error {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	agent := fs.String("agent", "", "the agent to run, as a shell command")
	runs := fs.Int("runs", 1, "how many attempts")
	keep := fs.String("keep", "", "keep the workspaces under this directory")
	results := fs.String("results", "", "append each attempt to this file as JSON")
	asJSON := fs.Bool("json", false, "print the attempts as JSON")
	fs.Usage = func() { fmt.Print(runUsage) }
	// The task is named before the flags, because the flag package stops
	// reading at the first argument that is not one.
	name := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	if err := fs.Parse(args); err != nil {
		return err
	}
	if name == "" || fs.NArg() != 0 {
		return fmt.Errorf("run needs one task: specgap run <task> --agent <command>")
	}
	if *agent == "" {
		return fmt.Errorf("run needs --agent, the command that does the work in the workspace")
	}
	if *runs < 1 {
		return fmt.Errorf("--runs must be at least 1")
	}
	task, err := loadTask(name)
	if err != nil {
		return err
	}

	var attempts []Attempt
	for i := 0; i < *runs; i++ {
		a, err := attempt(task, *agent, *keep, i)
		if err != nil {
			return err
		}
		attempts = append(attempts, a)
		if *results != "" {
			if err := appendResult(*results, a); err != nil {
				return err
			}
		}
		if !*asJSON {
			reportAttempt(a, i+1, *runs)
		}
	}
	if *asJSON {
		return json.NewEncoder(os.Stdout).Encode(attempts)
	}
	if len(attempts) > 1 {
		reportSpread(attempts)
	}
	return nil
}

// attempt prepares a workspace nothing has seen, lets the agent work in
// it, and grades what it left behind.
func attempt(task Task, agent, keep string, n int) (Attempt, error) {
	dir := ""
	if keep != "" {
		dir = filepath.Join(keep, fmt.Sprintf("%s-%d", task.Name, n+1))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return Attempt{}, err
		}
	} else {
		tmp, err := ioutil.TempDir("", "specgap-")
		if err != nil {
			return Attempt{}, err
		}
		defer os.RemoveAll(tmp)
		dir = tmp
	}
	if err := task.write(dir); err != nil {
		return Attempt{}, err
	}

	a := Attempt{Task: task.Name, Agent: agent, StartedAt: time.Now().UTC()}
	started := time.Now()
	cmd := exec.Command("sh", "-c", agent)
	cmd.Dir = dir
	cmd.Stdout = ioutil.Discard
	cmd.Stderr = ioutil.Discard
	if err := cmd.Run(); err != nil {
		// An agent that gave up still left work behind, so it is graded
		// rather than thrown away. What it exited with is recorded.
		a.Error = err.Error()
	}
	a.Seconds = time.Since(started).Seconds()

	visible, err := runTests(dir, nil)
	if err != nil {
		return a, err
	}
	a.VisiblePassed, a.VisibleTotal = len(visible.Passed), visible.total()

	names, remove, err := task.plant(dir)
	if err != nil {
		return a, err
	}
	hidden, err := runTests(dir, names)
	remove()
	if err != nil {
		return a, err
	}
	a.HiddenPassed, a.HiddenTotal = len(hidden.Passed), hidden.total()
	a.HiddenFailures = hidden.Failed
	return a, nil
}

func appendResult(path string, a Attempt) error {
	b, err := json.Marshal(a)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = f.Write(append(b, '\n'))
	return err
}

func reportAttempt(a Attempt, n, of int) {
	if of > 1 {
		fmt.Printf("\nattempt %d of %d\n", n, of)
	} else {
		fmt.Println("SPECGAP")
		fmt.Printf("task     %s\n", a.Task)
	}
	if a.Error != "" {
		fmt.Printf("agent    %s\n", a.Error)
	}
	fmt.Printf("visible  %3.0f%%  %d of %d   the tests the agent could see\n",
		a.visibleScore(), a.VisiblePassed, a.VisibleTotal)
	fmt.Printf("hidden   %3.0f%%  %d of %d   the tests it could not\n",
		a.hiddenScore(), a.HiddenPassed, a.HiddenTotal)
	fmt.Printf("gap      %3.0f points  in %.0fs\n", a.visibleScore()-a.hiddenScore(), a.Seconds)
	if len(a.HiddenFailures) > 0 {
		fmt.Println("failed on what it never saw:")
		for _, name := range a.HiddenFailures {
			fmt.Printf("  %s\n", name)
		}
	}
}

// reportSpread says what several attempts agreed on. One attempt is an
// anecdote: a task an agent fails once in five is a different finding
// from one it fails every time, and the list of what failed says which.
func reportSpread(attempts []Attempt) {
	var hidden []float64
	counts := map[string]int{}
	for _, a := range attempts {
		hidden = append(hidden, a.hiddenScore())
		for _, name := range a.HiddenFailures {
			counts[name]++
		}
	}
	sort.Float64s(hidden)
	fmt.Printf("\n%d attempts\n", len(attempts))
	fmt.Printf("hidden   lowest %.0f%%  median %.0f%%  highest %.0f%%\n",
		hidden[0], hidden[len(hidden)/2], hidden[len(hidden)-1])
	if len(counts) == 0 {
		fmt.Println("no hidden test failed in any attempt")
		return
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		if counts[names[i]] != counts[names[j]] {
			return counts[names[i]] > counts[names[j]]
		}
		return names[i] < names[j]
	})
	fmt.Println("failed on what it never saw:")
	for _, name := range names {
		fmt.Printf("  %d of %d   %s\n", counts[name], len(attempts), name)
	}
}

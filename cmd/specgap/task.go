package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

// Task describes one challenge. A files task writes a small workspace
// from templates. A repo task copies a real repository at a commit and
// removes the tests that will grade the work, which is the only way to
// pose a problem whose answer is not already in a model's training data.
type Task struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Summary string `json:"summary"`
	// Runner names the test tool this task needs, go or pytest. Empty
	// means go, so the tasks written before this stay as they were.
	Runner string `json:"runner,omitempty"`

	// files tasks
	GoMod string `json:"gomod"`

	// repo tasks
	Repo   string `json:"repo"`
	Commit string `json:"commit"`

	// Visible maps a file in the task directory to its place in the
	// workspace. Hidden does the same for the tests the agent never sees,
	// and for a repo task names paths to remove from the copy.
	Visible map[string]string `json:"visible"`
	Hidden  map[string]string `json:"hidden"`

	files taskFiles
	name  string
}

func loadTask(name string) (Task, error) {
	files := openTasks()
	b, err := files.read("tasks", name, "task.json")
	if err != nil {
		return Task{}, fmt.Errorf("no task named %s in %s", name, files.where())
	}
	var t Task
	if err := json.Unmarshal(b, &t); err != nil {
		return Task{}, err
	}
	t.files = files
	t.name = name
	return t, nil
}

func tasks() ([]Task, error) {
	names, err := openTasks().names()
	if err != nil {
		return nil, err
	}
	var out []Task
	for _, name := range names {
		t, err := loadTask(name)
		if err != nil {
			continue
		}
		out = append(out, t)
	}
	return out, nil
}

func (t Task) write(dir string) error {
	switch t.Kind {
	case "files":
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		for from, to := range t.Visible {
			b, err := t.files.read("tasks", t.name, from)
			if err != nil {
				return err
			}
			if err := ioutil.WriteFile(filepath.Join(dir, to), b, 0644); err != nil {
				return err
			}
		}
		if t.GoMod != "" {
			return ioutil.WriteFile(filepath.Join(dir, "go.mod"), []byte(t.GoMod), 0644)
		}
		return nil
	case "repo":
		return t.writeRepo(dir)
	}
	return fmt.Errorf("task %s has no kind I understand", t.Name)
}

// writeRepo copies the tracked files of a repository at one commit, so
// the workspace holds what a person would have checked out and nothing
// else, then removes the tests that grade the work.
func (t Task) writeRepo(dir string) error {
	repo, err := t.repoPath()
	if err != nil {
		return err
	}
	t.Repo = repo
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	commit := t.Commit
	if commit == "" {
		commit = "HEAD"
	}
	archive := exec.Command("git", "-C", t.Repo, "archive", commit)
	extract := exec.Command("tar", "-x", "-C", dir)
	pipe, err := archive.StdoutPipe()
	if err != nil {
		return err
	}
	extract.Stdin = pipe
	if err := extract.Start(); err != nil {
		return err
	}
	if err := archive.Run(); err != nil {
		return fmt.Errorf("could not read %s at %s: %v", t.Repo, commit, err)
	}
	if err := extract.Wait(); err != nil {
		return err
	}
	// A hidden test is removed by where it sits in the workspace. One
	// the repository already has is taken out of the copy; one the task
	// ships was never in it, so there is nothing to remove.
	for _, path := range t.Hidden {
		if err := os.Remove(filepath.Join(dir, path)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for from, to := range t.Visible {
		b, err := t.files.read("tasks", t.name, from)
		if err != nil {
			return err
		}
		if err := ioutil.WriteFile(filepath.Join(dir, to), b, 0644); err != nil {
			return err
		}
	}
	return nil
}

// repoPath resolves the repository a task is cut from. SPECGAP_REPO
// names it outright. A relative name is looked for beside the checkout
// when there is one, and otherwise beside the working directory, which
// is where somebody running an installed executable keeps their clones.
func (t Task) repoPath() (string, error) {
	if env := os.Getenv("SPECGAP_REPO"); env != "" {
		if _, err := os.Stat(env); err != nil {
			return "", fmt.Errorf("SPECGAP_REPO is %s, which is not there", env)
		}
		return env, nil
	}
	if filepath.IsAbs(t.Repo) {
		if _, err := os.Stat(t.Repo); err != nil {
			return "", fmt.Errorf("this task is cut from %s, which is not there", t.Repo)
		}
		return t.Repo, nil
	}
	var tried []string
	var bases []string
	if dir, err := diskTasks(); err == nil {
		bases = append(bases, dir)
	}
	if wd, err := os.Getwd(); err == nil {
		bases = append(bases, wd)
	}
	for _, base := range bases {
		candidate := filepath.Join(base, t.Repo)
		if _, err := os.Stat(candidate); err == nil {
			return candidate, nil
		}
		tried = append(tried, candidate)
	}
	return "", fmt.Errorf("this task is cut from %s, which was not found at %s. "+
		"Clone it beside this one, or set SPECGAP_REPO to where it is",
		t.Repo, strings.Join(tried, " or "))
}

// plant puts the hidden tests into the workspace and reports the test
// names they define, so grading runs those and nothing else.
func (t Task) plant(dir string) (names []string, remove func(), err error) {
	var written []string
	for from, to := range t.Hidden {
		var b []byte
		// A hidden test is either a file the repository already has,
		// which is where the strongest ones come from, or one the task
		// ships to say whether the work was done at all.
		b, err = t.files.read("tasks", t.name, from)
		if err != nil && t.Kind == "repo" {
			var repo string
			repo, err = t.repoPath()
			if err != nil {
				return nil, func() {}, err
			}
			b, err = ioutil.ReadFile(filepath.Join(repo, from))
		}
		if err != nil {
			return nil, func() {}, err
		}
		target := filepath.Join(dir, to)
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return nil, func() {}, err
		}
		if err := ioutil.WriteFile(target, b, 0644); err != nil {
			return nil, func() {}, err
		}
		written = append(written, target)
		for _, m := range namePattern(t.Runner).FindAllStringSubmatch(string(b), -1) {
			// TestMain is the package's entry point rather than a test.
			// Grading on it measures whether the harness started, which
			// is not what the agent was asked to do.
			if m[1] == "TestMain" {
				continue
			}
			names = append(names, m[1])
		}
	}
	sort.Strings(names)
	return names, func() {
		for _, p := range written {
			os.Remove(p)
		}
	}, nil
}

func (t Task) String() string {
	return fmt.Sprintf("%-20s %s", t.Name, strings.TrimSpace(t.Summary))
}

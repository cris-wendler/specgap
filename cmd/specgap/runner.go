package main

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strings"
)

// A runner knows how to ask one language's test tool to run a suite and
// how to read which tests passed. A task names the one it needs, so the
// environment is not tied to the language it was first written in.
type runner interface {
	// command builds the invocation, limited to the named tests when any
	// are given.
	command(only []string) []string
	// read turns whatever the tool printed into passes and failures.
	read(output string) result
}

func runnerFor(name string) (runner, error) {
	switch name {
	case "", "go":
		return goRunner{}, nil
	case "pytest":
		return pytestRunner{}, nil
	}
	return nil, fmt.Errorf("no runner named %s: this build knows go and pytest", name)
}

// goRunner reads the event stream go test writes with -json.
type goRunner struct{}

func (goRunner) command(only []string) []string {
	args := []string{"go", "test", "-json", "-count=1", "./..."}
	if len(only) > 0 {
		args = append(args, "-run", "^("+strings.Join(only, "|")+")$")
	}
	return args
}

func (goRunner) read(output string) result {
	seen := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(line, "{") {
			continue
		}
		var ev struct{ Action, Test string }
		if json.Unmarshal([]byte(line), &ev) != nil || ev.Test == "" {
			continue
		}
		if ev.Action == "pass" || ev.Action == "fail" {
			seen[ev.Test] = ev.Action
		}
	}
	return collect(seen)
}

// pytestRunner reads the lines pytest writes with -v, which name one
// test and its outcome. It is the only report pytest gives without a
// plugin, and a task should not need one installed to be graded.
type pytestRunner struct{}

func (pytestRunner) command(only []string) []string {
	args := []string{"python3", "-m", "pytest", "-v", "-p", "no:cacheprovider"}
	if len(only) > 0 {
		args = append(args, "-k", strings.Join(only, " or "))
	}
	return args
}

func (pytestRunner) read(output string) result {
	seen := map[string]string{}
	for _, line := range strings.Split(output, "\n") {
		// tests/test_cache.py::test_set_and_get PASSED       [ 12%]
		i := strings.Index(line, "::")
		if i < 0 {
			continue
		}
		rest := strings.TrimSpace(line[i+2:])
		name := rest
		if j := strings.IndexAny(rest, " \t"); j > 0 {
			name = rest[:j]
		}
		switch {
		case strings.Contains(rest, "PASSED"):
			seen[name] = "pass"
		case strings.Contains(rest, "FAILED"), strings.Contains(rest, "ERROR"):
			seen[name] = "fail"
		}
	}
	return collect(seen)
}

func collect(seen map[string]string) result {
	var r result
	for name, action := range seen {
		if action == "pass" {
			r.Passed = append(r.Passed, name)
		} else {
			r.Failed = append(r.Failed, name)
		}
	}
	sort.Strings(r.Passed)
	sort.Strings(r.Failed)
	return r
}

// runSuite runs the tests in dir and reports which passed. A suite that
// could not build or start counts every test asked for as failed,
// because code that does not run passes nothing.
func runSuite(dir string, kind string, only []string) (result, error) {
	run, err := runnerFor(kind)
	if err != nil {
		return result{}, err
	}
	args := run.command(only)
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir
	out, _ := cmd.CombinedOutput()
	r := run.read(string(out))

	ran := map[string]bool{}
	for _, n := range r.Passed {
		ran[n] = true
	}
	for _, n := range r.Failed {
		ran[n] = true
	}
	for _, name := range only {
		if !ran[name] {
			r.Failed = append(r.Failed, name)
		}
	}
	sort.Strings(r.Failed)
	return r, nil
}

package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// An agent that never returns used to stop the whole run. There was no
// limit anywhere in the environment, and nothing afterwards said what
// had happened: the attempt simply never finished. This is a harness for
// running commands it does not control, several times over, so that is
// the one outcome it cannot afford.
func TestAnAgentThatNeverFinishesIsGivenUpOn(t *testing.T) {
	withTasks(t)
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}

	started := time.Now()
	a, err := attempt(task, "sleep 600", "", 0, 300*time.Millisecond)
	if err != nil {
		t.Fatal(err)
	}
	took := time.Since(started)

	if !a.TimedOut {
		t.Error("the agent was killed and the attempt does not say so")
	}
	if took > 30*time.Second {
		t.Errorf("the limit was 300ms and the attempt took %s", took)
	}
	if !strings.Contains(a.Error, "did not finish") {
		t.Errorf("the error does not say what happened: %q", a.Error)
	}
	// It is still graded, because an agent that ran out of time left
	// whatever it had done behind, the same as one that gave up.
	if a.VisibleTotal == 0 {
		t.Error("nothing was graded, so a timed out attempt reports no score at all")
	}
}

// The agent runs through sh, so what it starts are grandchildren of this
// process. Killing the shell alone leaves them running, and they hold
// the write end of the pipe this process reads, so waiting for the agent
// would go on waiting after the agent was dead.
func TestWhatTheAgentStartedIsKilledToo(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("the agent is run through sh, and process groups differ here")
	}
	withTasks(t)
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}

	keep := t.TempDir()
	// The shell starts a background child that outlives it and writes a
	// file after the limit has passed. If the group was killed, that file
	// never appears.
	marker := filepath.Join(keep, "grandchild-survived")
	agent := "sh -c 'sleep 3; printf x > " + marker + "' & sleep 600"

	// The attempt is run off to the side with a deadline of its own.
	// Without the process group kill this does not fail, it HANGS: the
	// grandchild holds the write end of the pipe the attempt is reading,
	// so waiting for the agent goes on after the agent is dead. A test
	// that hangs stops continuous integration instead of reporting, so
	// the wait is bounded here rather than left to the suite timeout.
	type result struct {
		a   Attempt
		err error
	}
	done := make(chan result, 1)
	go func() {
		a, err := attempt(task, agent, "", 0, 300*time.Millisecond)
		done <- result{a, err}
	}()

	var a Attempt
	select {
	case r := <-done:
		if r.err != nil {
			t.Fatal(r.err)
		}
		a = r.a
	case <-time.After(60 * time.Second):
		t.Fatal("the attempt never returned: something the agent started is still holding it open")
	}
	if !a.TimedOut {
		t.Fatal("the attempt did not time out, so this test checks nothing")
	}

	// Well past when the grandchild would have written, had it lived.
	time.Sleep(4 * time.Second)
	if _, err := os.Stat(marker); err == nil {
		t.Error("something the agent started outlived the attempt that was given up on")
	}
}

// Nothing should change for an agent that finishes, and a limit of zero
// has to mean no limit rather than no time at all.
func TestAnAgentThatFinishesIsNotAffected(t *testing.T) {
	withTasks(t)
	task, err := loadTask("cache")
	if err != nil {
		t.Fatal(err)
	}

	for _, limit := range []time.Duration{0, time.Minute} {
		a, err := attempt(task, "true", "", 0, limit)
		if err != nil {
			t.Fatal(err)
		}
		if a.TimedOut {
			t.Errorf("limit %s: an agent that returned at once was called timed out", limit)
		}
		if a.Error != "" {
			t.Errorf("limit %s: %q", limit, a.Error)
		}
	}
}

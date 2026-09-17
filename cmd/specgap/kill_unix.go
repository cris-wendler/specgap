//go:build !windows
// +build !windows

package main

import (
	"os/exec"
	"syscall"
)

// ownGroup puts the agent in a process group of its own.
//
// The agent is run through sh, so what it starts are grandchildren of
// this process. Killing the shell alone leaves them running, and they
// still hold the write end of the pipe this process reads, so waiting
// for the agent would go on waiting after the agent is dead.
func ownGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killGroup kills the agent and everything it started. A negative pid
// addresses the whole process group.
func killGroup(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
}

//go:build windows
// +build windows

package main

import "os/exec"

// Windows has no process groups of this shape, and the agent is run
// through sh, which is not native there either. The direct child is
// killed and anything it started is left, which is the honest limit
// rather than a pretence that this works the same way.
func ownGroup(cmd *exec.Cmd) {}

func killGroup(cmd *exec.Cmd) {
	if cmd.Process != nil {
		cmd.Process.Kill()
	}
}

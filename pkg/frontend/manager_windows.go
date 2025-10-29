//go:build windows

package frontend

import (
	"os/exec"
)

// setupProcessGroup is a no-op on Windows
func setupProcessGroup(cmd *exec.Cmd) {
	// Windows doesn't use process groups the same way
}

// stopProcess stops the process on Windows
func stopProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}
	return cmd.Process.Kill()
}

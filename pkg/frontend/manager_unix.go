//go:build !windows

package frontend

import (
	"errors"
	"os/exec"
	"syscall"
	"time"
)

// setupProcessGroup configures the command to run in its own process group
func setupProcessGroup(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// stopProcess stops the process and its children gracefully on Unix systems
func stopProcess(cmd *exec.Cmd) error {
	if cmd == nil || cmd.Process == nil {
		return nil
	}

	pgid, pgErr := syscall.Getpgid(cmd.Process.Pid)
	if pgErr != nil {
		return cmd.Process.Kill()
	}

	// Attempt graceful shutdown with SIGTERM
	termErr := syscall.Kill(-pgid, syscall.SIGTERM)
	if termErr != nil && !errors.Is(termErr, syscall.ESRCH) {
		return termErr
	}

	// Give process a brief moment to exit before forcing
	time.Sleep(250 * time.Millisecond)

	// Force kill with SIGKILL
	killErr := syscall.Kill(-pgid, syscall.SIGKILL)
	if killErr != nil && !errors.Is(killErr, syscall.ESRCH) {
		return killErr
	}

	return nil
}

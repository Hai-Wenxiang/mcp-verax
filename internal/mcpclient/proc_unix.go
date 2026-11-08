//go:build !windows

package mcpclient

import (
	"fmt"
	"os/exec"
	"syscall"
)

// setSysProcAttr puts the child in its own process group so the whole tree
// can be signalled at once.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// killTree kills the process group (negative PID).
func killTree(pid int) error {
	if err := syscall.Kill(-pid, syscall.SIGKILL); err != nil {
		return fmt.Errorf("kill pgid %d: %w", pid, err)
	}
	return nil
}

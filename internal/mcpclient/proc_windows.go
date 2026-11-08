//go:build windows

package mcpclient

import (
	"fmt"
	"os/exec"
	"syscall"
)

// setSysProcAttr reserves the child in a job-like containment; Windows has no
// process groups, tree kill goes through taskkill /T.
func setSysProcAttr(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}

// killTree kills pid and all its descendants via taskkill.
func killTree(pid int) error {
	out, err := exec.Command("taskkill", "/T", "/F", "/PID", fmt.Sprint(pid)).CombinedOutput()
	if err != nil {
		return fmt.Errorf("taskkill %d: %v: %s", pid, err, out)
	}
	return nil
}

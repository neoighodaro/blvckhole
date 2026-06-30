package update

import (
	"os/exec"
	"syscall"
)

// Spawn launches a detached "<executable> __update-check" process that runs in
// the background and outlives the current process. Output is discarded; the
// child is placed in its own process group so it is not killed with the parent.
func Spawn(executable string) error {
	cmd := exec.Command(executable, "__update-check")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Release so we never wait on or reap the child.
	return cmd.Process.Release()
}

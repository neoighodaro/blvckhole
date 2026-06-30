package update

import (
	"os/exec"
	"syscall"
)

// Spawn launches a detached "<executable> __update-check" process that runs in
// the background and outlives the current process. Output is discarded. Setsid
// creates a new session so the child is fully detached from the parent's
// controlling terminal; a terminal hangup after the parent exits cannot deliver
// SIGHUP to the child. On Linux and Darwin, setsid() also creates a new process
// group, so Setpgid is not set (combining them is redundant and can error).
func Spawn(executable string) error {
	cmd := exec.Command(executable, "__update-check")
	cmd.Stdin = nil
	cmd.Stdout = nil
	cmd.Stderr = nil
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		return err
	}
	// Release so we never wait on or reap the child.
	return cmd.Process.Release()
}

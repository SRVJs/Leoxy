//go:build unix

package core

import (
	"errors"
	"syscall"
)

func RequestStop(pid int) error {
	return syscall.Kill(pid, syscall.SIGTERM)
}

func ForceStop(pid int) error {
	return syscall.Kill(pid, syscall.SIGKILL)
}

func ProcessRunning(pid int) bool {
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	return errors.Is(err, syscall.EPERM)
}

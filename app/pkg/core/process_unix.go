//go:build unix

package core

import (
	"errors"
	"fmt"
	"os"
	"strings"
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

func ProcessMatches(pid int, name string) bool {
	data, err := os.ReadFile(fmt.Sprintf("/proc/%d/cmdline", pid))
	if err != nil {
		return true
	}
	return strings.Contains(string(data), name)
}

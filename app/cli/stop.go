package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

var stopCmd = &cobra.Command{
	Use:     "stop",
	Short:   "Stop the server",
	Long:    "Stop the background Leoxy process started with the start command.",
	Example: "  leoxy stop",
	RunE: func(cmd *cobra.Command, args []string) error {
		data, err := os.ReadFile(pidFile)
		if err != nil {
			return fmt.Errorf("server is not running")
		}

		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err != nil {
			return fmt.Errorf("invalid PID file")
		}
		if pid <= 1 || pid == os.Getpid() {
			return fmt.Errorf("invalid PID file")
		}

		if runtime.GOOS == "windows" {
			err = exec.Command("taskkill", "/PID", strconv.Itoa(pid), "/T", "/F").Run()
		} else {
			process, findErr := os.FindProcess(pid)
			if findErr != nil {
				err = findErr
			} else {
				err = process.Kill()
			}
		}

		if err != nil {
			_ = os.Remove(pidFile)
			return fmt.Errorf("could not stop server: %w", err)
		}

		_ = os.Remove(pidFile)

		fmt.Printf("Leoxy stopped with PID %d\n", pid)
		return nil
	},
}

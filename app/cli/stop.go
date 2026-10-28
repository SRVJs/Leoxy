package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	platform "github.com/elrefai99/Leoxy/app/pkg/core"

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

		if !platform.ProcessRunning(pid) {
			_ = os.Remove(pidFile)
			return fmt.Errorf("server is not running")
		}

		if err := platform.RequestStop(pid); err != nil {
			if forceErr := platform.ForceStop(pid); forceErr != nil {
				_ = os.Remove(pidFile)
				return fmt.Errorf("could not stop server: %w", forceErr)
			}
		}

		for i := 0; i < 20; i++ {
			if !platform.ProcessRunning(pid) {
				break
			}
			time.Sleep(100 * time.Millisecond)
		}

		if platform.ProcessRunning(pid) {
			if err := platform.ForceStop(pid); err != nil {
				_ = os.Remove(pidFile)
				return fmt.Errorf("could not stop server: %w", err)
			}
		}

		_ = os.Remove(pidFile)

		fmt.Printf("Leoxy stopped with PID %d\n", pid)
		return nil
	},
}

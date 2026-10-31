package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	platform "github.com/elrefai99/Leoxy/app/pkg/core"

	"github.com/spf13/cobra"
)

const (
	pidFile = "leoxy/keys/leoxy.pid"
	pidLogs = "leoxy/log"
	pidKeys = "leoxy/keys"
)

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start the server in the background",
	Long:    "Start Leoxy in the background. Server output is appended to leoxy/log/leoxy.log, and the process ID is stored under leoxy/keys/.",
	Example: "  leoxy start",
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, dir := range []string{pidKeys, pidLogs} {
			if err := os.MkdirAll(dir, 0700); err != nil {
				return err
			}
			if err := os.Chmod(dir, 0700); err != nil {
				return err
			}
		}

		if data, err := os.ReadFile(pidFile); err == nil {
			if pid, err := strconv.Atoi(strings.TrimSpace(string(data))); err == nil &&
				platform.ProcessRunning(pid) && platform.ProcessMatches(pid, "leoxy") {
				return fmt.Errorf("server is already running with PID %d", pid)
			}
		}

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		runArgs := []string{"run"}
		if configFile != "" {
			runArgs = append(runArgs, "--config", configFile)
		}
		if dataDir != "" {
			runArgs = append(runArgs, "--data-dir", dataDir)
		}
		process := exec.Command(executable, runArgs...)

		logFile, err := os.OpenFile("leoxy/log/leoxy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer func() {
			if err := logFile.Close(); err != nil {
				fmt.Fprintf(os.Stderr, "close log file: %v\n", err)
			}
		}()

		process.Stdout = logFile
		process.Stderr = logFile
		detach(process)

		if err := process.Start(); err != nil {
			return err
		}

		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(process.Process.Pid)), 0600); err != nil {
			_ = process.Process.Kill()
			return err
		}
		_ = process.Process.Release()

		fmt.Println("Leoxy started proxy")
		return nil
	},
}

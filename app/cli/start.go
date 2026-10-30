package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"sync"

	"github.com/spf13/cobra"
)

const (
	pidFile = "leoxy/keys/leoxy.pid"
	pidLogs = "leoxy/log"
	pidKeys = "leoxy/keys"
)

var once sync.Once

var startCmd = &cobra.Command{
	Use:     "start",
	Short:   "Start the server in the background",
	Long:    "Start Leoxy in the background. Server output is appended to leoxy/log/leoxy.log, and the process ID is stored under leoxy/keys/.",
	Example: "  leoxy start",
	RunE: func(cmd *cobra.Command, args []string) error {
		once.Do(func() {
			if err := os.MkdirAll(pidKeys, 0700); err != nil {
				log.Fatal(err)
			}
			if err := os.Chmod(pidKeys, 0700); err != nil {
				log.Fatal(err)
			}
			if err := os.MkdirAll(pidLogs, 0700); err != nil {
				log.Fatal(err)
			}
			if err := os.Chmod(pidLogs, 0700); err != nil {
				log.Fatal(err)
			}
		})

		executable, err := os.Executable()
		if err != nil {
			return err
		}

		process := exec.Command(executable, "run")

		logFile, err := os.OpenFile("leoxy/log/leoxy.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			return err
		}
		defer func() {
			if err := logFile.Close(); err != nil {
				log.Printf("close log file: %v", err)
			}
		}()
		if err := logFile.Chmod(0600); err != nil {
			return err
		}

		process.Stdout = logFile
		process.Stderr = logFile

		if err := process.Start(); err != nil {
			return err
		}

		if err := os.WriteFile(pidFile, []byte(strconv.Itoa(process.Process.Pid)), 0644); err != nil {
			_ = process.Process.Kill()
			return err
		}
		if err := os.Chmod(pidFile, 0600); err != nil {
			_ = process.Process.Kill()
			_ = os.Remove(pidFile)
			return err
		}

		fmt.Println("Leoxy started proxy")
		return nil
	},
}

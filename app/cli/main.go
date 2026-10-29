package main

import (
	"fmt"
	"os"

	serverapp "github.com/elrefai99/Leoxy/app/pkg/server"
	"github.com/spf13/cobra"
)

const version = "v1.9.0"

var configFile, dataDir string

var rootCmd = &cobra.Command{
	Use:     "leoxy",
	Short:   "Leoxy reverse proxy service",
	Version: version,
	Run: func(cmd *cobra.Command, args []string) {
		serverapp.Run()
	},
}

var versionCmd = &cobra.Command{
	Use:     "version",
	Short:   "Print the Leoxy version",
	Long:    "Print the version of the Leoxy reverse proxy executable.",
	Example: "  leoxy version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("leoxy %s\n", version)
	},
}

var runCmd = &cobra.Command{
	Use:    "run",
	Short:  "Run the server",
	Hidden: true,
	Run: func(cmd *cobra.Command, args []string) {
		serverapp.Run()
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&configFile, "config", "", "Path to the configuration file")
	rootCmd.PersistentFlags().StringVar(&dataDir, "data-dir", "", "Directory for logs and runtime state")
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(stopCmd)
	rootCmd.AddCommand(runCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

package main

import (
	"fmt"
	"os"

	serverapp "github.com/elrefai99/Leoxy/app/pkg/server"
	"github.com/spf13/cobra"
)

const version = "v3.8.1"

var rootCmd = &cobra.Command{
	Use:   "leoxy",
	Short: "Leoxy reverse proxy service",
	Long: `Leoxy is a reverse proxy configured with leoxy/config.yaml, relative to the
current working directory. If the file is missing, Leoxy creates a default
configuration.

Configuration:
  server.port sets the HTTP listener address (default :8080). PORT overrides it.
  HTTP upstreams use path plus server_url or servers. Multiple servers use
  round-robin load balancing. HTTP upstreams support HTTP/1.1, negotiated
  upstream HTTP/2, WebSocket upgrades, and GraphQL over HTTP. HTTP/3 is not
  currently supported.
  TCP and UDP upstreams use protocol, listen, and server_url. Example:
    - name: TCPService
      protocol: tcp
      listen: ":7001"
      server_url: "127.0.0.1:7000"
  Raw TCP and UDP listeners do not use HTTP security middleware.

Security settings can be configured under server.security or upstream[].security.
They include rate limits, allowed methods and paths, CIDR rules, API keys, JWT,
mTLS, body limits, trusted proxy CIDRs, and optional Redis global rate limiting.
Set secrets using API_KEYS, JWT_SECRET, JWT_ISSUER, JWT_AUDIENCE, and
REDIS_PASSWORD environment variables. PORT and RATE_LIMIT also override config.

HTTP endpoints:
  GET /ping       Returns PONG.
  GET /health/live Returns the liveness status.
  GET /health/ready Checks configured HTTP upstreams.
  GET /metrics    Exposes request metrics.
  GET /leoxy      Serves the project page.

Run from the directory containing leoxy/config.yaml:
  leoxy             Run in the foreground.
  leoxy start       Start in the background; logs go to leoxy/log/leoxy.log.
  leoxy stop        Stop the background process.
  leoxy version     Print the executable version.

Build with: go build -o leoxy ./app/cli
On Windows: go build -o leoxy.exe ./app/cli`,
	Example: "  leoxy\n  leoxy start\n  leoxy stop\n  leoxy version",
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

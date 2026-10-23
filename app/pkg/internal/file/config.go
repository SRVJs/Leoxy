package file

import "os"

const defaultConfig = `
# Leoxy listens on this port. PORT can override it through the environment.
server:
  port: "8080"

# Global security and request limits for all routes.
  security:
    # HTTP methods accepted by the proxy.
    allowed_methods: [GET, POST, PUT, PATCH, DELETE, OPTIONS]

# Each upstream maps a request path to one or more backend servers.
upstream:
  - name: "server_runner"
    # Requests under /api are forwarded to one of these servers.
    path: /
    # Use server_url for a single backend; use servers for multiple backends.
    server_url: "http://localhost:3000"
`

func CreateConfig() error {
	if err := os.MkdirAll("leoxy", 0700); err != nil {
		return err
	}
	if err := os.Chmod("leoxy", 0700); err != nil {
		return err
	}

	configPath := "leoxy/config.yaml"
	if _, err := os.Stat(configPath); err == nil {
		return os.Chmod(configPath, 0600)
	} else if !os.IsNotExist(err) {
		return err
	}

	if err := os.WriteFile(configPath, []byte(defaultConfig), 0600); err != nil {
		return err
	}
	return os.Chmod(configPath, 0600)
}

package file

import (
	"os"
	"path/filepath"
)

const defaultConfig = `server:
  host: "127.0.0.1"
  port: "4444"

  security:
    rate_limit: 600
    rate_limit_burst: 100
    max_body: 10
    allowed_methods: [GET, POST, PUT, PATCH, DELETE, OPTIONS]
    trusted_proxy_cidrs: []
    internal_cidrs: []

upstream:
  - name: "server_runner"
    path: /
    server_url: "http://localhost:3000"
`

func CreateConfig() (err error) {
	dir := filepath.Dir(ConfigPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	_ = os.Chmod(dir, 0700)

	if _, err := os.Stat(ConfigPath); err == nil {
		_ = os.Chmod(ConfigPath, 0600)
		return nil
	} else if !os.IsNotExist(err) {
		return err
	}

	file, err := os.OpenFile(ConfigPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := file.Close(); err == nil {
			err = closeErr
		}
	}()
	_, err = file.WriteString(defaultConfig)
	return err
}

package core

import (
	"os"

	"github.com/elrefai99/Leoxy/app/pkg/internal/file"
)

func Configure(configPath, dataDir string) error {
	if configPath != "" {
		file.ConfigPath = configPath
	}
	if dataDir == "" {
		return nil
	}
	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return err
	}
	return os.Chdir(dataDir)
}

package fs

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

func SaveHeapProf(client *http.Client, url, output string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("pprof returned http %d", resp.StatusCode)
	}

	if err := os.MkdirAll(filepath.Dir(output), 0755); err != nil {
		return err
	}
	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = io.Copy(file, resp.Body)
	return err
}

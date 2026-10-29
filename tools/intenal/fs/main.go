package fs

import (
	"fmt"
	"io"
	"net/http"
	"os"
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

	file, err := os.Create(output)
	if err != nil {
		return err
	}
	defer file.Close()
	ile, err := io.Copy(file, resp.Body)
	fmt.Println(ile)
	return err
}

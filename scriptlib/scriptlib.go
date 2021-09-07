// Package scriptlib should be used as loadable scripts
package scriptlib

import (
	"fmt"
	"io"
	"net/http"
)

// HTTPGet gets file from url
func HTTPGet(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("file not found: %d error", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

//go:build stdhttp
// +build stdhttp

package gshellos

import (
	"fmt"
	"io"
	"net/http"
)

func init() {
	httpGet = httpGetFunc
}

func httpGetFunc(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("url: %s not found: %d error", url, resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return body, nil
}

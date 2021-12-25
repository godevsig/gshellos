// +build stdhttp

package gshellos

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"syscall"

	as "github.com/godevsig/adaptiveservice"
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

func init() {
	as.RegisterType((*net.OpError)(nil))
	as.RegisterType((*net.TCPAddr)(nil))
	as.RegisterType((*os.SyscallError)(nil))
	as.RegisterType(syscall.Errno(0))
}

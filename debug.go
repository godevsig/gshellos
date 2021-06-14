// +build debug

package gshellos

import (
	"net"
	"net/http"
	_ "net/http/pprof"

	"github.com/godevsig/gshellos/log"
)

func init() {
	debugService = debugSrv
}

func debugSrv(lgr *log.Logger) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		panic(err)
	}

	lgr.Infoln("debugging at:", listener.Addr().String())
	http.Serve(listener, nil)
}

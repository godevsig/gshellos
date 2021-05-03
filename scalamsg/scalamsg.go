// Package scalamsg is a message handling client server framework.
//
// It has the ability to auto scale.
package scalamsg

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"os/signal"
	"path"
	"runtime"
	"strings"
	"sync"
	"syscall"
)

var (
	maxErrNum = 1000
	// ErrClosed is an error where the connection was closed.
	ErrClosed = errors.New("connection closed")
	// ErrRecvTimeout is an error where receiving is timeout.
	ErrRecvTimeout = errors.New("recv timeout")
)

func errorHere(err interface{}) error {
	_, file, line, _ := runtime.Caller(1)
	return fmt.Errorf("(%s:%d): %v", path.Base(file), line, err)
}

type errs struct {
	sync.Mutex
	errs []error
}

func (es *errs) addError(err error) {
	es.Lock()
	es.errs = append(es.errs, err)
	es.Unlock()
	if len(es.errs) > maxErrNum {
		panic(es)
	}
}

func (es *errs) Error() string {
	var sb strings.Builder
	for _, e := range es.errs {
		fmt.Fprintf(&sb, "%v\n", e)
	}
	return sb.String()
}

// Processor represents a server or a client whose OnConnect method will be
// called upon a new connection is established.
type Processor interface {
	// OnConnect is called upon a new connection is established.
	// If the returned error is io.EOF, the connection will be closed.
	OnConnect(conn Conn) error
}

// ListenRun listens the network address and starts serving, upon a new
// connection is established, the server's OnConnect method is called.
// The returned error includes all occurred errors before the server
// is shutdown.
func ListenRun(network, address string, server Processor) (retErr error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return err
	}
	defer l.Close()

	var errall errs
	defer func() {
		if len(errall.errs) != 0 {
			retErr = &errall
		}
	}()

	// handle signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		errall.addError(errorHere(fmt.Errorf("got signal: %s", sig.String())))
		l.Close()
	}()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for {
		c, err := l.Accept()
		if err != nil {
			errall.addError(errorHere(err))
			break
		}
		conn := newConn(ctx, c)
		wg.Add(1)
		go func() {
			defer func() {
				conn.close()
				wg.Done()
			}()
			if err := server.OnConnect(conn); err != nil {
				if errors.Is(err, io.EOF) {
					conn.close()
				} else {
					errall.addError(errorHere(err))
				}
			}
			if err := conn.wait(); err != nil {
				errall.addError(errorHere(err))
			}
		}()
	}
	cancel()
	wg.Wait()
	return
}

// DialRun dials the network address and starts the client's OnConnect method.
// The returned error includes all occurred errors before the client finishes.
func DialRun(network, address string, client Processor) (retErr error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return err
	}
	conn := newConn(context.Background(), c)
	defer conn.close()

	var errall errs
	defer func() {
		if len(errall.errs) != 0 {
			retErr = &errall
		}
	}()

	// handle signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		errall.addError(errorHere(fmt.Errorf("got signal: %s", sig.String())))
		conn.close()
	}()

	if err := client.OnConnect(conn); err != nil {
		if errors.Is(err, io.EOF) {
			conn.close()
		} else {
			errall.addError(errorHere(err))
		}
	}
	if err := conn.wait(); err != nil {
		errall.addError(errorHere(err))
	}
	return
}

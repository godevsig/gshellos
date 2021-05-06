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

// OnConnectFunc is a wrapper to allow the use of ordinary functions as
// Processor.
type OnConnectFunc func(Conn) error

// OnConnect calls f(conn)
func (f OnConnectFunc) OnConnect(conn Conn) error {
	return f(conn)
}

// Logger is the logger interface.
type Logger interface {
	Debugf(format string, args ...interface{})
	Debugln(args ...interface{})
	Infof(format string, args ...interface{})
	Infoln(args ...interface{})
	Warnf(format string, args ...interface{})
	Warnln(args ...interface{})
	Errorf(format string, args ...interface{})
	Errorln(args ...interface{})
	Fatalf(format string, args ...interface{})
	Fatalln(args ...interface{})
}

type null struct{}

func (null) Debugf(format string, args ...interface{}) {}
func (null) Debugln(args ...interface{})               {}
func (null) Infof(format string, args ...interface{})  {}
func (null) Infoln(args ...interface{})                {}
func (null) Warnf(format string, args ...interface{})  {}
func (null) Warnln(args ...interface{})                {}
func (null) Errorf(format string, args ...interface{}) {}
func (null) Errorln(args ...interface{})               {}
func (null) Fatalf(format string, args ...interface{}) {}
func (null) Fatalln(args ...interface{})               {}

type conf struct {
	lgr Logger
}

// Option is used to set options.
type Option func(*conf)

// WithLogger sets logger.
func WithLogger(lgr Logger) Option {
	return func(c *conf) {
		c.lgr = lgr
	}
}

// Listener represents a server.
type Listener struct {
	l      net.Listener
	errall errs
	sync.Mutex
	conns map[*conn]struct{}
	lgr   Logger
}

func (lnr *Listener) addConn(cn *conn) {
	lnr.Lock()
	lnr.conns[cn] = struct{}{}
	lnr.Unlock()
}

func (lnr *Listener) delConn(cn *conn) {
	lnr.Lock()
	delete(lnr.conns, cn)
	lnr.Unlock()
}

// Listen listens the network address and starts serving.
func Listen(network, address string, options ...Option) (*Listener, error) {
	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}
	var cnf conf
	for _, o := range options {
		o(&cnf)
	}

	lnr := &Listener{l: l, conns: make(map[*conn]struct{})}
	if cnf.lgr == nil {
		cnf.lgr = null{}
	}
	lnr.lgr = cnf.lgr
	return lnr, nil
}

// Close closes the listener.
func (lnr *Listener) Close() {
	lnr.l.Close()
}

// Errors returns all errors occurred in the listener so far.
func (lnr *Listener) Errors() error {
	var sb strings.Builder
	fmt.Fprintln(&sb, "Current errors:")
	for conn := range lnr.conns {
		if err := conn.Errors(); err != nil {
			remote := conn.GetNetConn().RemoteAddr().String()
			fmt.Fprintf(&sb, "%s: %v\n", remote, err)
		}
	}
	fmt.Fprintln(&sb, "Historical errors:")
	if len(lnr.errall.errs) != 0 {
		fmt.Fprintf(&sb, "%v\n", &lnr.errall)
	}

	return errors.New(sb.String())
}

func (lnr *Listener) logError(err error) {
	lnr.lgr.Errorln(err)
	lnr.errall.addError(err)
}

// Run runs the server's OnConnect method upon each new connection is established.
// The returned error includes all occurred errors before the server is shutdown.
func (lnr *Listener) Run(server Processor) (retErr error) {
	defer func() {
		if len(lnr.errall.errs) != 0 {
			retErr = &lnr.errall
		}
		lnr.l.Close()
	}()

	// handle signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		lnr.logError(errorHere(fmt.Errorf("got signal: %s", sig.String())))
		lnr.l.Close()
	}()

	var wg sync.WaitGroup
	ctx, cancel := context.WithCancel(context.Background())

	for {
		c, err := lnr.l.Accept()
		if err != nil {
			if strings.Contains(err.Error(), "use of closed network connection") {
				break
			}
			lnr.logError(errorHere(err))
		}
		cn := newConn(ctx, c, lnr.lgr)
		wg.Add(1)
		go func(cn *conn) {
			lnr.addConn(cn)
			defer func() {
				cn.close()
				lnr.delConn(cn)
				wg.Done()
			}()
			if err := server.OnConnect(cn); err != nil {
				if errors.Is(err, io.EOF) {
					cn.close()
				} else {
					lnr.logError(errorHere(err))
				}
			}
			if err := cn.wait(); err != nil {
				lnr.errall.addError(errorHere(err))
			}
		}(cn)
	}
	cancel()
	wg.Wait()
	return
}

// ListenRun is a shortcut for Listen and Run.
func ListenRun(server Processor, network string, address string, options ...Option) (retErr error) {
	lnr, err := Listen(network, address, options...)
	if err != nil {
		return err
	}

	return lnr.Run(server)
}

// Dialer represents a client.
type Dialer struct {
	errall errs
	conn   *conn
	lgr    Logger
}

// Dial dials the network address and starts a background receiver
// waiting for incoming messages.
func Dial(network, address string, options ...Option) (*Dialer, error) {
	c, err := net.Dial(network, address)
	if err != nil {
		return nil, err
	}
	var cnf conf
	for _, o := range options {
		o(&cnf)
	}

	dlr := &Dialer{}
	if cnf.lgr == nil {
		cnf.lgr = null{}
	}
	dlr.lgr = cnf.lgr

	dlr.conn = newConn(context.Background(), c, dlr.lgr)
	return dlr, nil
}

// Close closes the dialer.
func (dlr *Dialer) Close() {
	dlr.conn.close()
}

// Errors returns all errors occurred in the Dialer so far.
func (dlr *Dialer) Errors() error {
	return dlr.conn.Errors()
}

func (dlr *Dialer) logError(err error) {
	dlr.lgr.Errorln(err)
	dlr.errall.addError(err)
}

// Run runs client's OnConnect method.
// The returned error includes all occurred errors before the client finishes.
func (dlr *Dialer) Run(client Processor) (retErr error) {
	defer func() {
		if len(dlr.errall.errs) != 0 {
			retErr = &dlr.errall
		}
		dlr.conn.close()
	}()

	// handle signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		dlr.errall.addError(errorHere(fmt.Errorf("got signal: %s", sig.String())))
		dlr.conn.close()
	}()

	if err := client.OnConnect(dlr.conn); err != nil {
		if errors.Is(err, io.EOF) {
			dlr.conn.close()
		} else {
			dlr.logError(errorHere(err))
		}
	}
	if err := dlr.conn.wait(); err != nil {
		dlr.errall.addError(errorHere(err))
	}
	return
}

// DialRun is a shortcut for Dial and Run.
func DialRun(client Processor, network string, address string, options ...Option) (retErr error) {
	dlr, err := Dial(network, address, options...)
	if err != nil {
		return err
	}
	return dlr.Run(client)
}

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
	"time"
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
	Tracef(format string, args ...interface{})
	Traceln(args ...interface{})
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

func (null) Tracef(format string, args ...interface{}) {}
func (null) Traceln(args ...interface{})               {}
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
	lgr         Logger
	dialTimeout time.Duration
	errorAsEOF  bool
	recvQlen    int
	qWeight     int
	scaleFactor int
}

func (cnf *conf) setPostDefault() {
	if cnf.lgr == nil {
		cnf.lgr = null{}
	}
	if cnf.recvQlen <= 0 {
		cnf.recvQlen = defaultQlen
	}
	if cnf.qWeight <= 0 {
		cnf.qWeight = defaultQWeight
	}
	if cnf.scaleFactor <= 0 {
		cnf.scaleFactor = defaultScaleFactor
	}
}

// Option is used to set options.
type Option func(*conf)

// WithLogger sets logger.
func WithLogger(lgr Logger) Option {
	return func(c *conf) {
		c.lgr = lgr
	}
}

// WithRecvQlen sets the receive queue length.
func WithRecvQlen(recvQlen int) Option {
	return func(c *conf) {
		c.recvQlen = recvQlen
	}
}

// WithScaleFactor sets the scale factors:
//  qWeight: smaller qWeight results higher message handling concurrency.
//  scaleFactor: the maximum concurrency per CPU core.
// You should know what you are doing with these parameters.
func WithScaleFactor(qWeight, scaleFactor int) Option {
	return func(c *conf) {
		c.qWeight = qWeight
		c.scaleFactor = scaleFactor
	}
}

// WithDialTimeout sets the timeout when dialing.
func WithDialTimeout(timout time.Duration) Option {
	return func(c *conf) {
		c.dialTimeout = timout
	}
}

// ErrorAsEOF sets the flag with which any error returned by OnConnect()
// or message's Handle() method is treated as io.EOF and triggers to close
// the connection.
func ErrorAsEOF() Option {
	return func(c *conf) {
		c.errorAsEOF = true
	}
}

// Listener represents a server.
type Listener struct {
	conf
	sync.Mutex
	l      net.Listener
	errall errs
	conns  map[*conn]struct{}
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
	var cnf conf
	for _, o := range options {
		o(&cnf)
	}
	cnf.setPostDefault()

	l, err := net.Listen(network, address)
	if err != nil {
		return nil, err
	}

	lnr := &Listener{conf: cnf, l: l, conns: make(map[*conn]struct{})}
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
// Run will keep running unless:
//  Signal such as SIGINT SIGHUP SIGTERM was captured.
//  Close() method of the listener is called by user.
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
		lnr.errall.addError(errorHere(fmt.Errorf("got signal: %s", sig.String())))
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
		cn := newConn(ctx, c, lnr.conf)
		wg.Add(1)
		go func(cn *conn) {
			lnr.addConn(cn)
			defer func() {
				cn.Close()
				lnr.delConn(cn)
				wg.Done()
			}()
			if err := server.OnConnect(cn); err != nil {
				if lnr.errorAsEOF || errors.Is(err, io.EOF) {
					cn.Close()
				}
				if !errors.Is(err, io.EOF) {
					lnr.logError(errorHere(err))
				}
			}
			if err := cn.wait(); err != nil {
				lnr.errall.addError(errorHere(err))
			}
		}(cn)
	}
	lnr.lgr.Traceln("listener closed, waiting for remaining work")
	cancel()
	wg.Wait()
	lnr.lgr.Traceln("listener done")
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
	conf
	errall errs
	c      net.Conn
	conn   *conn
}

// Dial dials the network address.
func Dial(network, address string, options ...Option) (*Dialer, error) {
	var cnf conf
	for _, o := range options {
		o(&cnf)
	}
	cnf.setPostDefault()

	var c net.Conn
	var err error
	if cnf.dialTimeout != 0 {
		c, err = net.DialTimeout(network, address, cnf.dialTimeout)
	} else {
		c, err = net.Dial(network, address)
	}
	if err != nil {
		return nil, err
	}

	dlr := &Dialer{conf: cnf, c: c}
	return dlr, nil
}

// Close closes the dialer.
func (dlr *Dialer) Close() {
	dlr.conn.Close()
}

// Errors returns all errors occurred in the Dialer so far.
func (dlr *Dialer) Errors() error {
	return dlr.conn.Errors()
}

func (dlr *Dialer) logError(err error) {
	dlr.lgr.Errorln(err)
	dlr.errall.addError(err)
}

// Run starts a background receiver waiting for incoming messages, and calls
// client's OnConnect method. Run will keep running unless:
//  The connection peer closed.
//  The OnConnect() method of client returns io.EOF as error.
//  The Handle() method of the received message returns io.EOF as error.
//  Signal such as SIGINT SIGHUP SIGTERM was captured.
//  Close() method of the dialer is called by user.
// Option ErrorAsEOF() can change the io.EOF behavior.
// The returned error includes all occurred errors before the client finishes.
func (dlr *Dialer) Run(client Processor) (retErr error) {
	dlr.conn = newConn(context.Background(), dlr.c, dlr.conf)
	defer func() {
		if len(dlr.errall.errs) != 0 {
			retErr = &dlr.errall
		}
		dlr.conn.Close()
	}()

	// handle signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGHUP, syscall.SIGTERM)
	go func() {
		sig := <-sigChan
		dlr.errall.addError(errorHere(fmt.Errorf("got signal: %s", sig.String())))
		dlr.conn.Close()
	}()

	if err := client.OnConnect(dlr.conn); err != nil {
		if dlr.errorAsEOF || errors.Is(err, io.EOF) {
			dlr.conn.Close()
		}
		if !errors.Is(err, io.EOF) {
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
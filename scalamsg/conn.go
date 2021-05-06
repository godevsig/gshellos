package scalamsg

import (
	"context"
	"encoding/gob"
	"errors"
	"io"
	"net"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

// Context represents a context.
type Context interface {
	// PutVar puts value v to the underlying map overriding the old value of the same type.
	PutVar(v interface{})

	// GetVar gets value that v points to from the underlying map, it panics if v
	// is not a non-nil pointer.
	// The value that v points to will be set to the value if value
	// of the same type has been putted to the map,
	// otherwise zero value will be set.
	GetVar(v interface{})

	// SetContext sets the context with value v which supposedly is a pointer to
	// an instance of the struct associated to the connection.
	// It panics if v is not a non-nil pointer.
	// It is supposed to be called only once upon a new connection is connected.
	SetContext(v interface{})
	// GetContext gets the context that has been set by SetContext.
	GetContext() interface{}
}

type contextImpl struct {
	sync.RWMutex
	kv  map[reflect.Type]interface{}
	ctx interface{}
}

func (c *contextImpl) PutVar(v interface{}) {
	c.Lock()
	c.kv[reflect.TypeOf(v)] = v
	c.Unlock()
}

func (c *contextImpl) GetVar(v interface{}) {
	c.RLock()
	defer c.RUnlock()
	rptr := reflect.ValueOf(v)
	if rptr.Kind() != reflect.Ptr || rptr.IsNil() {
		panic("not a pointer or nil pointer")
	}
	rv := rptr.Elem()
	tp := rv.Type()
	if i, ok := c.kv[tp]; ok {
		rv.Set(reflect.ValueOf(i))
	}
	rv.Set(reflect.Zero(tp))
}

func (c *contextImpl) SetContext(v interface{}) {
	rptr := reflect.ValueOf(v)
	if rptr.Kind() != reflect.Ptr || rptr.IsNil() {
		panic("not a pointer or nil pointer")
	}
	c.ctx = v
}

func (c *contextImpl) GetContext() interface{} {
	return c.ctx
}

// Conn represents a connection with context.
// When operating with Context, be careful to the concurrency,
// since messages from the same connection have the same context,
// and they may be executed concurrently.
type Conn interface {
	Context

	// Send sends an arbitrary message.
	Send(msg interface{}) error

	// Recv receives an unknown message.
	// Unknown messages are messages not implementing Message's Handle method.
	// Note known messages are handled automatically.
	Recv() (msg interface{}, err error)

	// RecvTimeout receives an unknown message within timeout.
	// It returns ErrRecvTimeout if timout happens, ErrClosed if the connection
	// was closed.
	RecvTimeout(timeout time.Duration) (msg interface{}, err error)

	// GetNetConn gets the raw network connection.
	GetNetConn() net.Conn
}

var (
	qWeight     = 16
	defaultQLen = qWeight * runtime.NumCPU()
	nilQ        <-chan time.Time
)

type conn struct {
	sync.RWMutex
	contextImpl
	netConn net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	recvQ   chan interface{} // for unknown msg
	closeQ  chan struct{}
	errall  errs
	done    chan struct{}
	lgr     Logger
}

func (c *conn) logError(err error) {
	c.lgr.Errorln(err)
	c.errall.addError(err)
}

func newConn(ctx context.Context, netConn net.Conn, lgr Logger) *conn {
	c := &conn{
		contextImpl: contextImpl{kv: make(map[reflect.Type]interface{})},
		netConn:     netConn,
		enc:         gob.NewEncoder(netConn),
		dec:         gob.NewDecoder(netConn),
		recvQ:       make(chan interface{}, defaultQLen),
		closeQ:      make(chan struct{}),
		done:        make(chan struct{}),
		lgr:         lgr,
	}

	handlemsg := func(msg Message, exclusive bool) (eof bool) {
		if exclusive {
			c.Lock()
		}
		reply, err := msg.Handle(c)
		if exclusive {
			c.Unlock()
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				c.close()
				eof = true
				return
			}
			c.logError(errorHere(err))
			return
		}
		if reply != nil {
			c.RLock()
			err := c.Send(reply)
			c.RUnlock()
			if err != nil {
				c.logError(errorHere(err))
			}
		}
		return
	}

	receiver := func() {
		msgs := make(chan Message, defaultQLen)
		worker := func(done <-chan struct{}) {
			for {
				select {
				case <-ctx.Done():
					c.close()
					return
				case <-done:
					return
				case msg := <-msgs:
					if eof := handlemsg(msg, false); eof {
						return
					}
				}
			}
		}
		wp := NewWorkerPool()
		wp.AddWorker(worker)
		for {
			var msg interface{}
			if err := c.dec.Decode(&msg); err != nil {
				if errors.Is(err, io.EOF) {
					c.close()
					break
				}
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				}
				c.logError(errorHere(err))
			} else if emsg, ok := msg.(ExclusiveMessage); ok {
				if eof := handlemsg(emsg, true); eof {
					break
				}
			} else if mmsg, ok := msg.(Message); ok {
				should := len(msgs)/qWeight + 1
				now := wp.Len()
				switch {
				case should > now:
					wp.AddWorker(worker)
				case should < now:
					wp.RmWorker()
				}
				msgs <- mmsg
			} else {
				select {
				case c.recvQ <- msg:
				default:
					c.logError(errorHere("message dropped"))
				}
			}
		}
		for len(msgs) != 0 {
			time.Sleep(time.Second)
		}
		wp.Close()
	}

	go func() {
		receiver()
		c.done <- struct{}{}
	}()
	return c
}

func (c *conn) close() {
	c.Lock()
	defer c.Unlock()
	if c.closeQ != nil {
		close(c.closeQ)
		c.closeQ = nil
		c.netConn.Close()
	}
}

// Wait waits the connection to finish, returns all occurred errors if any.
func (c *conn) wait() (err error) {
	<-c.done
	if len(c.errall.errs) != 0 {
		err = &c.errall
	}
	return
}

// Errors returns all errors occurred in the connection so far.
func (c *conn) Errors() (err error) {
	if len(c.errall.errs) != 0 {
		err = &c.errall
	}
	return
}

// Send sends an arbitrary message.
func (c *conn) Send(msg interface{}) error {
	// to be able to decode directly into an interface variable,
	// we need to encode it as reference of the interface
	err := c.enc.Encode(&msg)
	if err != nil {
		c.logError(errorHere(err))
	}
	return err
}

// RecvTimeout receives an unknown message within timeout.
// It returns ErrRecvTimeout if timout happens, ErrClosed if the connection
// was closed.
func (c *conn) RecvTimeout(timeout time.Duration) (msg interface{}, err error) {
	timeQ := nilQ
	if timeout > 0 {
		timeQ = time.After(timeout)
	}
	recvQ := c.recvQ
	closeQ := c.closeQ

	select {
	case <-closeQ:
		err = ErrClosed
	case <-timeQ:
		err = ErrRecvTimeout
	case m := <-recvQ:
		msg = m
	}

	if err != nil {
		c.logError(errorHere(err))
	}
	return
}

// Recv receives an unknown message.
// Unknown messages are messages not implementing Message's Handle method.
// Note known messages are handled automatically.
func (c *conn) Recv() (msg interface{}, err error) {
	return c.RecvTimeout(0)
}

// GetNetConn gets the raw network connection.
func (c *conn) GetNetConn() net.Conn {
	return c.netConn
}

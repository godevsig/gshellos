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
	// If the peer's message handlers return non-nil err, the error will be
	// returned by Recv() as err.
	Recv() (msg interface{}, err error)

	// Close closes the Conn, and triggers the termination process of this Conn.
	Close()

	// RecvTimeout receives an unknown message within timeout.
	// It returns ErrRecvTimeout if timout happens, ErrClosed if the connection
	// was closed.
	RecvTimeout(timeout time.Duration) (msg interface{}, err error)

	// GetNetConn gets the raw network connection.
	GetNetConn() net.Conn
}

var (
	defaultQWeight     = 8
	defaultScaleFactor = 8
	defaultQlen        = 128
	nilQ               <-chan time.Time
)

type conn struct {
	sync.RWMutex
	conf
	contextImpl
	netConn net.Conn
	enc     *gob.Encoder
	dec     *gob.Decoder
	recvQ   chan interface{} // for unknown msg
	closeQ  chan struct{}
	errall  errs
	done    chan struct{}
}

func (c *conn) logError(err error) {
	c.lgr.Errorln(err)
	c.errall.addError(err)
}

func newConn(ctx context.Context, netConn net.Conn, cnf conf) *conn {
	c := &conn{
		conf:        cnf,
		contextImpl: contextImpl{kv: make(map[reflect.Type]interface{})},
		netConn:     netConn,
		enc:         gob.NewEncoder(netConn),
		dec:         gob.NewDecoder(netConn),
		recvQ:       make(chan interface{}, cnf.recvQlen),
		closeQ:      make(chan struct{}),
		done:        make(chan struct{}),
	}
	c.lgr.Traceln("new connection established")

	handlemsg := func(msg Message, exclusive bool) (eof bool) {
		c.lgr.Traceln("handling message")
		if exclusive {
			c.Lock()
		}
		reply, err := msg.Handle(c)
		if exclusive {
			c.Unlock()
		}

		if errors.Is(err, io.EOF) {
			eof = true
			if reply != nil { // send reply first
				c.RLock()
				c.Send(reply)
				c.RUnlock()
			}
			return
		}
		if err != nil {
			c.logError(errorHere(err))
			if c.errorAsEOF {
				eof = true
			}
			em := ErrorMsg{Msg: reply, Err: err.Error()}
			c.RLock()
			c.Send(em)
			c.RUnlock()
		}
		if reply != nil {
			c.RLock()
			c.Send(reply)
			c.RUnlock()
		}
		return
	}

	qlen := cnf.qWeight * cnf.scaleFactor * runtime.NumCPU()
	receiver := func() {
		c.lgr.Traceln("receiver started")
		msgs := make(chan Message, qlen)
		worker := func(done <-chan struct{}) {
			for {
				select {
				case <-ctx.Done():
					c.Close()
					return
				case <-done:
					return
				case msg := <-msgs:
					if eof := handlemsg(msg, false); eof {
						c.Close()
						return
					}
				}
			}
		}
		wp := NewWorkerPool()
		wp.AddWorker(worker)
		c.lgr.Traceln("worker added")
		for {
			var msg interface{}
			if err := c.dec.Decode(&msg); err != nil {
				if errors.Is(err, io.EOF) {
					c.Close()
					break
				}
				if strings.Contains(err.Error(), "use of closed network connection") {
					break
				}
				c.logError(errorHere(err))
			} else if emsg, ok := msg.(ExclusiveMessage); ok {
				c.lgr.Traceln("exclusive message received")
				if eof := handlemsg(emsg, true); eof {
					c.Close()
					break
				}
			} else if mmsg, ok := msg.(Message); ok {
				c.lgr.Traceln("message received")
				should := len(msgs)/cnf.qWeight + 1
				now := wp.Len()
				c.lgr.Debugf("worker number: should: %d, now: %d\n", should, now)
				switch {
				case should > now:
					wp.AddWorker(worker)
					c.lgr.Traceln("worker added")
				case should < now:
					wp.RmWorker()
					c.lgr.Traceln("worker removed")
				}
				msgs <- mmsg
			} else {
				c.lgr.Traceln("unknown message received")
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
		c.lgr.Traceln("receiver done")
	}

	go func() {
		receiver()
		c.done <- struct{}{}
	}()
	return c
}

func (c *conn) Close() {
	c.Lock()
	defer c.Unlock()
	if c.closeQ != nil {
		c.lgr.Traceln("closing connection")
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
		if em, ok := m.(ErrorMsg); ok {
			msg = em.Msg
			err = errors.New(em.Err)
		} else {
			msg = m
		}
	}

	if err != nil {
		c.logError(errorHere(err))
	}
	return
}

// Recv receives an unknown message.
// Unknown messages are messages not implementing Message's Handle method.
// Note known messages are handled automatically.
// If the peer's message handlers return non-nil err, the error will be
// returned by Recv() as err.
func (c *conn) Recv() (msg interface{}, err error) {
	return c.RecvTimeout(0)
}

// GetNetConn gets the raw network connection.
func (c *conn) GetNetConn() net.Conn {
	return c.netConn
}

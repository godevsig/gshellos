package scalamsg

import "encoding/gob"

// Message represents a message, a handler is needed to process the message
type Message interface {
	// Handle handles the message, reply will be sent back to the connection if it is not nil.
	// The connection will be closed if the returned err is io.EOF.
	// If err is not nil, reply and err will be sent back to the connection as one unknown Message
	// so that they are returned by the peer's Conn.Recv() as if Handle() were called locally.
	//
	// The message may be marshaled or compressed.
	// Remember in golang assignment to interface is also value copy,
	// so return reply as &someStruct whenever possible in your handler implementation.
	Handle(conn Conn) (reply interface{}, err error)
}

// ExclusiveMessage represents a message that no other messages should be operating the underlying
// net connection while this exclusive message is being handled.
type ExclusiveMessage interface {
	Message
	// IsExclusive specifies whether the message should be executed exclusively per connection.
	IsExclusive()
}

// ErrorMsg is a special message that carries the error from a connection endpoint to its peer.
type ErrorMsg struct {
	Msg interface{}
	Err string
}

func (em ErrorMsg) Error() string {
	return em.Err
}

func init() {
	gob.Register(ErrorMsg{})
}

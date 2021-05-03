package scalamsg

// Message represents a message, a handler is needed to process the message
type Message interface {
	// Handle handles the message, if reply is not nil, it will be sent back to the connection.
	// The connection will be closed if the returned err is io.EOF.
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

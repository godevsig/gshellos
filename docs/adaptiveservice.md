# Service name

A service is identified by a tuple of {"publisher", "service"} where

- "service" is the service name
- "publisher" is the programing owner of the named service

# Scopes and transports

A server can publish services to all available scopes:

| Scope   | Transport          | Allowed Instance # | Discover method |
| ------- | ------------------ | ------------------ | --------------- |
| Process | go channel         | 1                  | table lookup    |
| OS      | unix domain socket | 1                  | file lookup     |
| LAN     | TCP socket         | many               | LAN broadcast   |
| WAN     | TCP socket         | many               | root registry   |

# Publisher name VS Provider ID

In scope LAN and scope WAN, there can be many instances of the same service,
they all have the same "publisher", but each one of them has an
unique provider ID which is then can be used to do service selection.

- publisher name is the programing "owner" of the named service
- provider ID is the runtime entity that actually running the named service

# Discover a service

A service is discovered in the order of scope Process then scope OS
then scope LAN then scope WAN, in the client's configured scopes.

## Example: find out my observed IP

When you are behind a NAT network, `ip address` command does not tell you how outside of
your local network see your IP. There is a builtin service can help you to find out your
IP.

```go
import as "github.com/godevsig/adaptiveservice"

func GetObservedIP() string {
	c := as.NewClient(as.WithScope(as.ScopeWAN)).SetDiscoverTimeout(3)
	conn := <-c.Discover("builtin", "IPObserver")
	if conn == nil {
		lg.Errorln("IPObserver service not found")
		return ""
	}
	defer conn.Close()
	var ip string
	if err := conn.SendRecv(as.GetObservedIP{}, &ip); err != nil {
		lg.Errorln("get observed ip failed: %v", err)
		return ""
	}
	return ip
}
```

- `NewClient()` API creates a client. You can specify in which scopes the client is supposed
  to run. The default is in ALL scopes.
- `client.Discover()` API finds the wanted service and returns the connection channel from which
  you can get the established connection torwards to that service.
  See [Discover API](https://pkg.go.dev/github.com/godevsig/adaptiveservice#Client.Discover)
- `connection.SendRecv()` API exchange the request message and the reply message with the server.

# Publish a service

To publish a service:

- Prepare your publisher and service name.
- Define the message structs in the service package and their handlers.
- Define optional `OnConnectFunc()` and/or `OnNewStreamFunc()`
- Call `server.Publish()` API to publish the service.
  Multiple services can be published under one server.
- Register the message types in `init()`

See [adaptiveservice echo example](https://github.com/godevsig/adaptiveservice/tree/master/examples/echo/server)

# Known message

Messages that satisfy Handle() interface are known messages. Typically
server defines Handle() method for every message type it can handle,
then when the known message arrived on one of the transports it is
listening, the message is delivered to one of the workers in which
the message's Handle() is called.

Below code is taken from [adaptiveservice](https://github.com/godevsig/adaptiveservice/blob/master/builtinservices.go)

```go
// GetObservedIP returns the observed IP of the client.
// The reply is string type.
type GetObservedIP struct{}

// Handle handles GetObservedIP message.
func (msg GetObservedIP) Handle(stream ContextStream) (reply interface{}) {
	rhost, _, err := net.SplitHostPort(stream.GetNetconn().RemoteAddr().String())
	if err != nil {
		return err
	}
	return rhost
}

// publishIPObserverService declares the IP observer service.
func (s *Server) publishIPObserverService() error {
	knownMsgs := []KnownMessage{GetObservedIP{}}
	return s.publish(ScopeWAN, "builtin", "IPObserver", knownMsgs)
}

func init() {
	RegisterType(GetObservedIP{})
}
```

- `GetObservedIP{}` is a known message to service "IPObserver".
- When client calls `conn.SendRecv(as.GetObservedIP{}, &ip)`, the request message is then
  delivered to the server's message queue waitting one "idle" worker from worker pool to pick
  up the message and call the handler `GetObservedIP.Handle()`.
- Many clients can send the request simultaneously and the handlers will be called also
  in parallel on server side.
- `GetObservedIP.Handle()` returns the reply to the client. In this case the string type IP
  address is returned or an error is returned. Adaptiveservice framework will detect if
  `reply` is an error value, in which case the reply is turned into return value on the client
  side `Recv()` API. e.g. `err := conn.SendRecv(as.GetObservedIP{}, &ip)`.
- `ContextStream` in `GetObservedIP.Handle()` provides "contexted stream", representing the
  dedicated channel(called stream) between the client and the server multiplexed from the
  underlying transport which is a TCP socket in this case. `ContextStream` can be used to
  set/get context variables of the same stream; It can be also used to `Send()` or `Recv()`
  directly to the stream peer. It is not recommended to mix use return value `reply` and
  `ContextStream`'s Send/Recv API. See [message.go](https://github.com/godevsig/adaptiveservice/blob/master/message.go).

# Subsequent message

Server can also handle subsequent messages in the known message's handler, where the
known message, e.g. `SubWhoElseEvent`, is an initiator(lead message) of sequential
messages exchanged by client and server.

Subsequent messages do not need to satisfy `Handle(stream ContextStream) (reply interface{})`,
client and server should know the message types to be exchanged.

Below code declares a known message `SubWhoElseEvent`, within its handler a new routine is
created to wait new connection event on the server and send the "who else" info back to the
client.

```go
// SubWhoElseEvent is used for clients to subscribe who else event which
// reports new incoming connection to the server.
// Return string.
type SubWhoElseEvent struct{}

// Handle handles msg.
func (msg SubWhoElseEvent) Handle(stream as.ContextStream) (reply interface{}) {
	si := stream.GetContext().(*sessionInfo)
	ch := make(chan string, 1)
	si.mgr.Lock()
	si.mgr.subscribers[ch] = struct{}{}
	si.mgr.Unlock()
	go func() {
		for {
			addr := <-ch
			if err := stream.Send(addr); err != nil {
				si.mgr.Lock()
				delete(si.mgr.subscribers, ch)
				si.mgr.Unlock()
				fmt.Println("channel deleted")
				return
			}
		}
	}()
	return
}
```

See [echo example](https://github.com/godevsig/adaptiveservice/tree/master/examples/echo/server) for details.
This is also an implementation of PUB/SUB communication pattern.

# Send() Recv() and SendRecv()

On the client side, `Discover()` finds the service:

```go
c := as.NewClient()
conn := <-c.Discover(echo.Publisher, echo.ServiceEcho)
```

The returned `conn` is the connection to the wanted service, using which user can
`Send()` `Recv()` and `SendRecv()` messages.

On the server side, as we already know, `stream ContextStream` in
`Handle(stream ContextStream) (reply interface{})` can `Send()` `Recv()` and `SendRecv()` messages.

- `Send(msg)` sends a message to the stream peer. If msg is an error value, it will be received
  and returned by peer's Recv() as error.
- `Recv(msgPtr)` receives a message from the stream peer and stores it into the value that msgPtr points to.
- `SendRecv(msgSnd, msgRcvPtr)` combines send and receive, making it similar to a RPC:
  client "call" a "function" which is defined by `msgSnd`, server "handle" the message, the client waits then
  receives the reply from server in this single function.

# Client side multiplexed connection

The connection returned by discover can be multiplexed to get separate virtual streams towards the server using
the same underlying connection, clients can use streams to increase requesting concurrency.

```go
    c := as.NewClient()
    conn := <-c.Discover(echo.Publisher, echo.ServiceEcho)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			stream := conn.NewStream() // create multiplexed stream over conn
			req := echo.MessageRequest{
				Msg: "ni hao",
				Num: 100 * int32(i),
			}
			var rep echo.MessageReply
			for i := 0; i < 9; i++ {
				req.Num += 10
				if err := stream.SendRecv(&req, &rep); err != nil {
					fmt.Println(err)
					return
				}
				if req.Num+1 != rep.Num {
					panic("wrong number")
				}
				fmt.Printf("%v ==> %v, %s\n", req, rep.MessageRequest, rep.Signature)
				//time.Sleep(time.Second)
			}
		}(i)
	}
	wg.Wait()
```

See [echo example](https://github.com/godevsig/adaptiveservice/tree/master/examples/echo/client)

# Server side auto scale worker pool

We already know message handlers are called in a worker routine backed by a worker pool.
This worker pool is auto scaled in a way that

- if workers are not enough, the pool increases, adding workers until balanced.
- if workers are too much, the pool shrinks, removeing workers until balanced.

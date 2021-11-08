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
they all have the same "publisher" name, but each one of them has an
unique provider ID which is then can be used to do service selection.

- publisher name is the programing "owner" of the named service
- provider ID is the runtime entity that actually running the named service

# Publish a service

To publish a service, you prepare:

- your publisher and service name
- the message types your service can handle
- optional OnConnectFunc and/or OnNewStreamFunc

See [adaptiveservice echo example](https://github.com/godevsig/adaptiveservice/tree/master/examples/echo/server)

# Discover a service

A service is discovered in the order of scope Process then scope OS
then scope LAN then scope WAN, in the client's configured scopes.

See [Discover API](https://pkg.go.dev/github.com/godevsig/adaptiveservice#Client.Discover)

# Known message

Messages that satisfy Handle() interface are known messages. Typically
server defines Handle() method for every message type it can handle,
then when the known message arrived on one of the transports it is
listening, the message is delivered to one of the workers in which
the message's Handle() is called.
Clients do not define Handle() method, they just send and receive message
in a natural synchronized fashion.

# Send() Recv() and SendRecv()

# Client side multiplexed connection

# Server side auto scale worker pool

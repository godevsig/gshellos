times := import("times")
fmt := import("fmt")

// Part 1: simple async call
var := 0

f1 := func(a,b) { var = 10; return a+b }
f2 := func(a,b,c) { var = 11; return a+b+c }

g1 := go(f1,1,2)
g2 := go(f2,1,2,5)

fmt.println(g1.result()) // 3
fmt.println(g2.result()) // 8
//fmt.println(var) // 10 or 11

// Part 2: 1 client 1 server
reqChan := makechan(8)
repChan := makechan(8)

client := func(interval) {
	reqChan.send("hello")
	for i := 0; true; i++ {
		fmt.println(repChan.recv())
		times.sleep(interval*times.second)
		reqChan.send(i)
	}
}

server := func() {
	for {
		req := reqChan.recv()
		if req == "hello" {
			fmt.println(req)
			repChan.send("world")
		} else {
			repChan.send(req+100)
		}
	}
}

gClient := go(client, 2)
gServer := go(server)

if ok := gClient.wait(5); !ok {
	gClient.abort()
}
gServer.abort()

// Part 3: n clients n servers, channel in channel
sharedReqChan := makechan(128)

client = func(name, interval, timeout) {
	print := func(s) {
		fmt.println(name, s)
	}
	print("started")

	repChan := makechan(1)
	msg := {chan:repChan}

	msg.data = "hello"
	sharedReqChan.send(msg)
	print(repChan.recv())

	for i := 0; i * interval < timeout; i++ {
		msg.data = i
		sharedReqChan.send(msg)
		print(repChan.recv())
		times.sleep(interval*times.second)
	}
}

server = func(name) {
	print := func(s) {
		fmt.println(name, s)
	}
	print("started")

	for {
		req := sharedReqChan.recv()
		if req.data == "hello" {
			req.chan.send("world")
		} else {
			req.chan.send(req.data+100)
		}
	}
}

clients := func() {
	for i :=0; i < 5; i++ {
		go(client, format("client %d: ", i), 1, 4)
	}
}

servers := func() {
	for i :=0; i < 2; i++ {
		go(server, format("server %d: ", i))
	}
}

// After 4 seconds, all clients should have exited normally
gclts := go(clients)
// If servers exit earlier than clients, then clients may be
// blocked forever waiting for the reply chan, because servers
// were aborted with the req fetched from sharedReqChan before
// sending back the reply.
// In such case, do below to abort() the clients manually
//go(func(){times.sleep(6*times.second); gclts.abort()})

// Servers are infinite loop, abort() them after 5 seconds
gsrvs := go(servers)
if ok := gsrvs.wait(5); !ok {
	gsrvs.abort()
}

// Main VM waits here until all the child "go" finish

// If somehow the main VM is stuck, that is because there is
// at least one child VM that has not exited as expected, we
// can do abort() to force exit.
abort()

//output:
//3
//8
//hello
//world
//100
//101

//unordered output:
//client 4: started
//server 0: started
//client 4: world
//client 4: 100
//client 3: started
//client 3: world
//client 3: 100
//client 2: started
//client 2: world
//client 2: 100
//client 0: started
//client 0: world
//client 0: 100
//client 1: started
//client 1: world
//client 1: 100
//server 1: started
//client 1: 101
//client 2: 101
//client 4: 101
//client 0: 101
//client 3: 101
//client 3: 102
//client 0: 102
//client 2: 102
//client 1: 102
//client 4: 102
//client 0: 103
//client 3: 103
//client 2: 103
//client 1: 103
//client 4: 103

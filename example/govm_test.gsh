times := import("times")
fmt := import("fmt")

// part 1: simple async call
var := 0

f1 := func(a,b) { var = 10; return a+b }
f2 := func(a,b,c) { var = 11; return a+b+c }

gvm1 := govm(f1,1,2)
gvm2 := govm(f2,1,2,5)

fmt.println(gvm1.result()) // 3
fmt.println(gvm2.result()) // 8
//fmt.println(var) // 10 or 11

// part 2: 1 client  1 server
reqChan := makechan(8)
repChan := makechan(8)

client := func(interval) {
	reqChan.send("hello")
	i := 0
	for {
		fmt.println(repChan.recv())
		times.sleep(interval*times.second)
		reqChan.send(i)
		i++
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

gvmClient := govm(client, 2)
gvmServer := govm(server)

if ok := gvmClient.wait(5); !ok {
	gvmClient.abort()
}
gvmServer.abort()

fmt.println("client: ", gvmClient.result())
fmt.println("server: ", gvmServer.result())

// part 3: n clients 3 servers of different types
sharedReqChan := makechan(128)

client = func(name, interval, timeout) {
	print := func(s) {
		fmt.println(name, s)
	}

	print("started")
	repChan := makechan(1)
	msg := {data:"hello", chan:repChan}
	sharedReqChan.send(msg)
	print(msg.chan.recv())

	i := 0
	for i * interval < timeout {
		msg := [i, repChan]
		sharedReqChan.send(msg)
		rep := msg[1].recv()
		print(rep)
		i++
		times.sleep(interval*times.second)
	}
}

server = func() {
	print := func(s) {
		fmt.println("server: ", s)
	}
	print("started")

	for {
		req := sharedReqChan.recv()
		if is_map(req) {
			req.chan.send("world")
		} else {
			req[1].send(req[0]+100)
		}
	}
}

asyncServer := func() {
	print := func(s) {
		fmt.println("asyncServer: ", s)
	}
	print("started")

	repChan := makechan(128)

	asyncHandle := func(req) {
		repChan.send([req[0]+100, req[1]])
	}

	responder := func() {
		for {
			rep := repChan.recv()
			rep[1].send(rep[0])
		}
	}

	dispatcher := func() {
		for {
			req := sharedReqChan.recv()
			if is_map(req) {
				req.chan.send("world")
			} else {
				govm(asyncHandle, req)
			}
		}
	}

	govm(responder)
	govm(dispatcher)
}

asyncPoolServer := func() {
	print := func(...s) {
		fmt.println("asyncPoolServer: ", s...)
	}
	print("started")

	repChan := makechan(128)
	reqChan := makechan(128)

	handle := func(req) {
		repChan.send([req[0]+100, req[1]])
	}

	responder := func() {
		for {
			rep := repChan.recv()
			rep[1].send(rep[0])
		}
	}

	worker := func(num) {
		for {
			req := reqChan.recv()
			//print("worker", num)
			handle(req)
		}
	}

	dispatcher := func() {
		for {
			req := sharedReqChan.recv()
			if is_map(req) {
				req.chan.send("world")
			} else {
				reqChan.send(req)
			}
		}
	}

	govm(worker, 1)
	govm(worker, 2)
	govm(worker, 3)

	govm(responder)
	govm(dispatcher)

	times.sleep(10*times.second)
	abort()
}

for i :=0; i < 5; i++ {
	// all clients will exit in 6 seconds
	govm(client, format("client %d: ", i), 1, 6)
}

s := govm(server)
as := govm(asyncServer)
// server and asyncServer will not exit voluntarily,
// so force them to abort from outside after 4 seconds.
// The exit of the 2 servers will not affect the service to the
// clients which will be still running and getting service from
// the remaining asyncPoolServer.
govm(func(){times.sleep(4*times.second); s.abort(); as.abort()})

// asyncPoolServer itself can exit in 10 seconds
govm(asyncPoolServer)

// all the descendent VMs will exit in 10 seconds,
// so we do not need to abort() here.
// main VM will exit after 10 seconds.

// if we want main VM to exit earlier, do below
// will cause main VM to exit in 8 seconds but with
// error: "virtual machine aborted"
//times.sleep(8*times.second); abort()

//output:
//3
//8
//hello
//world
//100
//101
//client: error: "virtual machine aborted"
//server: error: "virtual machine aborted"

//unordered output:
//client 1: started
//asyncPoolServer: started
//client 4: started
//client 2: started
//client 3: started
//server: started
//client 3: world
//client 3: 100
//asyncServer: started
//client 1: world
//client 1: 100
//client 4: world
//client 2: world
//client 2: 100
//client 4: 100
//client 0: started
//client 0: world
//client 0: 100
//client 3: 101
//client 1: 101
//client 4: 101
//client 2: 101
//client 0: 101
//client 4: 102
//client 3: 102
//client 2: 102
//client 1: 102
//client 0: 102
//client 4: 103
//client 3: 103
//client 2: 103
//client 0: 103
//client 1: 103
//client 3: 104
//client 2: 104
//client 1: 104
//client 0: 104
//client 4: 104
//client 2: 105
//client 3: 105
//client 1: 105
//client 0: 105
//client 4: 105

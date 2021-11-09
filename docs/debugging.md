# Interactive debugging

gshell enters interactive mode when starting with no paras and no args.
It executes exactly the same go code, line by line:

```shell
$ gshell
>>
>> fmt.Println("hello, gshell")
hello, gshell
>>
```

- stdlib was pre-loaded, import other packages on demand
- Ctrl+D to exit, Ctrl+C to interrupt the current line

# Example

Enter gshell interactive mode and then issue below code to get your observed IP:

```go
>> import as "github.com/godevsig/adaptiveservice"
>> c := as.NewClient()
>> conn := <-c.Discover("builtin", "IPObserver")
>> var ip string
>> conn.SendRecv(as.GetObservedIP{}, &ip)
>> fmt.Println(ip)
10.182.105.179
>>
```

Of course we should do more error checking, but the above snippet
is already enough for an interactive debugging to get correct result.
The complete code you will put in your final code:

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

- We don't need "observed" IP in scope Process, OS, or LAN, so we create the client
  only in `as.ScopeWAN`, the discovery procedure then searches the wanted service
  {"builtin", "IPObserver"} only in scope WAN.
- We set discover timeout to 3 seconds, meaning that if no such service found in WAN
  network after waiting for 3 seconds, discover API then returns nil connection. That's
  why we should check if `conn == nil`. By default the timeout is -1, means wait forever.
- The connection returned by discover should be closed after use.
- SendRecv() is a blocking API, it sends the request message `as.GetObservedIP{}` and
  waits for reply message - the IP in string type. You should know what type the peer
  will return as the reply, check the service package docs to find out the info. If
  the peer returns error type, that error value will be returned as return value, e.g.
  `err := conn.SendRecv(as.GetObservedIP{}, &ip)` means the server will return error
  if there was something wrong to obtain the client IP, the error will be the stored
  in `err` variable.

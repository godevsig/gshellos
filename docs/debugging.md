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

Of course we should do more error checking in our final code, but the above snippet
is already enough for an interactive debugging to get correct result.
The complete code then you put in

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

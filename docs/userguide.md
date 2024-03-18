- [Start root gshell daemon](#start-root-gshell-daemon)
- [Start none root gshell daemon](#start-none-root-gshell-daemon)
- [Run a task](#run-a-task)
- [Learn help of a task](#learn-help-of-a-task)
- [Stop a task](#stop-a-task)
- [List all services](#list-all-services)
- [Run a task on a remote node](#run-a-task-on-a-remote-node)
- [Check gshell daemon build info](#check-gshell-daemon-build-info)
- [gshell help](#gshell-help)


# Start root gshell daemon
A root [gshell daemon](daemon.md) is the only one that runs with `-root` option in gshell service mesh, which usually provides global service lookup in [WAN scope](adaptiveservice.md) and code repo services.

Below command starts a root(`-root`) gshell daemon with TCP port 11985(`:11985`) providing service registry service and UDP port(`-bcast 9923`) providing service broadcasting, it also provides a code repo service that later `gshell run` commands on local/remote nodes can get source code from 'github.com/godevsig/ghub' at its master branch(`-repo github.com/godevsig/ghub/master`).
```shell
$ alias gsh='bin/gshell'
$ gsh -loglevel info daemon -registry :11985 -bcast 9923 -root -repo github.com/godevsig/ghub/master &
```

# Start none root gshell daemon
A none root gshell daemon is the one runs without `-root` option. Each node that wants to join gshell service mesh must run a gshell daemon, which talks to other nodes in the mesh to exchange service information and execute local/remote tasks.

Below command starts a none root gshell daemon for a node, specifying the root registry address by `-registry 10.10.10.10:11985` and broadcasting port 9923 by `-bcast 9923`. `10.10.10.10` is the public IP that root gshell daemon runs on.
```shell
$ alias gsh='bin/gshell'
$ gsh -loglevel info daemon -registry 10.10.10.10:11985 -bcast 9923 &
```

# Run a task
Once a gshell daemon has been started, users can use `gshell run` commands to run a gshell task.
A gshell task is a plain go path/file located in `gshell repo`, which was configured by daemon's `-repo` option, which can be a local directory or a github/gitlab repo.

`gshell run` will first try to run local go path/file, if there is no such code, it will fetch the code from `gshell repo`.

Below commands first checked the configured code repo, and ran `example/hello`, which returned a GRE(gshell runtime environment) ID `d0872b5a346e`. The the log of the task was checked with `gsh log d0872b5a346e`.  
The task ran in a GRE, the GRE ran in group `nrtikd-$version`, `$version` is the gshell version.  
Tasks can run in the same group and separate groups.  

Use `gsh repo ls` to see the all the go files available in the code repo service.
```shell
$ alias gsh='bin/gshell'
$ gsh repo
github.com/godevsig/ghub master

$ gsh run example/hello
d0872b5a346e

$ gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
d0872b5a346e  nrtikd-v2.2.0       hello               2024/03/18 06:33:30  exited:OK  16.833528ms

$ gsh log d0872b5a346e
Hello, world!

$ gsh repo ls
.github
.gitignore
LICENSE
Makefile
README.md
benchmark
debug
example
go.mod
go.sum
perf
render
util
```
Further info please check `gsh -h`, `gsh repo -h`, `gsh run -h`.

# Learn help of a task
Before running a task, often we want to check its help message. Here we add `-i` to view the output of `util/fileserver/fileserver.go -h` instead of letting it outputs to log file. We also use `-rm` to automatically remove the GRE because we just want to take a little look of the help message, we don't want to keep its existence in `gsh ps`.
```shell
$ gsh run -i -rm util/fileserver/fileserver.go -h
Usage:
  -dir string
        absolute directory path to be served
  -logLevel string
        debug/info/warn/error (default "info")
  -port string
        set server port, default 0 means alloced by net Listener (default "0")
  -title string
        title of file server (default "file server")
```

# Stop a task
Use `gshell stop` to stop a running task. If there is a `Stop()` function defined in user code, it will be called to cancel the task in the way user wanted, otherwise the task will be aborted forcibly, the status will be "aborted" in the later case.

In below commands, we first started `testdata/sleep.go` with its source code showed below:
```go
package main

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

var stopChan = make(chan struct{})

// Stop stops the app
func Stop() {
	fmt.Println("stopping...")
	close(stopChan)
}

func main() {
	fmt.Println(os.Args)
	cnt := 30
	if len(os.Args) >= 2 {
		if i, err := strconv.Atoi(os.Args[1]); err == nil {
			cnt = i
		}
	}
	fmt.Printf("sleeping %d seconds\n", cnt)
	go func() {
		time.Sleep(time.Duration(cnt) * time.Second)
		stopChan <- struct{}{}
	}()

	_, ok := <-stopChan
	if ok {
		fmt.Println("wakeup")
	} else {
		fmt.Println("canceled")
	}
}
```

Then we checked its status by `gsh ps`, and it was running. Then we stopped the task by `gsh stop f7264babdaf3`, the task was stopped and became "cancelled". `gsh log f7264babdaf3` showed its output, because it was terminated by user, so it has "stopping..." and "canceled" prints.
```shell
$ alias gsh='bin/gshell'
$ gsh run testdata/sleep.go
f7264babdaf3

$ gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
f7264babdaf3  fmueuo-v2.2.0       sleep               2024/03/18 06:55:12  running    1.538414904s
d0872b5a346e  nrtikd-v2.2.0       hello               2024/03/18 06:33:30  exited:OK  16.833528ms

$ gsh stop f7264babdaf3
f7264babdaf3
stopped

$ gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
f7264babdaf3  fmueuo-v2.2.0       sleep               2024/03/18 06:55:12  cancelled  9.589076037s
d0872b5a346e  nrtikd-v2.2.0       hello               2024/03/18 06:33:30  exited:OK  16.833528ms

$ gsh ps f7264babdaf3
GRE ID       : f7264babdaf3
IN GROUP     : fmueuo-v2.2.0
NAME         : sleep
ARGS         : [testdata/sleep.go]
REQUESTED BY : 2de2a6888ef3
STATUS       : cancelled
RESTARTED    : 0
START AT     : 2024-03-18 06:55:12.849548461 +0000 UTC
END AT       : 2024-03-18 06:55:22.438624498 +0000 UTC
ERROR        :

$ gsh log f7264babdaf3
[testdata/sleep.go]
sleeping 30 seconds
stopping...
canceled
```

# List all services
`gshell list` can list all services available in the gshell service mesh. Services are identified by `PUBLISHER` and `SERVICE`, where `PUBLISHER` usually is the organization/team and `SERVICE` is the project.

Column `WLOP(SCOPE)` shows the avaliability of the service in scopes [WLOP: Wan/Lan/OS/Process](adaptiveservice.md), with `1` means the service is avaliable.

It supports wildcard matching.
```shell
$ gsh list
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
builtin                   IPObserver                self          1111
builtin                   LANRegistry               self            11
builtin                   messageTracing            244de4b51737   100
builtin                   messageTracing            self          1111
builtin                   providerInfo              self            11
builtin                   registryInfo              self            11
builtin                   reverseProxy              fa163ecfb434   100
builtin                   reverseProxy              self          1100
builtin                   serviceLister             self            11
godevsig                  codeRepo                  self          1111
godevsig                  grg-nrtikd-v2.2.0         self            10
godevsig                  gshellDaemon              244de4b51737   100
godevsig                  gshellDaemon              fa163ecfb434   100
godevsig                  gshellDaemon              self          1111

$ gsh list -s "grg*"
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
godevsig                  grg-nrtikd-v2.2.0         self            10
```

# Run a task on a remote node
Some sub commands like `gshell run` accepts a `-p` option to operate sub commands on the remote node identified by [Provider ID](adaptiveservice.md).

Here we took provider ID `244de4b51737` from above `gsh list` as the remote node, we ran `example/hello` and checked its output on that remote node.

This ability enables gshell to deploy gshell tasks on any distributed gshell node and debug it in remotely.
```shell
$ gsh -p 244de4b51737 ps
GRE ID        IN GROUP            NAME                START AT             STATUS

$ gsh -p 244de4b51737 run example/hello
5105f2aae7c0

$ gsh -p 244de4b51737 ps
GRE ID        IN GROUP            NAME                START AT             STATUS
5105f2aae7c0  ahaoxs-v2.2.0       hello               2024/03/18 07:52:34  exited:OK  11.465698ms

$ gsh -p 244de4b51737 log 5105f2aae7c0
Hello, world!
```

# Check gshell daemon build info
gshell is a single go binary, build tags indicate the features built in the binary.
```shell
$ gsh info
Version: v2.2.0
Commit: 04d943ea7d9c1630c3ee46b07ac35ef28d2f9e50
Build time: 2024.03.18 04:50:15
Build tags: stdbase,stdcommon,stdruntime,stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding,stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdtext,stdunicode,debug,adaptiveservice,shell,log,pidinfo,asbench,echo,fileserver,topidchart,docit,recorder
```

# gshell help
```shell
  gshell is a simple pure golang service framework for linux devices.

  Running a gshell daemon on a board/VM/container makes it a node in the gshell service mesh.
  Each node has an unique provider ID.

  Each job runs in one dedicated GRE(Gshell Runtime Environment) which runs in a GRG(Gshell Runtime Group).
  GREs can be grouped into one named GRG for better performance.

  gshell enters interactive mode if no options and no commands provided.

Usage: [OPTIONS] COMMAND ...
OPTIONS:
  -l, --loglevel
        loglevel, debug/info/warn/error (default "error")
  -p, --provider
        provider ID, run following command on the remote node with this ID (default "self")
  --trace
        Comma seprated messages to be traced, use "gshell mtrace list" to show possbile values
COMMANDS:
  id
        Print self provider ID
  exec <path[/file.go]> [args...]
        Run go file(s) in a local GRE
  daemon [options]
        Start local gshell daemon
  list [options]
        List services in all scopes
  repo [ls [path]]
        List contens of the code repo seen on local/remote node
  run [options] <path[/file.go]> [args...]
        Try local code path[/file.go] first or fetch the code from `gshell repo`,
        and run it in a new GRE in specified GRG on local/remote node
        Use `gshell repo ls [path]` to see available code files
  kill [options] names ...
        Terminate the named GRG(s) on local/remote node
        wildcard(*) is supported
  ps [options] [GRE IDs ...|names ...]
        Show jobs by GRE ID or name on local/remote node
  stop [options] [GRE IDs ...|names ...]
        Stop one or more jobs on local/remote node
  rm [options] [GRE IDs ...|names ...]
        Remove one or more stopped jobs on local/remote node
  start [options] [GRE IDs ...|names ...]
        Start one or more stopped jobs on local/remote node
  info
        Show gshell info on local/remote node
  log [options] <daemon|grg|GRE ID>
        Print target log on local/remote node
  joblist [options] <save|load>
        Save all current jobs to file or load them to run on local/remote node
  mtrace <list>
        List traceable message types
```

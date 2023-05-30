# Command line usage

## References

```
$ alias gsh='bin/gshell'

$ gsh -loglevel info daemon -wd rootregistry -registry :11985 -bcast 9923 -root -repo github.com/godevsig/grepo/master &

$ gsh list
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
builtin                   IPObserver                self          1111
builtin                   LANRegistry               self            11
builtin                   providerInfo              self            11
builtin                   registryInfo              self            11
builtin                   reverseProxy              self          1100
builtin                   serviceLister             self            11
godevsig                  codeRepo                  self          1111
godevsig                  grg-bemfus-v23.05.25      self            10
godevsig                  grg-ezbyic-v23.05.25      self            10
godevsig                  grg-jbczuj-v23.05.25      self            10
godevsig                  grg-nvemyl-v23.05.25      self            10
godevsig                  grg-ohhrco-v23.05.25      self            10
godevsig                  grg-rjahbf-v23.05.25      self            10
godevsig                  grg-rydpft-v23.05.25      self            10
godevsig                  grg-rzujxb-v23.05.25      self            10
godevsig                  gshellDaemon              00198f937353  1000
godevsig                  gshellDaemon              00198fbe8407  1000
godevsig                  gshellDaemon              00198fc8a52b  1000
godevsig                  gshellDaemon              00198fc8a549  1000
godevsig                  gshellDaemon              0847d0094b3f  1000
godevsig                  gshellDaemon              0847d00b7632  1000
godevsig                  gshellDaemon              20677ce3ec48  1000
godevsig                  gshellDaemon              781735a222d9  1000
godevsig                  gshellDaemon              self          1111
godevsig                  updater                   self          1111
platform                  docit                     self          1110
platform                  topidchart                self          1110

$ gsh list -s "grg*"
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
godevsig                  grg-bemfus-v23.05.25      self            10
godevsig                  grg-ezbyic-v23.05.25      self            10
godevsig                  grg-jbczuj-v23.05.25      self            10
godevsig                  grg-nvemyl-v23.05.25      self            10
godevsig                  grg-ohhrco-v23.05.25      self            10
godevsig                  grg-rjahbf-v23.05.25      self            10
godevsig                  grg-rydpft-v23.05.25      self            10
godevsig                  grg-rzujxb-v23.05.25      self            10

$ gsh repo ls
.github
benchmark
debug
example
lib
perf
render
util
.gitignore
.gitlab-ci.yml
Makefile
README.md
go.mod
go.sum


$ gsh run -i util/fileserver/cmd/fileserver.go -h
Usage:
  -dir string
        absolute directory path to be served
  -logLevel string
        debug/info/warn/error (default "info")
  -port string
        set server port, default 0 means alloced by net Listener (default "0")
  -title string
        title of file server (default "file server")

$ gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
f27e0071e35f  bemfus-v23.05.25    fileserver          2023/05/25 04:09:52  running    125h33m8.363278311s
f987918bf1bf  ohhrco-v23.05.25    topid               2023/05/30 09:06:05  running    36m55.012998167s
63055dfedf44  rydpft-v23.05.25    topidchart          2023/05/25 04:09:30  running    125h33m30.795757737s
9345dc2ce195  ezbyic-v23.05.25    docit               2023/05/25 04:09:28  running    125h33m31.961285295s


$ gsh info
Version: v23.05.25
Build tags: stdbase,stdcommon,stdruntime,stdext,stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding,stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdtext,stdunicode,debug,adaptiveservice,shell,log,pidinfo,asbench,echo,fileserver,topidchart,docit,recorder
Commit: 1e0a1f5a2d49a6dfe1baa7663d95e784b2d291c0

$ gsh -h
  gshell is gshellos based service management tool.
  gshellos is a simple pure golang service framework for linux devices.
  A system with one gshell daemon running is a node in the
service network, each node has an unique provider ID.
  Each job runs in one dedicated GRE(Gshell Runtime Environment)
which by default runs in a random GRG(Gshell Runtime Group). GREs can be
grouped into one named GRG for better performance.
  gshell enters interactive mode if no options and no commands provided.

Usage: [OPTIONS] COMMAND ...
OPTIONS:
  -l, --loglevel
        loglevel, debug/info/warn/error (default "error")
  -p, --provider
        provider ID, run following command on the remote node with this ID (default "self")
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
        list contens of the central code repo
  run [options] <path[/file.go]> [args...]
        fetch code path[/file.go] from `gshell repo`
        and run the go file(s) in a new GRE in specified GRG on local/remote node
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
```

## Remote deploy go apps/services
Supply the remote provider ID to gshell:
```
$ gsh list
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
builtin                   IPObserver                self          1111
builtin                   LANRegistry               self            11
builtin                   providerInfo              self            11
builtin                   registryInfo              self            11
builtin                   reverseProxy              self          1100
builtin                   serviceLister             self            11
godevsig                  codeRepo                  self          1111
godevsig                  grg-bemfus-v23.05.25      self            10
godevsig                  grg-ezbyic-v23.05.25      self            10
godevsig                  grg-jbczuj-v23.05.25      self            10
godevsig                  grg-nvemyl-v23.05.25      self            10
godevsig                  grg-rjahbf-v23.05.25      self            10
godevsig                  grg-rydpft-v23.05.25      self            10
godevsig                  grg-rzujxb-v23.05.25      self            10
godevsig                  gshellDaemon              00198f937353  1000
godevsig                  gshellDaemon              00198fbe8407  1000
godevsig                  gshellDaemon              00198fc8a52b  1000
godevsig                  gshellDaemon              00198fc8a549  1000
godevsig                  gshellDaemon              0847d0094b3f  1000
godevsig                  gshellDaemon              0847d00b7632  1000
godevsig                  gshellDaemon              20677ce3ec48  1000
godevsig                  gshellDaemon              781735a222d9  1000
godevsig                  gshellDaemon              self          1111
godevsig                  updater                   self          1111
platform                  docit                     self          1110
platform                  topidchart                self          1110

$ gsh -p 781735a222d9 run perf/topid/topid.go -chart -snapshot -sys -i 5 -tag lsr2306_dot1xTime
aa6c463e97fb

$ gsh -p 781735a222d9 ps
GRE ID        IN GROUP            NAME                START AT             STATUS
aa6c463e97fb  durvzl-v23.05.25    topid               2023/05/30 09:25:02  running    7.249270302s
595218a30dbd  tfhgbe-v23.05.25    topid               2023/05/28 23:16:21  exited:OK  5h2m2.866944639s
```

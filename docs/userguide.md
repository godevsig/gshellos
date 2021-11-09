# Command line usage

# References

```
$ alias gsh='bin/gshell'

$ gsh -v
v1.1.3

$ gsh -wd rootregistry -loglevel info daemon -registry :11985 -bcast 9923 -root -repogithub.com/godevsig/grepo/master &

$ gsh list
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
builtin                   IPObserver                self          1000
builtin                   LANRegistry               self            11
builtin                   providerInfo              self            11
builtin                   registryInfo              self            11
builtin                   reverseProxy              fa163ecfb434   100
builtin                   reverseProxy              self          1100
builtin                   serviceLister             self            11
godevsig                  codeRepo                  self          1111
godevsig                  grg-mebasz.v1.1.3         self            10
godevsig                  grg-vngfqq.v1.1.3         self            10
godevsig                  grg-yakbro.v1.1.3         self            10
godevsig                  gshellDaemon              fa163ecfb434   100
godevsig                  gshellDaemon              self          1111

$ gsh list -s "grg*"
PUBLISHER                 SERVICE                   PROVIDER      WLOP(SCOPE)
godevsig                  grg-mebasz.v1.1.3         self            10
godevsig                  grg-vngfqq.v1.1.3         self            10
godevsig                  grg-yakbro.v1.1.3         self            10

$ gsh run -i app-http/fileserver/fileserver.go -h
Usage:
-dir string
directory to be served
-port string
port for http (default "8088")

$ gsh run app-http/fileserver/fileserver.go -dir .
521e1d1b4d0b
$ gsh log 521e1d1b4d0b
file server for . @ :8088
file server running...

$ gsh ps
GRE ID        IN GROUP            NAME                START AT             STATUS
521e1d1b4d0b  vngfqq.v1.1.3       fileserver          2021/11/09 16:46:11  running    2m59.265772722s

$ gsh info
Version: v1.1.3
Build tags: stdbase,stdcommon,stdruntime,stdext,stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding,stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdtext,stdunicode,debug,adaptiveservice,shell,log
Commit: 6f579e5b1ad853c5789f946baf17585cbf99c68f

$ gsh -h
  gshell is gshellos based service management tool.
  gshellos is a simple pure golang service framework for linux devices.
  One gshell daemon must have been started in the system to join the
service network with an unique provider ID.
  Each app/service is run in one dedicated GRE(Gshell Runtime Environment)
which by default runs in a random GRG(Gshell Runtime Group). GREs can be
grouped into one named GRG for better performance.
  gshell enters interactive mode if no options and no commands provided.

Usage: [OPTIONS] COMMAND ...
OPTIONS:
  -loglevel string
        debug/info/warn/error (default "error")
  -p string
        provider ID to specify a remote system (default "self")
  -wd string
        set working directory (default "/var/tmp/gshell")
COMMANDS:
  daemon [options]
        Start local gshell daemon
  list [options]
        List services in all scopes
  id
        Print self provider ID
  exec <file.go> [args...]
        Run <file.go> in a local GRE
  repo
        Print central code repo https address
  kill [options] names ...
        Terminate the named GRG(s) on local/remote system
  run [options] <file.go> [args...]
        Look for file.go in local file system or else in `gshell repo`,
        run it in a new GRE in specified GRG on local/remote system
  ps [options] [GRE IDs ...|names ...]
        Show GRE instances by GRE ID or name on local/remote system
  stop [options] [GRE IDs ...|names ...]
        Call `func Stop()` to stop one or more GREs on local/remote system
  rm [options] [GRE IDs ...|names ...]
        Remove one or more stopped GREs on local/remote system
  restart [options] [GRE IDs ...|names ...]
        Restart one or more stopped GREs on local/remote system
  info
        Show gshell info on local/remote system
  log [options] <daemon|grg|GRE ID>
        Print target log on local/remote system
```

## Remote deploy go apps/services

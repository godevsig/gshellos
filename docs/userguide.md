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
```

## Remote deploy go apps/services

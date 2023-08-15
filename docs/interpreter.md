# Interpreted mode VS compiled mode

gshell supports both compiled code and interpreted code execution:

- Libraries are compiled into machine code binary using standard Golang compiler
- Other go files can be interpred by [Golang interpreter yaegi](https://github.com/traefik/yaegi)
- Compiled code can be directly called by interpreted code

For example, `gshell run testdata/fileserver.go` reads and interpres `testdata/fileserver.go`, but
the execution of `fmt.Println()` and `http.FileServer()` and other functions from stdlib is in binary
mode, because all those functions were already compiled and put into the gshell binary. In this way,
interpreted code controls the bussiness logic which often changes, compiled code performs something
that is stable to requirements.

- put your app logic code interpreted
- put your library code compiled
- put your performance sensitive code compiled
- adjust flexibility and performance balance according to your needs

See `testdata/fileserver.go`

```go
func Start(args []string) (err error) {
	...
	fmt.Printf("file server for %s @ :%s\n", *dir, *port)

	lnr, err = net.Listen("tcp", ":"+*port)
	if err != nil {
		return err
	}
	defer func() {
		lnr.Close()
		fmt.Println("file server stopped")
	}()

	fs := http.FileServer(http.Dir(*dir))
	fmt.Println("file server running...")
	if err := http.Serve(lnr, fs); err != nil {
		return err
	}

	return nil
}
```

## Add library callable by interpreted code

To be able to compile a library and call the library binary from other interpreted code, the
library must be put into gshell extension using `go:generate`.
See `extension/extension.go`

```go
//go:generate ../cmd/extract/extract -name extension -tag adaptiveservice github.com/godevsig/adaptiveservice
//go:generate ../cmd/extract/extract -name extension -tag shell github.com/godevsig/glib/sys/shell
//go:generate ../cmd/extract/extract -name extension -tag log github.com/godevsig/glib/sys/log
```

If in the `go:generate` command a tag was used, you should decide which build type should include the tag
and add it into Makefile.
See below.

## Build types

We use `make full` or `make lite` to build gshell binary, full and lite are the build type. As the name
indicating, full build is used on large machines like X86 servers, lite build is used on embedded boxes
with lite feature set to produce smaller binary size.

Current feature set table:

| lib tag         | lite | full |
| --------------- | ---- | ---- |
| stdbase         | Y    | Y    |
| stdcommon       | Y    | Y    |
| stdruntime      | Y    | Y    |
| stdext          | N    | Y    |
| stdarchive      | N    | Y    |
| stdcompress     | N    | Y    |
| stdcontainer    | N    | Y    |
| stdcrypto       | N    | Y    |
| stddatabase     | N    | Y    |
| stdencoding     | N    | Y    |
| stdhash         | N    | Y    |
| stdhtml         | N    | Y    |
| stdlog          | N    | Y    |
| stdmath         | N    | Y    |
| stdhttp         | N    | Y    |
| stdmail         | N    | Y    |
| stdrpc          | N    | Y    |
| stdregexp       | N    | Y    |
| stdtext         | N    | Y    |
| stdunicode      | N    | Y    |
| debug           | N    | Y    |
| adaptiveservice | Y    | Y    |
| shell           | Y    | Y    |
| log             | Y    | Y    |

The table will change over time, use `gshell info` to check the tags in use:

```
$ gshell info
Version: v1.1.3
Build tags: stdbase,stdcommon,stdruntime,stdext,stdarchive,stdcompress,stdcontainer,stdcrypto,stddatabase,stdencoding,stdhash,stdhtml,stdlog,stdmath,stdhttp,stdmail,stdrpc,stdregexp,stdtext,stdunicode,debug,adaptiveservice,shell,log
Commit: 6f579e5b1ad853c5789f946baf17585cbf99c68f
```

### stdbase

Includes

```
bufio bytes context errors expvar flag fmt io io/fs io/ioutil net os os/exec os/signal os/user path path/filepath reflect sort strconv strings sync sync/atomic time
```

### stdcommon

Includes

```
archive/tar compress/gzip crypto/md5 crypto/rand encoding/binary encoding/hex encoding/json net/http
```

### stdruntime

Includes

```
runtime runtime/debug runtime/metrics runtime/pprof runtime/trace
```

### other standard libraries

See `stdlib/stdlib.go`

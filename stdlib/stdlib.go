// Package stdlib provides wrappers of standard library packages to be imported natively in Yaegi.
package stdlib

import "reflect"

// Symbols variable stores the map of stdlib symbols per package.
var Symbols = map[string]map[string]reflect.Value{}

func init() {
	Symbols["github.com/godevsig/gshellos/stdlib"] = map[string]reflect.Value{
		"Symbols": reflect.ValueOf(Symbols),
	}
}

// Provide access to go standard library (http://golang.org/pkg/)
// go list std | grep -v internal | grep -v '\.' | grep -v unsafe | grep -v syscall

//go:generate ../cmd/extract/extract -name stdlib -tag stdbase bufio bytes context errors flag fmt
//go:generate ../cmd/extract/extract -name stdlib -tag stdbase io io/fs io/ioutil net os os/exec os/signal os/user
//go:generate ../cmd/extract/extract -name stdlib -tag stdbase path path/filepath reflect
//go:generate ../cmd/extract/extract -name stdlib -tag stdbase sort strconv strings sync sync/atomic time

//go:generate ../cmd/extract/extract -name stdlib -tag stdcommon archive/tar compress/gzip crypto/md5 crypto/rand
//go:generate ../cmd/extract/extract -name stdlib -tag stdcommon encoding/binary encoding/gob encoding/hex encoding/json

//go:generate ../cmd/extract/extract -name stdlib -tag stdext embed plugin

//go:generate ../cmd/extract/extract -name stdlib -tag stdarchive archive/zip

//go:generate ../cmd/extract/extract -name stdlib -tag stdcompress compress/bzip2 compress/flate compress/lzw compress/zlib

//go:generate ../cmd/extract/extract -name stdlib -tag stdcontainer container/heap container/list container/ring

//go:generate ../cmd/extract/extract -name stdlib -tag stdcrypto crypto crypto/aes crypto/cipher crypto/des crypto/dsa crypto/ecdsa crypto/ed25519
//go:generate ../cmd/extract/extract -name stdlib -tag stdcrypto crypto/elliptic crypto/hmac crypto/rc4 crypto/rsa crypto/sha1
//go:generate ../cmd/extract/extract -name stdlib -tag stdcrypto crypto/sha256 crypto/sha512 crypto/subtle crypto/tls crypto/x509 crypto/x509/pkix

//go:generate ../cmd/extract/extract -name stdlib -tag stddatabase database/sql database/sql/driver

//go:generate ../cmd/extract/extract -name stdlib -tag stddebug debug/dwarf debug/elf debug/gosym debug/macho debug/pe debug/plan9obj

//go:generate ../cmd/extract/extract -name stdlib -tag stdencoding encoding encoding/ascii85 encoding/asn1 encoding/base32 encoding/base64
//go:generate ../cmd/extract/extract -name stdlib -tag stdencoding encoding/csv encoding/pem encoding/xml

//go:generate ../cmd/extract/extract -name stdlib -tag stdgo go/ast go/build go/build/constraint go/constant go/doc go/format go/importer go/parser go/printer go/scanner go/token go/types

//go:generate ../cmd/extract/extract -name stdlib -tag stdhash hash hash/adler32 hash/crc32 hash/crc64 hash/fnv hash/maphash

//go:generate ../cmd/extract/extract -name stdlib -tag stdhtml html html/template

//go:generate ../cmd/extract/extract -name stdlib -tag stdimage image image/color image/color/palette image/draw image/gif image/jpeg image/png

//go:generate ../cmd/extract/extract -name stdlib -tag stdalgorithm index/suffixarray

//go:generate ../cmd/extract/extract -name stdlib -tag stdlog log log/syslog

//go:generate ../cmd/extract/extract -name stdlib -tag stdmath math math/big math/bits math/cmplx math/rand

//go:generate ../cmd/extract/extract -name stdlib -tag stdmime mime mime/multipart mime/quotedprintable

//go:generate ../cmd/extract/extract -name stdlib -tag stdhttp net/http net/http/cgi net/http/cookiejar net/http/fcgi net/http/httptest net/http/httptrace
//go:generate ../cmd/extract/extract -name stdlib -tag stdhttp net/http/httputil net/http/pprof net/textproto net/url expvar
//go:generate ../cmd/extract/extract -name stdlib -tag stdmail net/mail net/smtp

//go:generate ../cmd/extract/extract -name stdlib -tag stdrpc net/rpc net/rpc/jsonrpc

//go:generate ../cmd/extract/extract -name stdlib -tag stdregexp regexp regexp/syntax

//go:generate ../cmd/extract/extract -name stdlib -tag stdruntime runtime runtime/debug runtime/metrics runtime/pprof runtime/trace
//go:generate sed -i "/NumCgoCall/d" runtime.go
//go:generate sed -i "/SetCgoTraceback/d" runtime.go

//go:generate ../cmd/extract/extract -name stdlib -tag stdtesting testing testing/fstest testing/iotest testing/quick

//go:generate ../cmd/extract/extract -name stdlib -tag stdtext text/scanner text/tabwriter text/template text/template/parse

//go:generate ../cmd/extract/extract -name stdlib -tag stdunicode unicode unicode/utf16 unicode/utf8

//go:generate ../cmd/extract/extract -name stdlib -tag stdother time/tzdata

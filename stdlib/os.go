// Code generated by 'yaegi extract os'. DO NOT EDIT.

// +build go1.16,!go1.17,stdbase

package stdlib

import (
	"go/constant"
	"go/token"
	"io/fs"
	"os"
	"reflect"
	"time"
)

func init() {
	Symbols["os/os"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"Args":                reflect.ValueOf(&os.Args).Elem(),
		"Chdir":               reflect.ValueOf(os.Chdir),
		"Chmod":               reflect.ValueOf(os.Chmod),
		"Chown":               reflect.ValueOf(os.Chown),
		"Chtimes":             reflect.ValueOf(os.Chtimes),
		"Clearenv":            reflect.ValueOf(os.Clearenv),
		"Create":              reflect.ValueOf(os.Create),
		"CreateTemp":          reflect.ValueOf(os.CreateTemp),
		"DevNull":             reflect.ValueOf(constant.MakeFromLiteral("\"/dev/null\"", token.STRING, 0)),
		"DirFS":               reflect.ValueOf(os.DirFS),
		"Environ":             reflect.ValueOf(os.Environ),
		"ErrClosed":           reflect.ValueOf(&os.ErrClosed).Elem(),
		"ErrDeadlineExceeded": reflect.ValueOf(&os.ErrDeadlineExceeded).Elem(),
		"ErrExist":            reflect.ValueOf(&os.ErrExist).Elem(),
		"ErrInvalid":          reflect.ValueOf(&os.ErrInvalid).Elem(),
		"ErrNoDeadline":       reflect.ValueOf(&os.ErrNoDeadline).Elem(),
		"ErrNotExist":         reflect.ValueOf(&os.ErrNotExist).Elem(),
		"ErrPermission":       reflect.ValueOf(&os.ErrPermission).Elem(),
		"ErrProcessDone":      reflect.ValueOf(&os.ErrProcessDone).Elem(),
		"Executable":          reflect.ValueOf(os.Executable),
		"Exit":                reflect.ValueOf(osExit),
		"Expand":              reflect.ValueOf(os.Expand),
		"ExpandEnv":           reflect.ValueOf(os.ExpandEnv),
		"FindProcess":         reflect.ValueOf(osFindProcess),
		"Getegid":             reflect.ValueOf(os.Getegid),
		"Getenv":              reflect.ValueOf(os.Getenv),
		"Geteuid":             reflect.ValueOf(os.Geteuid),
		"Getgid":              reflect.ValueOf(os.Getgid),
		"Getgroups":           reflect.ValueOf(os.Getgroups),
		"Getpagesize":         reflect.ValueOf(os.Getpagesize),
		"Getpid":              reflect.ValueOf(os.Getpid),
		"Getppid":             reflect.ValueOf(os.Getppid),
		"Getuid":              reflect.ValueOf(os.Getuid),
		"Getwd":               reflect.ValueOf(os.Getwd),
		"Hostname":            reflect.ValueOf(os.Hostname),
		"Interrupt":           reflect.ValueOf(&os.Interrupt).Elem(),
		"IsExist":             reflect.ValueOf(os.IsExist),
		"IsNotExist":          reflect.ValueOf(os.IsNotExist),
		"IsPathSeparator":     reflect.ValueOf(os.IsPathSeparator),
		"IsPermission":        reflect.ValueOf(os.IsPermission),
		"IsTimeout":           reflect.ValueOf(os.IsTimeout),
		"Kill":                reflect.ValueOf(&os.Kill).Elem(),
		"Lchown":              reflect.ValueOf(os.Lchown),
		"Link":                reflect.ValueOf(os.Link),
		"LookupEnv":           reflect.ValueOf(os.LookupEnv),
		"Lstat":               reflect.ValueOf(os.Lstat),
		"Mkdir":               reflect.ValueOf(os.Mkdir),
		"MkdirAll":            reflect.ValueOf(os.MkdirAll),
		"MkdirTemp":           reflect.ValueOf(os.MkdirTemp),
		"ModeAppend":          reflect.ValueOf(os.ModeAppend),
		"ModeCharDevice":      reflect.ValueOf(os.ModeCharDevice),
		"ModeDevice":          reflect.ValueOf(os.ModeDevice),
		"ModeDir":             reflect.ValueOf(os.ModeDir),
		"ModeExclusive":       reflect.ValueOf(os.ModeExclusive),
		"ModeIrregular":       reflect.ValueOf(os.ModeIrregular),
		"ModeNamedPipe":       reflect.ValueOf(os.ModeNamedPipe),
		"ModePerm":            reflect.ValueOf(os.ModePerm),
		"ModeSetgid":          reflect.ValueOf(os.ModeSetgid),
		"ModeSetuid":          reflect.ValueOf(os.ModeSetuid),
		"ModeSocket":          reflect.ValueOf(os.ModeSocket),
		"ModeSticky":          reflect.ValueOf(os.ModeSticky),
		"ModeSymlink":         reflect.ValueOf(os.ModeSymlink),
		"ModeTemporary":       reflect.ValueOf(os.ModeTemporary),
		"ModeType":            reflect.ValueOf(os.ModeType),
		"NewFile":             reflect.ValueOf(os.NewFile),
		"NewSyscallError":     reflect.ValueOf(os.NewSyscallError),
		"O_APPEND":            reflect.ValueOf(os.O_APPEND),
		"O_CREATE":            reflect.ValueOf(os.O_CREATE),
		"O_EXCL":              reflect.ValueOf(os.O_EXCL),
		"O_RDONLY":            reflect.ValueOf(os.O_RDONLY),
		"O_RDWR":              reflect.ValueOf(os.O_RDWR),
		"O_SYNC":              reflect.ValueOf(os.O_SYNC),
		"O_TRUNC":             reflect.ValueOf(os.O_TRUNC),
		"O_WRONLY":            reflect.ValueOf(os.O_WRONLY),
		"Open":                reflect.ValueOf(os.Open),
		"OpenFile":            reflect.ValueOf(os.OpenFile),
		"PathListSeparator":   reflect.ValueOf(constant.MakeFromLiteral("58", token.INT, 0)),
		"PathSeparator":       reflect.ValueOf(constant.MakeFromLiteral("47", token.INT, 0)),
		"Pipe":                reflect.ValueOf(os.Pipe),
		"ReadDir":             reflect.ValueOf(os.ReadDir),
		"ReadFile":            reflect.ValueOf(os.ReadFile),
		"Readlink":            reflect.ValueOf(os.Readlink),
		"Remove":              reflect.ValueOf(os.Remove),
		"RemoveAll":           reflect.ValueOf(os.RemoveAll),
		"Rename":              reflect.ValueOf(os.Rename),
		"SEEK_CUR":            reflect.ValueOf(os.SEEK_CUR),
		"SEEK_END":            reflect.ValueOf(os.SEEK_END),
		"SEEK_SET":            reflect.ValueOf(os.SEEK_SET),
		"SameFile":            reflect.ValueOf(os.SameFile),
		"Setenv":              reflect.ValueOf(os.Setenv),
		"StartProcess":        reflect.ValueOf(os.StartProcess),
		"Stat":                reflect.ValueOf(os.Stat),
		"Stderr":              reflect.ValueOf(&os.Stderr).Elem(),
		"Stdin":               reflect.ValueOf(&os.Stdin).Elem(),
		"Stdout":              reflect.ValueOf(&os.Stdout).Elem(),
		"Symlink":             reflect.ValueOf(os.Symlink),
		"TempDir":             reflect.ValueOf(os.TempDir),
		"Truncate":            reflect.ValueOf(os.Truncate),
		"Unsetenv":            reflect.ValueOf(os.Unsetenv),
		"UserCacheDir":        reflect.ValueOf(os.UserCacheDir),
		"UserConfigDir":       reflect.ValueOf(os.UserConfigDir),
		"UserHomeDir":         reflect.ValueOf(os.UserHomeDir),
		"WriteFile":           reflect.ValueOf(os.WriteFile),

		// type definitions
		"DirEntry":     reflect.ValueOf((*os.DirEntry)(nil)),
		"File":         reflect.ValueOf((*os.File)(nil)),
		"FileInfo":     reflect.ValueOf((*os.FileInfo)(nil)),
		"FileMode":     reflect.ValueOf((*os.FileMode)(nil)),
		"LinkError":    reflect.ValueOf((*os.LinkError)(nil)),
		"PathError":    reflect.ValueOf((*os.PathError)(nil)),
		"ProcAttr":     reflect.ValueOf((*os.ProcAttr)(nil)),
		"Process":      reflect.ValueOf((*os.Process)(nil)),
		"ProcessState": reflect.ValueOf((*os.ProcessState)(nil)),
		"Signal":       reflect.ValueOf((*os.Signal)(nil)),
		"SyscallError": reflect.ValueOf((*os.SyscallError)(nil)),

		// interface wrapper definitions
		"_DirEntry": reflect.ValueOf((*_os_DirEntry)(nil)),
		"_FileInfo": reflect.ValueOf((*_os_FileInfo)(nil)),
		"_Signal":   reflect.ValueOf((*_os_Signal)(nil)),
	}
}

// _os_DirEntry is an interface wrapper for DirEntry type
type _os_DirEntry struct {
	IValue interface{}
	WInfo  func() (fs.FileInfo, error)
	WIsDir func() bool
	WName  func() string
	WType  func() fs.FileMode
}

func (W _os_DirEntry) Info() (fs.FileInfo, error) { return W.WInfo() }
func (W _os_DirEntry) IsDir() bool                { return W.WIsDir() }
func (W _os_DirEntry) Name() string               { return W.WName() }
func (W _os_DirEntry) Type() fs.FileMode          { return W.WType() }

// _os_FileInfo is an interface wrapper for FileInfo type
type _os_FileInfo struct {
	IValue   interface{}
	WIsDir   func() bool
	WModTime func() time.Time
	WMode    func() fs.FileMode
	WName    func() string
	WSize    func() int64
	WSys     func() interface{}
}

func (W _os_FileInfo) IsDir() bool        { return W.WIsDir() }
func (W _os_FileInfo) ModTime() time.Time { return W.WModTime() }
func (W _os_FileInfo) Mode() fs.FileMode  { return W.WMode() }
func (W _os_FileInfo) Name() string       { return W.WName() }
func (W _os_FileInfo) Size() int64        { return W.WSize() }
func (W _os_FileInfo) Sys() interface{}   { return W.WSys() }

// _os_Signal is an interface wrapper for Signal type
type _os_Signal struct {
	IValue  interface{}
	WSignal func()
	WString func() string
}

func (W _os_Signal) Signal()        { W.WSignal() }
func (W _os_Signal) String() string { return W.WString() }

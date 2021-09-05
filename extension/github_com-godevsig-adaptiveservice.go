// Code generated by 'yaegi extract github.com/godevsig/adaptiveservice'. DO NOT EDIT.

// +build adaptiveservice

package extension

import (
	"github.com/godevsig/adaptiveservice"
	"go/constant"
	"go/token"
	"net"
	"reflect"
)

func init() {
	Symbols["github.com/godevsig/adaptiveservice/adaptiveservice"] = map[string]reflect.Value{
		// function, constant and variable definitions
		"BuiltinPublisher":       reflect.ValueOf(constant.MakeFromLiteral("\"builtin\"", token.STRING, 0)),
		"ErrBadMessage":          reflect.ValueOf(&adaptiveservice.ErrBadMessage).Elem(),
		"ErrConnReset":           reflect.ValueOf(&adaptiveservice.ErrConnReset).Elem(),
		"ErrServerClosed":        reflect.ValueOf(&adaptiveservice.ErrServerClosed).Elem(),
		"ErrServiceNotFound":     reflect.ValueOf(&adaptiveservice.ErrServiceNotFound).Elem(),
		"ErrServiceNotReachable": reflect.ValueOf(&adaptiveservice.ErrServiceNotReachable).Elem(),
		"NewClient":              reflect.ValueOf(adaptiveservice.NewClient),
		"NewServer":              reflect.ValueOf(adaptiveservice.NewServer),
		"OK":                     reflect.ValueOf(constant.MakeFromLiteral("0", token.INT, 0)),
		"OnConnectFunc":          reflect.ValueOf(adaptiveservice.OnConnectFunc),
		"OnDisconnectFunc":       reflect.ValueOf(adaptiveservice.OnDisconnectFunc),
		"OnNewStreamFunc":        reflect.ValueOf(adaptiveservice.OnNewStreamFunc),
		"RegisterType":           reflect.ValueOf(adaptiveservice.RegisterType),
		"ScopeAll":               reflect.ValueOf(adaptiveservice.ScopeAll),
		"ScopeLAN":               reflect.ValueOf(adaptiveservice.ScopeLAN),
		"ScopeOS":                reflect.ValueOf(adaptiveservice.ScopeOS),
		"ScopeProcess":           reflect.ValueOf(adaptiveservice.ScopeProcess),
		"ScopeWAN":               reflect.ValueOf(adaptiveservice.ScopeWAN),
		"WithLogger":             reflect.ValueOf(adaptiveservice.WithLogger),
		"WithProviderID":         reflect.ValueOf(adaptiveservice.WithProviderID),
		"WithQsize":              reflect.ValueOf(adaptiveservice.WithQsize),
		"WithRegistryAddr":       reflect.ValueOf(adaptiveservice.WithRegistryAddr),
		"WithScope":              reflect.ValueOf(adaptiveservice.WithScope),

		// type definitions
		"Client":              reflect.ValueOf((*adaptiveservice.Client)(nil)),
		"Connection":          reflect.ValueOf((*adaptiveservice.Connection)(nil)),
		"Context":             reflect.ValueOf((*adaptiveservice.Context)(nil)),
		"ContextStream":       reflect.ValueOf((*adaptiveservice.ContextStream)(nil)),
		"HighPriorityMessage": reflect.ValueOf((*adaptiveservice.HighPriorityMessage)(nil)),
		"KnownMessage":        reflect.ValueOf((*adaptiveservice.KnownMessage)(nil)),
		"ListService":         reflect.ValueOf((*adaptiveservice.ListService)(nil)),
		"Logger":              reflect.ValueOf((*adaptiveservice.Logger)(nil)),
		"LoggerAll":           reflect.ValueOf((*adaptiveservice.LoggerAll)(nil)),
		"LoggerNull":          reflect.ValueOf((*adaptiveservice.LoggerNull)(nil)),
		"LowPriorityMessage":  reflect.ValueOf((*adaptiveservice.LowPriorityMessage)(nil)),
		"Netconn":             reflect.ValueOf((*adaptiveservice.Netconn)(nil)),
		"Option":              reflect.ValueOf((*adaptiveservice.Option)(nil)),
		"ReqProviderInfo":     reflect.ValueOf((*adaptiveservice.ReqProviderInfo)(nil)),
		"Scope":               reflect.ValueOf((*adaptiveservice.Scope)(nil)),
		"Server":              reflect.ValueOf((*adaptiveservice.Server)(nil)),
		"ServiceInfo":         reflect.ValueOf((*adaptiveservice.ServiceInfo)(nil)),
		"ServiceOption":       reflect.ValueOf((*adaptiveservice.ServiceOption)(nil)),
		"Stream":              reflect.ValueOf((*adaptiveservice.Stream)(nil)),

		// interface wrapper definitions
		"_Connection":          reflect.ValueOf((*_github_com_godevsig_adaptiveservice_Connection)(nil)),
		"_Context":             reflect.ValueOf((*_github_com_godevsig_adaptiveservice_Context)(nil)),
		"_ContextStream":       reflect.ValueOf((*_github_com_godevsig_adaptiveservice_ContextStream)(nil)),
		"_HighPriorityMessage": reflect.ValueOf((*_github_com_godevsig_adaptiveservice_HighPriorityMessage)(nil)),
		"_KnownMessage":        reflect.ValueOf((*_github_com_godevsig_adaptiveservice_KnownMessage)(nil)),
		"_Logger":              reflect.ValueOf((*_github_com_godevsig_adaptiveservice_Logger)(nil)),
		"_LowPriorityMessage":  reflect.ValueOf((*_github_com_godevsig_adaptiveservice_LowPriorityMessage)(nil)),
		"_Netconn":             reflect.ValueOf((*_github_com_godevsig_adaptiveservice_Netconn)(nil)),
		"_Stream":              reflect.ValueOf((*_github_com_godevsig_adaptiveservice_Stream)(nil)),
	}
}

// _github_com_godevsig_adaptiveservice_Connection is an interface wrapper for Connection type
type _github_com_godevsig_adaptiveservice_Connection struct {
	IValue     interface{}
	WClose     func()
	WNewStream func() adaptiveservice.Stream
	WRead      func(p []byte) (n int, err error)
	WRecv      func(msgPtr interface{}) error
	WSend      func(msg interface{}) error
	WSendRecv  func(msgSnd interface{}, msgRcvPtr interface{}) error
	WWrite     func(p []byte) (n int, err error)
}

func (W _github_com_godevsig_adaptiveservice_Connection) Close() { W.WClose() }
func (W _github_com_godevsig_adaptiveservice_Connection) NewStream() adaptiveservice.Stream {
	return W.WNewStream()
}
func (W _github_com_godevsig_adaptiveservice_Connection) Read(p []byte) (n int, err error) {
	return W.WRead(p)
}
func (W _github_com_godevsig_adaptiveservice_Connection) Recv(msgPtr interface{}) error {
	return W.WRecv(msgPtr)
}
func (W _github_com_godevsig_adaptiveservice_Connection) Send(msg interface{}) error {
	return W.WSend(msg)
}
func (W _github_com_godevsig_adaptiveservice_Connection) SendRecv(msgSnd interface{}, msgRcvPtr interface{}) error {
	return W.WSendRecv(msgSnd, msgRcvPtr)
}
func (W _github_com_godevsig_adaptiveservice_Connection) Write(p []byte) (n int, err error) {
	return W.WWrite(p)
}

// _github_com_godevsig_adaptiveservice_Context is an interface wrapper for Context type
type _github_com_godevsig_adaptiveservice_Context struct {
	IValue      interface{}
	WGetContext func() interface{}
	WGetVar     func(v interface{})
	WPutVar     func(v interface{})
	WSetContext func(v interface{})
}

func (W _github_com_godevsig_adaptiveservice_Context) GetContext() interface{} {
	return W.WGetContext()
}
func (W _github_com_godevsig_adaptiveservice_Context) GetVar(v interface{})     { W.WGetVar(v) }
func (W _github_com_godevsig_adaptiveservice_Context) PutVar(v interface{})     { W.WPutVar(v) }
func (W _github_com_godevsig_adaptiveservice_Context) SetContext(v interface{}) { W.WSetContext(v) }

// _github_com_godevsig_adaptiveservice_ContextStream is an interface wrapper for ContextStream type
type _github_com_godevsig_adaptiveservice_ContextStream struct {
	IValue      interface{}
	WGetContext func() interface{}
	WGetVar     func(v interface{})
	WPutVar     func(v interface{})
	WRead       func(p []byte) (n int, err error)
	WRecv       func(msgPtr interface{}) error
	WSend       func(msg interface{}) error
	WSendRecv   func(msgSnd interface{}, msgRcvPtr interface{}) error
	WSetContext func(v interface{})
	WWrite      func(p []byte) (n int, err error)
}

func (W _github_com_godevsig_adaptiveservice_ContextStream) GetContext() interface{} {
	return W.WGetContext()
}
func (W _github_com_godevsig_adaptiveservice_ContextStream) GetVar(v interface{}) { W.WGetVar(v) }
func (W _github_com_godevsig_adaptiveservice_ContextStream) PutVar(v interface{}) { W.WPutVar(v) }
func (W _github_com_godevsig_adaptiveservice_ContextStream) Read(p []byte) (n int, err error) {
	return W.WRead(p)
}
func (W _github_com_godevsig_adaptiveservice_ContextStream) Recv(msgPtr interface{}) error {
	return W.WRecv(msgPtr)
}
func (W _github_com_godevsig_adaptiveservice_ContextStream) Send(msg interface{}) error {
	return W.WSend(msg)
}
func (W _github_com_godevsig_adaptiveservice_ContextStream) SendRecv(msgSnd interface{}, msgRcvPtr interface{}) error {
	return W.WSendRecv(msgSnd, msgRcvPtr)
}
func (W _github_com_godevsig_adaptiveservice_ContextStream) SetContext(v interface{}) {
	W.WSetContext(v)
}
func (W _github_com_godevsig_adaptiveservice_ContextStream) Write(p []byte) (n int, err error) {
	return W.WWrite(p)
}

// _github_com_godevsig_adaptiveservice_HighPriorityMessage is an interface wrapper for HighPriorityMessage type
type _github_com_godevsig_adaptiveservice_HighPriorityMessage struct {
	IValue          interface{}
	WHandle         func(stream adaptiveservice.ContextStream) (reply interface{})
	WIsHighPriority func()
}

func (W _github_com_godevsig_adaptiveservice_HighPriorityMessage) Handle(stream adaptiveservice.ContextStream) (reply interface{}) {
	return W.WHandle(stream)
}
func (W _github_com_godevsig_adaptiveservice_HighPriorityMessage) IsHighPriority() {
	W.WIsHighPriority()
}

// _github_com_godevsig_adaptiveservice_KnownMessage is an interface wrapper for KnownMessage type
type _github_com_godevsig_adaptiveservice_KnownMessage struct {
	IValue  interface{}
	WHandle func(stream adaptiveservice.ContextStream) (reply interface{})
}

func (W _github_com_godevsig_adaptiveservice_KnownMessage) Handle(stream adaptiveservice.ContextStream) (reply interface{}) {
	return W.WHandle(stream)
}

// _github_com_godevsig_adaptiveservice_Logger is an interface wrapper for Logger type
type _github_com_godevsig_adaptiveservice_Logger struct {
	IValue  interface{}
	WDebugf func(format string, args ...interface{})
	WErrorf func(format string, args ...interface{})
	WInfof  func(format string, args ...interface{})
	WWarnf  func(format string, args ...interface{})
}

func (W _github_com_godevsig_adaptiveservice_Logger) Debugf(format string, args ...interface{}) {
	W.WDebugf(format, args...)
}
func (W _github_com_godevsig_adaptiveservice_Logger) Errorf(format string, args ...interface{}) {
	W.WErrorf(format, args...)
}
func (W _github_com_godevsig_adaptiveservice_Logger) Infof(format string, args ...interface{}) {
	W.WInfof(format, args...)
}
func (W _github_com_godevsig_adaptiveservice_Logger) Warnf(format string, args ...interface{}) {
	W.WWarnf(format, args...)
}

// _github_com_godevsig_adaptiveservice_LowPriorityMessage is an interface wrapper for LowPriorityMessage type
type _github_com_godevsig_adaptiveservice_LowPriorityMessage struct {
	IValue         interface{}
	WHandle        func(stream adaptiveservice.ContextStream) (reply interface{})
	WIsLowPriority func()
}

func (W _github_com_godevsig_adaptiveservice_LowPriorityMessage) Handle(stream adaptiveservice.ContextStream) (reply interface{}) {
	return W.WHandle(stream)
}
func (W _github_com_godevsig_adaptiveservice_LowPriorityMessage) IsLowPriority() { W.WIsLowPriority() }

// _github_com_godevsig_adaptiveservice_Netconn is an interface wrapper for Netconn type
type _github_com_godevsig_adaptiveservice_Netconn struct {
	IValue      interface{}
	WClose      func() error
	WLocalAddr  func() net.Addr
	WRemoteAddr func() net.Addr
}

func (W _github_com_godevsig_adaptiveservice_Netconn) Close() error         { return W.WClose() }
func (W _github_com_godevsig_adaptiveservice_Netconn) LocalAddr() net.Addr  { return W.WLocalAddr() }
func (W _github_com_godevsig_adaptiveservice_Netconn) RemoteAddr() net.Addr { return W.WRemoteAddr() }

// _github_com_godevsig_adaptiveservice_Stream is an interface wrapper for Stream type
type _github_com_godevsig_adaptiveservice_Stream struct {
	IValue    interface{}
	WRead     func(p []byte) (n int, err error)
	WRecv     func(msgPtr interface{}) error
	WSend     func(msg interface{}) error
	WSendRecv func(msgSnd interface{}, msgRcvPtr interface{}) error
	WWrite    func(p []byte) (n int, err error)
}

func (W _github_com_godevsig_adaptiveservice_Stream) Read(p []byte) (n int, err error) {
	return W.WRead(p)
}
func (W _github_com_godevsig_adaptiveservice_Stream) Recv(msgPtr interface{}) error {
	return W.WRecv(msgPtr)
}
func (W _github_com_godevsig_adaptiveservice_Stream) Send(msg interface{}) error { return W.WSend(msg) }
func (W _github_com_godevsig_adaptiveservice_Stream) SendRecv(msgSnd interface{}, msgRcvPtr interface{}) error {
	return W.WSendRecv(msgSnd, msgRcvPtr)
}
func (W _github_com_godevsig_adaptiveservice_Stream) Write(p []byte) (n int, err error) {
	return W.WWrite(p)
}

package gshellos

import (
	"fmt"
)

var globalFuncs = []struct {
	name string
	fn   func(args ...Object) (Object, error)
}{
	{"ex", ex},
	{"show", show},
	{"makechan", makechan},
}

func ex(args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	return ExtendObj(args[0])
}

// hasHelp is the one who implements Help
type hasHelp interface {
	Help() string
}

func show(args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}

	var help string
	switch v := args[0].(type) {
	case hasHelp:
		help = v.Help()
	default:
		help = v.String()
	}

	fmt.Println(help)
	return nil, nil
}

type objchan chan Object

func makechan(args ...Object) (Object, error) {
	var size int
	switch len(args) {
	case 0:
	case 1:
		if err := FromObject(&size, args[0]); err != nil {
			return nil, err
		}
	default:
		return nil, ErrWrongNumArguments
	}

	oc := make(objchan, size)
	obj := map[string]Object{
		"send": &UserFunction{
			Value:     oc.send,
			Signature: `send(obj)`,
			Usage: `Send an obj to the channel, will block if channel is full.
					Send to a closed channel causes panic.`,
			Example: `objchan.send("hello")`,
		},
		"recv": &UserFunction{
			Value:     oc.recv,
			Signature: `recv() -> obj`,
			Usage: `Receive an obj from the channel, will block if channel is empty.
					Receive from a closed channel returns undefined value.`,
			Example: `obj = objchan.recv()`,
		},
		"close": &UserFunction{
			Value:     oc.close,
			Signature: `close()`,
			Usage:     `Close the channel.`,
			Example:   `objchan.close()`,
		},
	}
	return MustToObject(obj), nil
}

func (oc objchan) send(args ...Object) (Object, error) {
	if len(args) != 1 {
		return nil, ErrWrongNumArguments
	}
	oc <- args[0]
	return nil, nil
}

func (oc objchan) recv(args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}

	obj, ok := <-oc
	if ok {
		return obj, nil
	}

	return UndefinedValue, nil
}

func (oc objchan) close(args ...Object) (Object, error) {
	if len(args) != 0 {
		return nil, ErrWrongNumArguments
	}

	close(oc)
	return nil, nil
}

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

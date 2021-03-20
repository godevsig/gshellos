package gshellos

import (
	"fmt"

	"github.com/d5/tengo/v2"
)

var globalFuncs = []struct {
	name string
	fn   tengo.CallableFunc
}{
	{"ex", ex},
	{"show", show},
}

func ex(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
	}
	return ExtendObj(args[0])
}

// hasHelp is the one who implements Help
type hasHelp interface {
	Help() string
}

func show(args ...tengo.Object) (tengo.Object, error) {
	if len(args) != 1 {
		return nil, tengo.ErrWrongNumArguments
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

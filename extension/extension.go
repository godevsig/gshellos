// Package extension provides wrappers to selected packages to be imported natively in Yaegi.
package extension

import "reflect"

// Symbols variable stores the map of symbols per package.
var Symbols = map[string]map[string]reflect.Value{}

func init() {
	Symbols["github.com/godevsig/gshellos/extension"] = map[string]reflect.Value{
		"Symbols": reflect.ValueOf(Symbols),
	}
}

//go:generate ./gen_symbols.sh github.com/godevsig/adaptiveservice
//go:generate ./gen_symbols.sh github.com/godevsig/glib/sys/shell
//go:generate ./gen_symbols.sh github.com/godevsig/glib/sys/log -fixlog
//go:generate ./gen_symbols.sh github.com/godevsig/glib/sys/pidinfo

////go:generate ./gen_symbols.sh github.com/godevsig/grepo/util/fileserver

////go:generate ./gen_symbols.sh github.com/godevsig/grepo/example/echo -extramsg
////go:generate ./gen_symbols.sh github.com/godevsig/grepo/benchmark/asbench

////go:generate ./gen_symbols.sh github.com/godevsig/grepo/perf/topidchart -extramsg
////go:generate ./gen_symbols.sh github.com/godevsig/grepo/util/recorder -extramsg
////go:generate ./gen_symbols.sh github.com/godevsig/grepo/render/docit

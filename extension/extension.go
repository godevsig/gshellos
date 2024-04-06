// Package extension provides wrappers to selected packages to be imported natively in Yaegi.
package extension

import "reflect"

// BuiltinSymbols variable stores the map of symbols per built-in package.
var BuiltinSymbols = map[string]map[string]reflect.Value{}

// PluginSymbols variable stores the map of symbols per plugin.
var PluginSymbols = map[string]map[string]reflect.Value{}

//go:generate ./gen_symbols.sh github.com/godevsig/adaptiveservice

//go:generate ./gen_symbols.sh github.com/godevsig/glib/sys/shell
//go:generate ./gen_symbols.sh github.com/godevsig/glib/sys/log -fixlog
//go:generate ./gen_symbols.sh github.com/godevsig/glib/sys/pidinfo

//go:generate ./gen_symbols.sh github.com/godevsig/grepo/fileserver
//go:generate ./gen_symbols.sh github.com/godevsig/grepo/echo -extramsg
//go:generate ./gen_symbols.sh github.com/godevsig/grepo/asbench
//go:generate ./gen_symbols.sh github.com/godevsig/grepo/topidchart -extramsg
//go:generate ./gen_symbols.sh github.com/godevsig/grepo/recorder -extramsg
//go:generate ./gen_symbols.sh github.com/godevsig/grepo/docit

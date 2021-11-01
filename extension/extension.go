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

//go:generate ../cmd/extract/extract -name extension -tag adaptiveservice github.com/godevsig/adaptiveservice
//go:generate ../cmd/extract/extract -name extension -tag shell github.com/godevsig/grepo/lib-sys/shell
//go:generate ../cmd/extract/extract -name extension -tag log github.com/godevsig/grepo/lib-sys/log
//go:generate sed -i "s/logLogger/log.Logger/" github_com-godevsig-grepo-lib-sys-log.go

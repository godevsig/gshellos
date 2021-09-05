// Package extension provides wrappers to selected packages to be imported natively in Yaegi.
// Generate wrapper: e.g. extract -name extension -tag adaptiveservice github.com/godevsig/adaptiveservice
package extension

import "reflect"

// Symbols variable stores the map of symbols per package.
var Symbols = map[string]map[string]reflect.Value{}

func init() {
	Symbols["github.com/godevsig/gshellos/extension"] = map[string]reflect.Value{
		"Symbols": reflect.ValueOf(Symbols),
	}
}

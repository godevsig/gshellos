//go:build !plugin

package gshellos

import (
	"errors"
	"reflect"
)

func loadPluginFile(path string) (map[string]map[string]reflect.Value, error) {
	return nil, errors.New("plugin not enabled")
}

func loadPlugins(pluginDir string) error {
	return errors.New("plugin not enabled")
}

func listPlugins() []string {
	return nil
}

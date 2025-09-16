//go:build !plugin

package gshellos

import "reflect"

func loadPluginFile(path string) (map[string]map[string]reflect.Value, error) {
	return nil, nil
}

func loadPlugins(pluginDir string) error {
	return nil
}

func listPlugins() []string {
	return nil
}

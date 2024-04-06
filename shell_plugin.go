//go:build plugin

package gshellos

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"sort"
	"strings"

	"github.com/godevsig/gshellos/extension"
)

func loadPlugins(pluginDir string) error {
	if _, err := os.Stat(pluginDir); err != nil {
		return nil // no such path, assume ok
	}
	filepath.WalkDir(pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() && strings.HasSuffix(d.Name(), ".gplugin.so") {
			p, err := plugin.Open(path)
			if err != nil {
				return fmt.Errorf("open plugin %s error: %w", path, err)
			}
			export, err := p.Lookup("Export")
			if err != nil {
				return fmt.Errorf("symbol lookup error in plugin %s: %w", path, err)
			}

			name, symbols := export.(func() (string, map[string]reflect.Value))()
			if len(name) != 0 && len(symbols) != 0 {
				if _, has := extension.PluginSymbols[name]; !has {
					extension.PluginSymbols[name] = symbols
				}
			}
		}
		return nil
	})
	return nil
}

func listPlugins() []string {
	var plugins []string
	for name := range extension.PluginSymbols {
		plugins = append(plugins, name)
	}
	sort.Strings(plugins)
	return plugins
}

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
	var allErr error
	if err := filepath.WalkDir(pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.Type().IsRegular() && strings.HasSuffix(d.Name(), ".gplugin") {
			p, err := plugin.Open(path)
			if err != nil {
				allErr = fmt.Errorf("open plugin %s error: %v; %v", path, err, allErr)
				return nil
			}
			export, err := p.Lookup("Export")
			if err != nil {
				allErr = fmt.Errorf("symbol lookup error in plugin %s: %v; %v", path, err, allErr)
				return nil
			}

			name, symbols := export.(func() (string, map[string]reflect.Value))()
			if len(name) != 0 && len(symbols) != 0 {
				if _, has := extension.PluginSymbols[name]; !has {
					extension.PluginSymbols[name] = symbols
				}
			}
		}
		return nil
	}); err != nil {
		allErr = fmt.Errorf("walk dir %s error: %v; %v", pluginDir, err, allErr)
	}
	if allErr != nil {
		return allErr
	}
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

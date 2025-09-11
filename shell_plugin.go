//go:build plugin

package gshellos

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"reflect"
	"sort"
	"strings"

	"github.com/godevsig/gshellos/extension"
)

var loadedPlugins = make(map[string]bool)

// Load a single plugin file, only if not already loaded
func loadPluginFile(path string) error {
	if loadedPlugins[path] {
		return nil
	}

	p, err := plugin.Open(path)
	if err != nil {
		return fmt.Errorf("open plugin %s error: %w", path, err)
	}

	exportSym, err := p.Lookup("Export")
	if err != nil {
		return fmt.Errorf("symbol lookup error in plugin %s: %w", path, err)
	}

	export, ok := exportSym.(func() (string, map[string]reflect.Value))
	if !ok {
		return fmt.Errorf("plugin %s has invalid Export signature", path)
	}

	name, symbols := export()
	if name != "" && len(symbols) > 0 {
		if _, has := extension.PluginSymbols[name]; !has {
			extension.PluginSymbols[name] = symbols
		}
	}

	loadedPlugins[path] = true
	return nil
}

// load .gp files
func loadPlugins(pluginDir string) error {
	if _, err := os.Stat(pluginDir); err != nil {
		if os.IsNotExist(err) {
			return nil // no such path, assume ok
		}
		return err
	}

	var allErr error

	// Preload existing .gp files
	if err := filepath.WalkDir(pluginDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr == nil && d.Type().IsRegular() && strings.HasSuffix(d.Name(), ".gp") {
			if err := loadPluginFile(path); err != nil {
				e := fmt.Errorf("load plugin %s error: %v", path, err)
				if allErr == nil {
					allErr = e
				} else {
					allErr = fmt.Errorf("%v; %v", allErr, e)
				}
			}
		}
		return nil
	}); err != nil {
		return err
	}

	return allErr
}

func listPlugins() []string {
	var plugins []string
	for name := range extension.PluginSymbols {
		plugins = append(plugins, name)
	}
	sort.Strings(plugins)
	return plugins
}

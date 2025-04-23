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

	"github.com/fsnotify/fsnotify"
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

// Watch a plugin directory and load new .gp files as they appear
func loadPlugins(pluginDir string) error {
	if _, err := os.Stat(pluginDir); err != nil {
		return nil // no such path, assume ok
	}

	// Preload existing .gp files
	filepath.WalkDir(pluginDir, func(path string, d os.DirEntry, err error) error {
		if err == nil && d.Type().IsRegular() && strings.HasSuffix(d.Name(), ".gp") {
			if err := loadPluginFile(path); err != nil {
				fmt.Fprintf(os.Stderr, "load plugin %s error: %v", path, err)
			}
		}
		return nil
	})

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}

	if err := watcher.Add(pluginDir); err != nil {
		return err
	}

	// Watch for new .gp files
	go func() {
		defer watcher.Close()
		for {
			select {
			case event := <-watcher.Events:
				if event.Op&fsnotify.Create != 0 && strings.HasSuffix(event.Name, ".gp") {
					if err := loadPluginFile(event.Name); err != nil {
						fmt.Fprintf(os.Stderr, "load plugin %s error: %v", event.Name, err)
					}
				}
			case err := <-watcher.Errors:
				fmt.Fprintf(os.Stderr, "watch error: %v\n", err)
			}
		}
	}()

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

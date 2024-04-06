//go:build !plugin

package gshellos

import (
	"errors"
	"os"
)

func loadExtensions(pluginDir string) error {
	if _, err := os.Stat(pluginDir); err != nil {
		return nil // no such path, assume ok
	}
	return errors.New("plugin is not supported")
}

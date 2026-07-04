package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/pflag"
	"gopkg.in/yaml.v3"
)

// applyConfigFile loads ~/.config/goshot/config.yaml (a flat map of flag
// names to values) and applies it to every flag not set on the command
// line. Flags always win over the config file.
func applyConfigFile(flags *pflag.FlagSet) error {
	rootFlags = flags

	path := configPath()
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	var values map[string]any
	if err := yaml.Unmarshal(data, &values); err != nil {
		return fmt.Errorf("config %s: %w", path, err)
	}
	for key, value := range values {
		flag := flags.Lookup(key)
		if flag == nil || flag.Changed {
			continue
		}
		if err := flag.Value.Set(fmt.Sprint(value)); err != nil {
			return fmt.Errorf("config %s: %s: %w", path, key, err)
		}
	}
	return nil
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return ""
		}
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "goshot", "config.yaml")
}

var rootFlags *pflag.FlagSet

func flagChanged(name string) bool {
	if rootFlags == nil {
		return false
	}
	f := rootFlags.Lookup(name)
	return f != nil && f.Changed
}

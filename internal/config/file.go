package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// File represents optional YAML config file structure.
type File struct {
	Filters *FilterSpec `yaml:"filters,omitempty"`
}

func loadFile(path string, c *Config) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var f File
	if err := yaml.Unmarshal(data, &f); err != nil {
		return fmt.Errorf("yaml: %w", err)
	}
	if f.Filters != nil {
		c.Filters = *f.Filters
	}
	return nil
}

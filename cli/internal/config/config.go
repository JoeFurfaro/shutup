// Package config loads, validates, and writes the committed shutup.config.yaml.
//
// The config is a project's declarations only — never values. It lists the var
// NAMES the project consumes, maps local env names to env ids, and picks a
// default env. It holds no values, no visibility, and no secrets, so it is safe
// to commit. Values live in envs (see package env); visibility lives on the var
// within its env.
package config

import (
	"fmt"
	"os"

	"github.com/joe/shutup/internal/id"
	"gopkg.in/yaml.v3"
)

// Filename is the committed config file, discovered by walking up from cwd.
const Filename = "shutup.config.yaml"

// Config is the parsed shutup.config.yaml for one project.
type Config struct {
	// Consumes is the set of variable NAMES this project needs (visibility-agnostic).
	Consumes []string `yaml:"consumes"`
	// Envs maps a local env name (e.g. "dev") to an env id.
	Envs map[string]string `yaml:"envs"`
	// DefaultEnv is the env name used when --env is omitted.
	DefaultEnv string `yaml:"default_env"`

	path string
}

// Load reads and parses a config from path. It does not validate; call Validate.
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	c.path = path
	if c.Envs == nil {
		c.Envs = map[string]string{}
	}
	return &c, nil
}

// Path returns the file this config is bound to.
func (c *Config) Path() string { return c.path }

// Consumes reports whether the project consumes name.
func (c *Config) ConsumesVar(name string) bool {
	for _, n := range c.Consumes {
		if n == name {
			return true
		}
	}
	return false
}

// AddConsume adds name to the consumed set if absent. Returns true if added.
func (c *Config) AddConsume(name string) bool {
	if c.ConsumesVar(name) {
		return false
	}
	c.Consumes = append(c.Consumes, name)
	return true
}

// RemoveConsume removes name from the consumed set. Returns true if removed.
func (c *Config) RemoveConsume(name string) bool {
	for i, n := range c.Consumes {
		if n == name {
			c.Consumes = append(c.Consumes[:i], c.Consumes[i+1:]...)
			return true
		}
	}
	return false
}

// EnvID resolves an env name to its id via the project's map.
func (c *Config) EnvID(name string) (string, bool) {
	envID, ok := c.Envs[name]
	return envID, ok
}

// Save writes the config back to its path, regenerating the document with the
// standard teaching comments and stable (sorted) ordering.
func (c *Config) Save() error {
	out, err := c.marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(c.path, out, 0o644)
}

// Validate enforces internal consistency. These are hard errors (the file
// contradicts itself), distinct from a config that is merely incomplete (a
// consumed var with no value in an env) — that is normal, surfaced by `missing`.
func (c *Config) Validate() error {
	for name, envID := range c.Envs {
		if !id.IsValid(envID) {
			return fmt.Errorf("envs.%s: %q is not a valid env id", name, envID)
		}
	}
	if c.DefaultEnv != "" {
		if _, ok := c.Envs[c.DefaultEnv]; !ok {
			return fmt.Errorf("default_env %q is not one of the project's envs", c.DefaultEnv)
		}
	}
	return nil
}

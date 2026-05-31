package config

import (
	"errors"
	"os"
	"path/filepath"
)

// ErrNotFound means no shutup.config.yaml was found in cwd or any ancestor.
var ErrNotFound = errors.New("not inside a shutup project (no " + Filename + " found); run `shutup init`")

// Discover walks up from start looking for Filename and returns its full path.
func Discover(start string) (string, error) {
	dir, err := filepath.Abs(start)
	if err != nil {
		return "", err
	}
	for {
		p := filepath.Join(dir, Filename)
		if info, err := os.Stat(p); err == nil && !info.IsDir() {
			return p, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir { // reached filesystem root
			return "", ErrNotFound
		}
		dir = parent
	}
}

// DiscoverCWD finds the project config starting from the current directory.
func DiscoverCWD() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return Discover(cwd)
}

// LoadFromCWD discovers and loads the project config for the current directory.
func LoadFromCWD() (*Config, error) {
	path, err := DiscoverCWD()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

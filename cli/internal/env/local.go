package env

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/joe/shutup/internal/id"
	"gopkg.in/yaml.v3"
)

// LocalEnvStore persists envs as YAML files at ~/.shutup/envs/<id>.yaml.
//
// v1 is unencrypted: the files live outside any repo (uncommittable) and the
// TTY-bypass — the actual safety mechanism — is independent of at-rest
// encryption. An encrypting implementation can replace this behind EnvStore.
//
// TODO: encrypt at rest (AES-GCM, key in OS keychain) behind this interface.
type LocalEnvStore struct {
	dir string
}

// NewLocalEnvStore returns a store rooted at ~/.shutup/envs.
func NewLocalEnvStore() (*LocalEnvStore, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return &LocalEnvStore{dir: filepath.Join(home, ".shutup", "envs")}, nil
}

func (s *LocalEnvStore) path(envID string) string {
	return filepath.Join(s.dir, envID+".yaml")
}

// Load returns the env with the given id, or ErrNotFound.
func (s *LocalEnvStore) Load(envID string) (*Env, error) {
	data, err := os.ReadFile(s.path(envID))
	if os.IsNotExist(err) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	var e Env
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("reading env %s: %w", envID, err)
	}
	if e.Vars == nil {
		e.Vars = map[string]Var{}
	}
	return &e, nil
}

// Save writes the env, creating the store dir (0700) and file (0600) as needed.
func (s *LocalEnvStore) Save(e *Env) error {
	if err := os.MkdirAll(s.dir, 0o700); err != nil {
		return err
	}
	data, err := yaml.Marshal(e)
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(e.ID), data, 0o600)
}

// Create makes a fresh local env with a new id and persists it.
func (s *LocalEnvStore) Create() (*Env, error) {
	e := &Env{ID: id.GenerateLocal(), Source: SourceLocal, Vars: map[string]Var{}}
	if err := s.Save(e); err != nil {
		return nil, err
	}
	return e, nil
}

// List returns every env in the store, sorted by id.
func (s *LocalEnvStore) List() ([]*Env, error) {
	entries, err := os.ReadDir(s.dir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".yaml") {
			ids = append(ids, strings.TrimSuffix(e.Name(), ".yaml"))
		}
	}
	sort.Strings(ids)
	out := make([]*Env, 0, len(ids))
	for _, envID := range ids {
		e, err := s.Load(envID)
		if err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

// Delete removes an env; deleting a missing one is not an error.
func (s *LocalEnvStore) Delete(envID string) error {
	err := os.Remove(s.path(envID))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

var _ EnvStore = (*LocalEnvStore)(nil)

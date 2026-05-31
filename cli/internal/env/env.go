// Package env persists environments — the named bags of variable values that
// projects consume.
//
// An Env is an anonymous, id-keyed bag of name -> {value, secret}. It owns both
// the values AND each var's visibility (secret-ness is intrinsic to the var, so
// it lives with the var, not with whoever consumes it). Envs live on this
// machine now (LocalEnvStore, ~/.shutup/envs/<id>.yaml) and behind the API
// later — the EnvStore interface is the seam that keeps that swappable. The
// command layer talks only to this interface.
package env

import "errors"

// Source marks where an env is backed.
const SourceLocal = "local"

// ErrNotFound is returned by Load when no env with the id exists.
var ErrNotFound = errors.New("env not found")

// Var is a single variable's value and visibility within an env.
type Var struct {
	Value  string `yaml:"value"`
	Secret bool   `yaml:"secret"`
}

// Env is a named bag of variables, identified by a stable id.
type Env struct {
	ID     string         `yaml:"id"`
	Source string         `yaml:"source"`
	Vars   map[string]Var `yaml:"vars"`
}

// HasValue reports whether name has a value set in this env. (A declared secret
// with no value yet returns false — that's what `missing` surfaces.)
func (e *Env) HasValue(name string) bool {
	v, ok := e.Vars[name]
	return ok && v.Value != ""
}

// EnvStore persists envs by id. LocalEnvStore is today's implementation; a
// cloud APIStore (or an encrypting wrapper) can slot in behind it unchanged.
type EnvStore interface {
	// Load returns the env with the given id, or ErrNotFound.
	Load(id string) (*Env, error)
	// Save writes the env (creating or overwriting).
	Save(e *Env) error
	// Create makes a fresh local env with a new id and persists it.
	Create() (*Env, error)
	// List returns every env in the store.
	List() ([]*Env, error)
	// Delete removes an env; deleting a missing one is not an error.
	Delete(id string) error
}

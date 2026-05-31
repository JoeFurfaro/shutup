package env

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// Bundle returns a copy of the env safe to hand to a teammate out-of-band:
// public values are kept, but secret VALUES are stripped (only the secret var
// NAMES remain). The id is preserved so both people end up on the same logical
// env — which makes the eventual API "sync this live" upgrade trivial.
func (e *Env) Bundle() *Env {
	out := &Env{ID: e.ID, Source: e.Source, Vars: map[string]Var{}}
	for name, v := range e.Vars {
		if v.Secret {
			out.Vars[name] = Var{Secret: true} // name only, no value
		} else {
			out.Vars[name] = v
		}
	}
	return out
}

// MarshalBundle serializes a secret-free bundle of the env.
func MarshalBundle(e *Env) ([]byte, error) {
	return yaml.Marshal(e.Bundle())
}

// UnmarshalBundle parses a bundle file into an Env.
func UnmarshalBundle(data []byte) (*Env, error) {
	var e Env
	if err := yaml.Unmarshal(data, &e); err != nil {
		return nil, fmt.Errorf("parsing env bundle: %w", err)
	}
	if e.Vars == nil {
		e.Vars = map[string]Var{}
	}
	// A bundle must never carry secret values; defensively strip any.
	return e.Bundle(), nil
}

// ImportBundle merges a bundle into the local store under the bundle's id,
// preserving any secret values already set locally. Public values from the
// bundle are applied; secret names without a local value become placeholders
// (declared, unset — surfaced by `missing`). Returns the merged env.
func (s *LocalEnvStore) ImportBundle(b *Env) (*Env, error) {
	existing, err := s.Load(b.ID)
	if err == ErrNotFound {
		existing = &Env{ID: b.ID, Source: SourceLocal, Vars: map[string]Var{}}
	} else if err != nil {
		return nil, err
	}

	for name, bv := range b.Vars {
		if bv.Secret {
			// Keep a locally-set secret value; otherwise record a placeholder.
			if cur, ok := existing.Vars[name]; ok && cur.Value != "" {
				continue
			}
			existing.Vars[name] = Var{Secret: true}
		} else {
			existing.Vars[name] = bv // public value from the bundle
		}
	}

	if err := s.Save(existing); err != nil {
		return nil, err
	}
	return existing, nil
}

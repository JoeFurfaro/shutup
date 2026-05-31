// Package project ties a project's committed config to the env store. Commands
// operate through a Project rather than touching config/env directly, so the
// rules (consume-list = least-privilege, values+visibility live on the env,
// names resolve via the project's map) live in one place.
package project

import (
	"fmt"
	"os"
	"sort"

	"github.com/joe/shutup/internal/config"
	"github.com/joe/shutup/internal/env"
)

// Project is an opened shutup project: its config plus the env store.
type Project struct {
	Config *config.Config
	Store  env.EnvStore
}

// Open discovers, loads, and validates the config for the current directory and
// opens the local env store.
func Open() (*Project, error) {
	cfg, err := config.LoadFromCWD()
	if err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", cfg.Path(), err)
	}
	st, err := env.NewLocalEnvStore()
	if err != nil {
		return nil, err
	}
	return &Project{Config: cfg, Store: st}, nil
}

// envName returns the target env name: the given name, or the project default
// if empty. Errors if neither resolves.
func (p *Project) envName(name string) (string, error) {
	if name == "" {
		name = p.Config.DefaultEnv
	}
	if name == "" {
		return "", fmt.Errorf("no env specified and no default_env set (pass --env or run `shutup env default <name>`)")
	}
	return name, nil
}

// ResolveEnv loads the env that the project knows by name (or its default).
func (p *Project) ResolveEnv(name string) (*env.Env, error) {
	name, err := p.envName(name)
	if err != nil {
		return nil, err
	}
	envID, ok := p.Config.EnvID(name)
	if !ok {
		return nil, fmt.Errorf("this project has no env named %q (see `shutup env list`)", name)
	}
	e, err := p.Store.Load(envID)
	if err == env.ErrNotFound {
		return nil, fmt.Errorf("env %q (%s) is not set up on this machine (run `shutup env import` or set its values)", name, envID)
	}
	return e, err
}

// VarStatus is the resolved state of one consumed variable in one env.
type VarStatus struct {
	Name   string
	Public bool
	Set    bool
	Value  string // populated for public vars only
}

// Status returns the state of every consumed var against the resolved env,
// sorted by name.
func (p *Project) Status(envNameArg string) ([]VarStatus, error) {
	e, err := p.ResolveEnv(envNameArg)
	if err != nil {
		return nil, err
	}
	names := append([]string(nil), p.Config.Consumes...)
	sort.Strings(names)

	out := make([]VarStatus, 0, len(names))
	for _, name := range names {
		v, ok := e.Vars[name]
		vs := VarStatus{Name: name, Public: ok && !v.Secret, Set: ok && v.Value != ""}
		if vs.Public {
			vs.Value = v.Value
		}
		// An unknown-to-env var is treated as a secret-by-default placeholder.
		if !ok {
			vs.Public = false
		}
		out = append(out, vs)
	}
	return out, nil
}

// Missing returns the consumed vars that have no value in the resolved env.
func (p *Project) Missing(envNameArg string) ([]VarStatus, error) {
	all, err := p.Status(envNameArg)
	if err != nil {
		return nil, err
	}
	var missing []VarStatus
	for _, vs := range all {
		if !vs.Set {
			missing = append(missing, vs)
		}
	}
	return missing, nil
}

// SetVar writes name's value into the resolved env (with visibility) and wires
// the current project to consume it. Returns whether the consume was newly added.
func (p *Project) SetVar(envNameArg, name, value string, public bool) (wired bool, err error) {
	e, err := p.ResolveEnv(envNameArg)
	if err != nil {
		return false, err
	}
	e.Vars[name] = env.Var{Value: value, Secret: !public}
	if err := p.Store.Save(e); err != nil {
		return false, err
	}
	wired = p.Config.AddConsume(name)
	if wired {
		if err := p.Config.Save(); err != nil {
			return false, err
		}
	}
	return wired, nil
}

// CopyEnv creates a fresh env in the store seeded with a copy of the values from
// the project env named srcName, and returns the new env id plus the names that
// were copied, partitioned into public and secret (sorted). With publicOnly,
// secret vars are skipped so they must be re-entered per env. Values are copied
// store-side and never returned. The caller maps the new id into the project.
func (p *Project) CopyEnv(srcName string, publicOnly bool) (newEnvID string, public, secret []string, err error) {
	srcID, ok := p.Config.EnvID(srcName)
	if !ok {
		return "", nil, nil, fmt.Errorf("this project has no env named %q to copy from (see `shutup env list`)", srcName)
	}
	src, err := p.Store.Load(srcID)
	if err == env.ErrNotFound {
		return "", nil, nil, fmt.Errorf("env %q (%s) is not set up on this machine", srcName, srcID)
	}
	if err != nil {
		return "", nil, nil, err
	}
	e, err := p.Store.Create()
	if err != nil {
		return "", nil, nil, err
	}
	for name, v := range src.Vars {
		if publicOnly && v.Secret {
			continue
		}
		e.Vars[name] = v
		if v.Secret {
			secret = append(secret, name)
		} else {
			public = append(public, name)
		}
	}
	if err := p.Store.Save(e); err != nil {
		return "", nil, nil, err
	}
	sort.Strings(public)
	sort.Strings(secret)
	return e.ID, public, secret, nil
}

// Use wires the current project to consume name (no value). Returns false if
// already consumed.
func (p *Project) Use(name string) (bool, error) {
	if !p.Config.AddConsume(name) {
		return false, nil
	}
	return true, p.Config.Save()
}

// Unuse removes name from the project's consumed set. Returns false if absent.
func (p *Project) Unuse(name string) (bool, error) {
	if !p.Config.RemoveConsume(name) {
		return false, nil
	}
	return true, p.Config.Save()
}

// Exists reports whether a consumed var has a value in the resolved env. Used by
// `check`; never surfaces the value.
func (p *Project) Exists(envNameArg, name string) (bool, error) {
	e, err := p.ResolveEnv(envNameArg)
	if err != nil {
		return false, err
	}
	return e.HasValue(name), nil
}

// Resolve returns the env map for `run`: ONLY the project's consumed vars,
// resolved against the env (least-privilege). Errors if any consumed var lacks
// a value.
func (p *Project) Resolve(envNameArg string) (map[string]string, error) {
	e, err := p.ResolveEnv(envNameArg)
	if err != nil {
		return nil, err
	}
	result := map[string]string{}
	var missing []string
	for _, name := range p.Config.Consumes {
		if !e.HasValue(name) {
			missing = append(missing, name)
			continue
		}
		result[name] = e.Vars[name].Value
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		return nil, fmt.Errorf("missing values: %v (set them with `shutup set`)", missing)
	}
	return result, nil
}

// ChildEnv merges the resolved project vars onto the current process env,
// returning a slice for exec.Cmd.Env. Project vars override inherited ones.
func (p *Project) ChildEnv(envNameArg string) ([]string, error) {
	vars, err := p.Resolve(envNameArg)
	if err != nil {
		return nil, err
	}
	base := os.Environ()
	out := make([]string, 0, len(base)+len(vars))
	for _, kv := range base {
		name := kv
		for i := 0; i < len(kv); i++ {
			if kv[i] == '=' {
				name = kv[:i]
				break
			}
		}
		if _, overridden := vars[name]; !overridden {
			out = append(out, kv)
		}
	}
	for k, v := range vars {
		out = append(out, k+"="+v)
	}
	return out, nil
}

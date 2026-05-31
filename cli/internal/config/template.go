package config

// New builds a starter config for `shutup init` at the given path: no consumed
// vars yet, one env mapping (name -> id) as the default. Vars are added on
// demand via `shutup set` / `shutup use`, so a project starts as a clean slate.
func New(path, defaultEnvName, defaultEnvID string) *Config {
	return &Config{
		Consumes:   []string{},
		Envs:       map[string]string{defaultEnvName: defaultEnvID},
		DefaultEnv: defaultEnvName,
		path:       path,
	}
}

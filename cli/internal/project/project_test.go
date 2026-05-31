package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/joe/shutup/internal/config"
	"github.com/joe/shutup/internal/env"
)

// newTestProject builds a Project with an isolated store and a fresh env named dev.
func newTestProject(t *testing.T) *Project {
	t.Helper()
	t.Setenv("HOME", t.TempDir())
	st, err := env.NewLocalEnvStore()
	if err != nil {
		t.Fatal(err)
	}
	e, err := st.Create()
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.New(filepath.Join(t.TempDir(), config.Filename), "dev", e.ID)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	return &Project{Config: cfg, Store: st}
}

func statusByName(s []VarStatus) map[string]VarStatus {
	m := map[string]VarStatus{}
	for _, vs := range s {
		m[vs.Name] = vs
	}
	return m
}

func TestSetVarWiresAndStores(t *testing.T) {
	p := newTestProject(t)
	wired, err := p.SetVar("dev", "DATABASE_URL", "postgres://x", false)
	if err != nil {
		t.Fatal(err)
	}
	if !wired {
		t.Error("first SetVar should wire the consume")
	}
	if !p.Config.ConsumesVar("DATABASE_URL") {
		t.Error("project should consume DATABASE_URL")
	}
	// value landed in the env as a secret
	e, _ := p.ResolveEnv("dev")
	if e.Vars["DATABASE_URL"].Value != "postgres://x" || !e.Vars["DATABASE_URL"].Secret {
		t.Errorf("env var wrong: %+v", e.Vars["DATABASE_URL"])
	}
	// second set doesn't re-wire
	wired, _ = p.SetVar("dev", "DATABASE_URL", "y", false)
	if wired {
		t.Error("re-setting an already-consumed var should not report wired")
	}
}

func TestSetPublic(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.SetVar("dev", "PORT", "3000", true); err != nil {
		t.Fatal(err)
	}
	e, _ := p.ResolveEnv("dev")
	if e.Vars["PORT"].Secret {
		t.Error("PORT should be public")
	}
}

func TestStatusAndMissing(t *testing.T) {
	p := newTestProject(t)
	_, _ = p.SetVar("dev", "PORT", "3000", true)
	_, _ = p.Use("STRIPE_KEY") // consumed but unset

	st, err := p.Status("dev")
	if err != nil {
		t.Fatal(err)
	}
	m := statusByName(st)
	if !m["PORT"].Set || !m["PORT"].Public || m["PORT"].Value != "3000" {
		t.Errorf("PORT status wrong: %+v", m["PORT"])
	}
	if m["STRIPE_KEY"].Set {
		t.Error("STRIPE_KEY should be unset")
	}

	missing, _ := p.Missing("dev")
	if len(missing) != 1 || missing[0].Name != "STRIPE_KEY" {
		t.Errorf("expected STRIPE_KEY missing, got %+v", missing)
	}
}

func TestResolveLeastPrivilege(t *testing.T) {
	p := newTestProject(t)
	// env has an extra var the project does NOT consume.
	e, _ := p.ResolveEnv("dev")
	e.Vars["OTHER_APP_SECRET"] = env.Var{Value: "nope", Secret: true}
	_ = p.Store.Save(e)

	_, _ = p.SetVar("dev", "PORT", "3000", true)
	resolved, err := p.Resolve("dev")
	if err != nil {
		t.Fatal(err)
	}
	if _, leaked := resolved["OTHER_APP_SECRET"]; leaked {
		t.Error("Resolve must only return consumed vars (least-privilege)")
	}
	if resolved["PORT"] != "3000" {
		t.Error("Resolve should include consumed PORT")
	}
}

func TestResolveErrorsOnMissing(t *testing.T) {
	p := newTestProject(t)
	_, _ = p.Use("STRIPE_KEY") // consumed, no value
	if _, err := p.Resolve("dev"); err == nil {
		t.Error("Resolve should error when a consumed var has no value")
	}
}

func TestUseUnuse(t *testing.T) {
	p := newTestProject(t)
	if added, _ := p.Use("X"); !added {
		t.Error("Use should add")
	}
	if added, _ := p.Use("X"); added {
		t.Error("Use should be idempotent")
	}
	if removed, _ := p.Unuse("X"); !removed {
		t.Error("Unuse should remove")
	}
}

func TestExists(t *testing.T) {
	p := newTestProject(t)
	_, _ = p.Use("STRIPE_KEY")
	if ok, _ := p.Exists("dev", "STRIPE_KEY"); ok {
		t.Error("unset secret should not exist")
	}
	_, _ = p.SetVar("dev", "STRIPE_KEY", "sk", false)
	if ok, _ := p.Exists("dev", "STRIPE_KEY"); !ok {
		t.Error("set secret should exist")
	}
}

func TestDefaultEnvResolution(t *testing.T) {
	p := newTestProject(t)
	// empty env arg uses default_env (dev)
	if _, err := p.ResolveEnv(""); err != nil {
		t.Errorf("empty env should resolve to default: %v", err)
	}
}

func TestResolveEnvErrors(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.ResolveEnv("ghost"); err == nil {
		t.Error("unknown env name should error")
	}
	// env mapped but not present in the store
	p.Config.Envs["broken"] = "envlocal_00000000000000000000000000000000"
	if _, err := p.ResolveEnv("broken"); err == nil {
		t.Error("env not in store should error")
	}
	// no default and no arg
	p.Config.DefaultEnv = ""
	if _, err := p.ResolveEnv(""); err == nil {
		t.Error("no env + no default should error")
	}
}

func TestExistsPublic(t *testing.T) {
	p := newTestProject(t)
	_, _ = p.SetVar("dev", "PORT", "3000", true)
	if ok, _ := p.Exists("dev", "PORT"); !ok {
		t.Error("public PORT should exist")
	}
	if ok, _ := p.Exists("dev", "ABSENT"); ok {
		t.Error("absent var should not exist")
	}
}

func TestUnuseAbsent(t *testing.T) {
	p := newTestProject(t)
	if removed, _ := p.Unuse("NOPE"); removed {
		t.Error("Unuse of absent var should report false")
	}
}

func TestChildEnvMergesAndOverrides(t *testing.T) {
	t.Setenv("SHUTUP_TEST_INHERITED", "from-parent")
	t.Setenv("PORT", "9999") // parent env has PORT; project value should override
	p := newTestProject(t)
	_, _ = p.SetVar("dev", "PORT", "3000", true)
	_, _ = p.SetVar("dev", "DATABASE_URL", "db", false)

	kv, err := p.ChildEnv("dev")
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, e := range kv {
		if i := indexByte(e, '='); i >= 0 {
			got[e[:i]] = e[i+1:]
		}
	}
	if got["PORT"] != "3000" {
		t.Errorf("project value should override inherited PORT, got %q", got["PORT"])
	}
	if got["DATABASE_URL"] != "db" {
		t.Error("consumed secret should be injected")
	}
	if got["SHUTUP_TEST_INHERITED"] != "from-parent" {
		t.Error("unrelated inherited env should pass through")
	}
}

func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}

func TestOpenFromCWD(t *testing.T) {
	// Exercises project.Open + config.DiscoverCWD/LoadFromCWD + env store wiring.
	t.Setenv("HOME", t.TempDir())
	st, err := env.NewLocalEnvStore()
	if err != nil {
		t.Fatal(err)
	}
	e, _ := st.Create()
	dir := t.TempDir()
	cfg := config.New(filepath.Join(dir, config.Filename), "dev", e.ID)
	if err := cfg.Save(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	p, err := Open()
	if err != nil {
		t.Fatalf("Open failed: %v", err)
	}
	if _, ok := p.Config.EnvID("dev"); !ok {
		t.Error("opened project should have dev env")
	}
	// Open from a nested subdir should still find the project (walk-up).
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(sub)
	if _, err := Open(); err != nil {
		t.Errorf("Open from nested dir should walk up: %v", err)
	}
}

func TestOpenNotInProject(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	t.Chdir(t.TempDir())
	if _, err := Open(); err == nil {
		t.Error("Open outside a project should error")
	}
}

func TestOpenInvalidConfig(t *testing.T) {
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	// default_env references an env the project doesn't have -> Validate fails.
	bad := "consumes: []\nenvs: {}\ndefault_env: dev\n"
	if err := os.WriteFile(filepath.Join(dir, config.Filename), []byte(bad), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	if _, err := Open(); err == nil {
		t.Error("Open should fail validation for default_env not in envs")
	}
}

func TestSetVarBadEnv(t *testing.T) {
	p := newTestProject(t)
	if _, err := p.SetVar("ghost", "X", "v", false); err == nil {
		t.Error("SetVar with an unknown env should error")
	}
}

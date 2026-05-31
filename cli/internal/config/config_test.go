package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/joe/shutup/internal/id"
)

func tempConfig(t *testing.T) *Config {
	t.Helper()
	return New(filepath.Join(t.TempDir(), Filename), "dev", id.GenerateLocal())
}

func TestRoundTrip(t *testing.T) {
	c := tempConfig(t)
	c.AddConsume("DATABASE_URL")
	c.AddConsume("PORT")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	got, err := Load(c.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.Consumes, c.Consumes) {
		t.Errorf("consumes: got %v want %v", got.Consumes, c.Consumes)
	}
	if !reflect.DeepEqual(got.Envs, c.Envs) {
		t.Errorf("envs: got %v want %v", got.Envs, c.Envs)
	}
	if got.DefaultEnv != c.DefaultEnv {
		t.Errorf("default_env: got %q want %q", got.DefaultEnv, c.DefaultEnv)
	}
}

func TestSavePreservesComments(t *testing.T) {
	c := tempConfig(t)
	c.AddConsume("PORT")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(c.Path())
	out := string(data)
	for _, want := range []string{"safe to commit", "Variable NAMES", "share an env"} {
		if !strings.Contains(out, want) {
			t.Errorf("generated config missing comment %q\n%s", want, out)
		}
	}
}

func TestConsumesHelpers(t *testing.T) {
	c := tempConfig(t)
	if !c.AddConsume("X") {
		t.Error("AddConsume should report added")
	}
	if c.AddConsume("X") {
		t.Error("AddConsume should be idempotent")
	}
	if !c.ConsumesVar("X") {
		t.Error("ConsumesVar should be true")
	}
	if !c.RemoveConsume("X") || c.ConsumesVar("X") {
		t.Error("RemoveConsume failed")
	}
}

func TestValidate(t *testing.T) {
	if err := tempConfig(t).Validate(); err != nil {
		t.Fatalf("fresh config should be valid: %v", err)
	}

	t.Run("bad env id", func(t *testing.T) {
		c := tempConfig(t)
		c.Envs["staging"] = "not-an-id"
		if c.Validate() == nil {
			t.Error("expected error for bad env id")
		}
	})

	t.Run("default_env not in envs", func(t *testing.T) {
		c := tempConfig(t)
		c.DefaultEnv = "ghost"
		if c.Validate() == nil {
			t.Error("expected error for default_env not in envs")
		}
	})

	t.Run("incomplete is valid", func(t *testing.T) {
		c := tempConfig(t)
		c.AddConsume("STRIPE_KEY") // consumed but no value anywhere — fine
		if err := c.Validate(); err != nil {
			t.Errorf("incomplete config should be valid: %v", err)
		}
	})
}

func TestEnvID(t *testing.T) {
	c := tempConfig(t)
	if _, ok := c.EnvID("dev"); !ok {
		t.Error("dev should resolve")
	}
	if _, ok := c.EnvID("nope"); ok {
		t.Error("unknown env should not resolve")
	}
}

func TestDiscoverWalksUp(t *testing.T) {
	root := t.TempDir()
	c := New(filepath.Join(root, Filename), "dev", id.GenerateLocal())
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	found, err := Discover(deep)
	if err != nil {
		t.Fatal(err)
	}
	if found != c.Path() {
		t.Errorf("Discover: got %q want %q", found, c.Path())
	}
}

func TestDiscoverNotFound(t *testing.T) {
	if _, err := Discover(t.TempDir()); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestDiscoverCWDAndLoadFromCWD(t *testing.T) {
	dir := t.TempDir()
	c := New(filepath.Join(dir, Filename), "dev", id.GenerateLocal())
	c.AddConsume("PORT")
	if err := c.Save(); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)

	found, err := DiscoverCWD()
	if err != nil {
		t.Fatal(err)
	}
	if found != c.Path() {
		t.Errorf("DiscoverCWD: got %q want %q", found, c.Path())
	}
	loaded, err := LoadFromCWD()
	if err != nil {
		t.Fatal(err)
	}
	if !loaded.ConsumesVar("PORT") {
		t.Error("LoadFromCWD did not load the config")
	}
}

func TestLoadMissingFile(t *testing.T) {
	if _, err := Load(filepath.Join(t.TempDir(), "nope.yaml")); err == nil {
		t.Error("expected error loading a missing file")
	}
}

func TestLoadMalformed(t *testing.T) {
	p := filepath.Join(t.TempDir(), Filename)
	if err := os.WriteFile(p, []byte("consumes: [unterminated\n  : :"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(p); err == nil {
		t.Error("expected parse error for malformed yaml")
	}
}

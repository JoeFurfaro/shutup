package env

import (
	"os"
	"strings"
	"testing"
)

func newTestStore(t *testing.T) *LocalEnvStore {
	t.Helper()
	t.Setenv("HOME", t.TempDir()) // isolate ~/.shutup from the real home
	s, err := NewLocalEnvStore()
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestCreateLoad(t *testing.T) {
	s := newTestStore(t)
	e, err := s.Create()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(e.ID, "envlocal_") {
		t.Errorf("expected envlocal_ id, got %q", e.ID)
	}
	got, err := s.Load(e.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != e.ID || got.Source != SourceLocal {
		t.Errorf("loaded env mismatch: %+v", got)
	}
}

func TestLoadMissing(t *testing.T) {
	s := newTestStore(t)
	if _, err := s.Load("envlocal_00000000000000000000000000000000"); err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestSaveLoadVars(t *testing.T) {
	s := newTestStore(t)
	e, _ := s.Create()
	e.Vars["DATABASE_URL"] = Var{Value: "postgres://x", Secret: true}
	e.Vars["PORT"] = Var{Value: "3000", Secret: false}
	if err := s.Save(e); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Load(e.ID)
	if got.Vars["DATABASE_URL"].Value != "postgres://x" || !got.Vars["DATABASE_URL"].Secret {
		t.Errorf("secret var roundtrip failed: %+v", got.Vars["DATABASE_URL"])
	}
	if !got.HasValue("PORT") || got.Vars["PORT"].Secret {
		t.Errorf("public var roundtrip failed: %+v", got.Vars["PORT"])
	}
	if got.HasValue("NOPE") {
		t.Error("HasValue should be false for absent var")
	}
}

func TestList(t *testing.T) {
	s := newTestStore(t)
	a, _ := s.Create()
	b, _ := s.Create()
	got, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 envs, got %d", len(got))
	}
	ids := map[string]bool{got[0].ID: true, got[1].ID: true}
	if !ids[a.ID] || !ids[b.ID] {
		t.Errorf("List missing created envs: %v", ids)
	}
}

func TestListEmpty(t *testing.T) {
	s := newTestStore(t)
	got, err := s.List()
	if err != nil {
		t.Fatalf("List on empty store should not error: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty, got %d", len(got))
	}
}

func TestDelete(t *testing.T) {
	s := newTestStore(t)
	e, _ := s.Create()
	if err := s.Delete(e.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Load(e.ID); err != ErrNotFound {
		t.Errorf("expected ErrNotFound after delete, got %v", err)
	}
	if err := s.Delete(e.ID); err != nil {
		t.Errorf("deleting missing env should not error: %v", err)
	}
}

func TestBundleStripsSecretValues(t *testing.T) {
	e := &Env{ID: "envlocal_x", Source: SourceLocal, Vars: map[string]Var{
		"DATABASE_URL": {Value: "postgres://secret", Secret: true},
		"PORT":         {Value: "3000", Secret: false},
	}}
	b := e.Bundle()
	if b.Vars["DATABASE_URL"].Value != "" {
		t.Error("bundle must not contain secret values")
	}
	if !b.Vars["DATABASE_URL"].Secret {
		t.Error("bundle should keep the secret flag/name")
	}
	if b.Vars["PORT"].Value != "3000" {
		t.Error("bundle should keep public values")
	}
	// marshaled bytes must not contain the secret value
	data, _ := MarshalBundle(e)
	if strings.Contains(string(data), "postgres://secret") {
		t.Errorf("marshaled bundle leaked a secret value:\n%s", data)
	}
}

func TestImportBundlePreservesLocalSecret(t *testing.T) {
	s := newTestStore(t)
	// Local env already has a secret value set.
	local := &Env{ID: "envlocal_shared", Source: SourceLocal, Vars: map[string]Var{
		"DATABASE_URL": {Value: "my-local-db", Secret: true},
	}}
	if err := s.Save(local); err != nil {
		t.Fatal(err)
	}
	// Incoming bundle: public PORT + secret DATABASE_URL (no value) + new secret API_KEY.
	bundle := &Env{ID: "envlocal_shared", Source: SourceLocal, Vars: map[string]Var{
		"PORT":         {Value: "8080", Secret: false},
		"DATABASE_URL": {Secret: true},
		"API_KEY":      {Secret: true},
	}}
	merged, err := s.ImportBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if merged.Vars["DATABASE_URL"].Value != "my-local-db" {
		t.Error("import must preserve a locally-set secret value")
	}
	if merged.Vars["PORT"].Value != "8080" {
		t.Error("import should apply public values from the bundle")
	}
	if merged.Vars["API_KEY"].Value != "" || !merged.Vars["API_KEY"].Secret {
		t.Error("import should add new secret as an unset placeholder")
	}
}

func TestMarshalUnmarshalBundleRoundTrip(t *testing.T) {
	e := &Env{ID: "envlocal_rt", Source: SourceLocal, Vars: map[string]Var{
		"DATABASE_URL": {Value: "secret-db", Secret: true},
		"PORT":         {Value: "3000", Secret: false},
	}}
	data, err := MarshalBundle(e)
	if err != nil {
		t.Fatal(err)
	}
	got, err := UnmarshalBundle(data)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != "envlocal_rt" {
		t.Errorf("id lost: %q", got.ID)
	}
	if got.Vars["PORT"].Value != "3000" {
		t.Error("public value lost in round-trip")
	}
	if got.Vars["DATABASE_URL"].Value != "" || !got.Vars["DATABASE_URL"].Secret {
		t.Error("secret should round-trip as name-only placeholder")
	}
}

func TestUnmarshalBundleStripsSecretDefensively(t *testing.T) {
	// A hand-crafted/malicious bundle that wrongly includes a secret value must
	// be stripped on import.
	raw := []byte("id: envlocal_x\nsource: local\nvars:\n  API_KEY:\n    value: leaked\n    secret: true\n")
	got, err := UnmarshalBundle(raw)
	if err != nil {
		t.Fatal(err)
	}
	if got.Vars["API_KEY"].Value != "" {
		t.Errorf("UnmarshalBundle must strip secret values, got %q", got.Vars["API_KEY"].Value)
	}
}

func TestUnmarshalBundleMalformed(t *testing.T) {
	if _, err := UnmarshalBundle([]byte(": : not valid yaml")); err == nil {
		t.Error("expected error for malformed bundle")
	}
}

func TestUnmarshalBundleEmptyVars(t *testing.T) {
	got, err := UnmarshalBundle([]byte("id: envlocal_x\nsource: local\n"))
	if err != nil {
		t.Fatal(err)
	}
	if got.Vars == nil {
		t.Error("Vars should be initialized to non-nil")
	}
}

func TestImportBundleNewEnv(t *testing.T) {
	s := newTestStore(t)
	bundle := &Env{ID: "envlocal_brandnew", Source: SourceLocal, Vars: map[string]Var{
		"PORT": {Value: "3000", Secret: false},
	}}
	if _, err := s.ImportBundle(bundle); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load("envlocal_brandnew")
	if err != nil {
		t.Fatalf("imported env should exist: %v", err)
	}
	if got.Vars["PORT"].Value != "3000" {
		t.Error("imported new env missing public value")
	}
}

func TestSavedFileMode(t *testing.T) {
	s := newTestStore(t)
	e, _ := s.Create()
	info, err := os.Stat(s.path(e.ID))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Errorf("env file should be 0600, got %o", info.Mode().Perm())
	}
}

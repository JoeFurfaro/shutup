package id

import "testing"

func TestGenerateLocalIsValid(t *testing.T) {
	for i := 0; i < 100; i++ {
		got := GenerateLocal()
		if !IsValid(got) {
			t.Fatalf("GenerateLocal produced invalid id: %q", got)
		}
		if !IsLocal(got) {
			t.Fatalf("GenerateLocal id not recognized as local: %q", got)
		}
	}
}

func TestGenerateUnique(t *testing.T) {
	if a, b := GenerateLocal(), GenerateLocal(); a == b {
		t.Fatalf("expected distinct ids, got %q twice", a)
	}
}

func TestIsValid(t *testing.T) {
	cases := map[string]bool{
		"envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47": true,
		"env_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47":      true,  // API prefix accepted (forward-compat)
		"envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c4":  false, // 31 hex
		"envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c477": false, // 33 hex
		"envlocal_7F3A9C2E4B1D4E8A9F6C2D5B8A1E3C47": false,  // uppercase
		"7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47":          false,  // no prefix
		"proj_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47":     false,  // wrong prefix
		"envlocal_":                                  false,
		"":                                           false,
	}
	for in, want := range cases {
		if got := IsValid(in); got != want {
			t.Errorf("IsValid(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestIsLocal(t *testing.T) {
	if !IsLocal("envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47") {
		t.Error("envlocal_ id should be local")
	}
	if IsLocal("env_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47") {
		t.Error("env_ id should not be local")
	}
}

// Package id generates and validates shutup environment identifiers.
//
// An env id is a prefix followed by 32 lowercase hex characters (a dashless v4
// UUID), e.g. envlocal_7f3a9c2e4b1d4e8a9f6c2d5b8a1e3c47.
//
// The prefix encodes the env's backing, visible from the id alone:
//   - "envlocal_" — a local env, stored on this machine (~/.shutup/envs/).
//   - "env_"      — reserved for future API-backed envs.
//
// An env id is the stable identity of an environment: collision-free across
// repos, survives renames, and is the join key the cloud API will use. Projects
// reference envs by id; the human-facing name is a per-project label.
package id

import (
	"regexp"
	"strings"

	"github.com/google/uuid"
)

// Prefixes for env ids.
const (
	LocalPrefix = "envlocal_" // local, machine-stored env
	APIPrefix   = "env_"      // reserved for future API-backed env
)

// pattern accepts either prefix followed by 32 hex chars. Generate only emits
// LocalPrefix in v1; APIPrefix is accepted for forward compatibility.
var pattern = regexp.MustCompile(`^(envlocal_|env_)[0-9a-f]{32}$`)

// GenerateLocal returns a fresh, random local env id (envlocal_<32hex>).
func GenerateLocal() string {
	return LocalPrefix + strings.ReplaceAll(uuid.NewString(), "-", "")
}

// IsValid reports whether s is a well-formed env id (local or API).
func IsValid(s string) bool {
	return pattern.MatchString(s)
}

// IsLocal reports whether s is a local (machine-stored) env id.
func IsLocal(s string) bool {
	return strings.HasPrefix(s, LocalPrefix) && IsValid(s)
}

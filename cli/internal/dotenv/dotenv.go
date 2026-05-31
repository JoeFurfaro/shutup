// Package dotenv parses a subset of .env files for `shutup import`.
//
// Supported: KEY=value, `export KEY=value`, # comments, blank lines, single/
// double-quoted values, and a trailing ` # comment` on unquoted values.
// NOT supported (out of scope for the demo): multiline values, ${VAR}
// interpolation, command substitution.
package dotenv

import (
	"bufio"
	"io"
	"strings"
)

// Pair is one parsed variable, preserving file order.
type Pair struct {
	Name  string
	Value string
}

// Parse reads .env content into ordered pairs. Malformed lines are skipped.
func Parse(r io.Reader) ([]Pair, error) {
	var pairs []Pair
	sc := bufio.NewScanner(r)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(line, "export ")
		eq := strings.IndexByte(line, '=')
		if eq <= 0 {
			continue // no key, skip
		}
		name := strings.TrimSpace(line[:eq])
		if name == "" {
			continue
		}
		value := strings.TrimSpace(line[eq+1:])
		value = unquote(value)
		pairs = append(pairs, Pair{Name: name, Value: value})
	}
	return pairs, sc.Err()
}

func unquote(v string) string {
	if len(v) >= 2 {
		if (v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'') {
			inner := v[1 : len(v)-1]
			if v[0] == '"' {
				inner = strings.NewReplacer(`\n`, "\n", `\t`, "\t", `\"`, `"`, `\\`, `\`).Replace(inner)
			}
			return inner
		}
	}
	// Unquoted: strip a trailing " # comment".
	if i := strings.Index(v, " #"); i >= 0 {
		v = strings.TrimSpace(v[:i])
	}
	return v
}

package dotenv

import (
	"strings"
	"testing"
)

func TestParse(t *testing.T) {
	in := `# comment
export DATABASE_URL=postgres://localhost/db
PORT=3000
LOG_LEVEL="debug"
QUOTED='single'
WITH_COMMENT=value # trailing comment

NOEQ
EMPTY=
`
	pairs, err := Parse(strings.NewReader(in))
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	var order []string
	for _, p := range pairs {
		got[p.Name] = p.Value
		order = append(order, p.Name)
	}
	want := map[string]string{
		"DATABASE_URL": "postgres://localhost/db",
		"PORT":         "3000",
		"LOG_LEVEL":    "debug",
		"QUOTED":       "single",
		"WITH_COMMENT": "value",
		"EMPTY":        "",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("%s = %q, want %q", k, got[k], v)
		}
	}
	if _, ok := got["NOEQ"]; ok {
		t.Error("line without = should be skipped")
	}
	// order preserved
	if order[0] != "DATABASE_URL" || order[1] != "PORT" {
		t.Errorf("order not preserved: %v", order)
	}
}

func TestParseEscapes(t *testing.T) {
	pairs, _ := Parse(strings.NewReader(`MSG="line1\nline2"`))
	if pairs[0].Value != "line1\nline2" {
		t.Errorf("escape not handled: %q", pairs[0].Value)
	}
}

package gdp

import (
	"testing"
)

// TODO: can use way more testing.
func TestParseGopkgPath(t *testing.T) {
	p := "gopkg.in/yaml.v2"
	owner, repo, major, err := ParseGopkgPath(p)
	if err != nil {
		t.Fatal(err)
	}

	eq(t, "go-yaml", owner)
	eq(t, "yaml", repo)
	eq(t, "v2", major)
}

func eq(t *testing.T, s1, s2 string) {
	t.Helper()
	if s1 != s2 {
		t.Fatalf("expected %v and %v to be equal", s1, s2)
	}
}

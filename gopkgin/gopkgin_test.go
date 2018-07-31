package gopkgin

import (
	"context"
	"testing"

	"github.com/marwan-at-work/gdp"
	"github.com/marwan-at-work/gdp/github"
)

var gch = github.New("") // TODO:
var gpi = New(gdp.New(gch), gch)

func TestList(t *testing.T) {
	tags, err := gpi.List(context.Background(), "gopkg.in/yaml.v2")
	if err != nil {
		t.Fatal(err)
	}

	t.Fatal(tags)
}

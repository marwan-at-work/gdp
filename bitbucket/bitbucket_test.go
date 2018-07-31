package bitbucket

import (
	"archive/zip"
	"context"
	"io"
	"testing"

	"github.com/marwan-at-work/gdp"
	"github.com/spf13/afero"
)

var c = gdp.New(New())
var ctx = context.Background()

// TODO: create test repos.
// TODO: test +incompatible and metadata
// TODO: test v1 as opposed to v1.2.3

func TestList(t *testing.T) {
	_, err := c.List(ctx, "bitbucket.org/pkg/inflect")
	if err != nil {
		t.Fatal(err)
	}
}

func TestLatest(t *testing.T) {
	info, err := c.Latest(ctx, "bitbucket.org/pkg/inflect")
	if err != nil {
		t.Fatal(err)
	}

	t.Fatal(info)
}

func TestInfo(t *testing.T) {
	info, err := c.Info(ctx, "bitbucket.org/pkg/inflect", "v0.0.0-20130829110746-8961c3750a47")
	if err != nil {
		t.Fatal(err)
	}

	t.Fatalf("%+v", info)
}

func TestGoMod(t *testing.T) {
	mod, err := c.GoMod(ctx, "bitbucket.org/pkg/inflect", "v0.0.0-20130829110746-8961c3750a47")
	if err != nil {
		t.Fatal(err)
	}

	t.Fatalf("%s", mod)
}

func TestZip(t *testing.T) {
	rdr, err := c.Zip(context.Background(), "bitbucket.org/pkg/inflect", "v0.0.0-20130829110746-8961c3750a47", "")
	if err != nil {
		t.Fatal(err)
	}

	fs := afero.NewMemMapFs()
	f, _ := fs.Create("./temp")
	defer f.Close()
	_, err = io.Copy(f, rdr)
	if err != nil {
		t.Fatal(err)
	}

	stat, err := f.Stat()
	if err != nil {
		t.Fatal(err)
	}

	_, err = zip.NewReader(f, stat.Size())
	if err != nil {
		t.Fatal(err)
	}
}

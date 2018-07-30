package github

import (
	"archive/zip"
	"bytes"
	"context"
	"io"
	"reflect"
	"testing"
	"time"

	"github.com/marwan-at-work/gdp"
	"github.com/spf13/afero"
)

var d = gdp.New(New("")) // TODO: token from env var.

// TODO: create test repos as these tests will eventually fail when they introduce new versions.

func TestGoMod(t *testing.T) {
	bts, err := d.GoMod(context.Background(), "github.com/kr/pretty", "v0.1.0")
	if err != nil {
		t.Fatal(err)
	}

	expected := []byte(`module "github.com/kr/pretty"

require "github.com/kr/text" v0.1.0
`)

	if !bytes.Equal(expected, bts) {
		t.Fatalf("unexpected mod file %s", bts)
	}
}

func TestList(t *testing.T) {
	ss, err := d.List(context.Background(), "github.com/pkg/errors")
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{"v0.8.0", "v0.7.1", "v0.7.0", "v0.6.0", "v0.5.1", "v0.5.0", "v0.4.0", "v0.3.0", "v0.2.0", "v0.1.0"}

	if !reflect.DeepEqual(ss, expected) {
		t.Fatalf("unexpected list versions %v", ss)
	}
}

func TestInfo(t *testing.T) {
	info, err := d.Info(context.Background(), "github.com/pkg/errors", "v0.8.0")
	if err != nil {
		t.Fatal(err)
	}

	expected := &gdp.RevInfo{
		Name:    "645ef00459ed84a119197bfb8d8205042c6df63d",
		Short:   "v0.8.0",
		Version: "v0.8.0",
		Time:    time.Date(2016, 9, 29, 1, 48, 1, 0, time.UTC),
	}

	if !reflect.DeepEqual(info, expected) {
		t.Fatalf("unexpected rev info %#v", info)
	}
}

func TestLatest(t *testing.T) {
	info, err := d.Latest(context.Background(), "github.com/pkg/errors")
	if err != nil {
		t.Fatal(err)
	}

	expected := &gdp.RevInfo{
		Name:    "816c9085562cd7ee03e7f8188a1cfd942858cded",
		Short:   "816c9085562c",
		Version: "v0.0.0-20180311214515-816c9085562c",
		Time:    time.Date(2018, 3, 11, 21, 45, 15, 0, time.UTC),
	}

	if !reflect.DeepEqual(info, expected) {
		t.Fatalf("unexpected rev info %#v", info)
	}
}

func TestZip(t *testing.T) {
	rdr, err := d.Zip(context.Background(), "github.com/pkg/errors", "v0.8.0")
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

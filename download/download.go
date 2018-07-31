package download

import (
	"context"
	"io"
	"strings"

	"github.com/marwan-at-work/gdp"
	"github.com/marwan-at-work/gdp/bitbucket"
	"github.com/marwan-at-work/gdp/github"
	"github.com/marwan-at-work/gdp/gopkgin"
)

const (
	gh  = "github.com"
	bb  = "bitbucket.org"
	gpi = "gopkg.in"
)

// New returns a DownloadProtocol that implements
// Github, Bitbucket, and Gopkg.in.
func New(githubToken string) gdp.DownloadProtocol {
	var d download
	gch := github.New(githubToken)
	g := gdp.New(gch)
	b := gdp.New(bitbucket.New())
	gpiDP := gopkgin.New(g, gch)
	d.protos = map[string]gdp.DownloadProtocol{
		gh:  g,
		bb:  b,
		gpi: gpiDP,
	}

	return &d
}

type download struct {
	protos map[string]gdp.DownloadProtocol
}

func (d *download) List(ctx context.Context, module string) ([]string, error) {
	return d.deduceProtocol(module).List(ctx, module)
}

func (d *download) Info(ctx context.Context, module, version string) (*gdp.RevInfo, error) {
	return d.deduceProtocol(module).Info(ctx, module, version)
}

func (d *download) Latest(ctx context.Context, module string) (*gdp.RevInfo, error) {
	return d.deduceProtocol(module).Latest(ctx, module)
}

func (d *download) GoMod(ctx context.Context, module, version string) ([]byte, error) {
	return d.deduceProtocol(module).GoMod(ctx, module, version)
}

func (d *download) Zip(ctx context.Context, module, version, zipPrefix string) (io.Reader, error) {
	return d.deduceProtocol(module).Zip(ctx, module, version, zipPrefix)
}

func (d *download) deduceProtocol(module string) gdp.DownloadProtocol {
	for prefix, dp := range d.protos {
		if strings.HasPrefix(module, prefix) {
			return dp
		}
	}

	return noOpProtocol{}
}

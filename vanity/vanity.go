package vanity

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/marwan-at-work/gdp"
	"github.com/pkg/errors"
)

// New returns a vanity deducer.
func New(gh, bb gdp.DownloadProtocol) gdp.DownloadProtocol {
	return &protocol{
		github:    gh,
		bitbucket: bb,
		nop:       gdp.NoOpProtocol(),
	}
}

type redir struct {
	vcs  string
	base string
	path string
}

func deduceVanity(path string) (redir, error) {
	var r redir
	u, err := url.Parse(path)
	if err != nil {
		return r, err
	}
	u.Scheme = "http"
	u.RawQuery = "go-get=1"

	resp, err := http.Get(u.String())
	if err != nil {
		return r, err
	}
	defer resp.Body.Close()
	document, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return r, err
	}

	document.Find("meta").Each(func(i int, s *goquery.Selection) {
		val, _ := s.Attr("name")
		if val != "go-import" {
			return
		}

		cnt, _ := s.Attr("content")
		fields := strings.Fields(cnt)
		if len(fields) != 3 {
			return // return err
		}
		r.base = fields[0]
		r.vcs = fields[1]
		u, err := url.Parse(fields[2])
		if err != nil {
			return
		}
		r.path = u.Hostname() + u.Path
	})

	if r.base != path {
		return r, fmt.Errorf("%v != %v", r.base, path)
	}

	return r, nil
}

type protocol struct {
	github    gdp.DownloadProtocol
	bitbucket gdp.DownloadProtocol
	nop       gdp.DownloadProtocol
}

func (p *protocol) List(ctx context.Context, module string) ([]string, error) {
	r, err := deduceVanity(module)
	if err != nil {
		return nil, errors.Wrap(err, "vanity.List")
	}

	return p.deduce(r).List(ctx, r.path)
}

func (p *protocol) deduce(r redir) gdp.DownloadProtocol {
	switch {
	case strings.HasPrefix(r.path, "github.com"):
		return p.github
	case strings.HasPrefix(r.path, "bitbucket.org"):
		return p.bitbucket
	}

	return p.nop
}

func (p *protocol) Info(ctx context.Context, module string, version string) (*gdp.RevInfo, error) {
	r, err := deduceVanity(module)
	if err != nil {
		return nil, errors.Wrap(err, "vanity.Info")
	}

	return p.deduce(r).Info(ctx, r.path, version)
}

func (p *protocol) Latest(ctx context.Context, module string) (*gdp.RevInfo, error) {
	r, err := deduceVanity(module)
	if err != nil {
		return nil, errors.Wrap(err, "vanity.Latest")
	}

	return p.deduce(r).Latest(ctx, r.path)
}

func (p *protocol) GoMod(ctx context.Context, module string, version string) ([]byte, error) {
	r, err := deduceVanity(module)
	if err != nil {
		return nil, errors.Wrap(err, "vanity.GoMod")
	}

	bts, err := p.deduce(r).GoMod(ctx, r.path, version)
	if err != nil {
		return nil, errors.Wrap(err, "vanity.GoMod")
	}

	emptyMod := []byte(fmt.Sprintf("module %v\n", r.path))
	if bytes.Equal(bts, emptyMod) {
		bts = []byte(fmt.Sprintf("module %v\n", module))
	}

	return bts, nil
}

func (p *protocol) Zip(ctx context.Context, module string, version string, zipPrefix string) (io.Reader, error) {
	r, err := deduceVanity(module)
	if err != nil {
		return nil, errors.Wrap(err, "vanity.Zip")
	}

	return p.deduce(r).Zip(ctx, r.path, version, r.base)
}

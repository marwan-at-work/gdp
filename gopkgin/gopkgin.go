package gopkgin

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/marwan-at-work/gdp"
	"github.com/marwan-at-work/vgop/semver"
	"github.com/pkg/errors"
)

// New returns a Gopkg.in DownloadProtocol
// it is not a codehose, and therefore it needs
// a githubProtocol to use for the real data.
func New(gdp gdp.DownloadProtocol) gdp.DownloadProtocol {
	return &downloadProtocol{gdp}
}

type downloadProtocol struct {
	gdp gdp.DownloadProtocol
}

func (ch *downloadProtocol) githubPath(module string) (string, string, error) {
	owner, repo, major, err := gdp.ParseGopkgPath(module)
	if err != nil {
		return "", "", errors.Wrap(err, "gopkgin.githubPath")
	}
	return "github.com/" + owner + "/" + repo, major, nil
}

func (ch *downloadProtocol) List(ctx context.Context, module string) ([]string, error) {
	path, major, err := ch.githubPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "gopkgin.List")
	}

	tags, err := ch.gdp.List(ctx, path)
	if err != nil {
		return nil, errors.Wrap(err, "gopkgin.List")
	}

	filtered := []string{}
	for _, t := range tags {
		if semver.Major(t) != major {
			continue
		}

		filtered = append(filtered, t)
	}

	return filtered, nil
}

// Info maybe panic here since gopkg.in always has versions, or handle v0?
func (ch *downloadProtocol) Info(ctx context.Context, module string, version string) (*gdp.RevInfo, error) {
	path, _, err := ch.githubPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "gopkgin.Info")
	}

	return ch.gdp.Info(ctx, path, version)
}

// maybe panic here since gopkg.in always has versions, or handle v0?
func (ch *downloadProtocol) Latest(ctx context.Context, module string) (*gdp.RevInfo, error) {
	path, _, err := ch.githubPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "gopkgin.Info")
	}

	return ch.gdp.Latest(ctx, path)
}

func (ch *downloadProtocol) GoMod(ctx context.Context, module string, version string) ([]byte, error) {
	path, _, err := ch.githubPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "gopkgin.Info")
	}

	bts, err := ch.gdp.GoMod(ctx, path, version)
	if err != nil {
		return nil, errors.Wrap(err, "gopkg.Info")
	}

	emptyMod := []byte(fmt.Sprintf("module %v\n", path))
	if bytes.Equal(bts, emptyMod) {
		bts = []byte(fmt.Sprintf("module %v\n", module))
	}

	return bts, nil
}

func (ch *downloadProtocol) Zip(ctx context.Context, module, version, zipPrefix string) (io.Reader, error) {
	path, _, err := ch.githubPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "gopkgin.Info")
	}

	return ch.gdp.Zip(ctx, path, version, module)
}

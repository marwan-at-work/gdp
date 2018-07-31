package gdp

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/marwan-at-work/vgop/semver"
	"github.com/pkg/errors"
)

type generic struct {
	ch CodeHost
}

// New takes a CodeHost interface and returns
// a generic DownloadProtocol that knows how to use
// the CodeHost to retrieve module info.
func New(ch CodeHost) DownloadProtocol {
	return &generic{ch}
}

func (g *generic) List(ctx context.Context, module string) ([]string, error) {
	tags := []string{}
	owner, repo, err := SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "generic.splitPath")
	}

	repoTags, err := g.ch.Tags(ctx, owner, repo)
	if err != nil {
		return nil, errors.Wrap(err, "generic.Tags")
	}

	for _, t := range repoTags {
		if semver.IsValid(t) && semver.Canonical(t) == t {
			tags = append(tags, t)
		}
	}

	return tags, nil

}

func (g *generic) Info(ctx context.Context, module string, version string) (*RevInfo, error) {
	version = strings.Replace(version, "+incompatible", "", 1)
	owner, repo, err := SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "info.SplitPath")
	}
	if IsPseudo(version) {
		sha, err := ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "info.shaFromPseudo")
		}
		return g.ch.CommitInfo(ctx, owner, repo, sha)
	}

	return g.ch.TagInfo(ctx, owner, repo, version)
}

func (g *generic) Latest(ctx context.Context, module string) (*RevInfo, error) {
	var ri RevInfo
	owner, repo, err := SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "latest.splitPath")
	}

	sha, t, err := g.ch.LatestCommit(ctx, owner, repo)
	if err != nil {
		return nil, errors.Wrap(err, "latest.LatestCommit")
	}

	ri.Name = sha
	ri.Short = ri.Name[:12]
	ri.Time = t
	ri.Version = Pseudo(ri.Time, ri.Short)

	return &ri, nil
}

func (g *generic) GoMod(ctx context.Context, module string, version string) ([]byte, error) {
	version = strings.Replace(version, "+incompatible", "", 1)
	var err error
	owner, repo, err := SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "goMod.splitPath")
	}
	if IsPseudo(version) {
		version, err = ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "goMod.shaFromPseudo")
		}
	}

	modBts, err := g.ch.GetModFile(ctx, owner, repo, version)
	if err == ErrNotFound {
		return []byte(fmt.Sprintf("module %v\n", module)), nil
	} else if err != nil {
		return nil, errors.Wrap(err, "goMod.GetModFile")
	}

	return modBts, nil
}

func (g *generic) Zip(ctx context.Context, module, version, zipPrefix string) (io.Reader, error) {
	ref := strings.Replace(version, "+incompatible", "", 1)
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		ref, err = ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "zip.shaFromPseudo")
		}
	}
	owner, repo, err := SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "zip.splitPath")
	}
	u, err := g.ch.TarURL(ctx, owner, repo, ref)
	if err != nil {
		return nil, errors.Wrap(err, "zip.getURL")
	}

	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "zip.httpGet")
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "zip.gzipNewReader")
	}

	t := tar.NewReader(gr)
	t.Next()             // pax
	dir, err := t.Next() // top level dir
	if err != nil {
		return nil, errors.Wrap(err, "zip.tarNext")
	}

	goModName := g.org(module) + "/" + owner + "/" + repo + "@" + version + "/"
	if zipPrefix != "" {
		goModName = zipPrefix + "@" + version + "/"
	}
	dirName := dir.Name
	pr, pw := io.Pipe()
	go func() {
		zw := zip.NewWriter(pw)
		for {
			h, err := t.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				zw.Close()
				pw.CloseWithError(errors.Wrap(err, "zip.tarNext"))
				return
			} else if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
				continue
			}

			path := strings.Replace(h.Name, dirName, goModName, 1)
			w, err := zw.Create(path)
			if err != nil {
				zw.Close()
				pw.CloseWithError(errors.Wrap(err, "zip.zipCreate"))
				return
			}

			_, err = io.Copy(w, t)
			if err != nil {
				zw.Close()
				pw.CloseWithError(errors.Wrap(err, "zip.ioCopy"))
				return
			}
		}
		zw.Close()
		pw.Close()
	}()

	return pr, nil
}

func (g *generic) org(path string) string {
	return strings.Split(path, "/")[0]
}

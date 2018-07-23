package gdp

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"

	"golang.org/x/oauth2"

	"github.com/google/go-github/github"
	"github.com/marwan-at-work/vgop/semver"
)

// DownloadProtocol of cmd/go
type DownloadProtocol interface {
	List(ctx context.Context, module string) ([]string, error)
	Info(ctx context.Context, module, version string) (*RevInfo, error)
	Latest(ctx context.Context, module string) (*RevInfo, error)
	GoMod(ctx context.Context, module, version string) ([]byte, error)
	Zip(ctx context.Context, module, version string) (io.Reader, error) // potentially zip.ReadCloser?
}

// RevInfo describes a single revision in a module repository.
type RevInfo struct {
	Version string    // version string
	Name    string    // complete ID in underlying repository
	Short   string    // shortened ID, for use in pseudo-version
	Time    time.Time // commit time
}

type dp struct {
	c *github.Client
}

// New returns a Github Implementation of the Download Protocol
func New(tok string) DownloadProtocol {
	var d dp
	var client *http.Client
	if tok != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
		client = oauth2.NewClient(oauth2.NoContext, ts)
	}

	d.c = github.NewClient(client)

	return &d
}

func (d *dp) List(ctx context.Context, module string) ([]string, error) {
	owner, repo, err := splitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}
	var allTags []*github.RepositoryTag
	var tags []string
	page := 1
	for {
		tags, _, err := d.c.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			return nil, errors.Wrapf(err, "list tags page %v:", page)
		}

		if len(tags) == 0 {
			break
		}

		allTags = append(allTags, tags...)
		page++
	}
	for _, t := range allTags {
		if tag := t.GetName(); semver.IsValid(tag) {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func (d *dp) Info(ctx context.Context, module, version string) (*RevInfo, error) {
	var ri RevInfo
	var untagged bool
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		untagged = true
		version, err = shaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "info.shaFromPseudo")
		}
	}
	owner, repo, err := splitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "info.SplitPath")
	}

	c, _, err := d.c.Repositories.GetCommit(ctx, owner, repo, version)
	if err != nil {
		return nil, errors.Wrapf(err, "info.GetCommit failed for %v", module)
	}
	ri.Name = c.GetSHA()
	ri.Short = version
	if untagged {
		ri.Short = ri.Name[:12]
	}
	ri.Time = c.GetCommit().GetCommitter().GetDate()
	ri.Version = version
	if untagged {
		ri.Version = pseudo(ri.Time, ri.Short)
	}
	return &ri, nil
}

func (d *dp) Latest(ctx context.Context, module string) (*RevInfo, error) {
	var ri RevInfo
	owner, repo, err := splitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "Latest.splitPath")
	}

	r, _, err := d.c.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, errors.Wrap(err, "Latest.repoGet")
	}

	ref := r.GetDefaultBranch()
	c, _, err := d.c.Repositories.GetCommit(ctx, owner, repo, ref)
	if err != nil {
		return nil, errors.Wrap(err, "Latest.repoGetCOmmit")
	}

	ri.Name = c.GetSHA()
	ri.Short = ri.Name[:12]
	ri.Time = c.GetCommit().GetCommitter().GetDate()
	ri.Version = pseudo(ri.Time, ri.Short)

	return &ri, nil
}

// format for a shortened commit sha: YYYYMMDDHHMMSS
const format = "20060102150405"

func pseudo(t time.Time, shortSha string) string {
	return "v0.0.0-" + t.Format(format) + "-" + shortSha
}

func shaFromPseudo(sv string) (string, error) {
	vinfo := strings.Split(sv, "-")
	if len(vinfo) < 3 {
		return "", errors.New("incorrect pseudo version: " + sv)
	}

	return vinfo[2], nil
}

func (d *dp) GoMod(ctx context.Context, module, version string) ([]byte, error) {
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		version, err = shaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "GoMod.shaFromPseudo")
		}
	}
	owner, repo, err := splitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "GoMod.splitPath")
	}
	fc, _, resp, err := d.c.Repositories.GetContents(ctx, owner, repo, "go.mod", &github.RepositoryContentGetOptions{
		Ref: version,
	})
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return []byte(fmt.Sprintf("module %v", module)), nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "GoMod.GetContents")
	}

	str, err := fc.GetContent()

	return []byte(str), errors.Wrap(err, "GoMod.GetContent")
}

func (d *dp) Zip(ctx context.Context, module, version string) (io.Reader, error) {
	ref := version
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		ref, err = shaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "Zip.shaFromPseudo")
		}
	}
	owner, repo, err := splitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "Zip.splitPath")
	}
	u, err := d.getURL(ctx, owner, repo, ref)
	if err != nil {
		return nil, errors.Wrap(err, "Zip.getURL")
	}

	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "Zip.httpGet")
	}

	gr, err := gzip.NewReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Zip.gzipNewReader")
	}

	t := tar.NewReader(gr)
	t.Next()             // pax
	dir, err := t.Next() // top level dir
	if err != nil {
		return nil, errors.Wrap(err, "Zip.tarNext")
	}

	goModName := "github.com/" + owner + "/" + repo + "@" + version + "/"
	dirName := dir.Name
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	defer zw.Close()
	for {
		h, err := t.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, errors.Wrap(err, "Zip.tarNext")
		} else if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
			continue
		}

		path := strings.Replace(h.Name, dirName, goModName, 1)
		w, err := zw.Create(path)
		if err != nil {
			return nil, errors.Wrap(err, "Zip.zipCreate")
		}

		_, err = io.Copy(w, t)
		if err != nil {
			return nil, errors.Wrap(err, "Zip.ioCopu")
		}
	}

	return &buf, nil
}

func (d *dp) getURL(ctx context.Context, owner, repo, ref string) (string, error) {
	url, _, err := d.c.Repositories.GetArchiveLink(
		ctx,
		owner,
		repo,
		github.Tarball,
		&github.RepositoryContentGetOptions{Ref: ref},
	)
	if err != nil {
		return "", errors.Wrap(err, "GetArchiveLink")
	}

	return url.String(), nil
}

func splitPath(path string) (owner, repo string, err error) {
	els := strings.Split(path, "/")
	switch els[0] {
	case "github.com":
		if len(els) != 3 {
			return "", "", errors.New("splitPath: unparsable github path: " + path)
		}
		owner = els[1]
		repo = els[2]

		return owner, repo, nil
	}

	return "", "", errors.New("splitPath: unsupported API")
}

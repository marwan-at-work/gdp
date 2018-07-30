package github

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/google/go-github/github"
	"github.com/marwan-at-work/gdp"
	"github.com/marwan-at-work/vgop/semver"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type dp struct {
	c *github.Client
}

// New returns a Github Implementation of the Download Protocol
func New(tok string) gdp.DownloadProtocol {
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
	owner, repo, err := gdp.SplitPath(module)
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
		if tag := t.GetName(); semver.IsValid(tag) && semver.Canonical(tag) == tag {
			tags = append(tags, tag)
		}
	}

	return tags, nil
}

func (d *dp) Info(ctx context.Context, module, version string) (*gdp.RevInfo, error) {
	version = strings.Replace(version, "+incompatible", "", 1)
	var ri gdp.RevInfo
	var untagged bool
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		untagged = true
		version, err = gdp.ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "info.shaFromPseudo")
		}
	}
	owner, repo, err := gdp.SplitPath(module)
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
		ri.Version = gdp.Pseudo(ri.Time, ri.Short)
	}
	return &ri, nil
}

func (d *dp) Latest(ctx context.Context, module string) (*gdp.RevInfo, error) {
	var ri gdp.RevInfo
	owner, repo, err := gdp.SplitPath(module)
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
	ri.Version = gdp.Pseudo(ri.Time, ri.Short)

	return &ri, nil
}

func (d *dp) GoMod(ctx context.Context, module, version string) ([]byte, error) {
	version = strings.Replace(version, "+incompatible", "", 1)
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		version, err = gdp.ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "GoMod.shaFromPseudo")
		}
	}
	owner, repo, err := gdp.SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "GoMod.splitPath")
	}
	fc, _, resp, err := d.c.Repositories.GetContents(ctx, owner, repo, "go.mod", &github.RepositoryContentGetOptions{
		Ref: version,
	})
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		x := fmt.Sprintf("module %v", module)
		if strings.Contains(module, "times") {
			fmt.Println("wtf", x)
		}
		return []byte(fmt.Sprintf("module %v", module)), nil
	}
	if err != nil {
		return nil, errors.Wrap(err, "GoMod.GetContents")
	}

	str, err := fc.GetContent()

	return []byte(str), errors.Wrap(err, "GoMod.GetContent")
}

func (d *dp) Zip(ctx context.Context, module, version string) (io.Reader, error) {
	ref := strings.Replace(version, "+incompatible", "", 1)
	var err error
	if strings.HasPrefix(version, "v0.0.0-") {
		ref, err = gdp.ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "Zip.shaFromPseudo")
		}
	}
	owner, repo, err := gdp.SplitPath(module)
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
	pr, pw := io.Pipe()
	go func() {
		zw := zip.NewWriter(pw)
		for {
			h, err := t.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				zw.Close()
				pw.CloseWithError(errors.Wrap(err, "Zip.tarNext"))
				return
			} else if h.Typeflag != tar.TypeReg && h.Typeflag != tar.TypeDir {
				continue
			}

			path := strings.Replace(h.Name, dirName, goModName, 1)
			w, err := zw.Create(path)
			if err != nil {
				zw.Close()
				pw.CloseWithError(errors.Wrap(err, "Zip.zipCreate"))
				return
			}

			_, err = io.Copy(w, t)
			if err != nil {
				zw.Close()
				pw.CloseWithError(errors.Wrap(err, "Zip.ioCopy"))
				return
			}
		}
		zw.Close()
		pw.Close()
	}()

	return pr, nil
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

package github

import (
	"context"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/marwan-at-work/gdp"
	"github.com/pkg/errors"
	"golang.org/x/oauth2"
)

type codeHost struct {
	c *github.Client
}

// New github implementation of the CodeHost api.
// Use gdp.New create a download protocol out of it.
func New(tok string) gdp.CodeHost {
	var d codeHost
	var client *http.Client
	if tok != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: tok})
		client = oauth2.NewClient(oauth2.NoContext, ts)
	}

	d.c = github.NewClient(client)

	return &d
}

func (d *codeHost) Tags(ctx context.Context, owner, repo string) ([]string, error) {
	var allTags []string
	page := 1
	for {
		tags, _, err := d.c.Repositories.ListTags(ctx, owner, repo, &github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			return nil, errors.Wrapf(err, "github.Tags page %v", page)
		}

		if len(tags) == 0 {
			break
		}

		for _, t := range tags {
			allTags = append(allTags, t.GetName())
		}

		page++
	}

	return allTags, nil
}

func (d *codeHost) Branches(ctx context.Context, owner, repo string) ([]string, error) {
	branches := []string{}
	page := 1
	for {
		bb, _, err := d.c.Repositories.ListBranches(ctx, owner, repo, &github.ListOptions{Page: page, PerPage: 100})
		if err != nil {
			return nil, errors.Wrapf(err, "github.Branches page %v", page)
		}
		if len(bb) == 0 {
			break
		}
		for _, b := range bb {
			branches = append(branches, b.GetName())
		}
		page++
	}

	return branches, nil
}

func (d *codeHost) CommitInfo(ctx context.Context, owner, repo, sha string) (*gdp.RevInfo, error) {
	var ri gdp.RevInfo
	c, _, err := d.c.Repositories.GetCommit(ctx, owner, repo, sha)
	if err != nil {
		return nil, errors.Wrapf(err, "info.GetCommit failed for %v/%v@%v", owner, repo, sha)
	}
	ri.Name = c.GetSHA()
	ri.Short = ri.Name[:12]
	ri.Time = c.GetCommit().GetCommitter().GetDate()
	ri.Version = gdp.Pseudo(ri.Time, ri.Short)
	return &ri, nil
}

func (d *codeHost) TagInfo(ctx context.Context, owner, repo, tag string) (*gdp.RevInfo, error) {
	var ri gdp.RevInfo
	c, _, err := d.c.Repositories.GetCommit(ctx, owner, repo, tag)
	if err != nil {
		return nil, errors.Wrapf(err, "info.GetCommit failed for %v/%v@%v", owner, repo, tag)
	}
	ri.Name = c.GetSHA()
	ri.Short = tag
	ri.Time = c.GetCommit().GetCommitter().GetDate()
	ri.Version = tag
	return &ri, nil
}

func (d *codeHost) LatestCommit(ctx context.Context, owner, repo string) (sha string, t time.Time, err error) {
	r, _, err := d.c.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "github.repoGet")
	}

	ref := r.GetDefaultBranch()
	c, _, err := d.c.Repositories.GetCommit(ctx, owner, repo, ref)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "github.repoGetCOmmit")
	}

	return c.GetSHA(), c.GetCommit().GetCommitter().GetDate(), nil
}

func (d *codeHost) GetModFile(ctx context.Context, owner, repo, version string) ([]byte, error) {
	fc, _, resp, err := d.c.Repositories.GetContents(ctx, owner, repo, "go.mod", &github.RepositoryContentGetOptions{
		Ref: version,
	})
	if resp != nil && resp.StatusCode == http.StatusNotFound {
		return nil, gdp.ErrNotFound
	}
	if err != nil {
		return nil, errors.Wrap(err, "github.GetContents")
	}

	str, err := fc.GetContent()

	return []byte(str), errors.Wrap(err, "err.fc.GetContent")
}

func (d *codeHost) TarURL(ctx context.Context, owner, repo, version string) (string, error) {
	u, err := d.getURL(ctx, owner, repo, version)
	if err != nil {
		return "", errors.Wrap(err, "github.getURL")
	}

	return u, nil
}

func (d *codeHost) getURL(ctx context.Context, owner, repo, ref string) (string, error) {
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

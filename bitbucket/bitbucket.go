package bitbucket

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/marwan-at-work/gdp"
	"github.com/pkg/errors"
)

// New maybe needs credentials?
func New() gdp.CodeHost {
	return &client{}
}

type client struct{}

func (c *client) Branches(ctx context.Context, owner string, repo string) ([]string, error) {
	return nil, errors.New("bitbucket: unimplemented")
}

func (c *client) Tags(ctx context.Context, owner, repo string) ([]string, error) {
	url := c.tagsURL(owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketList.httpGet")
	}
	defer resp.Body.Close()
	var tr tagsResponse
	if err = json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, errors.Wrap(err, "bitbucketList.decode")
	}
	tags := []string{}
	for _, t := range tr.Values {
		tags = append(tags, t.Name)
	}

	return tags, nil
}

func (c *client) CommitInfo(ctx context.Context, owner, repo, sha string) (*gdp.RevInfo, error) {
	var ri gdp.RevInfo
	u := c.commitURL(owner, repo, sha)
	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "infoFromSha.httpGet")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v unexpected code %v from infoFromSha", u, resp.StatusCode)
	}
	var cmt commit
	if err = json.NewDecoder(resp.Body).Decode(&cmt); err != nil {
		return nil, errors.Wrap(err, "infoFromSha.decode")
	}

	ri.Name = cmt.Hash
	ri.Short = ri.Name[:12]
	ri.Time = cmt.Date
	ri.Version = gdp.Pseudo(ri.Time, ri.Short)

	return &ri, nil
}

func (c *client) TagInfo(ctx context.Context, owner, repo, tag string) (*gdp.RevInfo, error) {
	var ri gdp.RevInfo
	u := c.tagRefURL(owner, repo, tag)
	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "infoFromTag.httpGet")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("infoFromTag %v unexpected status %v", u, resp.StatusCode)
	}
	var tr tagRefResponse
	if err = json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return nil, errors.Wrap(err, "infoFromTag.decode")
	}

	ri.Name = tr.Target.Hash
	ri.Short = tag
	ri.Version = tag
	ri.Time = tr.Target.Date

	return &ri, nil
}

func (c *client) LatestCommit(ctx context.Context, owner, repo string) (sha string, t time.Time, err error) {
	u := c.repoURL(owner, repo)
	resp, err := http.Get(u)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "bitbucketLatest.httpGet")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", time.Time{}, fmt.Errorf("bitbucketLatest.httpGet unexpected response code %v", resp.StatusCode)
	}
	var rr repoResponse
	if err = json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return "", time.Time{}, errors.Wrap(err, "bitbucketLatest.jsonDecode")
	}

	u = c.branchRefURL(owner, repo, rr.Mainbranch.Name)
	resp, err = http.Get(u)
	if err != nil {
		return "", time.Time{}, errors.Wrap(err, "bitbucketLatest.httpGetBranch")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", time.Time{}, fmt.Errorf("bitbucketLatest.httpGetBranch unexpected response code %v", resp.StatusCode)
	}
	var br branchRefResponse
	if err = json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return "", time.Time{}, errors.Wrap(err, "bitbucketLatest.jsonDecodeBranch")
	}

	return br.Target.Hash, br.Target.Date, nil
}

func (c *client) GetModFile(ctx context.Context, owner, repo, version string) ([]byte, error) {
	u := c.contentURL(owner, repo, version, "go.mod")
	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "goModFromTag.httpGet")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, gdp.ErrNotFound
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v returned %v at goModFromTag", u, resp.StatusCode)
	}
	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketGoMod.readAll")
	}

	return bts, nil
}

func (c *client) TarURL(ctx context.Context, owner, repo, version string) (string, error) {
	return c.tarURL(owner, repo, version), nil
}

func (c *client) contentURL(owner, repo, tag, path string) string {
	return fmt.Sprintf(
		"https://api.bitbucket.org/2.0/repositories/%v/%v/src/%v/%v",
		owner, repo, tag, path,
	)
}

func (c *client) tarURL(owner, repo, ref string) string {
	return fmt.Sprintf(
		"https://bitbucket.org/%v/%v/get/%v.tar.gz",
		owner,
		repo,
		ref,
	)
}

type commit struct {
	Hash string    `json:"hash"`
	Date time.Time `json:"date"` // 2013-08-29T11:07:46+00:00
}

func (c *client) commitURL(owner, repo, sha string) string {
	return fmt.Sprintf(
		"https://api.bitbucket.org/2.0/repositories/%v/%v/commit/%v",
		owner,
		repo,
		sha,
	)
}

type tagRefResponse struct {
	Name   string `json:"name"`
	Target commit `json:"target"`
}

func (c *client) tagRefURL(owner, repo, tag string) string {
	return fmt.Sprintf(
		"https://api.bitbucket.org/2.0/repositories/%v/%v/refs/tags/%v",
		owner,
		repo,
		tag,
	)
}

type tagsResponse struct {
	Values []struct {
		Name string `json:"name"`
	} `json:"values"`
}

func (c *client) tagsURL(owner, repo string) string {
	return fmt.Sprintf(
		"https://api.bitbucket.org/2.0/repositories/%v/%v/refs/tags",
		owner,
		repo,
	)
}

type repoResponse struct {
	Mainbranch struct {
		Name string `json:"name"`
	} `json:"mainbranch"`
}

func (c *client) repoURL(owner, repo string) string {
	return fmt.Sprintf(
		"https://api.bitbucket.org/2.0/repositories/%v/%v",
		owner,
		repo,
	)
}

type branchRefResponse struct {
	Target commit `json:"target"`
}

func (c *client) branchRefURL(owner, repo, branch string) string {
	return fmt.Sprintf(
		"https://api.bitbucket.org/2.0/repositories/%v/%v/refs/branches/%v",
		owner,
		repo,
		branch,
	)
}

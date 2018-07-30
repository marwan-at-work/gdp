package bitbucket

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/marwan-at-work/gdp"
	"github.com/marwan-at-work/vgop/semver"
	"github.com/pkg/errors"
)

// New maybe needs credentials?
func New() gdp.DownloadProtocol {
	return &client{}
}

type client struct{}

func (c *client) List(ctx context.Context, module string) ([]string, error) {
	tags := []string{}
	owner, repo, err := gdp.SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketList.splitPath")
	}
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

	for _, t := range tr.Values {
		if semver.IsValid(t.Name) {
			tags = append(tags, t.Name)
		}
	}

	return tags, nil
}

func (c *client) Info(ctx context.Context, module, version string) (*gdp.RevInfo, error) {
	version = strings.Replace(version, "+incompatible", "", 1)
	owner, repo, err := gdp.SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketInfo.SplitPath")
	}
	if gdp.IsPseudo(version) {
		sha, err := gdp.ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "bitbucketInfo.shaFromPseudo")
		}
		return c.infoFromSha(ctx, owner, repo, sha)
	}

	return c.infoFromTag(ctx, owner, repo, version)
}

func (c *client) infoFromTag(ctx context.Context, owner, repo, tag string) (*gdp.RevInfo, error) {
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

func (c *client) infoFromSha(ctx context.Context, owner, repo, sha string) (*gdp.RevInfo, error) {
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

func (c *client) Latest(ctx context.Context, module string) (*gdp.RevInfo, error) {
	var ri gdp.RevInfo
	owner, repo, err := gdp.SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketLatest.splitPath")
	}

	u := c.repoURL(owner, repo)
	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketLatest.httpGet")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bitbucketLatest.httpGet unexpected response code %v", resp.StatusCode)
	}
	var rr repoResponse
	if err = json.NewDecoder(resp.Body).Decode(&rr); err != nil {
		return nil, errors.Wrap(err, "bitbucketLatest.jsonDecode")
	}

	u = c.branchRefURL(owner, repo, rr.Mainbranch.Name)
	fmt.Println(u)
	resp, err = http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketLatest.httpGetBranch")
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("bitbucketLatest.httpGetBranch unexpected response code %v", resp.StatusCode)
	}
	var br branchRefResponse
	if err = json.NewDecoder(resp.Body).Decode(&br); err != nil {
		return nil, errors.Wrap(err, "bitbucketLatest.jsonDecodeBranch")
	}

	ri.Name = br.Target.Hash
	ri.Short = ri.Name[:12]
	ri.Time = br.Target.Date
	ri.Version = gdp.Pseudo(ri.Time, ri.Short)

	return &ri, nil
}

func (c *client) GoMod(ctx context.Context, module, version string) ([]byte, error) {
	version = strings.Replace(version, "+incompatible", "", 1)
	var err error
	owner, repo, err := gdp.SplitPath(module)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketGoMod.splitPath")
	}
	if gdp.IsPseudo(version) {
		version, err = gdp.ShaFromPseudo(version)
		if err != nil {
			return nil, errors.Wrap(err, "bitbucketGoMod.shaFromPseudo")
		}
	}

	u := c.contentURL(owner, repo, version, "go.mod")
	resp, err := http.Get(u)
	if err != nil {
		return nil, errors.Wrap(err, "goModFromTag.httpGet")
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return []byte(fmt.Sprintf("module %v", module)), nil
	} else if resp.StatusCode != 200 {
		return nil, fmt.Errorf("%v returned %v at goModFromTag", u, resp.StatusCode)
	}
	bts, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "bitbucketGoMod.readAll")
	}

	return bts, nil
}

func (c *client) Zip(ctx context.Context, module, version string) (io.Reader, error) {
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
	u := c.tarURL(owner, repo, ref)
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

	goModName := "bitbucket.org" + owner + "/" + repo + "@" + version + "/"
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

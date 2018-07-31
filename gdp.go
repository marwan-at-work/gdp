package gdp

import (
	"context"
	"io"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// ErrNotFound error
var ErrNotFound = errors.New("not found")

// DownloadProtocol of cmd/go
type DownloadProtocol interface {
	List(ctx context.Context, module string) ([]string, error)
	Info(ctx context.Context, module, version string) (*RevInfo, error)
	Latest(ctx context.Context, module string) (*RevInfo, error)
	GoMod(ctx context.Context, module, version string) ([]byte, error)
	Zip(ctx context.Context, module, version, zipPrefix string) (io.Reader, error) // potentially zip.ReadCloser?
}

// RevInfo describes a single revision in a module repository.
type RevInfo struct {
	Version string    // version string
	Name    string    // complete ID in underlying repository
	Short   string    // shortened ID, for use in pseudo-version
	Time    time.Time // commit time
}

// CodeHost describes a code hosting API (github, bitbucket etc)
// where they have common functionalities to deal with repositories,
// users, commits, and tags.
type CodeHost interface {
	Tags(ctx context.Context, owner, repo string) ([]string, error)
	CommitInfo(ctx context.Context, owner, repo, sha string) (*RevInfo, error)
	TagInfo(ctx context.Context, owner, repo, tag string) (*RevInfo, error)
	LatestCommit(ctx context.Context, owner, repo string) (sha string, t time.Time, err error)
	GetModFile(ctx context.Context, owner, repo, version string) ([]byte, error)
	TarURL(ctx context.Context, owner, repo, version string) (string, error)
}

// PseudoTime for a shortened commit sha: YYYYMMDDHHMMSS
const PseudoTime = "20060102150405"

// IsPseudo returns whether the tag
// comes from a sha or a valid semver tag
func IsPseudo(v string) bool {
	vinfo := strings.Split(v, "-")
	if len(vinfo) < 3 {
		return false
	}
	_, err := time.Parse(PseudoTime, vinfo[1])
	return vinfo[0] == "v0.0.0" && err == nil
}

// Pseudo takes a time and a short sha and returns
// v0.0.0-formattedTime-shortSha
func Pseudo(t time.Time, shortSha string) string {
	return "v0.0.0-" + t.Format(PseudoTime) + "-" + shortSha
}

// ShaFromPseudo takes v0.0.0-formattedTime-shortSha
// and returns shortSha
func ShaFromPseudo(sv string) (string, error) {
	vinfo := strings.Split(sv, "-")
	if len(vinfo) < 3 {
		return "", errors.New("incorrect pseudo version: " + sv)
	}

	return vinfo[2], nil
}

// SplitPath takes a valid import path such as
// github.com/a/b and returns the owner and repo (a, b)
func SplitPath(path string) (owner, repo string, err error) {
	els := strings.Split(path, "/")
	switch els[0] {
	case "github.com", "bitbucket.org":
		if len(els) != 3 {
			return "", "", errors.New("splitPath: unparsable path: " + path)
		}
		owner = els[1]
		repo = els[2]

		return owner, repo, nil
	case "gopkg.in":
		return "", "", ErrGopkg
	}

	return "", "", errors.New("splitPath: unsupported API")
}

// ErrGopkg so that a protocol can switch from SplitPath
// to ParseGopkgPath
var ErrGopkg = errors.New("use ParseGopkgPath")

// ParseGopkgPath takes a gopkg.in path and returns
// the github and version information
func ParseGopkgPath(path string) (owner, repo, major string, err error) {
	els := strings.Split(path, "/")
	if len(els) < 2 || len(els) > 3 {
		return "", "", "", errors.New("pkginSplit: unparsable path" + path)
	}

	owner = els[1]
	repo = els[1]
	vloc := els[1]
	if len(els) == 3 {
		vloc = els[2]
	}
	vidx := strings.Index(vloc, ".v")
	if vidx == -1 {
		return "", "", "", errors.New("gopkin missing version from path " + path)
	}
	major = vloc[vidx+1:]
	if len(els) == 2 {
		repo = repo[:vidx]
		owner = "go-" + repo
	}
	if len(els) == 3 {
		repo = els[2]
		repo = repo[:vidx]
	}

	return owner, repo, major, nil
}

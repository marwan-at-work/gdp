package gopkgin

import (
	"context"
	"time"

	"github.com/marwan-at-work/gdp"
)

// New bla
func New() gdp.CodeHost {
	return nil
}

type codeHost struct{}

func (ch *codeHost) Tags(ctx context.Context, owner string, repo string) ([]string, error) {
	panic("not implemented")
}

func (ch *codeHost) CommitInfo(ctx context.Context, owner string, repo string, sha string) (*gdp.RevInfo, error) {
	panic("not implemented")
}

func (ch *codeHost) TagInfo(ctx context.Context, owner string, repo string, tag string) (*gdp.RevInfo, error) {
	panic("not implemented")
}

func (ch *codeHost) LatestCommit(ctx context.Context, owner string, repo string) (sha string, t time.Time, err error) {
	panic("not implemented")
}

func (ch *codeHost) GetModFile(ctx context.Context, owner string, repo string, version string) ([]byte, error) {
	panic("not implemented")
}

func (ch *codeHost) TarURL(ctx context.Context, owner string, repo string, version string) (string, error) {
	panic("not implemented")
}

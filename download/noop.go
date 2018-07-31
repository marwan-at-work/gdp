package download

import (
	"context"
	"io"

	"github.com/marwan-at-work/gdp"
)

type noOpProtocol struct{}

func (noOpProtocol) List(context.Context, string) ([]string, error) {
	return nil, gdp.ErrNotFound
}

func (noOpProtocol) Info(context.Context, string, string) (*gdp.RevInfo, error) {
	return nil, gdp.ErrNotFound
}

func (noOpProtocol) Latest(context.Context, string) (*gdp.RevInfo, error) {
	return nil, gdp.ErrNotFound
}

func (noOpProtocol) GoMod(context.Context, string, string) ([]byte, error) {
	return nil, gdp.ErrNotFound
}

func (noOpProtocol) Zip(context.Context, string, string, string) (io.Reader, error) {
	return nil, gdp.ErrNotFound
}

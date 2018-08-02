package gdp

import (
	"context"
	"io"
)

// NoOpProtocol always returns 400
func NoOpProtocol() DownloadProtocol {
	return noOpProtocol{}
}

type noOpProtocol struct{}

func (noOpProtocol) List(context.Context, string) ([]string, error) {
	return nil, ErrNotFound
}

func (noOpProtocol) Info(context.Context, string, string) (*RevInfo, error) {
	return nil, ErrNotFound
}

func (noOpProtocol) Latest(context.Context, string) (*RevInfo, error) {
	return nil, ErrNotFound
}

func (noOpProtocol) GoMod(context.Context, string, string) ([]byte, error) {
	return nil, ErrNotFound
}

func (noOpProtocol) Zip(context.Context, string, string, string) (io.Reader, error) {
	return nil, ErrNotFound
}

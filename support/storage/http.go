package storage

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"

	"github.com/stellar/go/support/errors"
)

type HttpStorage struct {
	ctx    context.Context
	client http.Client
	base   url.URL
}

func NewHttpStorage(ctx context.Context, base *url.URL) Storage {
	return &HttpStorage{
		ctx:  ctx,
		base: *base,
	}
}

func checkResp(r *http.Response) error {
	if r.StatusCode >= 200 && r.StatusCode < 400 {
		return nil
	} else {
		return fmt.Errorf("Bad HTTP response '%s' for %s '%s'",
			r.Status, r.Request.Method, r.Request.URL.String())
	}
}

func (b *HttpStorage) GetFile(pth string) (io.ReadCloser, error) {
	derived := b.base
	derived.Path = path.Join(derived.Path, pth)
	req, err := http.NewRequest("GET", derived.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(b.ctx)
	logReq(req)
	resp, err := b.client.Do(req)
	logResp(resp)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	err = checkResp(resp)
	if err != nil {
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		return nil, err
	}
	return resp.Body, nil
}

func (b *HttpStorage) Head(pth string) (*http.Response, error) {
	derived := b.base
	derived.Path = path.Join(derived.Path, pth)
	req, err := http.NewRequest("HEAD", derived.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(b.ctx)
	logReq(req)
	resp, err := b.client.Do(req)
	logResp(resp)
	if err != nil {
		return nil, err
	}

	if resp.Body != nil {
		resp.Body.Close()
	}

	return resp, nil
}

func (b *HttpStorage) Exists(pth string) (bool, error) {
	resp, err := b.Head(pth)
	if err != nil {
		return false, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return true, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return false, nil
	} else {
		return false, errors.Errorf("Unkown status code=%d", resp.StatusCode)
	}
}

func (b *HttpStorage) Size(pth string) (int64, error) {
	resp, err := b.Head(pth)
	if err != nil {
		return 0, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return resp.ContentLength, nil
	} else if resp.StatusCode == http.StatusNotFound {
		return 0, nil
	} else {
		return 0, errors.Errorf("Unkown status code=%d", resp.StatusCode)
	}
}

func (b *HttpStorage) PutFile(pth string, in io.ReadCloser) error {
	in.Close()
	return errors.New("PutFile not available over HTTP")
}

func (b *HttpStorage) ListFiles(pth string) (chan string, chan error) {
	ch := make(chan string)
	er := make(chan error)
	close(ch)
	er <- errors.New("ListFiles not available over HTTP")
	close(er)
	return ch, er
}

func (b *HttpStorage) CanListFiles() bool {
	return false
}

func (b *HttpStorage) Close() error {
	return nil
}

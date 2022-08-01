package storage

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"path"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/stellar/go/support/errors"
)

type S3Storage struct {
	ctx              context.Context
	svc              *s3.S3
	bucket           string
	prefix           string
	unsignedRequests bool
}

func NewS3Storage(
	ctx context.Context,
	bucket string,
	prefix string,
	region string,
	endpoint string,
	unsignedRequests bool,
) (Storage, error) {
	log.WithFields(log.Fields{"bucket": bucket,
		"prefix":   prefix,
		"region":   region,
		"endpoint": endpoint}).Debug("s3: making backend")
	cfg := &aws.Config{
		Region:   aws.String(region),
		Endpoint: aws.String(endpoint),
	}
	cfg = cfg.WithS3ForcePathStyle(true)

	sess, err := session.NewSession(cfg)
	if err != nil {
		return nil, err
	}

	backend := S3Storage{
		ctx:              ctx,
		svc:              s3.New(sess),
		bucket:           bucket,
		prefix:           prefix,
		unsignedRequests: unsignedRequests,
	}
	return &backend, nil
}

func (b *S3Storage) GetFile(pth string) (io.ReadCloser, error) {
	key := path.Join(b.prefix, pth)
	params := &s3.GetObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}

	req, resp := b.svc.GetObjectRequest(params)
	if b.unsignedRequests {
		req.Handlers.Sign.Clear() // makes this request unsigned
	}
	req.SetContext(b.ctx)
	logReq(req.HTTPRequest)
	err := req.Send()
	logResp(req.HTTPResponse)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}

func (b *S3Storage) Head(pth string) (*http.Response, error) {
	key := path.Join(b.prefix, pth)
	params := &s3.HeadObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
	}

	req, _ := b.svc.HeadObjectRequest(params)
	if b.unsignedRequests {
		req.Handlers.Sign.Clear() // makes this request unsigned
	}
	req.SetContext(b.ctx)
	logReq(req.HTTPRequest)
	err := req.Send()
	logResp(req.HTTPResponse)

	if req.HTTPResponse != nil && req.HTTPResponse.StatusCode == http.StatusNotFound {
		// Lately the S3 SDK has started treating a 404 as generating a non-nil
		// 'err', so we have to test for this _before_ we test 'err' for
		// nil-ness. This is undocumented, as is the err.Code returned in that
		// error ("NotFound"), and it's a breaking change from what it used to
		// do, and not what one would expect, but who's counting? We'll just
		// turn it _back_ into what it used to be: 404 as a non-erroneously
		// received HTTP response.
		return req.HTTPResponse, nil
	}

	if err != nil {
		return nil, err
	}
	return req.HTTPResponse, nil
}

func (b *S3Storage) Exists(pth string) (bool, error) {
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

func (b *S3Storage) Size(pth string) (int64, error) {
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

func (b *S3Storage) PutFile(pth string, in io.ReadCloser) error {
	var buf bytes.Buffer
	_, err := buf.ReadFrom(in)
	in.Close()
	if err != nil {
		return err
	}
	key := path.Join(b.prefix, pth)
	params := &s3.PutObjectInput{
		Bucket: aws.String(b.bucket),
		Key:    aws.String(key),
		Body:   bytes.NewReader(buf.Bytes()),
	}
	req, _ := b.svc.PutObjectRequest(params)
	if b.unsignedRequests {
		req.Handlers.Sign.Clear() // makes this request unsigned
	}
	req.SetContext(b.ctx)
	logReq(req.HTTPRequest)
	err = req.Send()
	logResp(req.HTTPResponse)

	in.Close()
	return err
}

func (b *S3Storage) ListFiles(pth string) (chan string, chan error) {
	prefix := path.Join(b.prefix, pth)
	ch := make(chan string)
	errs := make(chan error)

	params := &s3.ListObjectsInput{
		Bucket:  aws.String(b.bucket),
		MaxKeys: aws.Int64(1000),
		Prefix:  aws.String(prefix),
	}
	req, resp := b.svc.ListObjectsRequest(params)
	if b.unsignedRequests {
		req.Handlers.Sign.Clear() // makes this request unsigned
	}
	req.SetContext(b.ctx)
	logReq(req.HTTPRequest)
	err := req.Send()
	logResp(req.HTTPResponse)
	if err != nil {
		errs <- err
		close(ch)
		close(errs)
		return ch, errs
	}
	go func() {
		for {
			for _, c := range resp.Contents {
				params.Marker = c.Key
				log.WithField("key", *c.Key).Trace("s3: ListFiles")
				ch <- *c.Key
			}
			if *resp.IsTruncated {
				req, resp = b.svc.ListObjectsRequest(params)
				if b.unsignedRequests {
					req.Handlers.Sign.Clear() // makes this request unsigned
				}
				req.SetContext(b.ctx)
				logReq(req.HTTPRequest)
				err := req.Send()
				logResp(req.HTTPResponse)
				if err != nil {
					errs <- err
				}
			} else {
				break
			}
		}
		close(ch)
		close(errs)
	}()
	return ch, errs
}

func (b *S3Storage) CanListFiles() bool {
	return true
}

func (b *S3Storage) Close() error {
	return nil
}

package index

import (
	"bytes"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"

	types "github.com/stellar/go/exp/lighthorizon/index/types"
)

type S3Backend struct {
	s3Session  *session.Session
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
	parallel   uint32
	pathPrefix string
	bucket     string
}

func NewS3Backend(awsConfig *aws.Config, bucket string, pathPrefix string, parallel uint32) (*S3Backend, error) {
	s3Session, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	return &S3Backend{
		s3Session:  s3Session,
		downloader: s3manager.NewDownloader(s3Session),
		uploader:   s3manager.NewUploader(s3Session),
		parallel:   parallel,
		pathPrefix: pathPrefix,
		bucket:     bucket,
	}, nil
}

func (s *S3Backend) FlushAccounts(accounts []string) error {
	var buf bytes.Buffer
	accountsString := strings.Join(accounts, "\n")
	_, err := buf.WriteString(accountsString)
	if err != nil {
		return err
	}

	path := filepath.Join(s.pathPrefix, "accounts")

	_, err = s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   &buf,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Backend) Flush(indexes map[string]types.NamedIndices) error {
	return parallelFlush(s.parallel, indexes, s.writeBatch)
}

func (s *S3Backend) writeBatch(b *batch) error {
	// TODO: re-use buffers in a pool
	var buf bytes.Buffer
	if _, err := writeGzippedTo(&buf, b.indexes); err != nil {
		// TODO: Should we retry or what here??
		return errors.Wrapf(err, "unable to serialize %s", b.account)
	}

	path := s.path(b.account)

	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
		Body:   &buf,
	})
	if err != nil {
		return errors.Wrapf(err, "unable to upload %s", b.account)
	}

	return nil
}

func (s *S3Backend) FlushTransactions(indexes map[string]*types.TrieIndex) error {
	// TODO: Parallelize this
	var buf bytes.Buffer
	for key, index := range indexes {
		buf.Reset()
		path := filepath.Join(s.pathPrefix, "tx", key)

		zw := gzip.NewWriter(&buf)
		if _, err := index.WriteTo(zw); err != nil {
			log.Errorf("Unable to serialize %s: %v", path, err)
			continue
		}

		if err := zw.Close(); err != nil {
			log.Errorf("Unable to serialize %s: %v", path, err)
			continue
		}

		_, err := s.uploader.Upload(&s3manager.UploadInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(path),
			Body:   &buf,
		})
		if err != nil {
			log.Errorf("Unable to upload %s: %v", path, err)
			// TODO: retries
			continue
		}
	}
	return nil
}

func (s *S3Backend) ReadAccounts() ([]string, error) {
	log.Debugf("Downloading accounts list")
	b := &aws.WriteAtBuffer{}
	path := filepath.Join(s.pathPrefix, "accounts")
	n, err := s.downloader.Download(b, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, os.ErrNotExist
		}
		return nil, errors.Wrapf(err, "Unable to download accounts list")
	}
	if n == 0 {
		return nil, os.ErrNotExist
	}
	body := b.Bytes()
	accounts := strings.Split(string(body), "\n")
	return accounts, nil
}

func (s *S3Backend) path(account string) string {
	return filepath.Join(s.pathPrefix, account[:10], account)
}

func (s *S3Backend) Read(account string) (types.NamedIndices, error) {
	// Check if index exists in S3
	log.Debugf("Downloading index: %s", account)
	var err error
	for i := 0; i < 10; i++ {
		b := &aws.WriteAtBuffer{}
		path := s.path(account)
		var n int64
		n, err = s.downloader.Download(b, &s3.GetObjectInput{
			Bucket: aws.String(s.bucket),
			Key:    aws.String(path),
		})
		if err != nil {
			if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
				return nil, os.ErrNotExist
			}
			err = errors.Wrapf(err, "Unable to download %s", account)
			time.Sleep(100 * time.Millisecond)
			continue
		}
		if n == 0 {
			return nil, os.ErrNotExist
		}
		var indexes map[string]*types.BitmapIndex
		indexes, _, err = readGzippedFrom(bytes.NewReader(b.Bytes()))
		if err != nil {
			log.Errorf("Unable to parse %s: %v", account, err)
			return nil, os.ErrNotExist
		}
		return indexes, nil
	}

	return nil, err
}

func (s *S3Backend) ReadTransactions(prefix string) (*types.TrieIndex, error) {
	// Check if index exists in S3
	log.Debugf("Downloading index: %s", prefix)
	b := &aws.WriteAtBuffer{}
	path := filepath.Join(s.pathPrefix, "tx", prefix)
	n, err := s.downloader.Download(b, &s3.GetObjectInput{
		Bucket: aws.String(s.bucket),
		Key:    aws.String(path),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, os.ErrNotExist
		}
		return nil, errors.Wrapf(err, "Unable to download %s", prefix)
	}
	if n == 0 {
		return nil, os.ErrNotExist
	}
	zr, err := gzip.NewReader(bytes.NewReader(b.Bytes()))
	if err != nil {
		log.Errorf("Unable to parse %s: %v", prefix, err)
		return nil, os.ErrNotExist
	}
	defer zr.Close()

	var index types.TrieIndex
	_, err = index.ReadFrom(zr)
	if err != nil {
		log.Errorf("Unable to parse %s: %v", prefix, err)
		return nil, os.ErrNotExist
	}
	return &index, nil
}

package index

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stellar/go/support/errors"
	"github.com/stellar/go/support/log"
)

const BUCKET = "horizon-index"

type S3Backend struct {
	s3Session  *session.Session
	downloader *s3manager.Downloader
	uploader   *s3manager.Uploader
	parallel   uint32
	prefix     string
}

func NewS3Store(awsConfig *aws.Config, prefix string, parallel uint32) (Store, error) {
	backend, err := NewS3Backend(awsConfig, prefix, parallel)
	if err != nil {
		return nil, err
	}
	return NewStore(backend)
}

func NewS3Backend(awsConfig *aws.Config, prefix string, parallel uint32) (*S3Backend, error) {
	s3Session, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	return &S3Backend{
		s3Session:  s3Session,
		downloader: s3manager.NewDownloader(s3Session),
		uploader:   s3manager.NewUploader(s3Session),
		parallel:   parallel,
		prefix:     prefix,
	}, nil
}

func (s *S3Backend) FlushAccounts(accounts []string) error {
	var buf bytes.Buffer
	accountsString := strings.Join(accounts, "\n")
	_, err := buf.WriteString(accountsString)
	if err != nil {
		return err
	}

	path := filepath.Join(s.prefix, "accounts")

	_, err = s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(path),
		Body:   &buf,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *S3Backend) Flush(indexes map[string]map[string]*CheckpointIndex) error {
	return parallelFlush(s.parallel, indexes, s.writeBatch)
}

func (s *S3Backend) writeBatch(b *batch, r retry) error {
	// TODO: re-use buffers in a pool
	var buf bytes.Buffer
	if _, err := writeGzippedTo(&buf, b.indexes); err != nil {
		// TODO: Should we retry or what here??
		return errors.Wrapf(err, "unable to serialize %s", b.account)
	}

	path := filepath.Join(s.prefix, b.account[:3], b.account)

	_, err := s.uploader.Upload(&s3manager.UploadInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(path),
		Body:   &buf,
	})
	if err != nil {
		log.Errorf("Unable to upload %s: %v", b.account, err)
		r(b)
		return nil
	}

	return nil
}

func (s *S3Backend) Read(account string) (map[string]*CheckpointIndex, error) {
	// Check if index exists in S3
	log.Debugf("Downloading index: %s", account)
	b := &aws.WriteAtBuffer{}
	path := filepath.Join(s.prefix, account[:3], account)
	n, err := s.downloader.Download(b, &s3.GetObjectInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(path),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok && aerr.Code() == s3.ErrCodeNoSuchKey {
			return nil, os.ErrNotExist
		}
		return nil, errors.Wrapf(err, "Unable to download %s", account)
	}
	if n == 0 {
		return nil, os.ErrNotExist
	}
	indexes, _, err := readGzippedFrom(bytes.NewReader(b.Bytes()))
	if err != nil {
		log.Errorf("Unable to parse %s: %v", account, err)
		return nil, os.ErrNotExist
	}
	return indexes, nil
}

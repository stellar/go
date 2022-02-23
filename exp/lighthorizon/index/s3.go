package index

import (
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/stellar/go/support/log"
)

const BUCKET = "horizon-index"

type S3IndexStore struct {
	mutex      sync.RWMutex
	indexes    map[string]*CheckpointIndex
	s3Session  *session.Session
	downloader *s3manager.Downloader
	parallel   uint32
}

func NewS3IndexStore(awsConfig *aws.Config, parallel uint32) (*S3IndexStore, error) {
	s3Session, err := session.NewSession(awsConfig)
	if err != nil {
		return nil, err
	}

	return &S3IndexStore{
		indexes:    map[string]*CheckpointIndex{},
		s3Session:  s3Session,
		downloader: s3manager.NewDownloader(s3Session),
		parallel:   uint32(parallel),
	}, nil
}

func (s *S3IndexStore) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	var wg sync.WaitGroup

	type upload struct {
		id    string
		index *CheckpointIndex
	}

	uch := make(chan upload, s.parallel)

	go func() {
		for id, index := range s.indexes {
			uch <- upload{
				id:    id,
				index: index,
			}
		}
		close(uch)
	}()

	uploader := s3manager.NewUploader(s.s3Session)

	written := uint64(0)
	for i := uint32(0); i < s.parallel; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for u := range uch {
				_, err := uploader.Upload(&s3manager.UploadInput{
					Bucket: aws.String(BUCKET),
					Key:    aws.String(u.id),
					Body:   u.index.Buffer(),
				})
				if err != nil {
					log.Errorf("Unable to upload %s, %v", u.id, err)
					uch <- u
					continue
				}

				nwritten := atomic.AddUint64(&written, 1)
				if nwritten%1000 == 0 {
					log.Infof("Writing indexes... %d/%d %.2f%%", nwritten, len(s.indexes), (float64(nwritten)/float64(len(s.indexes)))*100)
				}
			}
		}()
	}

	wg.Wait()

	// clear indexes to save memory
	s.indexes = map[string]*CheckpointIndex{}

	return nil
}

func (s *S3IndexStore) AddParticipantsToIndexes(checkpoint uint32, indexFormat string, participants []string) error {
	for _, participant := range participants {
		ind, err := s.getCreateIndex(fmt.Sprintf(indexFormat, participant))
		if err != nil {
			return err
		}
		err = ind.SetActive(checkpoint)
		if err != nil {
			return err
		}
	}
	return nil
}

func (s *S3IndexStore) getCreateIndex(id string) (*CheckpointIndex, error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	ind, ok := s.indexes[id]
	if ok {
		return ind, nil
	}

	// Check if index exists in S3
	log.Debugf("Downloading index: %v", id)
	b := &aws.WriteAtBuffer{}
	_, err := s.downloader.Download(b, &s3.GetObjectInput{
		Bucket: aws.String(BUCKET),
		Key:    aws.String(id),
	})
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			if aerr.Code() == s3.ErrCodeNoSuchKey {
				ind = &CheckpointIndex{}
			} else {
				return nil, err
			}
		} else {
			return nil, err
		}
	} else {
		ind, err = NewCheckpointIndexFromBytes(b.Bytes())
		if err != nil {
			return nil, err
		}
	}

	s.indexes[id] = ind

	return ind, nil
}

func (s *S3IndexStore) NextActive(indexId string, afterCheckpoint uint32) (uint32, error) {
	ind, err := s.getCreateIndex(indexId)
	if err != nil {
		return 0, err
	}
	return ind.NextActive(afterCheckpoint)
}

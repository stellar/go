package index

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"

	backend "github.com/stellar/go/exp/lighthorizon/index/backend"
)

func Connect(backendUrl string) (Store, error) {
	parsed, err := url.Parse(backendUrl)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "s3":
		config := &aws.Config{}
		query := parsed.Query()
		if region := query.Get("region"); region != "" {
			config.Region = aws.String(region)
		}

		return NewS3Store(config, parsed.Path, 20)

	case "file":
		return NewFileStore(filepath.Join(parsed.Host, parsed.Path), 20)

	default:
		return nil, fmt.Errorf("unknown URL scheme: '%s' (from %s)",
			parsed.Scheme, backendUrl)
	}
}

func NewFileStore(dir string, parallel uint32) (Store, error) {
	backend, err := backend.NewFileBackend(dir, parallel)
	if err != nil {
		return nil, err
	}
	return NewStore(backend)
}

func NewS3Store(awsConfig *aws.Config, prefix string, parallel uint32) (Store, error) {
	backend, err := backend.NewS3Backend(awsConfig, prefix, parallel)
	if err != nil {
		return nil, err
	}
	return NewStore(backend)
}

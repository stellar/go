package index

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
	backend "github.com/stellar/go/exp/lighthorizon/index/backend"
)

func Connect(backendUrl string) (Store, error) {
	return ConnectWithConfig(StoreConfig{URL: backendUrl})
}

func ConnectWithConfig(config StoreConfig) (Store, error) {
	if config.Workers <= 0 {
		config.Workers = 1
	}

	parsed, err := url.Parse(config.URL)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "s3":
		s3Url := fmt.Sprintf("%s/%s", config.URL, config.URLSubPath)
		parsed, err = url.Parse(s3Url)
		if err != nil {
			return nil, err
		}
		awsConfig := &aws.Config{}
		query := parsed.Query()
		if region := query.Get("region"); region != "" {
			awsConfig.Region = aws.String(region)
		}

		return NewS3Store(awsConfig, parsed.Host, parsed.Path, config)

	case "file":
		fileUrl := filepath.Join(config.URL, config.URLSubPath)
		parsed, err = url.Parse(fileUrl)
		if err != nil {
			return nil, err
		}
		return NewFileStore(filepath.Join(parsed.Host, parsed.Path), config)

	default:
		return nil, fmt.Errorf("unknown URL scheme: '%s' (from %s)",
			parsed.Scheme, config.URL)
	}
}

func NewFileStore(prefix string, config StoreConfig) (Store, error) {
	backend, err := backend.NewFileBackend(prefix, config.Workers)
	if err != nil {
		return nil, err
	}
	return NewStore(backend, config)
}

func NewS3Store(awsConfig *aws.Config, bucket string, prefix string, indexConfig StoreConfig) (Store, error) {
	backend, err := backend.NewS3Backend(awsConfig, bucket, prefix, indexConfig.Workers)
	if err != nil {
		return nil, err
	}
	return NewStore(backend, indexConfig)
}

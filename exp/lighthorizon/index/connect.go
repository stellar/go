package index

import (
	"fmt"
	"net/url"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"

	backend "github.com/stellar/go/exp/lighthorizon/index/backend"
)

func Connect(backendUrl string) (Store, error) {
	return ConnectWithConfig(StoreConfig{Url: backendUrl})
}

func ConnectWithConfig(config StoreConfig) (Store, error) {
	parsed, err := url.Parse(config.Url)
	if err != nil {
		return nil, err
	}
	switch parsed.Scheme {
	case "s3":
		awsConfig := &aws.Config{}
		query := parsed.Query()
		if region := query.Get("region"); region != "" {
			awsConfig.Region = aws.String(region)
		}

		config.Url = parsed.Path
		return NewS3Store(awsConfig, config)

	case "file":
		config.Url = filepath.Join(parsed.Host, parsed.Path)
		return NewFileStore(config)

	default:
		return nil, fmt.Errorf("unknown URL scheme: '%s' (from %s)",
			parsed.Scheme, config.Url)
	}
}

func NewFileStore(config StoreConfig) (Store, error) {
	backend, err := backend.NewFileBackend(config.Url, config.Workers)
	if err != nil {
		return nil, err
	}
	return NewStore(backend, config)
}

func NewS3Store(awsConfig *aws.Config, indexConfig StoreConfig) (Store, error) {
	backend, err := backend.NewS3Backend(awsConfig, indexConfig.Url, indexConfig.Workers)
	if err != nil {
		return nil, err
	}
	return NewStore(backend, indexConfig)
}

package index

import (
	"errors"
	"net/url"
	"path/filepath"

	"github.com/aws/aws-sdk-go/aws"
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
		return nil, errors.New("unknown URL scheme: '" + parsed.Scheme + "'")
	}
}

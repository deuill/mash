package imager

import (
	// Internal packages
	"fmt"

	// Third-party packages
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

// A Source represents an image source, which is usually matched against a URL endpoint, and
// provides options related to that endpoint.
type Source struct {
	bucket *s3.Bucket
}

func NewSource(region, bucket, accessKey, secretKey string) (*Source, error) {
	auth, err := aws.GetAuth(accessKey, secretKey)
	if err != nil {
		return nil, err
	}

	if _, exists := aws.Regions[region]; !exists {
		return nil, fmt.Errorf("S3 region by name '%s' not found", region)
	}

	s := &Source{
		bucket: s3.New(auth, aws.Regions[region]).Bucket(bucket),
	}

	return s, nil
}

func (s *Source) GetFile(path string) ([]byte, error) {
	data, err := s.bucket.Get(path)
	if err != nil {
		return nil, err
	}

	return data, nil
}

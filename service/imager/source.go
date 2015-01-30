package imager

import (
	// Internal packages
	"fmt"
	"os"
	"path"

	// Third-party packages
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

// A Source represents an image source, which is usually matched against a URL endpoint, and
// provides options related to that endpoint.
type Source struct {
	bucket *s3.Bucket
	cache  *FileCache
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

func (s *Source) InitCache(base string, size int64) error {
	path := path.Join(os.TempDir(), base, s.bucket.Region.Name, s.bucket.Name)

	c, err := NewFileCache(path, size)
	if err != nil {
		return err
	}

	s.cache = c
	return nil
}

func (s *Source) GetFile(path string) ([]byte, error) {
	// Check for locally cached data.
	if s.cache != nil {
		if data := s.cache.Get(path); data != nil {
			return data.([]byte), nil
		}
	}

	// Get data from S3 bucket.
	data, err := s.bucket.Get(path)
	if err != nil {
		return nil, err
	}

	// Cache data locally.
	if s.cache != nil {
		s.cache.Add(path, data)
	}

	return data, nil
}

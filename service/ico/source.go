package ico

import (
	// Standard library
	"fmt"
	"os"
	"path"
	"strings"
	"time"

	// Internal packages
	"github.com/deuill/mash/service/ico/image"

	// Third-party packages
	"github.com/goamz/goamz/aws"
	"github.com/goamz/goamz/s3"
)

// A Source represents an image source, which is usually matched against a URL endpoint, and
// provides options related to that endpoint.
type Source struct {
	bucket *s3.Bucket
	cache  *FileCache
}

// NewSource initializes a new source for region and bucket. Access is either provided by access and
// secret keys passed as parameters, or by IAM if the keys are invalid or empty. Any subsequent
// operations on the initialized source will affect the bucket pointed to.
func NewSource(region, bucket, accessKey, secretKey string) (*Source, error) {
	// Authorization token is set to expire 5 years in the future.
	auth, err := aws.GetAuth(accessKey, secretKey, "", time.Now().AddDate(5, 0, 0))
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

// InitCache initializes and attaches local cache to source.
func (s *Source) InitCache(base string, size int64) error {
	base = path.Join(os.TempDir(), base, s.bucket.Region.Name, s.bucket.Name)

	c, err := NewFileCache(base, size)
	if err != nil {
		return err
	}

	s.cache = c
	return nil
}

// Get fetches image data from local cache or S3 bucket for this source.
func (s *Source) Get(name string) (*image.Image, error) {
	// Check for locally cached data.
	if s.cache != nil {
		if data := s.cache.Get(name); data != nil {
			return image.New(data.([]byte))
		}
	}

	// Get data from S3 bucket.
	data, err := s.bucket.Get(name)
	if err != nil {
		return nil, err
	}

	// Cache data locally.
	if s.cache != nil {
		s.cache.Add(name, data)
	}

	return image.New(data)
}

// Put inserts data into local cache and remote S3 bucket for this source.
func (s *Source) Put(name string, data []byte, ctype string) error {
	// Store data locally.
	if s.cache != nil {
		s.cache.Add(name, data)
	}

	// Store data in S3 bucket. The initial upload is placed with a `.tmp` prefix, and is renamed
	// after it has uploaded successfully.
	if err := s.bucket.Put(name+".tmp", data, ctype, "", s3.Options{}); err != nil {
		return err
	}

	src := path.Join(s.bucket.Name, name+".tmp")
	if _, err := s.bucket.PutCopy(name, "", s3.CopyOptions{}, src); err != nil {
		return err
	}

	s.bucket.Del(name + ".tmp")

	return nil
}

// Delete removes one or more files from local cache and S3 bucket for this source.
func (s *Source) Delete(name ...string) error {
	// Delete from local cache.
	if s.cache != nil {
		for _, p := range name {
			s.cache.Remove(p)
		}
	}

	// Build objects list and delete from S3.
	objects := make([]s3.Object, len(name))
	for i := range objects {
		objects[i].Key = strings.TrimPrefix(name[i], "/")
	}

	if err := s.bucket.DelMulti(s3.Delete{true, objects}); err != nil {
		return err
	}

	return nil
}

// ListDirs returns the full paths to any directories contained in path name.
func (s *Source) ListDirs(name string) ([]string, error) {
	resp, err := s.bucket.List(strings.TrimPrefix(name, "/"), "/", "", 0)
	if err != nil {
		return nil, err
	}

	dirs := make([]string, len(resp.CommonPrefixes))
	for i := range resp.CommonPrefixes {
		dirs[i] = "/" + resp.CommonPrefixes[i]
	}

	return dirs, nil
}

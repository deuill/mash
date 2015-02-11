package imager

import (
	// Standard library
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"path"
	"strings"

	// Internal packages
	"github.com/Hearst-Digital/alfred/service"
)

// The Imager service, containing state shared between methods.
type Imager struct {
	Quota       *int64  // The image cache size maximum, in bytes.
	S3Region    *string // S3 region to use for bucket.
	S3Bucket    *string // S3 bucket to use for image access.
	S3AccessKey *string // Access key to use for bucket. If empty, access will be attempted with IAM.
	S3SecretKey *string // Secret key to use for bucket. If empty, access will be attempted with IAM.

	sources map[string]*Source // A map of sources, indexed under their region and bucket name.
}

func (m *Imager) Process(r *http.Request, w http.ResponseWriter) (interface{}, error) {
	// Get source for this request, pulling the region and bucket names from request headers.
	src, err := m.getSource(r.Header.Get("X-S3-Region"), r.Header.Get("X-S3-Bucket"))
	if err != nil {
		return nil, err
	}

	// Split request to its significant parts.
	fields := strings.SplitN(r.URL.Path, "/", 5)
	if len(fields) < 5 || fields[4] == "" {
		return nil, fmt.Errorf("image URL is unset or empty")
	}

	params, filepath := fields[3], fields[4]

	dir, file := path.Split(filepath)
	procpath := path.Join(dir, params, file)

	// Fetch existing processed file, if any.
	if data, _ := src.Get(procpath); data != nil {
		writeResponse(data, int64(len(data)), GetFileType(data), w)
		return nil, nil
	}

	// Unpack request parameters and prepare a pipeline for subsequent operations.
	req, err := base64.StdEncoding.DecodeString(params)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64-encoded parameters")
	}

	pipeline, err := NewPipeline()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pipeline: %s", err)
	}

	for _, p := range strings.Split(string(req), ",") {
		t := strings.Split(p, "=")
		if len(t) != 2 {
			return nil, fmt.Errorf("request parameter for pipeline is malformed: '%s'", p)
		}

		key, value := strings.TrimSpace(t[0]), strings.TrimSpace(t[1])
		if err = pipeline.SetString(key, value); err != nil {
			return nil, err
		}
	}

	// Fetch original image from remote server or local cache.
	origImg, err := src.Get(filepath)
	if err != nil {
		return nil, err
	}

	// Process image through pipeline.
	img, err := pipeline.Process(origImg)
	if err != nil {
		return nil, err
	}

	// Store image in local cache and upload to S3 bucket asynchronously.
	go src.Put(procpath, img.Data, img.Type)

	// Write response back to user.
	writeResponse(img.Data, img.Size, img.Type, w)

	return nil, nil
}

func (m *Imager) getSource(region, bucket string) (*Source, error) {
	var err error
	var access, secret string

	// Fall back to default values if either region name or bucket name is empty.
	if region == "" || bucket == "" {
		access, secret = *m.S3AccessKey, *m.S3SecretKey
		region, bucket = *m.S3Region, *m.S3Bucket
	}

	key := region + "/" + bucket

	// Check for existing source, or initialize new source for specified region and bucket.
	src, exists := m.sources[key]
	if !exists {
		if src, err = NewSource(region, bucket, access, secret); err != nil {
			return nil, err
		}

		src.InitCache("alfred/imager", *m.Quota)
		m.sources[key] = src
	}

	return m.sources[key], nil
}

func writeResponse(data []byte, size int64, ctype string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Cache-Control", "no-transform,public,max-age=86400,s-maxage=2592000")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (m *Imager) Start() error {
	// Register handler methods under their names.
	service.RegisterHandler("imager", "process", m.Process)

	return nil
}

func (m *Imager) Stop() error {
	return nil
}

// Package initialization, attaches options and registers service with Alfred.
func init() {
	fs := flag.NewFlagSet("imager", flag.ContinueOnError)
	serv := &Imager{
		Quota:       fs.Int64("quota", 0, ""),
		S3Region:    fs.String("s3-region", "", ""),
		S3Bucket:    fs.String("s3-bucket", "", ""),
		S3AccessKey: fs.String("s3-access-key", "", ""),
		S3SecretKey: fs.String("s3-secret-key", "", ""),
		sources:     make(map[string]*Source),
	}

	service.Register("imager", serv, fs)
}

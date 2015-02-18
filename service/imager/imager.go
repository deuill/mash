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

// Process request for image transformation, taking care caching both to local disk and S3.
func (m *Imager) Process(w http.ResponseWriter, r *http.Request, p service.Params) (interface{}, error) {
	// Get source for this request, pulling the region and bucket names from request headers.
	src, err := m.getSource(r.Header.Get("X-S3-Region"), r.Header.Get("X-S3-Bucket"))
	if err != nil {
		return nil, err
	}

	params, imgPath := p.Get("params"), p.Get("image")

	if imgPath == "" {
		return nil, fmt.Errorf("image URL is unset or empty")
	}

	dir, file := path.Split(imgPath)
	procPath := path.Join(dir, params, file)

	// Fetch existing processed file, if any.
	if data, _ := src.Get(procPath); data != nil {
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
		if err = pipeline.SetOption(key, value); err != nil {
			return nil, err
		}
	}

	// Fetch original image from remote server or local cache.
	origImg, err := src.Get(imgPath)
	if err != nil {
		return nil, err
	}

	// Process image through pipeline.
	img, err := pipeline.Process(origImg)
	if err != nil {
		return nil, err
	}

	// Store image in local cache and upload to S3 bucket asynchronously.
	go src.Put(procPath, img.Data, img.Type)

	// Write response back to user.
	writeResponse(img.Data, img.Size, img.Type, w)

	return nil, nil
}

// Gets source according to region and bucket, and initializes local cache on that source. Passing
// an empty region and bucket name will have Imager fall back to the configuration defaults, if any.
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

		if err = src.InitCache("alfred/imager", *m.Quota); err != nil {
			return nil, err
		}

		m.sources[key] = src
	}

	return m.sources[key], nil
}

// Writes image data back to user.
func writeResponse(data []byte, size int64, ctype string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Cache-Control", "no-transform,public,max-age=86400,s-maxage=2592000")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

// Package initialization, attaches options and registers service with Alfred.
func init() {
	flags := flag.NewFlagSet("imager", flag.ContinueOnError)
	serv := &Imager{
		Quota:       flags.Int64("quota", 0, ""),
		S3Region:    flags.String("s3-region", "", ""),
		S3Bucket:    flags.String("s3-bucket", "", ""),
		S3AccessKey: flags.String("s3-access-key", "", ""),
		S3SecretKey: flags.String("s3-secret-key", "", ""),
		sources:     make(map[string]*Source),
	}

	// Register Imager service along with handler methods.
	service.Register("imager", flags, []service.Handler{
		{"GET", "/:params/*image", serv.Process},
	})
}

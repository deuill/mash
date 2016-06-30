package ico

import (
	// Standard library
	"flag"
	"fmt"
	"net/http"
	"path"
	"strings"

	// Internal packages
	"github.com/deuill/mash/service"
)

// The Ico service, containing state shared between methods.
type Ico struct {
	Quota       *int64  // The image cache size maximum, in bytes.
	S3Region    *string // S3 region to use for bucket.
	S3Bucket    *string // S3 bucket to use for image access.
	S3AccessKey *string // Access key to use for bucket. If empty, access will be attempted with IAM.
	S3SecretKey *string // Secret key to use for bucket. If empty, access will be attempted with IAM.

	sources map[string]*Source // A map of sources, indexed under their region and bucket name.
}

// Process request for image transformation, taking care caching both to local disk and S3.
func (m *Ico) Process(w http.ResponseWriter, r *http.Request, p service.Params) (*service.Response, error) {
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

	// Prepare pipeline and set parameters from user request.
	pipeline, err := NewPipeline()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pipeline: %s", err)
	}

	for _, p := range strings.Split(params, ",") {
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
		return nil, fmt.Errorf("failed to fetch from source: %s", err)
	}

	// Process image through pipeline.
	img, err := pipeline.Process(origImg)
	if err != nil {
		return nil, fmt.Errorf("failed to process image: %s", err)
	}

	// If processing a GET request, store image locally and upload to S3 bucket asynchronously, then
	// write image back to user. Otherwise, wait for upload process to complete and return nothing.
	if r.Method == "GET" {
		go src.Put(procPath, img.Data, img.Type)
		writeResponse(img.Data, img.Size, img.Type, w)
	} else {
		src.Put(procPath, img.Data, img.Type)
	}

	return nil, nil
}

// Purge removes the original image pointed to by the request, along with any processed child images
// in the local cache and the remote server.
func (m *Ico) Purge(w http.ResponseWriter, r *http.Request, p service.Params) (*service.Response, error) {
	// Get source for this request, pulling the region and bucket names from request headers.
	src, err := m.getSource(r.Header.Get("X-S3-Region"), r.Header.Get("X-S3-Bucket"))
	if err != nil {
		return nil, err
	}

	// Get image URL from request.
	imgPath := p.Get("image")
	if imgPath == "" {
		return nil, fmt.Errorf("image URL is unset or empty")
	}

	imgDir, imgName := path.Split(imgPath)

	// Fetch list of directories in image path and append image name to each directory.
	dirList, err := src.ListDirs(imgDir)
	if err != nil {
		return nil, err
	}

	dirList = append(dirList, imgDir)
	for i := range dirList {
		dirList[i] = path.Join(dirList[i], imgName)
	}

	// Delete images from local and remote cache.
	if err = src.Delete(dirList...); err != nil {
		return nil, err
	}

	return &service.Response{http.StatusOK, map[string]bool{"result": true}}, nil
}

// Gets source according to region and bucket, and initializes local cache on that source. Passing
// an empty region and bucket name will have Ico fall back to the configuration defaults, if any.
func (m *Ico) getSource(region, bucket string) (*Source, error) {
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

		if err = src.InitCache("mash/ico", *m.Quota); err != nil {
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

// Package initialization, attaches options and registers service with Mash.
func init() {
	flags := flag.NewFlagSet("ico", flag.ContinueOnError)
	serv := &Ico{
		Quota:       flags.Int64("quota", 0, ""),
		S3Region:    flags.String("s3-region", "", ""),
		S3Bucket:    flags.String("s3-bucket", "", ""),
		S3AccessKey: flags.String("s3-access-key", "", ""),
		S3SecretKey: flags.String("s3-secret-key", "", ""),
		sources:     make(map[string]*Source),
	}

	// Register Ico service along with handler methods.
	service.Register("ico", flags, []service.Handler{
		{"HEAD", "/:params/*image", serv.Process},
		{"GET", "/:params/*image", serv.Process},
		{"DELETE", "/*image", serv.Purge},
	})
}

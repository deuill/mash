package imager

import (
	// Standard library
	"encoding/base64"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os/user"
	"strings"

	// Internal packages
	"github.com/Hearst-Digital/alfred/service"

	// Third-party packages
	ini "github.com/rakyll/goini"
)

// The Imager service, containing state shared between methods.
type Imager struct {
	CacheSize *int64  // The image cache size maximum, in bytes.
	Config    *string // Path to imager.ini file, used for defining sources and their options.

	sources map[string]*Source // A map of sources, indexed under their name.
}

func (i *Imager) Process(r *http.Request, w http.ResponseWriter) (interface{}, error) {
	// Split request to its significant parts.
	fields := strings.SplitN(r.URL.Path, "/", 5)
	if len(fields) < 5 || fields[4] == "" {
		return nil, fmt.Errorf("image URL is unset or empty")
	}

	data, path := fields[3], fields[4]

	// Unpack request parameters and prepare a pipeline for subsequent operations.
	params, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64-encoded parameters")
	}

	pipeline, err := NewPipeline()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pipeline: %s", err)
	}

	for _, p := range strings.Split(string(params), ",") {
		t := strings.Split(p, "=")
		if len(t) != 2 {
			return nil, fmt.Errorf("request parameter for pipeline is malformed: '%s'", p)
		}

		key, value := t[0], strings.TrimSpace(t[1])
		if err = pipeline.SetString(key, value); err != nil {
			return nil, err
		}
	}

	// Select source to fetch from and push to depending on the request Host field.
	// If the field is empty or invalid, the default source is used instead.
	src := i.getSource(r.Host)

	// Fetch original image from remote server or local cache.
	orig, err := src.Get(path)
	if err != nil {
		return nil, err
	}

	// Process image through pipeline.
	img, err := pipeline.Process(orig)
	if err != nil {
		return nil, err
	}

	writeResponse(img.Data, img.Size, img.Type, w)

	return nil, nil
}

func (i *Imager) getSource(host string) *Source {
	src := i.sources["default"]
	if host != "" {
		h, _, _ := net.SplitHostPort(host)
		if s, ok := i.sources[h]; ok {
			src = s
		}
	}

	return src
}

func writeResponse(data []byte, size int64, ctype string, w http.ResponseWriter) {
	w.Header().Set("Content-Type", ctype)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", size))
	w.Header().Set("Cache-Control", "no-transform,public,max-age=86400,s-maxage=2592000")
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (i *Imager) Start() error {
	// Register handler methods under their names.
	service.RegisterHandler("imager", "process", i.Process)

	// Load Imager configuration file and initialize sources.
	dict, err := ini.Load(*i.Config)
	if err != nil {
		return err
	}

	for _, sect := range dict.GetSections() {
		region, _ := dict.GetString(sect, "s3_region")
		bucket, _ := dict.GetString(sect, "s3_bucket")
		access, _ := dict.GetString(sect, "s3_access_key")
		secret, _ := dict.GetString(sect, "s3_secret_key")

		s, err := NewSource(region, bucket, access, secret)
		if err != nil {
			return err
		}

		if sect == "" {
			i.sources["default"] = s
		} else {
			i.sources[sect] = s
		}

		if err = s.InitCache("alfred/imager", *i.CacheSize); err != nil {
			return err
		}
	}

	return nil
}

func (i *Imager) Stop() error {
	return nil
}

// Package initialization, attaches options and registers service with Alfred.
func init() {
	// Fallback configuration directory is '/etc/alfred'.
	confPath := "/etc"

	// Attempt to set default configuration directory to user's home.
	if u, _ := user.Current(); u != nil {
		confPath = u.HomeDir + "/.config"
	}

	fs := flag.NewFlagSet("imager", flag.ContinueOnError)
	serv := &Imager{
		CacheSize: fs.Int64("cachesize", 0, ""),
		Config:    fs.String("config", confPath+"/alfred/imager.ini", ""),
		sources:   make(map[string]*Source),
	}

	service.Register("imager", serv, fs)
}

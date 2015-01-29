package imager

import (
	// Standard library
	"encoding/base64"
	"flag"
	"fmt"
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
	Config  *string            // Path to imager.ini file, used for defining sources and their options.
	Sources map[string]*Source // A map of sources, indexed under their name.
}

func (i *Imager) Call(method string, r *http.Request, w http.ResponseWriter) (interface{}, error) {
	// Split request to its significant parts.
	fields := strings.SplitN(r.URL.Path, "/", 4)
	if len(fields) < 4 || fields[3] == "" {
		return nil, fmt.Errorf("image URL is unset or empty")
	}

	data, _ := fields[2], fields[3]

	// Unpack request parameters and prepare a pipeline for subsequent operations.
	params, err := base64.StdEncoding.DecodeString(data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode base64-encoded parameters")
	}

	pline := NewPipeline()

	for _, p := range strings.Split(string(params), ",") {
		t := strings.Split(p, "=")
		if len(t) != 2 {
			return nil, fmt.Errorf("request parameter is malformed: '%s'", p)
		}

		key, value := t[0], strings.TrimSpace(t[1])
		if err = pline.SetString(key, value); err != nil {
			return nil, err
		}
	}

	return pline, nil
}

func (i *Imager) Start() error {
	dict, err := ini.Load(*i.Config)
	if err != nil {
		return err
	}

	for _, name := range dict.GetSections() {
		bucket, _ := dict.GetString(name, "s3_bucket")
		access, _ := dict.GetString(name, "s3_access_key")
		secret, _ := dict.GetString(name, "s3_secret_key")

		i.Sources[name] = NewSource(bucket, access, secret)
	}

	// A default source should always be set.
	if _, exists := i.Sources["default"]; !exists {
		return fmt.Errorf("no default source found in config")
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
		Config:  fs.String("config", confPath+"/alfred/imager.ini", ""),
		Sources: make(map[string]*Source),
	}

	service.Register("imager", serv, fs)
}

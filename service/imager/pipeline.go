package imager

// #cgo pkg-config: vips
// #include "vips.h"
import "C"

import (
	"bytes"
	"fmt"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"unsafe"
)

// A Pipeline represents all data required for converting an image from its original format to the
// processed result.
type Pipeline struct {
	Width   int64   `default:"0"`
	Height  int64   `default:"0"`
	Density float64 `default:"1"`
	Quality int64   `default:"75"`
	Fit     string  `default:"crop"`
}

func NewPipeline() (*Pipeline, error) {
	p := &Pipeline{}
	pt := reflect.ValueOf(p).Elem().Type()

	// Set default values from field tags.
	for i := 0; i < pt.NumField(); i++ {
		f := pt.Field(i)
		if err := p.SetString(f.Name, f.Tag.Get("default")); err != nil {
			return nil, err
		}
	}

	return p, nil
}

func (p *Pipeline) SetString(field, value string) error {
	pv := reflect.ValueOf(p).Elem()
	f := pv.FieldByName(strings.Title(field))
	if f.Kind() == reflect.Invalid {
		return fmt.Errorf("field with name '%s' not found", field)
	}

	switch f.Kind() {
	case reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to convert value to concrete 'int64' type: %s", err)
		}

		f.SetInt(v)
	case reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("unable to convert value to concrete 'float64' type: %s", err)
		}

		f.SetFloat(v)
	case reflect.String:
		f.SetString(value)
	default:
		return fmt.Errorf("field '%s' with type '%s' does not match supported types", field, f.Kind().String())
	}

	return nil
}

// Image represents a processed image, and contains the image data as a byte slice along with other
// useful information about the image.
type Image struct {
	Data   []byte
	Width  int
	Height int
	Type   string
}

// A map of supported image MIME types against their magic numbers.
var imageTypes = map[string][]byte{
	"image/jpeg": []byte{0xff, 0xd8},
	"image/png":  []byte{0x89, 0x50},
	"image/gif":  []byte{0x47, 0x49},
}

func (p *Pipeline) Process(buf []byte) (*Image, error) {
	// Image definition for generated image.
	img := Image{make([]byte, 0), 0, 0, ""}

	// Detect file type for image in buffer.
	for t, sig := range imageTypes {
		if bytes.Equal(buf[:2], sig) {
			img.Type = t
		}
	}

	switch img.Type {
	// GIF images are not supported by VIPS directly, and as such must be handled as a special case.
	// This is done by extracting the frames in a GIF, processing them as PNG images, and converting
	// back to a GIF once VIPS is done processing each frame individually.
	case "image/gif":
		return nil, fmt.Errorf("GIF format support upcoming")
	case "":
		return nil, fmt.Errorf("unknown image type, cannot process")
	}

	vipsImg := C.vips_image_new()
	defer C.vips_thread_shutdown()

	switch img.Type {
	case "image/jpeg":
		C.vips_load_buffer_jpeg(unsafe.Pointer(&buf[0]), C.size_t(len(buf)), &vipsImg)
	case "image/png":
		C.vips_load_buffer_png(unsafe.Pointer(&buf[0]), C.size_t(len(buf)), &vipsImg)
	}

	img.Width = int(vipsImg.Xsize)
	img.Height = int(vipsImg.Ysize)

	// Calculate resulting dimensions based on pipeline parameters.

	return &img, nil
}

func init() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := C.vips_initialize(); err != 0 {
		C.vips_shutdown()
		panic("unable to initialize VIPS library")
	}

	C.vips_concurrency_set(1)
	C.vips_cache_set_max_mem(1024 * 128) // 128MB
	C.vips_cache_set_max(1024 * 512)     // 512MB
}

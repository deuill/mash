package imager

// #cgo pkg-config: vips
// #include "vips.h"
import "C"

import (
	"bytes"
	"fmt"
	"math"
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
	Size   int64
	Type   string
	Width  int64
	Height int64
}

// A map of supported image MIME types against their magic numbers.
var imageTypes = map[string][]byte{
	"image/jpeg": []byte{0xff, 0xd8},
	"image/png":  []byte{0x89, 0x50},
	"image/gif":  []byte{0x47, 0x49},
}

func (p *Pipeline) Process(buf []byte) (*Image, error) {
	// Image definition for generated image.
	img := Image{buf, int64(len(buf)), "", 0, 0}

	// Detect file type for image in buffer.
	for t, sig := range imageTypes {
		if bytes.Equal(img.Data[:2], sig) {
			img.Type = t
		}
	}

	switch img.Type {
	// GIF images are not supported by VIPS directly, and as such must be handled as a special case.
	// This is done by extracting the frames in a GIF, processing them as PNG images, and converting
	// back to a GIF once VIPS is done processing each frame individually.
	case "image/gif":
		return nil, fmt.Errorf("images in GIF format are not supported yet")
	case "":
		return nil, fmt.Errorf("unknown image type, cannot process")
	}

	vipsImg := C.vips_image_new()

	defer C.vips_error_clear()
	defer C.vips_thread_shutdown()

	switch img.Type {
	case "image/jpeg":
		C.Vips_load_jpeg(unsafe.Pointer(&img.Data[0]), C.size_t(img.Size), &vipsImg)
	case "image/png":
		C.Vips_load_png(unsafe.Pointer(&img.Data[0]), C.size_t(img.Size), &vipsImg)
	}

	img.Width = int64(vipsImg.Xsize)
	img.Height = int64(vipsImg.Ysize)

	factor := 0.0

	// If the pipeline requests an enlarged image, or dimensions equal to original image, return original.
	if (p.Width > img.Width || p.Height > img.Height) || (p.Width == img.Width && p.Height == img.Height) {
		return &img, nil
	}

	// Calculate resize factor based on pipeline parameters.
	switch {
	// Fixed width and height.
	case p.Width > 0 && p.Height > 0:
		xf := float64(img.Width) / float64(p.Width)
		yf := float64(img.Height) / float64(p.Height)

		// We choose the smallest delta when cropping, and the largest when we're not.
		if p.Fit == "crop" {
			factor = math.Min(xf, yf)
		} else {
			factor = math.Max(xf, yf)
		}
	// Fixed width, auto height.
	case p.Width > 0:
		factor = float64(img.Width) / float64(p.Width)
		p.Height = int64(math.Floor(float64(img.Height) / factor))
	// Fixed height, auto width.
	case p.Height > 0:
		factor = float64(img.Height) / float64(p.Height)
		p.Width = int64(math.Floor(float64(img.Width) / factor))
	// No change requested, return original image.
	default:
		return &img, nil
	}

	// We resize images in a two-step operation, first shrinking the image by an integer factor,
	// then calculating the floating-point residual and interpolating the result.
	shrink := int(math.Floor(factor))
	if shrink < 1 {
		shrink = 1
	}

	residual := float64(shrink) / factor

	// VIPS supports a shrink-on-load operation for JPEG images, which is much more efficient than
	// generating a full-size image and shrinking afterwards.
	if shrink > 1 && img.Type == "image/jpeg" {
		var shrinkLoad int

		switch {
		case shrink >= 8:
			factor = factor / 8
			shrinkLoad = 8
		case shrink >= 4:
			factor = factor / 4
			shrinkLoad = 4
		case shrink >= 2:
			factor = factor / 2
			shrinkLoad = 2
		}

		// Recalculate shrink and residual values for shrunk image.
		factor = math.Max(factor, 1.0)
		shrink = int(math.Floor(factor))
		residual = float64(shrink) / factor

		vipsShrunk := C.vips_image_new()
		ptr := unsafe.Pointer(&img.Data[0])
		err := C.Vips_shrink_load_jpeg(ptr, C.size_t(img.Size), &vipsShrunk, C.int(shrinkLoad))
		if err != 0 {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to shrink JPEG image: %s", e)
		}

		C.Vips_copy_clear(vipsShrunk, &vipsImg)
	}

	// We shrink the image by an integer factor, if the factor is bigger than 1.
	if shrink > 1 {
		vipsShrunk := C.vips_image_new()
		err := C.Vips_shrink(vipsImg, &vipsShrunk, C.double(float64(shrink)), C.double(float64(shrink)))
		if err != 0 {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to shrink image: %s", e)
		}

		// Recalculate residual factor.
		rx := float64(p.Width) / float64(int(vipsShrunk.Xsize))
		ry := float64(p.Height) / float64(int(vipsShrunk.Ysize))

		if p.Fit == "crop" {
			residual = math.Max(rx, ry)
		} else {
			residual = math.Min(rx, ry)
		}

		C.Vips_copy_clear(vipsShrunk, &vipsImg)
	}

	// Resize image by the residual factor, if any is left over.
	if residual != 0 {
		vipsAffined := C.vips_image_new()
		err := C.Vips_affine_bilinear(vipsImg, &vipsAffined, C.double(residual), 0, 0, C.double(residual))
		if err != 0 {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to resize image: %s", e)
		}

		C.Vips_copy_clear(vipsAffined, &vipsImg)
	}

	// Crop image if required.
	if p.Fit == "crop" && (int64(vipsImg.Xsize) != p.Width || int64(vipsImg.Ysize) != p.Height) {
		p.Width = int64(math.Min(float64(vipsImg.Xsize), float64(p.Width)))
		p.Height = int64(math.Min(float64(vipsImg.Ysize), float64(p.Height)))

		vipsCropped := C.vips_image_new()
		err := C.Vips_crop(vipsImg, &vipsCropped, C.int(0), C.int(0), C.int(p.Width), C.int(p.Height))
		if err != 0 {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to crop image: %s", e)
		}

		C.Vips_copy_clear(vipsCropped, &vipsImg)
	}

	// Convert to sRGB colour space.
	vipsColourspaced := C.vips_image_new()
	C.Vips_colourspace(vipsImg, &vipsColourspaced, C.VIPS_INTERPRETATION_sRGB)
	C.Vips_copy_clear(vipsColourspaced, &vipsImg)

	// Save image to buffer.
	length := C.size_t(0)
	ptr := C.malloc(C.size_t(len(buf)))

	switch img.Type {
	case "image/jpeg":
		C.Vips_save_jpeg(vipsImg, &ptr, &length, C.int(p.Quality))
	case "image/png":
		C.Vips_save_png(vipsImg, &ptr, &length, C.int(9))
	}

	img.Data = C.GoBytes(ptr, C.int(length))
	img.Size = int64(len(img.Data))
	img.Width = int64(vipsImg.Xsize)
	img.Height = int64(vipsImg.Ysize)

	// Clean up data.
	C.g_object_unref(C.gpointer(vipsImg))
	C.free(ptr)

	return &img, nil
}

func init() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if err := C.Vips_init(); err != 0 {
		C.vips_shutdown()
		panic("unable to initialize VIPS library")
	}

	C.vips_concurrency_set(1)
	C.vips_cache_set_max_mem(1048576 * 128) // 128MB
	C.vips_cache_set_max(500)               // 500 operations
}

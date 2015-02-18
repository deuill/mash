package imager

// #cgo pkg-config: vips
// #include "vips.h"
import "C"

import (
	// Standard library
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/gif"
	"image/jpeg"
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
	Width   int64     `default:"0"`                    // Image width. If 0, calculate from height.
	Height  int64     `default:"0"`                    // Image height. If 0, calculate from width.
	Quality int64     `default:"75" min:"1" max:"100"` // The quality of the image, for JPEG output.
	Fit     string    `default:"crop"`                 // The fit mode. Values: "clip" and "crop".
	Crop    string    `default:"top"`                  // Cropping strategy. Values: "top", "bottom", "left", "right" and "focus".
	Focus   []float64 `default:"0:0:0:0" delim:":"`    // Focus bounding box. Values: "x", "y", "w" and "h".
	Frame   bool      `default:"false"`                // If true, only return first frame of animated GIF.
}

// NewPipeline initializes a new pipeline, along with defaults as set in the `Pipeline` structure
// definition.
func NewPipeline() (*Pipeline, error) {
	p := &Pipeline{}
	pt := reflect.ValueOf(p).Elem().Type()

	// Set default values from field tags.
	for i := 0; i < pt.NumField(); i++ {
		f := pt.Field(i)
		if err := p.SetOption(f.Name, f.Tag.Get("default")); err != nil {
			return nil, err
		}
	}

	return p, nil
}

// SetOption sets pipeline option specified by `value`, silently converting the string value to
// the type required by the `Pipeline` structure field. If the field is not found, or setting the
// value is impossible, an error is returned.
func (p *Pipeline) SetOption(field, value string) error {
	fname := strings.Title(field)

	pv := reflect.ValueOf(p).Elem()
	ft, exists := pv.Type().FieldByName(fname)
	if !exists {
		return fmt.Errorf("field with name '%s' not found", field)
	}

	f := pv.FieldByName(fname)
	switch {
	case f.Kind() == reflect.Slice && f.Type().Elem().Kind() == reflect.Float64:
		f.Set(reflect.Zero(f.Type()))

		s := strings.Split(value, ft.Tag.Get("delim"))
		for _, sv := range s {
			v, err := strconv.ParseFloat(sv, 64)
			if err != nil {
				return fmt.Errorf("unable to convert value to concrete type: %s", err)
			}

			f.Set(reflect.Append(f, reflect.ValueOf(v)))
		}
	case f.Kind() == reflect.Int64:
		v, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fmt.Errorf("unable to convert value to concrete type: %s", err)
		}

		// Check for minimum and maximum values.
		if ft.Tag.Get("min") != "" && ft.Tag.Get("max") != "" {
			min, _ := strconv.ParseInt(ft.Tag.Get("min"), 10, 64)
			max, _ := strconv.ParseInt(ft.Tag.Get("max"), 10, 64)

			if v < min || v > max {
				return fmt.Errorf("value passed for '%s' is outside the limit '%d - %d': %d", field, min, max, v)
			}
		}

		f.SetInt(v)
	case f.Kind() == reflect.Float64:
		v, err := strconv.ParseFloat(value, 64)
		if err != nil {
			return fmt.Errorf("unable to convert value to concrete type: %s", err)
		}

		f.SetFloat(v)
	case f.Kind() == reflect.Bool:
		var v bool
		if value == "true" {
			v = true
		}

		f.SetBool(v)
	case f.Kind() == reflect.String:
		f.SetString(value)
	default:
		return fmt.Errorf("field '%s' with type '%s' does not match supported types", field, f.Kind().String())
	}

	return nil
}

// Image represents a processed image, and contains the image data as a byte slice along with other
// useful information about the image.
type Image struct {
	Data []byte // The image data buffer
	Size int64  // The image size, in bytes.
	Type string // The image MIME type.
}

// Process image in `data` according to the pipeline setup, and return resulting Image structure as
// a pointer. The original data passed to the function is never modified directly, and is safe for
// reuse.
func (p *Pipeline) Process(data []byte) (*Image, error) {
	imgType := GetFileType(data)

	switch imgType {
	// GIF images are not supported by VIPS directly, and as such must be handled as a special case.
	// This is done by extracting the frames in a GIF, processing them as JPEG images, and converting
	// back to a GIF once VIPS is done processing each frame individually.
	case "image/gif":
		buf := bytes.NewBuffer(data)
		gifdec, err := gif.DecodeAll(buf)
		if err != nil {
			return nil, fmt.Errorf("unable to decode image: %s", err)
		}

		if p.Frame {
			gifdec.Image = gifdec.Image[:1]
		}

		frames := make([]*Image, len(gifdec.Image))

		// Convert and process available frames in GIF one-by-one.
		for i, gifimg := range gifdec.Image {
			b := new(bytes.Buffer)
			if err = jpeg.Encode(b, gifimg, &jpeg.Options{Quality: int(p.Quality)}); err != nil {
				return nil, fmt.Errorf("unable to decode image: %s", err)
			}

			im, err := p.Process(b.Bytes())
			if err != nil {
				return nil, err
			}

			frames[i] = im
		}

		// Return first frame if we're in "frame" mode.
		if p.Frame {
			return frames[0], nil
		}

		// Otherwise, process each frame and rebuild GIF file.
		gifenc := &gif.GIF{make([]*image.Paletted, len(frames)), gifdec.Delay, gifdec.LoopCount}
		for i, frm := range frames {
			b := bytes.NewBuffer(frm.Data)
			im, err := jpeg.Decode(b)
			if err != nil {
				return nil, fmt.Errorf("unable to decode image: %s", err)
			}

			pimg := image.NewPaletted(im.Bounds(), gifdec.Image[0].Palette)
			draw.Draw(pimg, im.Bounds(), im, image.Point{0, 0}, 0)
			gifenc.Image[i] = pimg
		}

		// Encode final GIF file.
		b := new(bytes.Buffer)
		if err = gif.EncodeAll(b, gifenc); err != nil {
			return nil, fmt.Errorf("unable to encode image: %s", err)
		}

		img := &Image{b.Bytes(), int64(b.Len()), imgType}
		return img, nil
	case "application/octet-stream":
		return nil, fmt.Errorf("unknown image type, cannot process")
	}

	img := &Image{data, int64(len(data)), imgType}
	vipsImg := C.Vips_image_init()

	defer C.vips_error_clear()
	defer C.vips_thread_shutdown()

	switch img.Type {
	case "image/jpeg":
		vipsImg = C.Vips_load_jpeg(unsafe.Pointer(&img.Data[0]), C.size_t(img.Size))
	case "image/png":
		vipsImg = C.Vips_load_png(unsafe.Pointer(&img.Data[0]), C.size_t(img.Size))
	}

	if vipsImg == nil {
		return nil, fmt.Errorf("failed to load image of type '%s'", img.Type)
	}

	defer C.g_object_unref(C.gpointer(vipsImg))

	var factor float64
	imgWidth := int64(vipsImg.Xsize)
	imgHeight := int64(vipsImg.Ysize)

	// If the pipeline requests an enlarged image, or dimensions equal to original image, return original.
	if (p.Width > imgWidth || p.Height > imgHeight) || (p.Width == imgWidth && p.Height == imgHeight) {
		return img, nil
	}

	// Calculate resize factor based on pipeline parameters.
	switch {
	// Fixed width and height.
	case p.Width > 0 && p.Height > 0:
		xf := float64(imgWidth) / float64(p.Width)
		yf := float64(imgHeight) / float64(p.Height)

		// We choose the smallest delta when cropping, and the largest when we're not.
		if p.Fit == "crop" {
			factor = math.Min(xf, yf)
		} else {
			factor = math.Max(xf, yf)
		}
	// Fixed width, auto height.
	case p.Width > 0:
		factor = float64(imgWidth) / float64(p.Width)
		p.Height = int64(math.Floor(float64(imgHeight) / factor))
	// Fixed height, auto width.
	case p.Height > 0:
		factor = float64(imgHeight) / float64(p.Height)
		p.Width = int64(math.Floor(float64(imgWidth) / factor))
	// No change requested, return original image.
	default:
		return img, nil
	}

	// We resize images in a two-step operation, first shrinking the image by an integer factor,
	// then calculating the floating-point residual and interpolating the result.
	shrink := int64(math.Floor(factor))
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
		shrink = int64(math.Floor(factor))
		residual = float64(shrink) / factor

		ptr := unsafe.Pointer(&img.Data[0])
		vipsShrunk := C.Vips_shrink_load_jpeg(ptr, C.size_t(img.Size), C.int(shrinkLoad))
		if vipsShrunk == nil {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to shrink JPEG image: %s", e)
		}

		defer C.g_object_unref(C.gpointer(vipsShrunk))
		vipsImg = vipsShrunk
	}

	// We shrink the image by an integer factor, if the factor is bigger than 1.
	if shrink > 1 {
		vipsShrunk := C.Vips_shrink(vipsImg, C.double(float64(shrink)), C.double(float64(shrink)))
		if vipsShrunk == nil {
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

		defer C.g_object_unref(C.gpointer(vipsShrunk))
		vipsImg = vipsShrunk
	}

	// Resize image by the residual factor, if any is left over.
	if residual != 0 {
		vipsAffined := C.Vips_affine_bilinear(vipsImg, C.double(residual), 0, 0, C.double(residual))
		if vipsAffined == nil {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to resize image: %s", e)
		}

		defer C.g_object_unref(C.gpointer(vipsAffined))
		vipsImg = vipsAffined
	}

	// Crop image if required.
	if p.Fit == "crop" && (int64(vipsImg.Xsize) != p.Width || int64(vipsImg.Ysize) != p.Height) {
		var cx, cy int64
		w, h := int64(vipsImg.Xsize), int64(vipsImg.Ysize)

		switch p.Crop {
		// Crop using specified bounding box as center of focus.
		case "focus":
			if len(p.Focus) != 4 {
				return nil, fmt.Errorf("failed to crop image: invalid format for focus box")
			}

			bx, bw := p.Focus[0], p.Focus[2]
			by, bh := p.Focus[1], p.Focus[3]

			// Recalculate bounding box position and dimensions based on the resize factor.
			factor = math.Max(float64((imgWidth / w)), float64((imgHeight / h)))
			cx = int64(math.Floor(((bx + (bw / 2)) / factor))) - (p.Width / 2)
			cy = int64(math.Floor(((by + (bh / 2)) / factor))) - (p.Height / 2)

			// Find X and Y offset for the crop bounding box and keep the value within constraints.
			cx = int64(math.Min(math.Max(0, float64(cx)), float64((w - p.Width))))
			cy = int64(math.Min(math.Max(0, float64(cy)), float64((h - p.Height))))
		// Crop from the right to left.
		case "right":
			cy = (h - p.Height + 1) / 2
		// Crop from the left to right.
		case "left":
			cx = w - p.Width
			cy = (h - p.Height + 1) / 2
		// Crop from the bottom up.
		case "bottom":
			cx = (w - p.Width + 1) / 2
		// Crop from the top down.
		default:
			cx = (w - p.Width + 1) / 2
			cy = h - p.Height
		}

		p.Width = int64(math.Min(float64(w), float64(p.Width)))
		p.Height = int64(math.Min(float64(h), float64(p.Height)))

		vipsCropped := C.Vips_crop(vipsImg, C.int(cx), C.int(cy), C.int(p.Width), C.int(p.Height))
		if vipsCropped == nil {
			e := C.GoString(C.vips_error_buffer())
			return nil, fmt.Errorf("failed to crop image: %s", e)
		}

		defer C.g_object_unref(C.gpointer(vipsCropped))
		vipsImg = vipsCropped
	}

	// Convert to sRGB colour space.
	vipsColourspaced := C.Vips_colourspace(vipsImg, C.VIPS_INTERPRETATION_sRGB)
	if vipsColourspaced == nil {
		e := C.GoString(C.vips_error_buffer())
		return nil, fmt.Errorf("failed to convert colour space for image: %s", e)
	}

	defer C.g_object_unref(C.gpointer(vipsColourspaced))
	vipsImg = vipsColourspaced

	// Save image to buffer.
	var ptr unsafe.Pointer
	length := C.size_t(0)

	switch img.Type {
	case "image/jpeg":
		ptr = C.Vips_save_jpeg(vipsImg, &length, C.int(p.Quality))
	case "image/png":
		q := math.Min(math.Floor(float64(p.Quality/10)), 9)
		ptr = C.Vips_save_png(vipsImg, &length, C.int(q))
	}

	defer C.free(ptr)

	img.Data = C.GoBytes(ptr, C.int(length))
	img.Size = int64(len(img.Data))

	return img, nil
}

// A map of supported MIME types against their magic numbers.
var fileTypes = map[string][]byte{
	"image/jpeg": []byte{0xff, 0xd8},
	"image/png":  []byte{0x89, 0x50},
	"image/gif":  []byte{0x47, 0x49},
}

// GetFileType detects and returns MIME type for file data in `data`, or returns
// an "application/octet-stream" MIME type if file type could not be detected.
func GetFileType(data []byte) string {
	fileType := "application/octet-stream"

	for t, sig := range fileTypes {
		if bytes.Equal(data[:2], sig) {
			fileType = t
		}
	}

	return fileType
}

// Initialize package variables and set up VIPS library for future processing.
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

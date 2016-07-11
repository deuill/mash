package pipeline

// #cgo pkg-config: vips
// #cgo CFLAGS: -Iinclude
// #cgo LDFLAGS: -lm
//
// #include <stdlib.h>
// #include <vips/vips.h>
//
// #include "pipeline.h"
// #include "resize.h"
import "C"

import (
	// Standard library.
	"fmt"
	"math"
)

// Resize is an operation for manipulating image dimensions, including clipping,
// cropping and focusing within images.
type Resize struct {
	Width  int64 `key:"width"`
	Height int64 `key:"height"`
	Fit    struct {
		Kind string `key:"fit" default:"clip" valid:"crop"`
		Crop struct {
			Gravity string `key:"fit=crop" default:"center" valid:"top|bottom|left|right|point"`
			Point   struct {
				X int64 `key:"fit=crop:point" index:"0"`
				Y int64 `key:"fit=crop:point" index:"1"`
			}
		}
	}
}

// Process applies the pre-defined constraints for this operation onto the image
// provided, changing the data in-place and freeing any additional allocations
// made automatically. Returns an error if processing fails for any reason.
func (r *Resize) Process(img *C.ico_image) error {
	// Do not process image if pipeline requests an identical or enlarged image.
	w, h := int64(C.ico_image_width(img)), int64(C.ico_image_height(img))
	if (r.Width > w || r.Height > h) || (r.Width == w && r.Height == h) {
		return nil
	}

	// Get base resize factor for resulting image.
	factor := r.resizeFactor(img)

	// Shrink image by integer factor, if needed.
	if factor >= 2 {
		if _, err := C.ico_image_shrink(img, C.double(factor)); err != nil {
			return fmt.Errorf("failed to shrink image")
		}

		// Recalculate crop point for shrunk image.
		r.Fit.Crop.Point.X, r.Fit.Crop.Point.Y = r.cropPoint(factor)

		// Recalculate resize factor for shrunk image.
		factor = r.resizeFactor(img)
	}

	// Resize image by remaining factor, if any.
	if factor > 1 {
		if _, err := C.ico_image_affine(img, C.double(factor)); err != nil {
			return fmt.Errorf("failed to affine resize image")
		}

		// Recalculate crop point for resized image.
		r.Fit.Crop.Point.X, r.Fit.Crop.Point.Y = r.cropPoint(factor)
	}

	// Apply specified fit mode
	switch r.Fit.Kind {
	case "crop":
		w, h := int64(C.ico_image_width(img)), int64(C.ico_image_height(img))
		bx, by, bw, bh := r.cropBounds(img)

		// Do not crop image if crop boundaries are same as image size.
		if bx == 0 && by == 0 && bw == w && bh == h {
			break
		}

		_, err := C.ico_image_crop(img, C.int(bx), C.int(by), C.int(bw), C.int(bh))
		if err != nil {
			return fmt.Errorf("failed to crop image")
		}
	}

	return nil
}

// Returns the resize factor (the difference between image size and requested
// final size) as a floating point number. For example, requesting a 500x500
// crop of a 1000x1000 image would return a factor of 2.
func (r *Resize) resizeFactor(img *C.ico_image) float64 {
	var factor float64
	w, h := int64(C.ico_image_width(img)), int64(C.ico_image_height(img))

	// Calculate resize factor based on pipeline parameters.
	switch {
	// Fixed width and height.
	case r.Width > 0 && r.Height > 0:
		xf := float64(w) / float64(r.Width)
		yf := float64(h) / float64(r.Height)

		// We choose the smallest delta when cropping, and the largest when we're not.
		if r.Fit.Kind == "crop" {
			factor = math.Min(xf, yf)
		} else {
			factor = math.Max(xf, yf)
		}
	// Fixed width, auto height.
	case r.Width > 0:
		factor = float64(w) / float64(r.Width)
	// Fixed height, auto width.
	case r.Height > 0:
		factor = float64(h) / float64(r.Height)
	}

	return factor
}

// Returns the pre-defined center of gravity as a pair of X/Y coordinates.
func (r *Resize) cropPoint(factor float64) (int64, int64) {
	x, y := r.Fit.Crop.Point.X, r.Fit.Crop.Point.Y
	return int64(float64(x) / factor), int64(float64(y) / factor)
}

// Returns the boundaries for the area to extract from the provided image.
func (r *Resize) cropBounds(img *C.ico_image) (int64, int64, int64, int64) {
	var x, y int64
	w, h := int64(C.ico_image_width(img)), int64(C.ico_image_height(img))

	// Set crop bounds for specified crop gravity.
	switch r.Fit.Crop.Gravity {
	case "point":
		// Set X and Y coordinates for bounding box, based on the pre-defined
		// center point, and modify the box for image constraints.
		x = ((r.Fit.Crop.Point.X) - (r.Width / 2))
		y = ((r.Fit.Crop.Point.Y) - (r.Height / 2))

		x = int64(math.Min(math.Max(0, float64(x)), float64((w - r.Width))))
		y = int64(math.Min(math.Max(0, float64(y)), float64((h - r.Height))))
	case "left":
		y = (h - r.Height) / 2
	case "right":
		x = w - r.Width
		y = (h - r.Height) / 2
	case "top":
		x = (w - r.Width) / 2
	case "bottom":
		x = (w - r.Width) / 2
		y = h - r.Height
	default:
		x = (w - r.Width) / 2
		y = (h - r.Height) / 2
	}

	return x, y, r.Width, r.Height
}

// NewResize attempts to initialize a resize operation from the parameters
// provided. Width and/or height parameters have to be provided, otherwise the
// resize operation is skipped.
func NewResize(p *Params) (Operation, error) {
	// Instantiate and unpack pipeline parameters into operation.
	r := &Resize{}
	if err := p.Unpack(r); err != nil {
		return nil, err
	}

	// Check for required pipeline parameters.
	if r.Width == 0 && r.Height == 0 {
		return nil, nil
	}

	return r, nil
}

package pipeline

// #cgo pkg-config: vips
// #cgo CFLAGS: -Iinclude
//
// #include <stdlib.h>
// #include <vips/vips.h>
//
// #include "pipeline.h"
import "C"

import (
	// Standard library.
	"fmt"
	"runtime"
	"unsafe"

	// Internal packages.
	"github.com/deuill/mash/service/ico/image"
)

// An Operation represents a set of related image manipulation tasks, e.g.
// resizing cropping. The results of processing an operation against a specific
// image are guaranteed to be deterministic.
type Operation interface {
	Process(*C.ico_image) error
}

// An ordered list of all possible operations in a pipeline.
var operations = []func(*Params) (Operation, error){
	NewResize,
}

// A Pipeline represents all data required for converting an image from its
// original format to the processed result.
type Pipeline struct {
	operations []Operation
}

// Process applies the set of operations defined for the pipeline against the
// provided image data. An error is returned if processing fails at any point,
// otherwise the image provided is modified in-place and nil is returned.
func (p *Pipeline) Process(img *image.Image) error {
	// Initialize internal image representation.
	ptr, err := C.ico_image_new(unsafe.Pointer(&img.Data[0]), C.size_t(img.Size), C.int(img.Type))
	if err != nil {
		return fmt.Errorf("failed to initialize image for pipeline")
	}

	// Apply ordered list of operations in turn.
	for _, op := range p.operations {
		if err = op.Process(ptr); err != nil {
			return err
		}
	}

	// Write internal image representation to buffer.
	var buf unsafe.Pointer
	var len C.size_t

	if _, err = C.ico_image_write(ptr, &buf, &len); err != nil {
		return fmt.Errorf("failed to write to image")
	}

	// Copy internal buffer to byte slice.
	img.Data = C.GoBytes(buf, C.int(len))
	img.Size = int64(len)

	// Clean up references to internal buffers.
	C.ico_image_destroy(ptr)
	C.g_free(buf)

	return nil
}

// New parses the parameter list provided and initializes a Pipeline and
// supporting list of Operations stored within.
func New(params string) (*Pipeline, error) {
	// Initialize and prepare pipeline.
	p := &Pipeline{operations: make([]Operation, 0)}

	// Prepare parameter list for distribution amongst operations.
	prm, err := Parse(params)
	if err != nil {
		return nil, fmt.Errorf("unable to parse parameters: %s", err)
	}

	// Iterate through ordered list of operations, checking for eligibility with
	// regards to the request parameters used. Operations that are to be executed
	// are initialized and appended to the pipeline's list of operations.
	for _, init := range operations {
		op, err := init(prm)
		if err != nil {
			return nil, err
		} else if op == nil {
			// Check for valid operation. Initializing an operation will return a
			// nil result if operation is not applicable for the parameters passed.
			continue
		}

		p.operations = append(p.operations, op)
	}

	return p, nil
}

// Initialize package variables and set up VIPS library for future processing.
func init() {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	if ok := C.ico_init(); ok != 0 {
		panic("Unable to initialize VIPS library")
	}
}

package image

import (
	// Standard library.
	"fmt"
)

// Kind represents the MIME type for an image file.
type Kind int

const (
	JPEG Kind = iota
	PNG
	GIF
)

var kindTypeLookup = map[Kind]string{
	JPEG: "image/jpeg",
	PNG:  "image/png",
	GIF:  "image/gif",
}

// String returns the internal representation of the image Kind as a MIME type.
func (k *Kind) String() string {
	return kindTypeLookup[*k]
}

// Image represents a processed image, and contains the image data as a byte
// slice along with other useful information about the image.
type Image struct {
	Data []byte // The image data buffer
	Size int64  // The image size, in bytes.
	Type Kind   // The image MIME type.
}

// The file signature, used for determining the type of file.
type magicHeader [2]byte

// A lookup table of magic numbers against image file types.
var magicHeaderLookup = map[magicHeader]Kind{
	magicHeader{0xff, 0xd8}: JPEG,
	magicHeader{0x89, 0x50}: PNG,
	magicHeader{0x47, 0x49}: GIF,
}

// New creates a new image representation for the data buffer provided. It returns
// an error if the data buffer is empty or does not correspond to any known image
// type handled by Ico.
func New(data []byte) (*Image, error) {
	// Check for valid image length before processing.
	l := int64(len(data))
	if l < 2 {
		return nil, fmt.Errorf("cannot use data buffer of length '%d' as image", l)
	}

	// Check for valid image MIME type.
	var m magicHeader
	copy(m[:], data[:2])

	if _, ok := magicHeaderLookup[m]; !ok {
		return nil, fmt.Errorf("unknown or unhandled file type for data buffer")
	}

	return &Image{Data: data, Size: l, Type: magicHeaderLookup[m]}, nil
}

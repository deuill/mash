package imager

// A Source represents an image source, which is usually matched against a URL endpoint, and
// provides options related to that endpoint.
type Source struct {
	bucket    string
	accessKey string
	secretKey string
}

func NewSource(bucket, accessKey, secretKey string) *Source {
	s := &Source{
		bucket:    bucket,
		accessKey: accessKey,
		secretKey: secretKey,
	}

	return s
}

package imager

import (
	// Standard library
	"container/list"
	"io/ioutil"
	"os"
	"path"
	"sync"
)

// FileCache implements a simple filesystem-based cache for arbitrary data.
type FileCache struct {
	path  string // The path to the directory in which to place cached files.
	quota int64  // The disk quota size, in bytes. A value of zero means no limit.
	usage int64  // The current disk usage, in bytes.

	order *list.List               // A doubly-linked list of items, ordered by access time.
	cache map[string]*list.Element // A reverse lookup table of item names to list elements.

	sync.RWMutex // Used for controlling concurrent access to item list and cache table.
}

// A file represents all information required for operating on a file in the context of the cache.
type file struct {
	size int64
	key  string
}

// A map of initialized caches, indexed under their path names. This is checked against every time
// a new cache is initialized, and is used to provide exclusivity guarantees for local access.
var caches map[string]*FileCache

// NewFileCache initializes a file cache under a specific path, most commonly a temporary directory,
// with an optional quota on the cache size. If the size of the quota is zero, the limit is assumed
// to be infinite.
func NewFileCache(name string, quota int64) (*FileCache, error) {
	// Check if a cache already exists for this path and return it, if any exists.
	if f, exists := caches[name]; exists {
		// Update quota size for cache, if the new quota size is greater than the existing one.
		if quota == 0 || f.quota > 0 && f.quota < quota {
			f.quota = quota
		}

		return f, nil
	}

	// Remove directory structure first, if any.
	if err := os.RemoveAll(name); err != nil {
		return nil, err
	}

	// Create directory structure for cached files.
	if err := os.MkdirAll(name, 0755); err != nil {
		return nil, err
	}

	caches[name] = &FileCache{
		path:  name,
		quota: quota,
		order: list.New(),
		cache: make(map[string]*list.Element),
	}

	return caches[name], nil
}

// Add inserts in `value` to file pointed to by `key`. Variable `value` is assumed to be a `[]byte`
// type, but is passed as an `interface{}` type to satisfy the generic `Cacher` interface.
func (f *FileCache) Add(key string, value interface{}) {
	var ok bool
	var data []byte
	var el *list.Element

	// Refuse to store non-byte-slice data.
	if data, ok = value.([]byte); !ok {
		return
	}

	// Do not store data whose size is equal to or larger than the quota size.
	if f.quota > 0 && int64(len(data)) >= f.quota {
		return
	}

	f.Lock()
	defer f.Unlock()

	// If entry already exists, move to front and return.
	if el, ok = f.cache[key]; ok {
		f.order.MoveToFront(el)
		return
	}

	// Create path heirarchy for file.
	p := path.Join(f.path, key)
	if err := os.MkdirAll(path.Dir(p), 0755); err != nil {
		return
	}

	// Push file pointer to front of file list.
	el = f.order.PushFront(&file{
		size: int64(len(data)),
		key:  key,
	})

	// If writing the file would bring us above quota, remove oldest files as required.
	// NOTE: If the call to write the data below fails, affected files will STILL be removed.
	for f.quota > 0 && f.usage+el.Value.(*file).size > f.quota {
		f.RemoveOldest()
	}

	if err := ioutil.WriteFile(p, data, 0644); err != nil {
		f.order.Remove(el)
		return
	}

	f.usage += el.Value.(*file).size
	f.cache[key] = el
}

// Get returns data stored under `key`, or `nil` if no data exists.
func (f *FileCache) Get(key string) interface{} {
	f.RLock()
	defer f.RUnlock()

	// Check reverse lookup table for file entry.
	if el, exists := f.cache[key]; exists {
		if buf, err := ioutil.ReadFile(path.Join(f.path, key)); err == nil {
			f.order.MoveToFront(el)
			return buf
		}

		return nil
	}

	return nil
}

// Remove removes file stored under `key`.
func (f *FileCache) Remove(key string) {
	f.Lock()
	defer f.Unlock()

	if el, exists := f.cache[key]; exists {
		f.removeElement(el)
	}
}

// RemoveOldest removes the oldest file in cache, as determined by access time.
func (f *FileCache) RemoveOldest() {
	f.Lock()
	defer f.Unlock()

	if el := f.order.Back(); el != nil {
		f.removeElement(el)
	}
}

// Delete file stored on disk as well as any internal state related to file.
func (f *FileCache) removeElement(el *list.Element) {
	// Remove file and subtract file size from total usage.
	os.Remove(path.Join(f.path, el.Value.(*file).key))
	f.usage -= el.Value.(*file).size

	// Remove internal book-keeping entries.
	delete(f.cache, el.Value.(*file).key)
	f.order.Remove(el)
}

// Initialize common package variables.
func init() {
	caches = make(map[string]*FileCache)
}

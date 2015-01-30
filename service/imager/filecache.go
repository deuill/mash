package imager

import (
	"container/list"
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

	sync.Mutex // Used for controlling concurrent access to item list and cache table.
}

type file struct {
	fp   *os.File
	size int64
	key  string
}

func NewFileCache(path string, size int64) (*FileCache, error) {
	// If directory structure already exists, remove it first, but only if we have a size limit.
	if fi, err := os.Stat(path); err != nil && !os.IsNotExist(err) {
		return nil, err
	} else if fi != nil && fi.IsDir() && size > 0 {
		if err = os.RemoveAll(path); err != nil {
			return nil, err
		}
	}

	// Create directory structure for cached files.
	if err := os.MkdirAll(path, 0755); err != nil {
		return nil, err
	}

	f := &FileCache{
		path:  path,
		quota: size,
		order: list.New(),
		cache: make(map[string]*list.Element),
	}

	return f, nil
}

func (f *FileCache) Add(key string, value interface{}) {
	var ok bool
	var data []byte
	var el *list.Element

	// Refuse to store non-byte-slice data.
	if data, ok = value.([]byte); !ok {
		return
	}

	f.Lock()
	defer f.Unlock()

	// If entry already exists, move to front and return. Otherwise, add a new element in front.
	if el, ok = f.cache[key]; ok {
		f.order.MoveToFront(el)
		return
	} else {
		p := path.Join(f.path, key)
		if err := os.MkdirAll(path.Dir(p), 0755); err != nil {
			return
		}

		fp, err := os.Create(p)
		if err != nil {
			return
		}

		el = f.order.PushFront(&file{
			fp:   fp,
			size: int64(len(data)),
			key:  key,
		})
	}

	// If writing the file would bring us above quota, remove oldest files as required.
	// NOTE: If the call to write the data below fails, any files removed WILL be lost.
	for f.quota > 0 && f.usage+el.Value.(*file).size > f.quota {
		f.RemoveOldest()
	}

	// Write data to file corresponding to key.
	_, err := el.Value.(*file).fp.Write(data)
	if err != nil {
		f.order.Remove(el)
		return
	}

	el.Value.(*file).fp.Seek(0, 0)
	f.usage += el.Value.(*file).size
	f.cache[key] = el
}

func (f *FileCache) Get(key string) interface{} {
	if el, exists := f.cache[key]; exists {
		f.Lock()
		defer f.Unlock()

		f.order.MoveToFront(el)

		buf := make([]byte, el.Value.(*file).size)
		el.Value.(*file).fp.Read(buf)
		el.Value.(*file).fp.Seek(0, 0)

		return buf
	}

	return nil
}

func (f *FileCache) Remove(key string) {
	if el, exists := f.cache[key]; exists {
		f.removeElement(el)
	}
}

func (f *FileCache) RemoveOldest() {
	if el := f.order.Back(); el != nil {
		f.removeElement(el)
	}
}

func (f *FileCache) removeElement(el *list.Element) {
	f.Lock()
	defer f.Unlock()

	// Remove file, close file descriptor and remove filesize from total usage.
	os.Remove(el.Value.(*file).fp.Name())
	el.Value.(*file).fp.Close()
	f.usage -= el.Value.(*file).size

	// Remove internal entries.
	delete(f.cache, el.Value.(*file).key)
	f.order.Remove(el)
}

package imager

import (
	"container/list"
	"os"
	"path"
)

// FileCache implements a simple filesystem-based cache for arbitrary data.
type FileCache struct {
	path  string // The path to the directory in which to place cached files.
	quota int64  // The disk quota size, in bytes. A value of zero means no limit.
	usage int64  // The current disk usage, in bytes.

	order *list.List               // A doubly-linked list of items, ordered by access time.
	cache map[string]*list.Element // A reverse lookup table of item names to list elements.
}

type file struct {
	fp   *os.File
	size int64
	key  string
}

func NewFileCache(path string, quota int64) (*FileCache, error) {
	// Create cache directory if it does not exist.
	fi, _ := os.Stat(path)
	if !fi.IsDir() {
		if err := os.MkdirAll(path, 0755); err != nil {
			return nil, err
		}
	}

	return &FileCache{
		path:  path,
		quota: quota,
		order: list.New(),
		cache: make(map[string]*list.Element),
	}, nil
}

func (f *FileCache) Add(key string, value interface{}) {
	var ok bool
	var data []byte
	var el *list.Element

	// Refuse to store non-byte-slice data.
	if data, ok = value.([]byte); !ok {
		return
	}

	// If entry already exists, move to front and return. Otherwise, add a new element in front.
	if el, ok = f.cache[key]; ok {
		f.order.MoveToFront(el)
		return
	} else {
		fp, err := os.Create(path.Join(f.path, key))
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
	if f.quota > 0 {
		for f.usage+el.Value.(*file).size > f.quota {
			f.RemoveOldest()
		}
	}

	// Write data to file corresponding to key.
	_, err := el.Value.(*file).fp.Write(data)
	if err != nil {
		f.order.Remove(el)
		return
	}

	f.cache[key] = el
	f.usage += el.Value.(*file).size
}

func (f *FileCache) Get(key string) interface{} {
	if el, exists := f.cache[key]; exists {
		f.order.MoveToFront(el)

		data := make([]byte, el.Value.(*file).size)
		el.Value.(*file).fp.Read(data)

		return data
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
	// Remove file, close file descriptor and remove filesize from total usage.
	os.Remove(el.Value.(*file).fp.Name())
	el.Value.(*file).fp.Close()
	f.usage -= el.Value.(*file).size

	// Remove internal entries.
	delete(f.cache, el.Value.(*file).key)
	f.order.Remove(el)
}

package filestore

import (
	"fmt"
	"io"
	"sync"
)

// File group together important attributes for general file store useage
type File struct {
	Name           string
	Store          FileWriter
	IsIntermediate bool
	UseCached      bool

	// cachable cached the cache checked value
	cachable *bool

	lck sync.Mutex
	// file is the internal file storage
	file io.WriteCloser
}

// IsOpen returns if the file has been opened already
func (file File) IsOpen() bool {
	file.lck.Lock()
	defer file.lck.Unlock()
	return file.file != nil
}

// Open returns an io.WriteCloser handle to a file on the store
func (file *File) Open() (err error) {
	if file.Store == nil {
		return fmt.Errorf("filestore is nil")
	}

	file.lck.Lock()
	if file.file != nil {
		file.file.Close()
	}
	file.file, err = file.Store.Writer(file.Name, file.IsIntermediate)
	file.lck.Unlock()
	return err
}

// Write to the file store
func (file File) Write(p []byte) (n int, err error) {
	file.lck.Lock()
	f := file.file
	if f == nil {
		file.lck.Unlock()
		return 0, io.EOF
	}
	n, err = f.Write(p)
	file.lck.Unlock()
	return n, err
}

// Close the file
func (file *File) Close() (err error) {

	file.lck.Lock()
	f := file.file
	file.file = nil
	file.lck.Unlock()

	if f == nil {
		return nil
	}
	return f.Close()
}

// Cached returns if the file.Name is cached on the Store
func (file *File) Cached() bool {
	if file.cachable != nil {
		return *(file.cachable)
	}
	if !file.UseCached {
		file.cachable = &(file.UseCached)
		return false
	}
	// check to see if file.Store even supports caching
	exister, ok := file.Store.(Exister)
	if !ok {
		file.cachable = &ok
		return false
	}
	ok = exister.Exists(file.Name)
	file.cachable = &ok
	return ok
}

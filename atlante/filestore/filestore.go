package filestore

import (
	"io"
	"log"
	"sync"
)

const (
	// ConfigKeyName is the config key name for the name of a filestore entry
	ConfigKeyName = "name"
	// ConfigKeyType is the config key type for the type of a filestore entry
	ConfigKeyType = "type"
)

// FileWriter returns a writer object
type FileWriter interface {
	// Writer should return an io.Writer that can be used to write the file to.
	// If the file should not be written to the filestore, return nil for
	// the io.WriteCloser.
	Writer(filepath string, isIntermediate bool) (io.WriteCloser, error)
}

// Provider returns a filestore that can be used to store files.
type Provider interface {
	// Writer provides a file writer that can be used to write file into the store.
	FileWriter(group string) (FileWriter, error)
}

// globalWaitGroupPipe is used by pipe to keep the process running
// till all the piped writes have had a chance to close and finish
// writing.
var globalWaitGroupPipe sync.WaitGroup

// Pipe creates a pipe that can be use to connect something that requires a io.Reader
func Pipe(typ, name string, fn func(r io.Reader) error) io.WriteCloser {
	r, w := io.Pipe()
	globalWaitGroupPipe.Add(1)
	go func() {
		// TODO(gdey): Need to figure something better for handeling errors here.
		// one idea is to use the CloseWithError() method, to pass the error to
		// the write side of the pipe.
		err := fn(r)
		if err != nil {
			log.Printf("error putting to %v (%v): %v", name, typ, err)
		}
		globalWaitGroupPipe.Done()
	}()
	return w
}

func cleanup() {
	globalWaitGroupPipe.Wait()
}

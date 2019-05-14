package filestore

import (
	"io"

	"github.com/gdey/errors"
)

const (
	// ErrSkipWrite is use to indicate that the file is being skipped.
	ErrSkipWrite = errors.String("skip")
)

// FileWriter returns a writer object
type FileWriter interface {
	// Writer should return an io.Writer that can be used to write the file to.
	// If the file should not be written to the filestore, return nil for
	// the io.WriteCloser, and an error of filestore.ErrSkipWrite
	Writer(filepath string, isIntermediate bool) (io.WriteCloser, error)
}

// Provider returns a filestore that can be used to store files.
type Provider interface {
	// Writer provides a file writer that can be used to write file into the store.
	FileWriter(group string) (FileWriter, error)
}

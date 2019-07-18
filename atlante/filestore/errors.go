package filestore

import "github.com/gdey/errors"

const (
	// ErrUnsupportedOperation is returned when the files store does not support
	// the operation for the fileapth or type.
	ErrUnsupportedOperation = errors.String("unsupported operation")

	// ErrFileDoesNotExist is returned when the file request does not exist
	// usually wrapped by a ErrPath object
	ErrFileDoesNotExist = errors.String("file does not exist")
)

// ErrPath records the error and the operation and file that caused it.
// timeout errors should have a Timeout() bool on it.
type ErrPath struct {
	Filepath       string
	IsIntermediate bool
	FilestoreType  string
	Err            error
}

// Timeout reports if the error represents a timeout
func (err ErrPath) Timeout() bool {
	t, ok := err.Err.(timeout)
	return ok && t.Timeout()
}

func (err ErrPath) Error() string { return err.Err.Error() }

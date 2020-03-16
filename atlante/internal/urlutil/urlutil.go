package urlutil

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/prometheus/common/log"
)

// ErrRemoteFile is an error that was caused when trying to access a remote file.
// the remote location is provided
type ErrRemoteFile struct {
	// Location is the name of the file that was attempted
	Location *url.URL
	Err      error
}

// Error implements the error interface
func (e ErrRemoteFile) Error() string {
	return fmt.Sprintf("error obtaining remote file (%v): %v", e.Location, e.Err)
}

// ErrUnsupportedScheme is returned when the scheme is not understood
type ErrUnsupportedScheme ErrRemoteFile

// Error implements the error interface
func (e ErrUnsupportedScheme) Error() string {
	return fmt.Sprintf("unsupported scheme (%v), for location %v", strings.ToLower(e.Location.Scheme), e.Location)
}

// ErrFile is retuned when an error related to a local file is return
type ErrFile struct {
	// Filename is the name of the file that was attempted
	Filename string
	Err      error
}

// Error implements the error interface
func (e ErrFile) Error() string {
	return fmt.Sprintf("error opening local file (%v): %v", e.Filename, e.Err)
}

// ErrFileNotExists is returned when the requested local file is not found
type ErrFileNotExists ErrFile

// Error implements the error interface
func (e ErrFileNotExists) Error() string {
	return fmt.Sprintf("file at location (%v) not found!", e.Filename)
}

// noCloserReader is a simple wraper to provide a Close method to Readers
// that does nothing, but allows the object to fullfil the ReaderCloser interface.
type noCloserReader struct {
	reader io.Reader
}

// Read implements the Reader interface
func (ncr noCloserReader) Read(p []byte) (int, error) { return ncr.reader.Read(p) }

// Close implements the Close method of the ReaderCloser interface
func (ncr noCloserReader) Close() error { return nil }

// NewReader will create a reader for the appropriate type of file described by
// the url provided.
func NewReader(location *url.URL) (io.ReadCloser, error) {

	if location == nil {
		return nil, errors.New("nil url provided")
	}
	switch strings.ToLower(location.Scheme) {
	case "", "file":

		// check the conf file exists
		filename := location.EscapedPath()
		if _, err := os.Stat(filename); os.IsNotExist(err) {
			wd, err := os.Getwd()
			log.Errorf("Could not find file: %v \n pwd: %v : %v", filename, wd, err)
			return nil, ErrFileNotExists{Filename: filename, Err: err}
		}

		file, err := os.Open(filename)
		if err != nil {
			return nil, ErrFile{Filename: filename, Err: err}
		}
		return file, nil

	case "http", "https":

		var httpClient = &http.Client{
			Timeout: 10 * time.Second,
		}

		res, err := httpClient.Get(location.String())
		if err != nil {
			return nil, ErrRemoteFile{
				Location: location,
				Err:      err,
			}
		}
		return noCloserReader{reader: res.Body}, nil

	default:

		return nil, ErrUnsupportedScheme{Location: location}

	}

}

// IsRemote returns if the url points to a remote resource
func IsRemote(location *url.URL) bool {
	if location == nil {
		return false
	}
	loc := strings.ToLower(location.Scheme)
	switch loc {
	case "", "file":
		return false
	default:
		return true
	}
}

// ReadAll will return all bytes from the url location
func ReadAll(location *url.URL) (b []byte, err error) {
	if location == nil {
		return nil, errors.New("nil url provided")
	}
	r, err := NewReader(location)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return ioutil.ReadAll(r)
}

// VisitReader will call the given function with an initialize io.Reader
func VisitReader(location *url.URL, fn func(io.Reader) error) error {
	if location == nil {
		return errors.New("nil url provided")
	}

	r, err := NewReader(location)
	if err != nil {
		return err
	}
	defer r.Close()
	return fn(r)
}

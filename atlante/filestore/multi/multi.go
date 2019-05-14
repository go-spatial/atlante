package multi

import (
	"fmt"
	"io"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante/filestore"
)

const (
	// TYPE is the name of the provider
	TYPE = "multi"

	// ConfigKeyFileStore is a config list of previously declared file stores
	ConfigKeyFileStore = "filestore"
)

// ErrUnknownFileStore is returned when a unknown file store is referenced
type ErrUnknownFileStore string

func (err ErrUnknownFileStore) Error() string {
	return fmt.Sprintf("error unknown filestore %v", string(err))

}

func initFunc(cfg filestore.Config) (filestore.Provider, error) {
	filestores, err := cfg.StringSlice(ConfigKeyFileStore)
	if err != nil {
		return nil, errors.Wrapf(err, "error for %v expected list of filestore providers", ConfigKeyFileStore)
	}
	var provider Provider

	// Go through each of the filestores and get them from the config.
	for _, fs := range filestores {
		p, err := cfg.ProviderFor(fs)
		if err != nil {
			return nil, ErrUnknownFileStore(fs)
		}
		if p == nil {
			continue
		}
		// Flatten other multi.Providers
		if mp, ok := p.(*Provider); ok {
			provider.providers = append(provider.providers, mp.providers...)
			continue
		}
		provider.providers = append(provider.providers, p)
	}
	return provider, nil
}

// New returns a new multi provider
func New(providers ...filestore.Provider) (provider Provider) {
	for _, p := range providers {
		if p == nil {
			continue
		}
		// Flatten other multi.Providers
		if mp, ok := p.(*Provider); ok {
			provider.providers = append(provider.providers, mp.providers...)
			continue
		}
		provider.providers = append(provider.providers, p)
	}
	return provider
}

func init() {
	filestore.Register(TYPE, initFunc, nil)
}

// Writer creates a writer that duplicates its writes to all the
// provided writers, similar to the Unix tee(1) command.
//
// Each write is written to each listed writer, one at a time.
// It a listed writer returns an error (that is not filestore.ErrSkipWrite),
// that overall write operation stops and returns the error; it does not continue
// down the list.
//
// This is heavily influnced by io.MultiWriter
type Writer struct {
	writers []io.WriteCloser
}

// Write implements the io.Writer interface
func (t *Writer) Write(p []byte) (n int, err error) {
	for _, w := range t.writers {
		n, err = w.Write(p)
		if err != nil {

			if err == filestore.ErrSkipWrite {
				continue
			}
			return n, err
		}
		if n != len(p) {
			return n, io.ErrShortWrite
		}
	}
	return len(p), nil
}

// Close implements the io.Closer interface
func (t *Writer) Close() error {
	for _, w := range t.writers {
		w.Close()
	}
	return nil
}

// Provider duplexes writes to multiple other filestore providers
type Provider struct {
	providers []filestore.Provider
}

// FileWriter implements the filestore.Provider interface
// Each call to the FileWriter will be sent to each listed provider to
// obtain it's FileWriter. If there is an error other then
// filestore.ErrSkipWrite, then the operation will stop and return that error; it
// does not continue down the list.
func (t Provider) FileWriter(grp string) (filestore.FileWriter, error) {
	var filewriter FileWriter
	for _, p := range t.providers {
		fw, err := p.FileWriter(grp)
		if err != nil {
			if err == filestore.ErrSkipWrite {
				continue
			}
			return nil, err
		}
		filewriter.writers = append(filewriter.writers, fw)
	}
	if len(filewriter.writers) == 0 {
		return nil, filestore.ErrSkipWrite
	}
	return filewriter, nil
}

// FileWriter returns writers that can write files to various locations
type FileWriter struct {
	writers []filestore.FileWriter
}

//Writer implements the filestore.FileWriter interface
func (t FileWriter) Writer(fpath string, isIntermediate bool) (io.WriteCloser, error) {
	var writer Writer
	for _, fw := range t.writers {
		w, err := fw.Writer(fpath, isIntermediate)
		if err != nil {
			if err == filestore.ErrSkipWrite {
				continue
			}
			return nil, err
		}
		writer.writers = append(writer.writers, w)
	}
	// No writers, no need to write this file.
	if len(writer.writers) == 0 {
		return nil, filestore.ErrSkipWrite
	}
	return &writer, nil
}

var _ io.WriteCloser = (*Writer)(nil)

package multi

import (
	"io"
	"net/url"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante/filestore"
)

const (
	// TYPE is the name of the provider
	TYPE = "multi"

	// ConfigKeyFileStore is a config list of previously declared file stores
	ConfigKeyFileStore = "file_stores"

	// ErrZeroFilestoresConfigured is returned when a multi filestore does not have
	// any filestore configured.
	ErrZeroFilestoresConfigured = errors.String("zero filestores configured")
)

func initFunc(cfg filestore.Config) (filestore.Provider, error) {
	filestores, err := cfg.StringSlice(ConfigKeyFileStore)
	if err != nil {
		return nil, errors.Wrapf(err, "error for %v expected list of filestore providers", ConfigKeyFileStore)
	}
	var provider Provider

	// Go through each of the filestores and get them from the config.
	for _, fs := range filestores {
		p, err := cfg.FileStoreFor(fs)
		if err != nil {
			return nil, filestore.ErrUnknownProvider(fs)
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
	switch len(provider.providers) {
	case 0:
		// There are two ways this can happen.
		// 1. filestores is zero (most common)
		// 2. we have a bunch of nil's in the FileStoreFor()
		return nil, ErrZeroFilestoresConfigured
	case 1:
		// If there is only one provider just return that provider,
		// no need to run through the multi provider.
		return provider.providers[0], nil
	default:
		return provider, nil
	}
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
func (p Provider) FileWriter(grp string) (filestore.FileWriter, error) {
	var filewriter FileWriter
	for _, p := range p.providers {
		fw, err := p.FileWriter(grp)
		if err != nil {
			return nil, err
		}
		if fw == nil {
			continue
		}
		filewriter.Writers = append(filewriter.Writers, fw)
	}
	if len(filewriter.Writers) == 0 {
		return nil, nil
	}
	return filewriter, nil
}

// FileWriter returns writers that can write files to various locations
type FileWriter struct {
	Writers []filestore.FileWriter
}

//Writer implements the filestore.FileWriter interface
func (t FileWriter) Writer(fpath string, isIntermediate bool) (io.WriteCloser, error) {
	var writer Writer
	for _, fw := range t.Writers {
		w, err := fw.Writer(fpath, isIntermediate)
		if err != nil {
			return nil, err
		}
		if w == nil {
			continue
		}
		writer.writers = append(writer.writers, w)
	}
	// No writers, no need to write this file.
	if len(writer.writers) == 0 {
		return nil, nil
	}
	return &writer, nil
}

// PathURL will go through each of the filestore looking for the first filestore that
// support the Pather interface and has the file and returns that url
func (p Provider) PathURL(group string, filepath string, isIntermediate bool) (*url.URL, error) {
	var firstError error
	// Search through our filestores and find the first one that supportes the
	// pather interface and has a url for it.
	for _, fs := range p.providers {
		pather, ok := fs.(filestore.Pather)
		if !ok {
			continue
		}
		// Try and get url.
		furl, err := pather.PathURL(group, filepath, isIntermediate)
		if err != nil {
			if err == filestore.ErrUnsupportedOperation {
				continue
			}
			if firstError != nil {
				firstError = err
			}
			continue
		}
		if furl == nil {
			continue
		}
		return furl, nil
	}
	if firstError != nil {
		return nil, firstError
	}
	return nil, filestore.ErrUnsupportedOperation
}

var _ filestore.Provider = Provider{}
var _ filestore.Pather = Provider{}

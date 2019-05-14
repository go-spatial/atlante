package null

import (
	"io"
	"log"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
)

const (
	// TYPE is the name of the provider
	TYPE = "null"

	// ConfigKeyLog is the key used in the config to select a logging null filestore
	ConfigKeyLog = "log"
	// ConfigKeyIntermediate is the key used in the config to select logging of all
	// file types -- only valid if Log is enabled as well.
	ConfigKeyIntermediate = "intermediate"
)

func initFunc(cfg filestore.Config) (filestore.Provider, error) {
	if log, _ := cfg.Bool(ConfigKeyLog, nil); !log {
		return Provider{}, nil
	}
	intermediate, _ := cfg.Bool(ConfigKeyIntermediate, nil)
	return LogProvider{
		intermediate: intermediate,
	}, nil
}

func init() {
	filestore.Register(TYPE, initFunc, nil)
}

// Writer is a null writer
type Writer struct{}

// Write implements io.Writer
func (Writer) Write(p []byte) (int, error) { return len(p), nil }

// Close implements io.Closer
func (Writer) Close() error { return nil }

// Provider provides a filestore that throws away any file written to it.
type Provider struct{}

// Writer implements the filestore.FileWriter interface
func (p Provider) Writer(string, bool) (io.WriteCloser, error) { return Writer{}, nil }

// FileWriter implements to the filestore.Provider interface
func (p Provider) FileWriter(string) (filestore.FileWriter, error) { return p, nil }

// LogProvider provides a filestore that throws away any file written to it, but logs to stderr
// the files it's throwing away
type LogProvider struct {
	intermediate bool
}
type writer struct {
	grp          string
	intermediate bool
}

// FileWriter confirms to the filestore.Provider interface
func (l LogProvider) FileWriter(grp string) (filestore.FileWriter, error) {
	return writer{
		grp:          grp,
		intermediate: l.intermediate,
	}, nil
}

// Writer confirms to the filestore.FileWriter interface
func (l writer) Writer(filepath string, isIntermediate bool) (io.WriteCloser, error) {
	if !l.intermediate && isIntermediate {
		return nil, filestore.ErrSkipWrite
	}
	log.Printf("%v would write: %v\n", l.grp, filepath)
	return nil, filestore.ErrSkipWrite
}

// make sure we are always adhering to the interface.
var (
	_ = filestore.Provider(Provider{})
	_ = filestore.Provider(LogProvider{})

	_ = filestore.FileWriter(Provider{})
	_ = filestore.FileWriter(writer{})
)

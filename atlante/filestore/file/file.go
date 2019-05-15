package file

import (
	"io"
	"os"
	"path/filepath"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante/filestore"
)

const (
	// TYPE is the name of the provider
	TYPE = "file"

	// ConfigKeyBasepath is the base directory where the file will be placed.
	ConfigKeyBasepath = "base_path"
	// ConfigKeyGroup indicates weather we should group assets in a subdirectory
	// based on the group name (This is will be the mgdid)
	ConfigKeyGroup = "group"
	// ConfigKeyIntermediate is the key used to tell the system to write out the intermediate
	// files as well.
	ConfigKeyIntermediate = "intermediate"

	// ErrMissingBasePath is returned when the configured value for the base path is missing.
	ErrMissingBasePath = errors.String("error " + ConfigKeyBasepath + " missing value")
)

func intiFunc(cfg filestore.Config) (filestore.Provider, error) {
	basepath, err := cfg.String(ConfigKeyBasepath, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "error invalid for config key: %v", ConfigKeyBasepath)
	}
	if basepath == "" {
		return nil, ErrMissingBasePath
	}
	basepath = filepath.Clean(basepath)
	if basepath != "." {
		if err = os.MkdirAll(basepath, os.ModePerm); err != nil {
			return nil, errors.Wrapf(err, "error failed to write to %v", basepath)
		}
	}

	grp, _ := cfg.Bool(ConfigKeyGroup, nil)
	intermediate, _ := cfg.Bool(ConfigKeyIntermediate, nil)

	return Provider{
		Base:         basepath,
		Group:        grp,
		Intermediate: intermediate,
	}, nil
}

func init() {
	filestore.Register(TYPE, intiFunc, nil)
}

// Provider provides a filestore that write to the local file system.
type Provider struct {
	Base         string
	Group        bool
	Intermediate bool
}

// FileWriter implements the filestore.Provider interface
func (p Provider) FileWriter(grp string) (filestore.FileWriter, error) {
	base := p.Base
	if p.Group {
		// We are going to need to group things.
		// Let's append the grp to end of the base to make a new base
		// path.
		base = filepath.Join(base, grp)
		base = filepath.Clean(base)
		if err := os.MkdirAll(base, os.ModePerm); err != nil {
			return nil, errors.Wrapf(err, "error failed to write to %v", base)
		}
	}
	// Let's make sure we can write to the group
	return Writer{
		Base:         base,
		Intermediate: p.Intermediate,
	}, nil
}

// Writer writes the given file to the location
type Writer struct {
	Base         string
	Intermediate bool
}

// Writer implements the filestore.FileWriter interface
func (w Writer) Writer(fpath string, isIntermediate bool) (io.WriteCloser, error) {
	// If we are not writing out intermediate file, skip.
	if !w.Intermediate && isIntermediate {
		return nil, filestore.ErrSkipWrite
	}
	// We are writing this file. First thing to do is
	// combine the file path with the base path.
	path := filepath.Join(w.Base, fpath)
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return nil, errors.Wrapf(err, "error failed create base dir %v", dir)
	}
	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrapf(err, "error failed to create file %v", path)
	}
	return f, nil
}

// make sure we are always adhering to the interface.
var (
	_ = filestore.Provider(Provider{})
	_ = filestore.FileWriter(Writer{})
)

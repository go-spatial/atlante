package s3

import (
	"bytes"
	"fmt"
	"io"
	"path/filepath"

	cfgaws "github.com/go-spatial/maptoolkit/atlante/config/aws"

	"github.com/aws/aws-sdk-go/service/s3/s3manager"

	"github.com/go-spatial/maptoolkit/atlante/filestore"
)

const (

	// TYPE is the name of the provider
	TYPE = "s3"

	// ConfigKeyGroup is the key used in the config to indicate if the files should be grouped. [optional]
	ConfigKeyGroup = "group"

	// ConfigKeyBucket is the key used in the config set the bucket [required]
	ConfigKeyBucket = "bucket"

	// ConfigKeyBasePath is the key used for the base path. [optional] defaults to ""
	ConfigKeyBasePath = "base_path"

	// ConfigKeyIntermediate is the key used to indicate if intermediate files should be copied. [optional]
	ConfigKeyIntermediate = "intermediate"

	// ConfigKeyIntermediateBucket is the key used to state a different bucket for intermediate files. [optional]
	ConfigKeyIntermediateBucket = "intermediate_bucket"

	// ConfigKeyIntermediateBasePath is the key used for a different base path for the intermediate files. [optional]
	ConfigKeyIntermediateBasePath = "intermediate_base_path"
)

var (
	// testData is used during New() to confirm the ability to write, read and purge the cache
	testData = []byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x2a, 0xce, 0xcc, 0x49, 0x2c, 0x6, 0x4, 0x0, 0x0, 0xff, 0xff, 0xaf, 0x9d, 0x59, 0xca, 0x5, 0x0, 0x0, 0x0}
)

func initFunc(cfg filestore.Config) (filestore.Provider, error) {

	name, _ := cfg.String(filestore.ConfigKeyName, nil)
	bucket, err := cfg.String(ConfigKeyBucket, nil)
	if err != nil {
		return nil, err
	}
	if bucket == "" {
		return nil, fmt.Errorf("bucket is required")
	}
	basepath, _ := cfg.String(ConfigKeyBasePath, nil)
	intermediate, _ := cfg.Bool(ConfigKeyIntermediate, nil)
	intermediateBucket, _ := cfg.String(ConfigKeyIntermediateBucket, &bucket)
	intermediateBasePath, _ := cfg.String(ConfigKeyIntermediateBasePath, &basepath)
	group, _ := cfg.Bool(ConfigKeyGroup, nil)

	sess, err := cfgaws.NewSession(cfg)
	if err != nil {
		return nil, err
	}

	uploader := s3manager.NewUploader(sess)

	p := &Provider{
		name:                 name,
		bucket:               bucket,
		basepath:             basepath,
		group:                group,
		intermediate:         intermediate,
		intermediateBucket:   intermediateBucket,
		intermediateBasePath: intermediateBasePath,
		uploader:             uploader,
	}

	testPath := filepath.Join(p.basePath("upload_test"), "testdata")
	testObj := &s3manager.UploadInput{
		Body:   bytes.NewReader(testData),
		Bucket: &bucket,
		Key:    &testPath,
	}
	_, err = p.uploader.Upload(testObj)
	if err != nil {
		return nil, err
	}

	return p, nil

}

func init() {
	filestore.Register(TYPE, initFunc, nil)
}

// Provider provides a filestore that can write to s3 object stores
type Provider struct {
	name                 string
	bucket               string
	basepath             string
	group                bool
	intermediate         bool
	intermediateBucket   string
	intermediateBasePath string
	uploader             *s3manager.Uploader
}

// FileWriter implements the filestore.Provider interface
func (p *Provider) FileWriter(grp string) (filestore.FileWriter, error) {
	return Writer{
		name:         p.name,
		bucket:       p.bucket,
		ibucket:      p.intermediateBucket,
		bpath:        p.basePath(grp),
		ibpath:       p.iBasePath(grp),
		intermediate: p.intermediate,
		uploader:     p.uploader,
	}, nil
}

func (p Provider) basePath(grp string) string {
	if p.group {
		return filepath.Join(p.basepath, grp)
	}
	return p.basepath
}

func (p Provider) iBasePath(grp string) string {
	bp := p.intermediateBasePath
	if p.intermediateBasePath == "" {
		bp = p.basepath
	}
	if p.group {
		return filepath.Join(bp, grp)
	}
	return bp
}

// Writer is a s3 writer
type Writer struct {
	name         string
	bucket       string
	bpath        string
	intermediate bool
	ibucket      string
	ibpath       string
	uploader     *s3manager.Uploader
}

func (wrt Writer) bucketPath(p string, intermediate bool) (string, string) {
	if intermediate {
		return wrt.ibucket, filepath.Join(wrt.ibpath, p)
	}
	return wrt.bucket, filepath.Join(wrt.bpath, p)
}

// PutObject will create an s3 object based on the params and put the object.
func (wrt Writer) PutObject(bucket, fpath string, r io.Reader) error {
	obj := &s3manager.UploadInput{
		Body:   r,
		Bucket: &bucket,
		Key:    &fpath,
	}
	_, err := wrt.uploader.Upload(obj)
	return err
}

// Writer implements the filestore.Writer
func (wrt Writer) Writer(fpath string, isIntermediate bool) (io.WriteCloser, error) {
	// Not interested in intermediate files
	if isIntermediate && !wrt.intermediate {
		return nil, nil
	}
	bucket, path := wrt.bucketPath(fpath, isIntermediate)
	return filestore.Pipe(TYPE, wrt.name, func(r io.Reader) error {
		return wrt.PutObject(bucket, path, r)
	}), nil
}

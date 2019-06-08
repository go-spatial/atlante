package s3

import (
	"bytes"
	"fmt"
	"io"
	"net/url"
	"path/filepath"
	"time"

	"github.com/gdey/errors"
	cfgaws "github.com/go-spatial/maptoolkit/atlante/config/aws"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
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

	// ConfigKeyURLTimeout is the timeout for pre-signed urls from AWS s3
	ConfigKeyURLTimeout = "url_timeout"

	// ConfigKeyGenPresigned is a key to tell the system to let the s3 provider
	// generate the presigned urls
	ConfigKeyGenPresigned = "generate_presigned_urls"
)

var (
	// testData is used during New() to confirm the ability to write, read and purge the cache
	testData = []byte{0x1f, 0x8b, 0x8, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0xff, 0x2a, 0xce, 0xcc, 0x49, 0x2c, 0x6, 0x4, 0x0, 0x0, 0xff, 0xff, 0xaf, 0x9d, 0x59, 0xca, 0x5, 0x0, 0x0, 0x0}

	// DefaultURLTimeout is going to be 10 minutes
	DefaultURLTimeout = 10
	falseValue        = false
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
	urlTimeout, _ := cfg.Int(ConfigKeyURLTimeout, &DefaultURLTimeout)
	genPresigned, _ := cfg.Bool(ConfigKeyGenPresigned, &falseValue)
	group, _ := cfg.Bool(ConfigKeyGroup, nil)

	sess, err := cfgaws.NewSession(cfg)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		name:                 name,
		bucket:               bucket,
		basepath:             basepath,
		group:                group,
		intermediate:         intermediate,
		intermediateBucket:   intermediateBucket,
		intermediateBasePath: intermediateBasePath,
		uploader:             s3manager.NewUploader(sess),
		urlTimeout:           time.Duration(urlTimeout) * time.Minute,
	}

	if genPresigned {
		p.s3 = s3.New(sess)
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

	// used for getting signed urls
	s3 *s3.S3
	// time for urls that are generated if zero, then default is used.
	urlTimeout time.Duration
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

func (p Provider) bucketPath(group, filepth string, isIntermediate bool) (bucket, path string) {
	if isIntermediate {
		return p.bucket, filepath.Join(p.iBasePath(group), filepth)
	}
	return p.intermediateBucket, filepath.Join(p.basePath(group), filepth)
}

// PathURL will get a pre-signed URL from aws for supported files.
func (p Provider) PathURL(group string, filepth string, isIntermediate bool) (*url.URL, error) {
	if p.s3 == nil {
		return nil, filestore.ErrUnsupportedOperation
	}
	// We don't support intermediate files for this operation
	if !p.intermediate && isIntermediate {
		return nil, filestore.ErrUnsupportedOperation
	}
	bucket, key := p.bucketPath(group, filepth, isIntermediate)
	abucket, akey := aws.String(bucket), aws.String(key)

	// Check to see if the key exists.
	headObjInput := &s3.HeadObjectInput{
		Bucket: abucket,
		Key:    akey,
	}
	if _, err := p.s3.HeadObject(headObjInput); err != nil {
		return nil, filestore.ErrPath{
			Filepath:       filepth,
			IsIntermediate: isIntermediate,
			FilestoreType:  TYPE,
			Err:            errors.String("file does not exist"),
		}
	}

	req, _ := p.s3.GetObjectRequest(&s3.GetObjectInput{
		Bucket: abucket,
		Key:    akey,
	})
	urlStr, err := req.Presign(p.urlTimeout)
	if err != nil {
		return nil, err
	}
	return url.Parse(urlStr)
}

var _ filestore.Pather = Provider{}

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

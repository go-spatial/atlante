//go:generate protoc "--go_out=paths=source_relative:." "job.proto"

package atlante

import (
	"encoding/base64"

	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/gogo/protobuf/proto"
	"github.com/pkg/errors"
)

type Notifier interface {
	Notify(map[string]string) error
}

// NewJob returns a new job object for the given sheet, grid and metadata
func NewJob(sheet string, cell *grids.Cell, metadata map[string]string) *Job {
	return &Job{
		SheetName: sheet,
		Cell:      cell,
		MetaData:  metadata,
	}
}

// Base64Marshal returns the job encode in a based64 string
func (j *Job) Base64Marshal() (string, error) {
	// first marshal to pbf
	data, err := proto.Marshal(j)
	if err != nil {
		return "", errors.Wrap(err, "failed to marshal")
	}

	// Now marshal the []byte to base64
	return base64.StdEncoding.EncodeToString(data), nil
}

// Base64UnmarshalJob will return a Job object for the encode job string
func Base64UnmarshalJob(str string) (*Job, error) {
	data, err := base64.StdEncoding.DecodeString(str)
	if err != nil {
		return nil, errors.Wrap(err, "failed to base64 decode")
	}

	var jb Job
	if err := proto.Unmarshal(data, &jb); err != nil {
		return nil, errors.Wrap(err, "failed to unmarshal protobuf")
	}

	return &jb, nil
}

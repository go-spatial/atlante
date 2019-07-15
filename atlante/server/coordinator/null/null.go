package null

import (
	"fmt"

	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
)

const (
	// TYPE is the name of the provider
	TYPE = "null"
)

func initFunc(cfg coordinator.Config) (coordinator.Provider, error) { return &Provider{}, nil }

func init() {
	coordinator.Register(TYPE, initFunc, nil)
}

type Provider struct{}

func (Provider) NewJob(job *atlante.Job) (jb *coordinator.Job, err error) {
	if job == nil {
		return nil, coordinator.ErrNilAtlanteJob
	}
	jbID := fmt.Sprintf("%v:%v", job.SheetName, job.Cell.Mdgid.AsString())
	return coordinator.NewJob(jbID, job), nil
}

func (Provider) UpdateField(job *coordinator.Job, fields ...field.Value) error {
	return nil
}

func (Provider) FindByJob(job *atlante.Job) (jb *coordinator.Job, found bool) {
	return nil, false
}

func (Provider) FindByJobID(jobid string) (jb *coordinator.Job, found bool) {
	return nil, false
}
func (Provider) Jobs(uint) ([]*coordinator.Job, error) {
	return nil, nil
}

var _ = coordinator.Provider(Provider{})

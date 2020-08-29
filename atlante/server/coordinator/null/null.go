package null

import (
	"fmt"

	"github.com/go-spatial/atlante/atlante"
	"github.com/go-spatial/atlante/atlante/server/coordinator"
	"github.com/go-spatial/atlante/atlante/server/coordinator/field"
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

func (Provider) UpdateField(*coordinator.Job, ...field.Value) error {
	return nil
}

func (Provider) FindByJob(*atlante.Job, string) (jobs []*coordinator.Job) {
	return nil
}

func (Provider) FindByJobID(string) (jb *coordinator.Job, found bool) {
	return nil, false
}
func (Provider) Jobs(uint) ([]*coordinator.Job, error) {
	return nil, nil
}

var _ = coordinator.Provider(Provider{})

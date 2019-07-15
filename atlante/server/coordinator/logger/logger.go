package logger

import (
	"fmt"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"
	"github.com/go-spatial/tegola/dict"
	"github.com/prometheus/common/log"
)

const (
	// TYPE is the name of the provider
	TYPE = "logger"

	// ConfigKeyProvider is the key used in the config to select the Name of the provider to proxy calls to.
	ConfigKeyProvider = "provider"
)

func initFunc(cfg coordinator.Config) (coordinator.Provider, error) {
	var (
		pConfig dict.Dicter
		err     error
	)
	pConfig, err = cfg.Map(ConfigKeyProvider)
	if err != nil {
		return &Provider{}, err
	}

	subProvider, err := coordinator.From(pConfig)
	if err != nil {
		return nil, err
	}
	log.Infof("initalizing log coordinator with: %T ", subProvider)
	return &Provider{
		Provider: subProvider,
	}, nil
}

func init() {
	coordinator.Register(TYPE, initFunc, nil)
}

type Provider struct {
	Provider coordinator.Provider
}

func (p *Provider) NewJob(job *atlante.Job) (jb *coordinator.Job, err error) {
	if job == nil {
		log.Infof("job is nil")
		return nil, coordinator.ErrNilAtlanteJob
	}
	jbID := fmt.Sprintf("%v:%v", job.SheetName, job.Cell.Mdgid.AsString())
	log.Infof("created a new jobID: %v", jbID)
	if p == nil || p.Provider != nil {
		return p.Provider.NewJob(job)
	}
	return coordinator.NewJob(jbID, job), nil
}

func (p *Provider) UpdateField(job *coordinator.Job, fields ...field.Value) error {
	if job == nil {
		log.Infof("job is nil")
		return coordinator.ErrNilJob
	}

	log.Infof("update fields in job: %v", job.JobID)
	for i, f := range fields {
		switch fld := f.(type) {
		case field.QJobID:
			log.Infof("update q job id to: %v", string(fld))
		case field.Status:
			switch status := fld.Status.(type) {
			case field.Requested:
				log.Infof("update status to requested")
			case field.Started:
				log.Infof("update status to started")
			case field.Processing:
				log.Infof("update status to processing %v", status.Description)
			case field.Failed:
				log.Infof("update status to failed - reason %v", status.Error)
			default:
				log.Infof("unknown status: %t", status)
			}
		default:
			log.Infof("unkown field[%v] %t", i, fld)
			return errors.String("unknown field type")
		}
	}
	if p == nil || p.Provider != nil {
		return p.Provider.UpdateField(job, fields...)
	}
	return nil
}

func (p *Provider) FindByJob(job *atlante.Job) (jobs []*coordinator.Job) {
	if job == nil {
		log.Infof("job is nil")
		return nil
	}
	log.Infof("looking for job via sheet: %v mdgid: %v ", job.SheetName, job.Cell.Mdgid.AsString())
	if p == nil || p.Provider != nil {
		return p.Provider.FindByJob(job)
	}
	return nil
}

func (p *Provider) FindByJobID(jobid string) (jb *coordinator.Job, found bool) {
	log.Infof("looking for job : %v ", jobid)
	if p == nil || p.Provider != nil {
		return p.Provider.FindByJobID(jobid)
	}
	return nil, false
}

func (p *Provider) Jobs(uint) ([]*coordinator.Job, error) {
	log.Infof("getting all jobs")
	return nil, nil
}

var _ = coordinator.Provider(&Provider{})

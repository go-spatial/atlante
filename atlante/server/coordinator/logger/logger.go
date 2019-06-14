package logger

import (
	"fmt"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/server/coordinator"
	"github.com/prometheus/common/log"
)

const (
	// TYPE is the name of the provider
	TYPE = "logger"

	// ConfigKeyProvider is the key used in the config to select the Name of the provider to proxy calls to.
	ConfigKeyProvider = "provider"
)

func initFunc(cfg coordinator.Config) (coordinator.Provider, error) {
	var pName string
	if pName, _ = cfg.String(ConfigKeyProvider, &pName); pName == "" {
		return &Provider{}, nil
	}
	subProvider, err := cfg.CoordinatorFor(pName)
	if err != nil {
		return nil, err
	}
	log.Infof("initalizing log coordinator with: %v ", pName)
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
	return coordinator.NewJob(jbID, job.Cell.Mdgid), nil
}

func (p *Provider) UpdateField(job *coordinator.Job, fields ...coordinator.FieldValue) error {
	if job == nil {
		log.Infof("job is nil")
		return coordinator.ErrNilJob
	}

	log.Infof("update fields in job: %v", job.JobID)
	for i, f := range fields {
		switch field := f.(type) {
		case coordinator.FieldQJobID:
			log.Infof("update q job id to: %v", string(field))
		case coordinator.FieldStatus:
			if field.Description != "" {
				log.Infof("update status to: %s —— %s", field.Status, field.Description)
			} else {
				log.Infof("update status to: %s", field.Status)
			}
		default:
			log.Infof("unkown field[%v] %t", i, field)
			return errors.String("unknown field type")
		}
	}
	if p == nil || p.Provider != nil {
		return p.Provider.UpdateField(job, fields...)
	}
	return nil
}

func (p *Provider) FindJob(job *atlante.Job) (jb *coordinator.Job, found bool) {
	if job == nil {
		log.Infof("job is nil")
		return nil, false
	}
	log.Infof("Looking for job via sheet: %v mdgid: %v ", job.SheetName, job.Cell.Mdgid.AsString())
	if p == nil || p.Provider != nil {
		return p.Provider.FindJob(job)
	}
	return nil, false
}

func (p *Provider) FindJobID(jobid string) (jb *coordinator.Job, err error) {
	log.Infof("Looking for job : %v ", jobid)
	if p == nil || p.Provider != nil {
		return p.Provider.FindJobID(jobid)
	}
	return nil, nil
}

var _ = coordinator.Provider(&Provider{})

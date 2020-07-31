// Package screen outputs the changes in job state to the screen.
// All output is currently at the info level of the logger.
// This module is most useful for debugging
package screen

import (
	"github.com/gdey/errors"
	"github.com/go-spatial/atlante/atlante/notifiers"
	"github.com/go-spatial/atlante/atlante/server/coordinator/field"
	"github.com/prometheus/common/log"
)

const (
	// TYPE of the notifier
	TYPE = "screen"
)

func initFunc(cfg notifiers.Config) (notifiers.Provider, error) {
	return &Provider{}, nil
}

func init() {
	notifiers.Register(TYPE, initFunc, nil)
}

// Provider supports the notifier Provider interface
type Provider struct{}

// NewEmitter returns a new emitter for the job id
func (*Provider) NewEmitter(jobid string) (notifiers.Emitter, error) {
	return &emitter{
		jobid:  jobid,
		logger: log.Base().With("job-id", jobid),
	}, nil
}

// Emitter support the notifier Emitter interface
type emitter struct {
	jobid  string
	logger log.Logger
}

// Emit notifiers the screen of the status change for the configured job
func (e *emitter) Emit(se field.StatusEnum) error {
	if e == nil {
		return errors.String("emitter is nil")
	}
	logger := e.logger
	switch s := se.(type) {
	case field.Requested:
		logger.Infoln("job requested")
	case field.Started:
		logger.Infoln("job started")
	case field.Processing:
		logger.Infof("job processing : %v", s.Description)
	case field.Failed:
		logger.Infof("job failed: %v , err: %v", s.Description, s.Error)
	case field.Completed:
		logger.Infof("job compleated")
	}
	return nil
}

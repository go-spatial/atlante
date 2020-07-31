package notifiers

import "github.com/go-spatial/atlante/atlante/server/coordinator/field"

// Emitter emits the status of the job to the notifier
type Emitter interface {
	Emit(field.StatusEnum) error
}

// Provider creates a new Emitter for a given jobid
type Provider interface {
	NewEmitter(jobid string) (Emitter, error)
}

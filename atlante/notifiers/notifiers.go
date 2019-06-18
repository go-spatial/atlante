package notifiers

import "github.com/go-spatial/maptoolkit/atlante/server/coordinator/field"

type Emitter interface {
	Emit(field.StatusEnum) error
}

type Provider interface {
	NewEmitter(jobid string) (Emitter, error)
}

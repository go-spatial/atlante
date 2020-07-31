package queuer

import (
	"github.com/go-spatial/atlante/atlante"
)

// Status is the status of the job
type Status uint8

const (
	// ConfigKeyType is the config key for the type of the provider
	ConfigKeyType = "type"
)

const (
	// Enqueded means that the job is in the queue but has not started
	Enqueded = Status(iota)
	// Started means that the job is starting or about to start
	Started
	// Processing  the job is being worked on
	Processing
	// Compleated the job is ready
	Compleated
	// Cancelled the job was cancelled
	Cancelled
	// Unknown the job is not in the system
	Unknown
)

// Provider allows things to enqueue jobs to be done
type Provider interface {
	Enqueue(key string, job *atlante.Job) (jobid string, err error)
}

// InfoProvider can report on the progress of a job,
// the jobid is the one that is provided by the
// Enqueue function
type InfoProvider interface {
	Provider
	Info(jobid string) Status
}

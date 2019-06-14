package coordinator

import (
	"github.com/go-spatial/maptoolkit/atlante/grids"
	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante"
	"time"
)

const (
	ErrNilAtlanteJob = errors.String("nil atlante job provided")
	ErrNilJob = errors.String("nil job provided")
)


type Status int
const (
	StatusUnknown = Status(iota)
	StatusRequested 
	StatusStarted
	StatusProcessing
	StatusCompleted
)

func (s Status)String() string {
	switch s {
	case StatusRequested:
		return "requested"
	case StatusStarted:
		return "started"
	case StatusProcessing:
		return "processing"
	case StatusCompleted:
		return "completed"
	default:
		return "unknown"
	}
}

type FieldValue interface {
	field()
}

// FieldStatus is used to update the status field
type FieldStatus struct {
	Status Status
	Description string
}
func (FieldStatus) field() {}

// FieldQJobID is used to update the Queue Job ID field
type FieldQJobID string
func (FieldQJobID) field() {}

type Job struct {
	JobID string `json:"job_id"`
	// QJobID is the job id returned by the queue when
	// the item was enqueued
	QJobID    string    `json:"-"`
	MdgID     string    `json:"mdgid"`
	MdgIDPart uint32    `json:"sheet_number,omitempty"`
	Status    Status    `json:"status,omitempty"`
	StatusDesc string `json:"status_description,omitempty"`
	EnquedAt  time.Time `json:"enqued_at,omitempty"`
	UpdatedAt time.Time `json:"updated_at,omitempty"`
}

type Provider interface {

	// NewJob is expected to return a Job with the ID, JobID, MdgID, MdgIDPart, Status, EnquedAt, and UpdatedAt fields filled in.
	// If there is already a job in the system then it should return that job, otherwise it should return a new job
	NewJob(job *atlante.Job) (jb *Job, err error)

	// FindJob will look for a job described by the given atlante.Job (MDGID/SheetName) and return it, or return nil, and a false for
	// found
	FindJob(job *atlante.Job) (jb *Job, found bool )

	// FindJobID will attempt to locate the job by the given jobId, if a job is found non-nil job will be returned. If an error
	// occurs then err will be non-nil. If job is not found, the both jb and err will be nil
	FindJobID(jobid string)(jb *Job, err error)

	// UpdateField will attempt to update the job field info for the given job.
	UpdateField(job *Job, fields ...FieldValue) error
}

// NewJob is a helper function that will create a new job object with basic fields filled in.
func NewJob(jobID string, mdgid *grids.MDGID) *Job {
	t := time.Now()
	return &Job{
		JobID: jobID,
		MdgID: mdgid.Id,
		MdgIDPart: mdgid.Part,
		EnquedAt: t,
		UpdatedAt: t,
	}
}

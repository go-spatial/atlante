package coordinator

import (
	"time"

	"github.com/gdey/errors"
	"github.com/go-spatial/atlante/atlante"
	"github.com/go-spatial/atlante/atlante/grids"
	"github.com/go-spatial/atlante/atlante/server/coordinator/field"
)

const (
	ErrNilAtlanteJob = errors.String("nil atlante job provided")
	ErrNilJob        = errors.String("nil job provided")
)

type Job struct {
	JobID string `json:"job_id"`
	// QJobID is the job id returned by the queue when
	// the item was enqueued
	QJobID        string       `json:"-"`
	MdgID         string       `json:"mdgid"`
	MdgIDPart     uint32       `json:"sheet_number,omitempty"`
	SheetName     string       `json:"sheet_name"`
	StyleLocation string       `json:"style_location"`
	Status        field.Status `json:"status"`
	EnqueuedAt    time.Time    `json:"enqueued_at"`
	UpdatedAt     time.Time    `json:"updated_at"`
	AJob          *atlante.Job `json:"-"`
	PDF           string       `json:"pdf_url"`
	LastGen       string       `json:"last_generated"` // RFC 3339 format
}

type Provider interface {

	// NewJob is expected to return a Job with the ID, JobID, MdgID, MdgIDPart, Status, EnquedAt, and UpdatedAt fields filled in.
	// If there is already a job in the system then it should return that job, otherwise it should return a new job
	NewJob(job *atlante.Job) (jb *Job, err error)

	// FindJob will look for a jobs described by the given atlante.Job (MDGID/SheetName), this should return only the latest
	// two jobs.
	FindByJob(job *atlante.Job, defaultStyleLocation string) (jobs []*Job)

	// FindJobID will attempt to locate the job by the given jobId, if a job is found non-nil job will be returned and found will be true.
	// If the job is not found; a nil job will be return and found will be false.
	FindByJobID(jobid string) (jobs *Job, found bool)

	// UpdateField will attempt to update the job field info for the given job.
	UpdateField(job *Job, fields ...field.Value) error

	// Jobs returns the currently known jobs in the system. If limit is not 0
	// then the number of jobs will be limited to that number of jobs. Jobs, should
	// be sorted newest request job to oldest
	Jobs(limit uint) ([]*Job, error)
}

// NewJob is a helper function that will create a new job object with basic fields filled in.
func NewJob(jobID string, ajob *atlante.Job) *Job {
	t := time.Now()
	var mdgid grids.MDGID
	var sheetName string
	var styleLocation string

	if ajob != nil {
		if ajob.Cell.Mdgid != nil {
			mdgid = *ajob.Cell.Mdgid
		}
		sheetName = ajob.SheetName
		if ajob.MetaData != nil {
			styleLocation = ajob.MetaData["styleLocation"]
		}
	}

	return &Job{
		JobID:         jobID,
		MdgID:         mdgid.Id,
		MdgIDPart:     mdgid.Part,
		SheetName:     sheetName,
		StyleLocation: styleLocation,
		Status:        field.Status{field.Requested{}},
		EnqueuedAt:    t,
		UpdatedAt:     t,
		AJob:          ajob,
	}
}

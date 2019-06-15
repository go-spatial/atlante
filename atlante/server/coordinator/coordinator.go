package coordinator

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gdey/errors"
	"github.com/go-spatial/maptoolkit/atlante"
	"github.com/go-spatial/maptoolkit/atlante/grids"
)

const (
	ErrNilAtlanteJob = errors.String("nil atlante job provided")
	ErrNilJob        = errors.String("nil job provided")
)

type ErrInvalidStatusString string

func (err ErrInvalidStatusString) Error() string {
	return fmt.Sprintf("unknown status string: %v", string(err))
}

type Status int

const (
	StatusUnknown = Status(iota)
	StatusRequested
	StatusStarted
	StatusProcessing
	StatusCompleted
	endOfStatuses
)

func (s Status) String() string {
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

func (s *Status) MarshalJSON() ([]byte, error) {
	v := ""
	if s != nil {
		v = s.String()
	}
	return json.Marshal(v)
}

func (s *Status) UnmarshalJSON(b []byte) error {
	var val string
	if err := json.Unmarshal(b, &val); err != nil {
		return err
	}
	val = strings.TrimSpace(strings.ToLower(val))
	for i := StatusRequested; i < endOfStatuses; i++ {
		if val == i.String() {
			*s = i
			return nil
		}
	}
	*s = StatusUnknown
	return ErrInvalidStatusString(val)
}

type FieldValue interface {
	field()
}

// FieldStatus is used to update the status field
type FieldStatus struct {
	Status      Status `json:"status"`
	Description string `json:"status_description"`
}

func (FieldStatus) field() {}

// FieldQJobID is used to update the Queue Job ID field
type FieldQJobID string

func (FieldQJobID) field() {}

type Job struct {
	JobID string `json:"job_id"`
	// QJobID is the job id returned by the queue when
	// the item was enqueued
	QJobID     string    `json:"-"`
	MdgID      string    `json:"mdgid"`
	MdgIDPart  uint32    `json:"sheet_number,omitempty"`
	SheetName  string    `json:"sheet_name"`
	Status     Status    `json:"status"`
	StatusDesc string    `json:"status_description"`
	EnquedAt   time.Time `json:"enqued_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Provider interface {

	// NewJob is expected to return a Job with the ID, JobID, MdgID, MdgIDPart, Status, EnquedAt, and UpdatedAt fields filled in.
	// If there is already a job in the system then it should return that job, otherwise it should return a new job
	NewJob(job *atlante.Job) (jb *Job, err error)

	// FindJob will look for a job described by the given atlante.Job (MDGID/SheetName) and return it, or return nil, and a false for
	// found
	FindJob(job *atlante.Job) (jb *Job, found bool)

	// FindJobID will attempt to locate the job by the given jobId, if a job is found non-nil job will be returned. If an error
	// occurs then err will be non-nil. If job is not found, the both jb and err will be nil
	FindJobID(jobid string) (jb *Job, found bool)

	// UpdateField will attempt to update the job field info for the given job.
	UpdateField(job *Job, fields ...FieldValue) error
}

// NewJob is a helper function that will create a new job object with basic fields filled in.
func NewJob(jobID string, ajob *atlante.Job) *Job {
	t := time.Now()
	var mdgid grids.MDGID
	var sheetName string

	if ajob != nil {
		if ajob.Cell.Mdgid != nil {
			mdgid = *ajob.Cell.Mdgid
		}
		sheetName = ajob.SheetName
	}

	return &Job{
		JobID:     jobID,
		MdgID:     mdgid.Id,
		MdgIDPart: mdgid.Part,
		SheetName: sheetName,
		Status:    StatusRequested,
		EnquedAt:  t,
		UpdatedAt: t,
	}
}

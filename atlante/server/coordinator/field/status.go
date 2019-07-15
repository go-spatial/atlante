package field

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	requested  = "requested"
	completed  = "completed"
	started    = "started"
	processing = "processing"
	failed     = "failed"

	errorKey       = "error"
	descriptionKey = "description"
	statusKey      = "status"
)

type (

	// Status is used to hold a status for searilization
	Status struct {
		Status StatusEnum
	}

	// StatusEnum is the status reference type
	StatusEnum interface {
		fmt.Stringer

		statusenum()
	}

	// Requested is the status of the job when it is first requested
	Requested struct{}
	// Started is the status of a started job
	Started struct{}
	// Processing is the status of a job that is processing
	Processing struct {
		// Description of what processing is being done.
		Description string `json:"description"`
	}
	// Failed is the status of a job that failed
	Failed struct {
		// The Error to send to the user
		Description string `json:"description"`
		// Error as to why it failed
		Error error `json:"error"`
	}
	// Completed is the status of a successful completed job
	Completed struct{}
)

func (s Status) String() string { return s.Status.String() }
func (s Status) field()         {}

// MarshalJSON implements the json.Marshaler interface
func (s Status) MarshalJSON() ([]byte, error) {

	const (
		stageRequested  = 0
		stageStarted    = 1
		stageProcessing = 2
		stageFailed     = 3
		stageCompleted  = 3
		totalStages     = 3
	)

	if s.Status == nil {
		return json.Marshal(nil)
	}

	type sentinalEnum struct {
		Type  string `json:"status"`
		Stage int    `json:"stage"`
		Total int    `json:"total"`
	}
	type processingEnum struct {
		Type        string `json:"status"`
		Stage       int    `json:"stage"`
		Total       int    `json:"total"`
		Description string `json:"description"`
	}
	type failedEnum struct {
		Type        string `json:"status"`
		Stage       int    `json:"stage"`
		Total       int    `json:"total"`
		Description string `json:"description"`
		Error       string `json:"error"`
	}

	var jsonval interface{}
	switch senum := s.Status.(type) {
	case Started:
		jsonval = sentinalEnum{
			Type:  started,
			Stage: stageStarted,
			Total: totalStages,
		}
	case Requested:
		jsonval = sentinalEnum{
			Type:  requested,
			Stage: stageRequested,
			Total: totalStages,
		}
	case Processing:
		jsonval = processingEnum{
			Type:        processing,
			Stage:       stageProcessing,
			Total:       totalStages,
			Description: senum.Description,
		}
	case Failed:
		jsonval = failedEnum{
			Type:  failed,
			Stage: stageFailed,
			Total: totalStages,
			Error: senum.Error.Error(),
		}
	case Completed:
		jsonval = sentinalEnum{
			Type:  completed,
			Stage: stageCompleted,
			Total: totalStages,
		}
	default:
		return []byte{}, fmt.Errorf("Unknown type %t", s.Status)

	}
	return json.Marshal(jsonval)
}

// UnmarshalJSON implements the json.Marshaler interface
func (s *Status) UnmarshalJSON(b []byte) error {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(b, &obj); err != nil {
		return err
	}
	var typ string
	if err := json.Unmarshal(obj[statusKey], &typ); err != nil {
		return err
	}

	switch typ {
	case started:
		s.Status = Started{}
	case requested:
		s.Status = Requested{}
	case processing:
		var p Processing
		if err := json.Unmarshal(obj[descriptionKey], &p.Description); err != nil {
			return nil
		}
		s.Status = p
	case failed:

		var errStr string
		if err := json.Unmarshal(obj[errorKey], &errStr); err != nil {
			return nil
		}
		s.Status = Failed{
			Error: errors.New(errStr),
		}

	case completed:
		s.Status = Completed{}

	default:
		return fmt.Errorf("Unknown status type: %v", typ)

	}
	return nil
}

func NewStatusFor(status, desc string) (StatusEnum, error) {
	switch strings.ToLower(status) {
	case started:
		return Started{}, nil
	case requested:
		return Requested{}, nil
	case completed:
		return Completed{}, nil
	case processing:
		return Processing{Description: desc}, nil
	case failed:
		return Failed{Error: errors.New(desc)}, nil
	default:
		return nil, fmt.Errorf("Unknown status type: %v", status)
	}
}

func (Requested) statusenum()    {}
func (Requested) String() string { return requested }

func (Started) statusenum()    {}
func (Started) String() string { return started }

func (p Processing) statusenum()    {}
func (p Processing) String() string { return processing + ":" + p.Description }

func (f Failed) statusenum()    {}
func (f Failed) String() string { return failed + ":" + f.Error.Error() }

func (Completed) statusenum()    {}
func (Completed) String() string { return completed }

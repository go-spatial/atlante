package field

type Value interface {
	field()
}

// QJobID is used to update the Queue Job ID field
type QJobID string

func (QJobID) field() {}

type JobData string

func (JobData) field() {}

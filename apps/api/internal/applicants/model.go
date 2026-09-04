package applicants

import "time"

type Status string

const (
	StatusScreening          Status = "screening"
	StatusDocumentCollection Status = "document_collection"
	StatusSubmitted          Status = "submitted"
	StatusProcessing         Status = "processing"
	StatusApproved           Status = "approved"
	StatusRejected           Status = "rejected"
)

func (s Status) Valid() bool {
	switch s {
	case StatusScreening, StatusDocumentCollection, StatusSubmitted, StatusProcessing, StatusApproved, StatusRejected:
		return true
	}
	return false
}

type Applicant struct {
	ID          int64
	FirstName   string
	LastName    string
	Email       string
	Phone       string
	Nationality string
	VisaType    string
	Status      Status
	CreatedAt   time.Time
}

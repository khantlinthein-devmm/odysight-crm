package visacases

import "time"

type Status string

const (
	StatusDraft                  Status = "draft"
	StatusInReview               Status = "in_review"
	StatusSubmitted              Status = "submitted"
	StatusAdditionalDocsRequired Status = "additional_docs_required"
	StatusApproved               Status = "approved"
	StatusRejected               Status = "rejected"
	StatusClosed                 Status = "closed"
)

func (s Status) Valid() bool {
	switch s {
	case StatusDraft, StatusInReview, StatusSubmitted, StatusAdditionalDocsRequired,
		StatusApproved, StatusRejected, StatusClosed:
		return true
	}
	return false
}

type VisaCase struct {
	ID            int64
	CaseNumber    string
	ApplicantName string
	VisaType      string
	Destination   string
	AssignedTo    string
	Status        Status
	CreatedAt     time.Time
}

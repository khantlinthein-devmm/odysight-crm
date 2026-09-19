package attendance

import (
	"time"

	"github.com/odysight/crm/pkg/response"
)

type RecordDTO struct {
	ID         int64      `json:"id"`
	PersonType PersonType `json:"personType"`
	PersonID   int64      `json:"personId"`
	PersonName string     `json:"personName"`
	WorkDate   string     `json:"workDate"`
	CheckInAt  *time.Time `json:"checkInAt"`
	CheckOutAt *time.Time `json:"checkOutAt"`
	Note       string     `json:"note"`
	CreatedAt  time.Time  `json:"createdAt"`
}

func toDTO(r Record) RecordDTO {
	return RecordDTO{
		ID:         r.ID,
		PersonType: r.PersonType,
		PersonID:   r.PersonID,
		PersonName: r.PersonName,
		WorkDate:   r.WorkDate,
		CheckInAt:  r.CheckInAt,
		CheckOutAt: r.CheckOutAt,
		Note:       r.Note,
		CreatedAt:  r.CreatedAt,
	}
}

// CheckActionRequest is the body for POST /check-in and POST /check-out.
type CheckActionRequest struct {
	PersonType PersonType `json:"personType"`
	PersonID   int64      `json:"personId"`
}

func (r *CheckActionRequest) Validate() error {
	if r.PersonType == "" {
		r.PersonType = PersonCleaner
	}
	if !r.PersonType.Valid() {
		return response.NewAPIError(400, "personType must be cleaner or staff")
	}
	if r.PersonID < 1 {
		return response.NewAPIError(400, "personId is required")
	}
	return nil
}

func (r *CheckActionRequest) Person() Person {
	return Person{Type: r.PersonType, ID: r.PersonID}
}

package attendance

import (
	"time"

	"github.com/odysight/crm/pkg/response"
)

type RecordDTO struct {
	ID          int64      `json:"id"`
	CleanerID   int64      `json:"cleanerId"`
	CleanerName string     `json:"cleanerName"`
	WorkDate    string     `json:"workDate"`
	CheckInAt   *time.Time `json:"checkInAt"`
	CheckOutAt  *time.Time `json:"checkOutAt"`
	Note        string     `json:"note"`
	CreatedAt   time.Time  `json:"createdAt"`
}

func toDTO(r Record) RecordDTO {
	return RecordDTO{
		ID:          r.ID,
		CleanerID:   r.CleanerID,
		CleanerName: r.CleanerName,
		WorkDate:    r.WorkDate,
		CheckInAt:   r.CheckInAt,
		CheckOutAt:  r.CheckOutAt,
		Note:        r.Note,
		CreatedAt:   r.CreatedAt,
	}
}

// CheckActionRequest is the body for POST /check-in and POST /check-out.
type CheckActionRequest struct {
	CleanerID int64 `json:"cleanerId"`
}

func (r *CheckActionRequest) Validate() error {
	if r.CleanerID < 1 {
		return response.NewAPIError(400, "cleanerId is required")
	}
	return nil
}

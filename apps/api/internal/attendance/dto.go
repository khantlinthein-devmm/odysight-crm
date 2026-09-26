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
	CheckInLat       *float64 `json:"checkInLat"`
	CheckInLng       *float64 `json:"checkInLng"`
	CheckOutLat      *float64 `json:"checkOutLat"`
	CheckOutLng      *float64 `json:"checkOutLng"`
	CheckInDistanceM *int     `json:"checkInDistanceM"`
	CheckInSiteName  string   `json:"checkInSiteName"`
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
		CheckInLat:       r.CheckInLat,
		CheckInLng:       r.CheckInLng,
		CheckOutLat:      r.CheckOutLat,
		CheckOutLng:      r.CheckOutLng,
		CheckInDistanceM: r.CheckInDistanceM,
		CheckInSiteName:  r.CheckInSiteName,
	}
}

// CheckActionRequest is the body for POST /check-in and POST /check-out.
type CheckActionRequest struct {
	PersonType PersonType `json:"personType"`
	PersonID   int64      `json:"personId"`
	// Optional phone position; required for cleaner self check-in when
	// geofencing is on and the day's sites have coordinates.
	Latitude  *float64 `json:"latitude"`
	Longitude *float64 `json:"longitude"`
	Accuracy  *float64 `json:"accuracy"`
}

// Location returns the reported position, or nil when none was sent.
func (r *CheckActionRequest) Location() *Location {
	if r.Latitude == nil || r.Longitude == nil {
		return nil
	}
	loc := &Location{Lat: *r.Latitude, Lng: *r.Longitude}
	if r.Accuracy != nil {
		loc.Accuracy = *r.Accuracy
	}
	return loc
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
	if (r.Latitude == nil) != (r.Longitude == nil) {
		return response.NewAPIError(400, "latitude and longitude must be sent together")
	}
	if r.Latitude != nil && (*r.Latitude < -90 || *r.Latitude > 90 || *r.Longitude < -180 || *r.Longitude > 180) {
		return response.NewAPIError(400, "latitude/longitude out of range")
	}
	return nil
}

func (r *CheckActionRequest) Person() Person {
	return Person{Type: r.PersonType, ID: r.PersonID}
}

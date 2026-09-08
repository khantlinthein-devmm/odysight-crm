package feedback

import (
	"strings"
	"time"

	"github.com/odysight/crm/pkg/response"
)

type FeedbackDTO struct {
	ID            int64   `json:"id"`
	BookingID     int64   `json:"bookingId"`
	BookingNumber string  `json:"bookingNumber"`
	CustomerID    *int64  `json:"customerId"`
	CustomerName  string  `json:"customerName"`
	Rating        int     `json:"rating"`
	Comment       string  `json:"comment"`
	CreatedAt     time.Time `json:"createdAt"`
}

func toDTO(f Feedback) FeedbackDTO {
	return FeedbackDTO{
		ID:            f.ID,
		BookingID:     f.BookingID,
		BookingNumber: f.BookingNumber,
		CustomerID:    f.CustomerID,
		CustomerName:  f.CustomerName,
		Rating:        f.Rating,
		Comment:       f.Comment,
		CreatedAt:     f.CreatedAt,
	}
}

type CreateFeedbackRequest struct {
	BookingID int64  `json:"bookingId"`
	Rating    int    `json:"rating"`
	Comment   string `json:"comment"`
}

func (r *CreateFeedbackRequest) Validate() error {
	r.Comment = strings.TrimSpace(r.Comment)
	if r.BookingID <= 0 {
		return response.NewAPIError(400, "bookingId is required")
	}
	if r.Rating < 1 || r.Rating > 5 {
		return response.NewAPIError(400, "rating must be between 1 and 5")
	}
	return nil
}

type UpdateFeedbackRequest struct {
	Rating  *int    `json:"rating"`
	Comment *string `json:"comment"`
}

func (r *UpdateFeedbackRequest) Validate() error {
	if r.Rating != nil && (*r.Rating < 1 || *r.Rating > 5) {
		return response.NewAPIError(400, "rating must be between 1 and 5")
	}
	if r.Comment != nil {
		v := strings.TrimSpace(*r.Comment)
		r.Comment = &v
	}
	return nil
}

func (r *UpdateFeedbackRequest) IsEmpty() bool {
	return r.Rating == nil && r.Comment == nil
}
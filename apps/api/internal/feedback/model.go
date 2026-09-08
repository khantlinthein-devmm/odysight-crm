package feedback

import "time"

type Feedback struct {
	ID          int64
	BookingID   int64
	BookingNumber string
	CustomerID  *int64
	CustomerName string
	Rating      int
	Comment     string
	CreatedAt   time.Time
}